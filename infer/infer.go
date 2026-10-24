// Package infer generates migration schemas from sample CSVs, in four steps:
//
//  1. Profile every column: data type, fixed values (categorical) or not,
//     and its exact list of values.
//  2. If both samples contain some of the same records (found through a
//     shared key such as product_id = id), learn column and value mappings
//     directly from those records.
//  3. Match the remaining names and values with string rules (prefixes,
//     initials, abbreviations, yes/no words).
//  4. Optionally compare meaning with a small local embedding model.
//
// Every mapping records the evidence it came from, so a person can review
// the uncertain ones.
package infer

import (
	"fmt"
	"sort"
	"strings"

	types "github.com/elrizwiraswara/godwit/types"
)

// Embedder turns texts into L2-normalized vectors (see package embedding).
type Embedder interface {
	EmbedAll(texts []string) [][]float32
}

type Options struct {
	// Embedder enables step 4. Nil runs steps 1-3 only.
	Embedder Embedder
	// IgnoreSharedRecords skips step 2 and value-overlap checks, to measure
	// how well matching works when the samples contain different records.
	IgnoreSharedRecords bool
	// Overrides fixes the source column for target columns after a person
	// reviewed them (target column -> source column, "" for none).
	Overrides map[string]string
}

type Result struct {
	Target []types.ColumnSchema
	Source []types.ColumnSchema
	// SourceColumns is the source sample's header
	SourceColumns []string
	// SharedKey describes the key used to line up records, if one was found
	SharedKey string
	// UnusedSource lists source columns that map to no target column
	UnusedSource []string
}

// Pairs scoring below this are left unmatched rather than guessed
const minPairScore = 0.4

type candidate struct {
	s, t        int
	score       float64
	evidence    string
	categorical bool              // the pair maps fixed values
	learned     map[string]string // value mapping seen in shared records
}

type generator struct {
	src, tgt []*column
	opts     Options
	pairs    [][2]int
	nameSim  [][]float64 // rule-based, [source][target]
	nameEmb  [][]float64 // embedding cosine, nil without an embedder
}

// Generate builds the target schema and the source-to-target mapping schema
// from a source sample and a target sample (CSV records, header first).
func Generate(source, target [][]string, opts Options) (Result, error) {
	if len(source) < 2 || len(target) < 2 {
		return Result{}, fmt.Errorf("both samples need a header and at least one data row")
	}

	g := &generator{src: profileTable(source), tgt: profileTable(target), opts: opts}

	var result Result
	if !opts.IgnoreSharedRecords {
		g.pairs, result.SharedKey = findSharedRecords(g.src, g.tgt)
	}
	g.compareNames()

	chosen := make(map[int]candidate)
	usedSource := make(map[int]bool)
	manualNone := make(map[int]bool)
	if err := g.applyOverrides(chosen, usedSource, manualNone); err != nil {
		return Result{}, err
	}

	var cands []candidate
	for si := range g.src {
		for ti := range g.tgt {
			if c, ok := g.scorePair(si, ti); ok {
				cands = append(cands, c)
			}
		}
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].score > cands[j].score })

	// Best-scoring pairs first; each column is used at most once
	for _, c := range cands {
		if _, taken := chosen[c.t]; taken || manualNone[c.t] || usedSource[c.s] {
			continue
		}
		chosen[c.t] = c
		usedSource[c.s] = true
	}

	for ti, t := range g.tgt {
		c, ok := chosen[ti]
		if !ok {
			evidence := "no matching source column"
			if manualNone[ti] {
				evidence = "left unmatched manually"
			}
			result.Source = append(result.Source, types.ColumnSchema{
				TargetColumn: t.name,
				Values:       []string{},
				Evidence:     evidence,
			})
			continue
		}
		result.Source = append(result.Source, g.buildMapping(c))
		if c.categorical {
			t.categorical = true
		}
	}

	for _, t := range g.tgt {
		values := []string{}
		if t.categorical {
			values = t.distinct
		}
		result.Target = append(result.Target, types.ColumnSchema{Column: t.name, Values: values})
	}

	for si, s := range g.src {
		result.SourceColumns = append(result.SourceColumns, s.name)
		if !usedSource[si] {
			result.UnusedSource = append(result.UnusedSource, s.name)
		}
	}

	return result, nil
}

// applyOverrides fixes the pairs a person chose before automatic matching.
func (g *generator) applyOverrides(chosen map[int]candidate, usedSource, manualNone map[int]bool) error {
	index := func(cols []*column, name string) int {
		for i, c := range cols {
			if c.name == name {
				return i
			}
		}
		return -1
	}

	for _, targetName := range sortedKeys(g.opts.Overrides) {
		sourceName := g.opts.Overrides[targetName]
		ti := index(g.tgt, targetName)
		if ti < 0 {
			return fmt.Errorf("override: unknown target column %q", targetName)
		}
		if sourceName == "" {
			manualNone[ti] = true
			continue
		}
		si := index(g.src, sourceName)
		if si < 0 {
			return fmt.Errorf("override: unknown source column %q", sourceName)
		}
		if usedSource[si] {
			return fmt.Errorf("override: source column %q is chosen for more than one target column", sourceName)
		}

		c, ok := g.scorePair(si, ti)
		if !ok || c.evidence == "" {
			// The data argues against this pair; keep it, but map values from scratch
			c = candidate{s: si, t: ti, categorical: g.src[si].categorical && g.tgt[ti].categorical}
		}
		c.evidence = strings.TrimSuffix("chosen manually; "+c.evidence, "; ")
		chosen[ti] = c
		usedSource[si] = true
	}
	return nil
}

