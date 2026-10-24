package infer

import (
	"fmt"
	"sort"
	"strings"

	utils "github.com/elrizwiraswara/godwit/utils"
)

// Mapping categorical values: shared records first, then string rules, then
// the embedding model (step 4), then elimination.

const (
	howRecords = "shared records"
	howLeft    = "only value left"

	// An embedding match must be similar enough and clearly ahead of the
	// runner-up; otherwise the value stays unresolved for a person to decide.
	minEmbedSim    = 0.35
	minEmbedMargin = 0.08
)

type valueMapping struct {
	to         map[string]string
	how        map[string]string
	tokens     map[string]string // per-part mapping for grouped values
	sep        string            // separator for joining mapped parts
	unresolved []string
}

func (vm *valueMapping) set(v, t, how string) {
	vm.to[v], vm.how[v] = t, how
}

func mapValues(src, tgt []string, learned map[string]string, emb Embedder) valueMapping {
	vm := valueMapping{to: make(map[string]string), how: make(map[string]string)}

	if sep := compositeSeparator(src, tgt); sep != "" {
		r := resolve(partsOf(src), partsOf(tgt), learnParts(learned), emb)
		if len(r.to) > 0 {
			vm.tokens, vm.sep = r.to, sep
		}
		for _, v := range src {
			if t, ok := learned[v]; ok {
				vm.set(v, t, howRecords)
				continue
			}
			parts := utils.SplitComposite(v)
			mapped := make([]string, len(parts))
			hows := make(map[string]bool)
			complete := true
			for i, p := range parts {
				m, ok := r.to[p]
				if !ok {
					complete = false
					break
				}
				mapped[i] = m
				hows[r.how[p]] = true
			}
			if complete {
				vm.set(v, strings.Join(mapped, sep), "parts by "+strings.Join(sortedKeys(hows), " + "))
			} else {
				vm.unresolved = append(vm.unresolved, v)
			}
		}
		return vm
	}

	r := resolve(src, tgt, learned, emb)
	for _, v := range src {
		if t, ok := r.to[v]; ok {
			vm.set(v, t, r.how[v])
		} else {
			vm.unresolved = append(vm.unresolved, v)
		}
	}
	return vm
}

// ruleCoverage is the share of source values that string rules alone can map.
func ruleCoverage(src, tgt []string) float64 {
	if len(src) == 0 {
		return 0
	}
	vm := mapValues(src, tgt, nil, nil)
	n := 0
	for _, how := range vm.how {
		if !strings.Contains(how, howLeft) {
			n++
		}
	}
	return float64(n) / float64(len(src))
}

func (vm valueMapping) summary() string {
	fromRecords, byRules := 0, 0
	var check []string
	for _, v := range sortedKeys(vm.how) {
		how := vm.how[v]
		switch {
		case how == howRecords:
			fromRecords++
		case strings.Contains(how, "meaning") || strings.Contains(how, howLeft):
			check = append(check, fmt.Sprintf("%s→%s (%s)", v, vm.to[v], how))
		default:
			byRules++
		}
	}

	var parts []string
	if fromRecords > 0 {
		parts = append(parts, fmt.Sprintf("%d from shared records", fromRecords))
	}
	if byRules > 0 {
		parts = append(parts, fmt.Sprintf("%d by rules", byRules))
	}
	summary := "values: " + strings.Join(parts, ", ")
	if len(parts) == 0 {
		summary = "values: none mapped with confidence"
	}
	if len(check) > 0 {
		summary += "; CHECK: " + strings.Join(check, ", ")
	}
	if len(vm.unresolved) > 0 {
		summary += "; UNRESOLVED: " + strings.Join(vm.unresolved, ", ")
	}
	return summary
}

type resolution struct {
	to, how map[string]string
}

// resolve maps each source value to one target value, or leaves it out.
func resolve(src, tgt []string, known map[string]string, emb Embedder) resolution {
	r := resolution{to: make(map[string]string), how: make(map[string]string)}
	claimed := make(map[string]bool)
	// With no more source values than target values, assume each target
	// value is used once, which narrows the choices for the rest
	exclusive := len(src) <= len(tgt)

	assign := func(s, t, how string) {
		r.to[s], r.how[s] = t, how
		claimed[t] = true
	}
	open := func() []string {
		if !exclusive {
			return tgt
		}
		var out []string
		for _, t := range tgt {
			if !claimed[t] {
				out = append(out, t)
			}
		}
		return out
	}
	pending := func() []string {
		var out []string
		for _, s := range src {
			if _, ok := r.to[s]; !ok {
				out = append(out, s)
			}
		}
		return out
	}

	for _, s := range src {
		if t, ok := known[s]; ok {
			assign(s, t, howRecords)
		}
	}

	for progress := true; progress; {
		progress = false
		for _, s := range pending() {
			if t, how := matchByRules(s, open()); t != "" {
				assign(s, t, how)
				progress = true
			}
		}
	}

	if emb != nil && len(pending()) > 0 {
		vec := embedValues(emb, src, tgt)
		for {
			cands := open()
			bestS, bestT, bestSim, bestLead := "", "", 0.0, -1.0
			for _, s := range pending() {
				first, second, firstT := -1.0, -1.0, ""
				for _, t := range cands {
					sim := cosine(vec["s:"+s], vec["t:"+t])
					if sim > first {
						first, second, firstT = sim, first, t
					} else if sim > second {
						second = sim
					}
				}
				lead := first - second
				if first >= minEmbedSim && lead >= minEmbedMargin && lead > bestLead {
					bestS, bestT, bestSim, bestLead = s, firstT, first, lead
				}
			}
			if bestS == "" {
				break
			}
			assign(bestS, bestT, fmt.Sprintf("meaning %.2f", bestSim))
		}
	}

	if p, o := pending(), open(); exclusive && len(p) == 1 && len(o) == 1 {
		assign(p[0], o[0], howLeft)
	}

	return r
}

func embedValues(emb Embedder, src, tgt []string) map[string][]float32 {
	var keys, texts []string
	for _, s := range src {
		text := expandValue(s)
		if text == "" {
			text = humanize(s)
		}
		keys, texts = append(keys, "s:"+s), append(texts, text)
	}
	for _, t := range tgt {
		keys, texts = append(keys, "t:"+t), append(texts, humanize(t))
	}

	vecs := emb.EmbedAll(texts)
	out := make(map[string][]float32, len(keys))
	for i, k := range keys {
		out[k] = vecs[i]
	}
	return out
}

// compositeSeparator returns the target's separator (such as ", ") when both
// columns hold grouped values like "R+W+D" and "view, edit, delete".
func compositeSeparator(src, tgt []string) string {
	hasGroups := func(values []string) bool {
		for _, v := range values {
			if len(utils.SplitComposite(v)) > 1 {
				return true
			}
		}
		return false
	}
	if !hasGroups(src) || !hasGroups(tgt) {
		return ""
	}
	for _, v := range tgt {
		if sep := utils.CompositeSeparator.FindString(v); sep != "" {
			return sep
		}
	}
	return ""
}

func partsOf(values []string) []string {
	seen := make(map[string]bool)
	var parts []string
	for _, v := range values {
		for _, p := range utils.SplitComposite(v) {
			if p != "" && !seen[p] {
				seen[p] = true
				parts = append(parts, p)
			}
		}
	}
	return parts
}

func cosine(a, b []float32) float64 {
	var dot float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}
	return dot
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
