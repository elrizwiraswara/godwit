package infer

import (
	"regexp"
	"strconv"
	"strings"
)

// Step 1: profile every column.

const maxCategories = 30

const (
	kindEmpty   = "empty"
	kindBool    = "boolean"
	kindInt     = "integer"
	kindDecimal = "decimal"
	kindDate    = "date"
	kindEmail   = "email"
	kindURL     = "url"
	kindText    = "text"
)

var (
	intRe   = regexp.MustCompile(`^[+-]?\d+$`)
	dateRe  = regexp.MustCompile(`^(\d{4}[-/.]\d{1,2}[-/.]\d{1,2}|\d{1,2}[-/.]\d{1,2}[-/.]\d{2,4})([ T]\d{1,2}:\d{2}(:\d{2})?\S*)?$`)
	emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	urlRe   = regexp.MustCompile(`(?i)^(https?://|www\.)`)
	// Short all-caps codes such as ELEC, F&B or R+W+D
	codeRe = regexp.MustCompile(`^[A-Z][A-Z&+/ ._-]{0,5}$`)

	boolWords = map[string]bool{"true": true, "false": true, "yes": true, "no": true, "y": true, "n": true, "t": true, "f": true, "1": true, "0": true}

	// Name endings that mark identifiers (always dynamic)
	idSuffixes = map[string]bool{"id": true, "uuid": true, "key": true, "sku": true, "no": true, "num": true, "number": true}
	// Name endings that mark a column as a key for the "X_id + X_name" rule
	keySuffixes = map[string]bool{"id": true, "uuid": true, "key": true, "code": true, "no": true, "num": true, "number": true}
	// Name endings that mark measurements (always dynamic)
	measureSuffixes = map[string]bool{"count": true, "total": true, "amount": true, "price": true, "quantity": true, "qty": true, "cost": true, "sum": true}
)

type column struct {
	name        string
	values      []string // one per row, trimmed
	distinct    []string // non-empty values in first-seen order
	nonEmpty    int
	kind        string
	categorical bool
	unique      bool
}

func profileTable(records [][]string) []*column {
	header, rows := records[0], records[1:]

	cols := make([]*column, len(header))
	for i, name := range header {
		c := &column{name: strings.TrimSpace(name)}
		seen := make(map[string]bool)
		for _, row := range rows {
			v := ""
			if i < len(row) {
				v = strings.TrimSpace(row[i])
			}
			c.values = append(c.values, v)
			if v == "" {
				continue
			}
			c.nonEmpty++
			if !seen[v] {
				seen[v] = true
				c.distinct = append(c.distinct, v)
			}
		}
		c.unique = c.nonEmpty >= 2 && len(c.distinct) == c.nonEmpty
		c.kind = detectKind(c.distinct)
		c.categorical = isCategorical(c)
		cols[i] = c
	}

	markEntityPairs(cols)
	return cols
}

func detectKind(distinct []string) string {
	if len(distinct) == 0 {
		return kindEmpty
	}
	if isBoolean(distinct) {
		return kindBool
	}
	checks := []struct {
		kind  string
		match func(string) bool
	}{
		{kindInt, intRe.MatchString},
		{kindDecimal, isNumber},
		{kindDate, dateRe.MatchString},
		{kindEmail, emailRe.MatchString},
		{kindURL, urlRe.MatchString},
	}
	for _, check := range checks {
		if all(distinct, check.match) {
			return check.kind
		}
	}
	return kindText
}

func isBoolean(distinct []string) bool {
	if len(distinct) > 2 || (len(distinct) == 1 && intRe.MatchString(distinct[0])) {
		return false
	}
	return all(distinct, func(v string) bool { return boolWords[strings.ToLower(v)] })
}

func isNumber(v string) bool {
	_, err := strconv.ParseFloat(strings.ReplaceAll(v, ",", ""), 64)
	return err == nil
}

// isCategorical decides whether a column holds fixed values: booleans,
// statuses, types and short codes are categorical; identifiers, numbers,
// dates, emails, URLs and free text are dynamic.
func isCategorical(c *column) bool {
	switch c.kind {
	case kindBool:
		return true
	case kindText:
	default:
		return false
	}

	if isIDName(c.name) || hasNameSuffix(c.name, measureSuffixes) {
		return false
	}

	d := len(c.distinct)
	if d > maxCategories {
		return false
	}
	// Values repeat across rows, or are all short codes
	return d*10 <= c.nonEmpty*6 || all(c.distinct, codeRe.MatchString)
}

// markEntityPairs applies the "X_id + X_name" rule: a key column and a
// same-prefix column whose values correspond one-to-one describe a related
// entity (store_id + store_name), not a fixed category.
func markEntityPairs(cols []*column) {
	for _, key := range cols {
		tokens := nameTokens(key.name)
		if len(tokens) < 2 || !keySuffixes[tokens[len(tokens)-1]] || len(key.distinct) < 2 {
			continue
		}
		prefix := strings.Join(tokens[:len(tokens)-1], "_")
		for _, other := range cols {
			if other == key || !strings.HasPrefix(strings.Join(nameTokens(other.name), "_"), prefix) {
				continue
			}
			if oneToOne(key.values, other.values) {
				key.categorical, other.categorical = false, false
			}
		}
	}
}

func oneToOne(a, b []string) bool {
	forward := make(map[string]string)
	backward := make(map[string]string)
	rows := 0
	for i := range a {
		if a[i] == "" || b[i] == "" {
			continue
		}
		rows++
		if prev, ok := forward[a[i]]; ok && prev != b[i] {
			return false
		}
		if prev, ok := backward[b[i]]; ok && prev != a[i] {
			return false
		}
		forward[a[i]], backward[b[i]] = b[i], a[i]
	}
	return rows >= 2
}

func isIDName(name string) bool {
	tokens := nameTokens(name)
	return len(tokens) > 0 && idSuffixes[tokens[len(tokens)-1]]
}

func hasNameSuffix(name string, suffixes map[string]bool) bool {
	tokens := nameTokens(name)
	return len(tokens) > 0 && suffixes[tokens[len(tokens)-1]]
}

func all(values []string, pred func(string) bool) bool {
	for _, v := range values {
		if !pred(v) {
			return false
		}
	}
	return true
}
