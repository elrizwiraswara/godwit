package infer

import (
	"fmt"
	"strconv"
	"strings"

	utils "github.com/elrizwiraswara/godwit/utils"
)

// Step 2: when both samples contain some of the same records, line them up
// and read the mappings straight from the data instead of guessing.

type evidenceKind int

const (
	evUnknown evidenceKind = iota
	evSame
	evSameIgnoringCase
	evMapping
	evContradicted
)

type recordEvidence struct {
	kind     evidenceKind
	rows     int
	mapping  map[string]string
	oneToOne bool
}

// findSharedRecords looks for a key present in both samples (such as
// product_id = id) and returns the matching (source row, target row) pairs.
func findSharedRecords(src, tgt []*column) ([][2]int, string) {
	var best [][2]int
	note := ""
	for si, s := range src {
		if !s.unique {
			continue
		}
		for ti, t := range tgt {
			if !t.unique {
				continue
			}
			rowOf := make(map[string]int)
			for r, v := range t.values {
				if v != "" {
					rowOf[normValue(v)] = r
				}
			}
			var pairs [][2]int
			for r, v := range s.values {
				if tr, ok := rowOf[normValue(v)]; ok && v != "" {
					pairs = append(pairs, [2]int{r, tr})
				}
			}
			if len(pairs) < 3 || len(pairs)*2 < min(s.nonEmpty, t.nonEmpty) || len(pairs) <= len(best) {
				continue
			}
			// Auto-increment ids can coincide between unrelated samples, so
			// require some other column to agree on the lined-up records too.
			if !otherColumnAgrees(src, tgt, si, ti, pairs) {
				continue
			}
			best = pairs
			note = fmt.Sprintf("%s = %s (%d shared records)", s.name, t.name, len(pairs))
		}
	}
	return best, note
}

func otherColumnAgrees(src, tgt []*column, keyS, keyT int, pairs [][2]int) bool {
	for si, s := range src {
		for ti, t := range tgt {
			if si == keyS || ti == keyT {
				continue
			}
			if ev := compareOnSharedRecords(s, t, pairs); ev.kind == evSame && len(ev.mapping) >= 2 {
				return true
			}
		}
	}
	return false
}

// compareOnSharedRecords checks how a source column's values relate to a
// target column's values across the lined-up records.
func compareOnSharedRecords(s, t *column, pairs [][2]int) recordEvidence {
	ev := recordEvidence{mapping: make(map[string]string), oneToOne: true}
	backward := make(map[string]string)
	same, sameIgnoringCase, consistent := true, true, true

	for _, p := range pairs {
		sv, tv := s.values[p[0]], t.values[p[1]]
		if sv == "" || tv == "" {
			continue
		}
		ev.rows++
		if prev, ok := ev.mapping[sv]; ok && prev != tv {
			consistent = false
		}
		ev.mapping[sv] = tv
		if prev, ok := backward[tv]; ok && prev != sv {
			ev.oneToOne = false
		}
		backward[tv] = sv
		if normValue(sv) != normValue(tv) {
			same = false
		}
		if !strings.EqualFold(sv, tv) {
			sameIgnoringCase = false
		}
	}

	switch {
	case ev.rows < 2:
		ev.kind = evUnknown
	case same:
		ev.kind = evSame
	case sameIgnoringCase:
		ev.kind = evSameIgnoringCase
	case !consistent:
		ev.kind = evContradicted
	case len(ev.mapping) == ev.rows:
		// Every source value appears once, so a lookup learned here would
		// not generalize to other rows
		ev.kind = evUnknown
	default:
		ev.kind = evMapping
	}
	return ev
}

// learnParts turns whole-value examples of grouped values into a per-part
// mapping ("R+W" -> "view, edit" gives R -> view, W -> edit), so that unseen
// combinations convert too. Returns nil if the examples disagree.
func learnParts(learned map[string]string) map[string]string {
	parts := make(map[string]string)
	for s, t := range learned {
		sp, tp := utils.SplitComposite(s), utils.SplitComposite(t)
		if len(sp) != len(tp) {
			return nil
		}
		for i := range sp {
			if prev, ok := parts[sp[i]]; ok && prev != tp[i] {
				return nil
			}
			parts[sp[i]] = tp[i]
		}
	}
	return parts
}

// normValue makes "65.00" and "65" compare equal.
func normValue(v string) string {
	v = strings.TrimSpace(v)
	if intRe.MatchString(v) || strings.Contains(v, ".") {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return strconv.FormatFloat(f, 'f', -1, 64)
		}
	}
	return v
}
