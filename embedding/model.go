// Package embedding runs a small BERT sentence-embedding model (such as
// all-MiniLM-L6-v2) in pure Go: no CGo, no native libraries, no network.
// Unlike a generative model, it can only turn text into vectors, so text
// inside the CSV cannot instruct it to do anything.
package embedding

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
)

const (
	maxTokens       = 128
	maxCacheEntries = 20000
)

type config struct {
	Hidden int     `json:"hidden_size"`
	Heads  int     `json:"num_attention_heads"`
	Layers int     `json:"num_hidden_layers"`
	Inter  int     `json:"intermediate_size"`
	Eps    float64 `json:"layer_norm_eps"`
	MaxPos int     `json:"max_position_embeddings"`
}

type linear struct {
	w       []float32 // [out][in], row-major
	b       []float32
	in, out int
}

type layerNorm struct {
	gamma, beta []float32
}

type encoderLayer struct {
	query, key, value, attnOut linear
	attnNorm                   layerNorm
	inter, out                 linear
	outNorm                    layerNorm
}

type Model struct {
	cfg     config
	tok     *tokenizer
	wordEmb []float32
	posEmb  []float32
	typeEmb []float32
	embNorm layerNorm
	layers  []encoderLayer

	mu    sync.Mutex
	cache map[string][]float32
}

// Load reads config.json, vocab.txt and model.safetensors from dir.
func Load(dir string) (*Model, error) {
	var cfg config
	raw, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("config.json: %v", err)
	}
	if cfg.Hidden == 0 || cfg.Heads == 0 || cfg.Hidden%cfg.Heads != 0 {
		return nil, fmt.Errorf("config.json: unsupported hidden_size/num_attention_heads")
	}

	tok, err := loadVocab(filepath.Join(dir, "vocab.txt"))
	if err != nil {
		return nil, err
	}

	tensors, err := readSafetensors(filepath.Join(dir, "model.safetensors"))
	if err != nil {
		return nil, err
	}

	var loadErr error
	get := func(name string, shape ...int) []float32 {
		t, ok := tensors[name]
		if loadErr == nil && (!ok || !slices.Equal(t.Shape, shape)) {
			loadErr = fmt.Errorf("model.safetensors: tensor %s missing or not shaped %v", name, shape)
		}
		return t.Data
	}
	lin := func(prefix string, out, in int) linear {
		return linear{w: get(prefix+".weight", out, in), b: get(prefix+".bias", out), in: in, out: out}
	}
	norm := func(prefix string) layerNorm {
		return layerNorm{gamma: get(prefix+".weight", cfg.Hidden), beta: get(prefix+".bias", cfg.Hidden)}
	}

	h := cfg.Hidden
	m := &Model{
		cfg:     cfg,
		tok:     tok,
		wordEmb: get("embeddings.word_embeddings.weight", len(tok.vocab), h),
		posEmb:  get("embeddings.position_embeddings.weight", cfg.MaxPos, h),
		typeEmb: get("embeddings.token_type_embeddings.weight", 2, h),
		embNorm: norm("embeddings.LayerNorm"),
		cache:   make(map[string][]float32),
	}
	for i := range cfg.Layers {
		p := fmt.Sprintf("encoder.layer.%d.", i)
		m.layers = append(m.layers, encoderLayer{
			query:    lin(p+"attention.self.query", h, h),
			key:      lin(p+"attention.self.key", h, h),
			value:    lin(p+"attention.self.value", h, h),
			attnOut:  lin(p+"attention.output.dense", h, h),
			attnNorm: norm(p + "attention.output.LayerNorm"),
			inter:    lin(p+"intermediate.dense", cfg.Inter, h),
			out:      lin(p+"output.dense", h, cfg.Inter),
			outNorm:  norm(p + "output.LayerNorm"),
		})
	}
	if loadErr != nil {
		return nil, loadErr
	}

	return m, nil
}

// Embed returns the L2-normalized, mean-pooled sentence embedding of text.
func (m *Model) Embed(text string) []float32 {
	m.mu.Lock()
	cached, ok := m.cache[text]
	m.mu.Unlock()
	if ok {
		return cached
	}

	vec := m.forward(m.tok.encode(text, min(maxTokens, m.cfg.MaxPos)))

	m.mu.Lock()
	// Keep memory bounded in a long-running server
	if len(m.cache) >= maxCacheEntries {
		clear(m.cache)
	}
	m.cache[text] = vec
	m.mu.Unlock()
	return vec
}

