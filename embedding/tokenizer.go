package embedding

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"
)

// tokenizer is a BERT WordPiece tokenizer (uncased), matching the
// Hugging Face BertNormalizer + BertPreTokenizer + WordPiece pipeline.
type tokenizer struct {
	vocab         map[string]int
	unk, cls, sep int
}

const maxCharsPerWord = 100

func loadVocab(path string) (*tokenizer, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	vocab := make(map[string]int)
	scanner := bufio.NewScanner(file)
	for id := 0; scanner.Scan(); id++ {
		vocab[strings.TrimRight(scanner.Text(), "\r")] = id
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	t := &tokenizer{vocab: vocab}
	for token, dst := range map[string]*int{"[UNK]": &t.unk, "[CLS]": &t.cls, "[SEP]": &t.sep} {
		id, ok := vocab[token]
		if !ok {
			return nil, fmt.Errorf("%s: missing special token %s", path, token)
		}
		*dst = id
	}

	return t, nil
}

// encode returns [CLS] tokens... [SEP], truncated to maxLen ids.
func (t *tokenizer) encode(text string, maxLen int) []int {
	ids := []int{t.cls}
	for _, word := range basicTokenize(text) {
		ids = append(ids, t.wordpiece(word)...)
	}
	if len(ids) > maxLen-1 {
		ids = ids[:maxLen-1]
	}
	return append(ids, t.sep)
}

// wordpiece splits a word greedily into the longest vocabulary pieces.
func (t *tokenizer) wordpiece(word string) []int {
	runes := []rune(word)
	if len(runes) > maxCharsPerWord {
		return []int{t.unk}
	}

	var ids []int
	for start := 0; start < len(runes); {
		id := -1
		end := len(runes)
		for ; end > start; end-- {
			piece := string(runes[start:end])
			if start > 0 {
				piece = "##" + piece
			}
			if v, ok := t.vocab[piece]; ok {
				id = v
				break
			}
		}
		if id < 0 {
			return []int{t.unk}
		}
		ids = append(ids, id)
		start = end
	}

	return ids
}

func basicTokenize(text string) []string {
	var cleaned strings.Builder
	for _, r := range text {
		switch {
		case r == 0 || r == 0xfffd || isControl(r):
			continue
		case isWhitespace(r):
			cleaned.WriteRune(' ')
		case isCJK(r):
			cleaned.WriteRune(' ')
			cleaned.WriteRune(r)
			cleaned.WriteRune(' ')
		default:
			cleaned.WriteRune(r)
		}
	}

	var tokens []string
	for _, word := range strings.Fields(cleaned.String()) {
		var current []rune
		for _, r := range stripAccents(strings.ToLower(word)) {
			if isPunct(r) {
				if len(current) > 0 {
					tokens = append(tokens, string(current))
					current = nil
				}
				tokens = append(tokens, string(r))
				continue
			}
			current = append(current, r)
		}
		if len(current) > 0 {
			tokens = append(tokens, string(current))
		}
	}

	return tokens
}

// accentFold covers precomposed Latin letters that decompose canonically
// (what NFD + dropping combining marks would produce), without pulling in
// golang.org/x/text.
var accentFold = func() map[rune]rune {
	fold := make(map[rune]rune)
	for base, variants := range map[rune]string{
		'a': "àáâãäåāăą", 'c': "çćĉċč", 'd': "ď", 'e': "èéêëēĕėęě",
		'g': "ĝğġģ", 'h': "ĥ", 'i': "ìíîïĩīĭį", 'j': "ĵ", 'k': "ķ",
		'l': "ĺļľ", 'n': "ñńņň", 'o': "òóôõöōŏő", 'r': "ŕŗř",
		's': "śŝşš", 't': "ţť", 'u': "ùúûüũūŭůűų", 'w': "ŵ",
		'y': "ýÿŷ", 'z': "źżž",
	} {
		for _, v := range variants {
			fold[v] = base
		}
	}
	return fold
}()

func stripAccents(s string) string {
	var out strings.Builder
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if base, ok := accentFold[r]; ok {
			r = base
		}
		out.WriteRune(r)
	}
	return out.String()
}

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || unicode.Is(unicode.Zs, r)
}

func isControl(r rune) bool {
	if r == '\t' || r == '\n' || r == '\r' {
		return false
	}
	return unicode.In(r, unicode.Cc, unicode.Cf)
}

func isPunct(r rune) bool {
	if (r >= 33 && r <= 47) || (r >= 58 && r <= 64) || (r >= 91 && r <= 96) || (r >= 123 && r <= 126) {
		return true
	}
	return unicode.IsPunct(r)
}

func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x20000 && r <= 0x2A6DF) || (r >= 0x2A700 && r <= 0x2B73F) ||
		(r >= 0x2B740 && r <= 0x2B81F) || (r >= 0x2B820 && r <= 0x2CEAF) ||
		(r >= 0xF900 && r <= 0xFAFF) || (r >= 0x2F800 && r <= 0x2FA1F)
}
