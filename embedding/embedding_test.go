package embedding

import (
	"math"
	"os"
	"slices"
	"testing"
)

const modelDir = "../models/all-MiniLM-L6-v2"

func loadModel(t *testing.T) *Model {
	m, err := Load(modelDir)
	if os.IsNotExist(err) {
		t.Skip("embedding model not present")
	}
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestHalfFloat(t *testing.T) {
	exact := []struct {
		f float32
		h uint16
	}{
		{0, 0x0000}, {1, 0x3c00}, {-2, 0xc000}, {0.5, 0x3800},
		{65504, 0x7bff},         // largest half
		{6.1035156e-05, 0x0400}, // smallest normal
		{5.9604645e-08, 0x0001}, // smallest subnormal
		{float32(math.Inf(1)), 0x7c00},
	}
	for _, c := range exact {
		if got := floatToHalf(c.f); got != c.h {
			t.Errorf("floatToHalf(%g) = %#04x, want %#04x", c.f, got, c.h)
		}
		if got := halfToFloat(c.h); got != c.f {
			t.Errorf("halfToFloat(%#04x) = %g, want %g", c.h, got, c.f)
		}
	}
	if got := floatToHalf(1e6); got != 0x7c00 {
		t.Errorf("overflow should give +Inf, got %#04x", got)
	}
	for _, f := range []float32{0.1, -3.14159, 123.456, 1e-3} {
		if got := halfToFloat(floatToHalf(f)); math.Abs(float64(got-f)) > math.Abs(float64(f))*1e-3 {
			t.Errorf("round trip of %g gave %g", f, got)
		}
	}
}

// Token ids produced by the Hugging Face tokenizer for all-MiniLM-L6-v2
func TestTokenizerMatchesHuggingFace(t *testing.T) {
	m := loadModel(t)
	for text, want := range map[string][]int{
		"Food & Beverage":   {101, 2833, 1004, 19645, 102},
		"R+W+D":             {101, 1054, 1009, 1059, 1009, 1040, 102},
		"Café Crème brûlée": {101, 7668, 13675, 21382, 7987, 9307, 2063, 102},
		"in_stock":          {101, 1999, 1035, 4518, 102},
	} {
		if got := m.tok.encode(text, maxTokens); !slices.Equal(got, want) {
			t.Errorf("encode(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestSimilarityOrdering(t *testing.T) {
	m := loadModel(t)
	vecs := m.EmbedAll([]string{"supplier name", "vendor company", "unit price"})
	related, unrelated := Cosine(vecs[0], vecs[1]), Cosine(vecs[0], vecs[2])
	if related <= unrelated {
		t.Errorf("supplier/vendor similarity %.2f should beat supplier/price %.2f", related, unrelated)
	}
}