// EmbedAll embeds texts in parallel across CPU cores.
func (m *Model) EmbedAll(texts []string) [][]float32 {
	out := make([][]float32, len(texts))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range runtime.NumCPU() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				out[i] = m.Embed(texts[i])
			}
		}()
	}
	for i := range texts {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return out
}

// Cosine is the cosine similarity of two normalized embeddings.
func Cosine(a, b []float32) float64 {
	var dot float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}
	return dot
}

func (m *Model) forward(ids []int) []float32 {
	n, h := len(ids), m.cfg.Hidden

	x := make([]float32, n*h)
	for i, id := range ids {
		row := x[i*h : (i+1)*h]
		for j := range row {
			row[j] = m.wordEmb[id*h+j] + m.posEmb[i*h+j] + m.typeEmb[j]
		}
	}
	m.embNorm.apply(x, n, m.cfg.Eps)

	for _, l := range m.layers {
		ctx := m.attention(l.query.apply(x, n), l.key.apply(x, n), l.value.apply(x, n), n)

		attn := l.attnOut.apply(ctx, n)
		addInPlace(attn, x)
		l.attnNorm.apply(attn, n, m.cfg.Eps)

		inter := l.inter.apply(attn, n)
		for i, v := range inter {
			inter[i] = float32(0.5 * float64(v) * (1 + math.Erf(float64(v)/math.Sqrt2)))
		}

		out := l.out.apply(inter, n)
		addInPlace(out, attn)
		l.outNorm.apply(out, n, m.cfg.Eps)
		x = out
	}

	// Mean pooling over all tokens, then L2 normalization
	pooled := make([]float32, h)
	for i := range n {
		for j := range h {
			pooled[j] += x[i*h+j]
		}
	}
	var norm float64
	for j := range pooled {
		pooled[j] /= float32(n)
		norm += float64(pooled[j]) * float64(pooled[j])
	}
	norm = math.Max(math.Sqrt(norm), 1e-12)
	for j := range pooled {
		pooled[j] = float32(float64(pooled[j]) / norm)
	}

	return pooled
}

func (m *Model) attention(q, k, v []float32, n int) []float32 {
	h, heads := m.cfg.Hidden, m.cfg.Heads
	hd := h / heads
	scale := float32(1 / math.Sqrt(float64(hd)))

	ctx := make([]float32, n*h)
	scores := make([]float32, n)
	for head := range heads {
		off := head * hd
		for i := range n {
			qi := q[i*h+off : i*h+off+hd]
			maxScore := float32(math.Inf(-1))
			for j := range n {
				kj := k[j*h+off : j*h+off+hd]
				var s float32
				for d := range hd {
					s += qi[d] * kj[d]
				}
				scores[j] = s * scale
				maxScore = max(maxScore, scores[j])
			}
			var sum float32
			for j := range n {
				scores[j] = float32(math.Exp(float64(scores[j] - maxScore)))
				sum += scores[j]
			}
			ci := ctx[i*h+off : i*h+off+hd]
			for j := range n {
				p := scores[j] / sum
				vj := v[j*h+off : j*h+off+hd]
				for d := range hd {
					ci[d] += p * vj[d]
				}
			}
		}
	}

	return ctx
}

func (l linear) apply(x []float32, n int) []float32 {
	y := make([]float32, n*l.out)
	for i := range n {
		xi := x[i*l.in : (i+1)*l.in]
		yi := y[i*l.out : (i+1)*l.out]
		for o := range l.out {
			w := l.w[o*l.in : (o+1)*l.in]
			s := l.b[o]
			for j, xv := range xi {
				s += xv * w[j]
			}
			yi[o] = s
		}
	}
	return y
}

func (ln layerNorm) apply(x []float32, n int, eps float64) {
	h := len(ln.gamma)
	for i := range n {
		row := x[i*h : (i+1)*h]
		var mean, variance float64
		for _, v := range row {
			mean += float64(v)
		}
		mean /= float64(h)
		for _, v := range row {
			d := float64(v) - mean
			variance += d * d
		}
		inv := 1 / math.Sqrt(variance/float64(h)+eps)
		for j, v := range row {
			row[j] = float32((float64(v)-mean)*inv)*ln.gamma[j] + ln.beta[j]
		}
	}
}

func addInPlace(dst, src []float32) {
	for i := range dst {
		dst[i] += src[i]
	}
}