func (g *generator) compareNames() {
	g.nameSim = make([][]float64, len(g.src))
	for si, s := range g.src {
		g.nameSim[si] = make([]float64, len(g.tgt))
		for ti, t := range g.tgt {
			g.nameSim[si][ti] = nameSimilarity(expandName(s.name), expandName(t.name))
		}
	}

	if g.opts.Embedder == nil {
		return
	}
	var texts []string
	for _, c := range g.src {
		texts = append(texts, humanName(c.name))
	}
	for _, c := range g.tgt {
		texts = append(texts, humanName(c.name))
	}
	vecs := g.opts.Embedder.EmbedAll(texts)

	g.nameEmb = make([][]float64, len(g.src))
	for si := range g.src {
		g.nameEmb[si] = make([]float64, len(g.tgt))
		for ti := range g.tgt {
			g.nameEmb[si][ti] = cosine(vecs[si], vecs[len(g.src)+ti])
		}
	}
}

func (g *generator) scorePair(si, ti int) (candidate, bool) {
	s, t := g.src[si], g.tgt[ti]
	c := candidate{s: si, t: ti}

	nameScore := g.nameSim[si][ti]
	nameNote := fmt.Sprintf("name match %.2f", nameScore)
	if g.nameEmb != nil {
		// Unrelated names typically score 0.1-0.3 with MiniLM, so rescale
		meaning := min(max((g.nameEmb[si][ti]-0.2)/0.6, 0), 1)
		nameNote += fmt.Sprintf(", meaning %.2f", g.nameEmb[si][ti])
		nameScore = max(nameScore, meaning)
	}

	// Step 2: the shared records decide whenever they can
	if g.pairs != nil {
		ev := compareOnSharedRecords(s, t, g.pairs)
		switch ev.kind {
		case evSame, evSameIgnoringCase:
			c.score = 10 + nameScore
			c.evidence = fmt.Sprintf("same values in %d shared records", ev.rows)
			if ev.kind == evSameIgnoringCase {
				c.score--
				c.evidence = fmt.Sprintf("same values ignoring case in %d shared records", ev.rows)
			}
			c.categorical = s.categorical && t.categorical
			if c.categorical {
				c.learned = ev.mapping
			}
			return c, true
		case evMapping:
			// A lookup into an identifier column is not a fixed-value mapping
			if !isIDName(t.name) {
				c.score = 8 + nameScore
				if ev.oneToOne {
					c.score += 0.5
				}
				c.evidence = fmt.Sprintf("consistent value mapping in %d shared records", ev.rows)
				c.categorical = true
				c.learned = ev.mapping
				return c, true
			}
		case evContradicted:
			return c, false
		}
	}

	// Steps 3 and 4: names, value rules and value formats
	valueLike := func(col *column) bool {
		return (col.kind == kindText || col.kind == kindBool) && !isIDName(col.name) && len(col.distinct) <= maxCategories
	}
	coverage := 0.0
	if valueLike(s) && valueLike(t) {
		coverage = ruleCoverage(s.distinct, t.distinct)
	}
	overlap := 0.0
	if !g.opts.IgnoreSharedRecords {
		overlap = jaccard(s.distinct, t.distinct, strings.ToLower)
	}
	format := jaccard(s.distinct, t.distinct, shape)

	c.categorical = (s.categorical && t.categorical) || (coverage >= 0.5 && valueLike(s) && valueLike(t))
	var valueScore float64
	switch {
	case c.categorical:
		valueScore = coverage
	case !s.categorical && !t.categorical:
		valueScore = max(overlap, 0.5*format)
	default:
		valueScore = max(overlap, 0.3*format)
	}

	c.score = typeCompatibility(s.kind, t.kind) * (nameScore + 1.5*valueScore)
	if s.categorical != t.categorical && !c.categorical {
		c.score *= 0.6
	}

	notes := []string{nameNote}
	if coverage > 0 {
		notes = append(notes, fmt.Sprintf("%.0f%% of values matched by rules", 100*coverage))
	}
	if overlap > 0 {
		notes = append(notes, fmt.Sprintf("value overlap %.2f", overlap))
	}
	c.evidence = fmt.Sprintf("score %.2f (%s)", c.score, strings.Join(notes, ", "))

	return c, c.score >= minPairScore
}

func typeCompatibility(a, b string) float64 {
	numeric := func(k string) bool { return k == kindInt || k == kindDecimal }
	switch {
	case a == b:
		return 1
	case a == kindEmpty || b == kindEmpty:
		return 0.5
	case numeric(a) && numeric(b):
		return 0.9
	case (a == kindBool && b == kindText) || (a == kindText && b == kindBool):
		return 0.7
	case a == kindText || b == kindText:
		// Formats can change between systems ("S001" vs 1)
		return 0.4
	}
	return 0.2
}

func (g *generator) buildMapping(c candidate) types.ColumnSchema {
	s, t := g.src[c.s], g.tgt[c.t]
	col := types.ColumnSchema{Column: s.name, TargetColumn: t.name, Values: []string{}, Evidence: c.evidence}
	if !c.categorical {
		return col
	}

	vm := mapValues(s.distinct, t.distinct, c.learned, g.opts.Embedder)
	col.Values = s.distinct
	col.ValuesMapping = vm.to
	col.TokenMapping, col.TokenSeparator = vm.tokens, vm.sep
	col.Unresolved = vm.unresolved
	col.Evidence += "; " + vm.summary()
	return col
}
