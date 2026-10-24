package infer

import (
	"strings"
	"unicode"
)

// Step 3: string rules for column names and values.

// Common abbreviations in column names
var nameAbbrev = map[string]string{
	"qty": "quantity", "uom": "unit of measure", "desc": "description", "descr": "description",
	"prod": "product", "cat": "category", "loc": "location", "amt": "amount", "num": "number",
	"no": "number", "nbr": "number", "addr": "address", "dt": "date", "img": "image",
	"inv": "inventory", "srp": "retail price", "msrp": "retail price", "wh": "warehouse",
	"whse": "warehouse", "stat": "status", "cd": "code", "nm": "name", "pct": "percent",
	"avail": "available", "ref": "reference", "org": "organization", "dept": "department",
	"emp": "employee", "cust": "customer", "acct": "account", "tel": "telephone",
	"supp": "supplier", "vend": "vendor", "mfr": "manufacturer", "mfg": "manufacturer",
}

// Common abbreviations in values: units, stock statuses and permission letters.
// Used only when the raw value matches nothing.
var valueAbbrev = map[string]string{
	"ea": "each", "pc": "piece", "pcs": "pieces", "pk": "pack", "pkg": "package", "bx": "box",
	"btl": "bottle", "kg": "kilogram", "g": "gram", "gr": "gram", "mg": "milligram",
	"lb": "pound", "lbs": "pounds", "oz": "ounce", "l": "liter", "ltr": "liter",
	"ml": "milliliter", "m": "meter", "cm": "centimeter", "mm": "millimeter", "dz": "dozen",
	"doz": "dozen", "rm": "ream", "ct": "count", "pr": "pair", "rl": "roll", "bg": "bag",
	"cs": "case", "avail": "available", "unavail": "unavailable", "disc": "discontinued",
	"oos": "out of stock", "inact": "inactive",
	"r": "read", "w": "write", "x": "execute", "d": "delete", "c": "create", "u": "update",
}

var (
	truthy = map[string]bool{"true": true, "yes": true, "y": true, "t": true, "1": true, "active": true, "enabled": true, "on": true}
	falsy  = map[string]bool{"false": true, "no": true, "n": true, "f": true, "0": true, "inactive": true, "disabled": true, "off": true}
)

// nameTokens splits snake_case, kebab-case, camelCase and spaced names into
// lowercase words.
func nameTokens(name string) []string {
	var tokens []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			tokens = append(tokens, strings.ToLower(string(current)))
			current = nil
		}
	}

	runes := []rune(name)
	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		if i > 0 && unicode.IsUpper(r) && unicode.IsLower(runes[i-1]) {
			flush()
		}
		current = append(current, r)
	}
	flush()

	return tokens
}

// expandName returns the name's words with abbreviations spelled out.
func expandName(name string) []string {
	var words []string
	for _, token := range nameTokens(name) {
		if full, ok := nameAbbrev[token]; ok {
			words = append(words, strings.Fields(full)...)
		} else {
			words = append(words, token)
		}
	}
	return words
}

func humanName(name string) string {
	return strings.Join(expandName(name), " ")
}

// nameSimilarity is the share of words (0..1) that the two names have in
// common, counting "stock"/"stocks" style prefixes as a match.
func nameSimilarity(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	matched := func(x, y []string) int {
		n := 0
		for _, p := range x {
			for _, q := range y {
				if wordsMatch(p, q) {
					n++
					break
				}
			}
		}
		return n
	}
	return float64(matched(a, b)+matched(b, a)) / float64(len(a)+len(b))
}

func wordsMatch(a, b string) bool {
	if a == b {
		return true
	}
	return len(a) >= 3 && len(b) >= 3 && (strings.HasPrefix(a, b) || strings.HasPrefix(b, a))
}

// humanize lowercases a value and turns "in_stock" / "in-stock" into "in stock".
func humanize(v string) string {
	v = strings.NewReplacer("_", " ", "-", " ").Replace(strings.ToLower(v))
	return strings.Join(strings.Fields(v), " ")
}

// expandValue spells out abbreviated words ("pcs" -> "pieces"), or returns
// "" when there is nothing to expand.
func expandValue(v string) string {
	words := strings.Fields(humanize(v))
	changed := false
	for i, w := range words {
		if full, ok := valueAbbrev[w]; ok {
			words[i] = full
			changed = true
		}
	}
	if !changed {
		return ""
	}
	return strings.Join(words, " ")
}

type valueRule struct {
	name  string
	match func(src, cand string) bool
}

// Ordered from strictest to loosest
var valueRules = []valueRule{
	{"same value", func(s, c string) bool { return humanize(s) == humanize(c) }},
	{"yes/no word", func(s, c string) bool {
		ls, lc := strings.ToLower(s), strings.ToLower(c)
		return (truthy[ls] && truthy[lc]) || (falsy[ls] && falsy[lc])
	}},
	{"prefix", func(s, c string) bool {
		return wordPrefix(humanize(s), humanize(c)) || wordPrefix(humanize(c), humanize(s))
	}},
	{"initials", func(s, c string) bool { return initialsMatch(s, c) || initialsMatch(c, s) }},
	{"abbreviation", func(s, c string) bool { return isAbbreviation(humanize(s), humanize(c)) }},
}

// ruleMatch returns the single candidate matched by the strictest rule that
// matches anything. It returns "" when nothing matches or the rule matches
// several candidates, so it abstains instead of guessing.
func ruleMatch(value string, cands []string) (match, rule string) {
	for _, r := range valueRules {
		var hits []string
		for _, c := range cands {
			if r.match(value, c) {
				hits = append(hits, c)
			}
		}
		if len(hits) == 1 {
			return hits[0], r.name
		}
		if len(hits) > 1 {
			return "", ""
		}
	}
	return "", ""
}

// matchByRules tries the raw value, then its expanded abbreviation.
func matchByRules(value string, cands []string) (match, how string) {
	if m, rule := ruleMatch(value, cands); m != "" {
		return m, "rule: " + rule
	}
	if expanded := expandValue(value); expanded != "" {
		if m, rule := ruleMatch(expanded, cands); m != "" {
			return m, "rule: " + value + " = " + expanded + ", " + rule
		}
	}
	return "", ""
}

// wordPrefix reports whether a single word (short) starts some word of long:
// "low" -> "low stock", "elec" -> "electronics".
func wordPrefix(short, long string) bool {
	if short == "" || short == long || strings.Contains(short, " ") {
		return false
	}
	for _, w := range strings.Fields(long) {
		if strings.HasPrefix(w, short) {
			return true
		}
	}
	return false
}

// initialsMatch: "F&B" -> "Food & Beverage", "OOS" -> "out of stock".
func initialsMatch(short, long string) bool {
	letters := strings.Join(alnumWords(strings.ToLower(short)), "")
	words := alnumWords(strings.ToLower(long))
	if len(letters) < 2 || len(words) < 2 {
		return false
	}
	var initials strings.Builder
	for _, w := range words {
		initials.WriteString(w[:1])
	}
	return initials.String() == letters
}

// isAbbreviation: "bx" -> "box", "btl" -> "bottle" (same first letter, letters in order).
func isAbbreviation(short, long string) bool {
	long = strings.ReplaceAll(long, " ", "")
	if len(short) < 2 || len(short) >= len(long) || strings.Contains(short, " ") || short[0] != long[0] {
		return false
	}
	i := 0
	for j := 0; j < len(long) && i < len(short); j++ {
		if long[j] == short[i] {
			i++
		}
	}
	return i == len(short)
}

func alnumWords(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// shape reduces a value to its format: "WH01" -> "a9", "65.00" -> "9.9".
func shape(v string) string {
	var b strings.Builder
	var last rune
	for _, r := range v {
		c := r
		switch {
		case unicode.IsLetter(r):
			c = 'a'
		case unicode.IsDigit(r):
			c = '9'
		}
		if c != last || (c != 'a' && c != '9') {
			b.WriteRune(c)
		}
		last = c
	}
	return b.String()
}

func jaccard(a, b []string, key func(string) string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	setA := make(map[string]bool)
	for _, v := range a {
		setA[key(v)] = true
	}
	setB := make(map[string]bool)
	for _, v := range b {
		setB[key(v)] = true
	}
	common := 0
	for k := range setA {
		if setB[k] {
			common++
		}
	}
	return float64(common) / float64(len(setA)+len(setB)-common)
}
