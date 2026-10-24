package infer

import (
	"fmt"
	"os"
	"testing"

	embedding "github.com/elrizwiraswara/godwit/embedding"
	types "github.com/elrizwiraswara/godwit/types"
	utils "github.com/elrizwiraswara/godwit/utils"
)

type want struct {
	source string
	values map[string]string
}

// Correct mappings for input/samples, checked against the shared records.
var expected = map[int]map[string]want{
	1: {
		"id": {source: "product_id"}, "name": {source: "product_name"},
		"category":       {source: "product_type", values: map[string]string{"ELEC": "Electronics", "F&B": "Food & Beverage", "FURN": "Furniture", "STAT": "Stationery"}},
		"price":          {source: "unit_price"},
		"stock_quantity": {source: "qty_available"},
		"is_active":      {source: "active_flag", values: map[string]string{"Y": "true", "N": "false"}},
		"status":         {source: "inventory_status", values: map[string]string{"available": "in_stock", "low": "low_stock", "unavailable": "out_of_stock"}},
		"unit":           {source: "measurement_unit", values: map[string]string{"box": "box", "jar": "jar", "kg": "kilogram", "pack": "pack", "pcs": "piece", "ream": "ream"}},
		"supplier_id":    {source: "vendor_code"}, "supplier_name": {source: "vendor_company"},
	},
	2: {
		"item_id": {source: "prod_code"}, "item_name": {source: "prod_description"},
		"item_category":   {source: "category_code", values: map[string]string{"ACC": "Accessories", "BEV": "Beverages", "FUR": "Furniture", "GRO": "Groceries", "OFF": "Office Supplies"}},
		"unit_price":      {source: "price_per_unit"},
		"available_stock": {source: "qty_in_stock"},
		"is_available":    {source: "active_status", values: map[string]string{"1": "true", "0": "false"}},
		"stock_status":    {source: "availability", values: map[string]string{"A": "available", "D": "discontinued", "L": "low_stock"}},
		"unit_type":       {source: "measure_unit", values: map[string]string{"bar": "bar", "btl": "bottle", "can": "can", "ea": "unit", "pc": "piece", "pk": "pack", "rm": "ream", "set": "set"}},
		"warehouse_id":    {source: "storage_loc_id"}, "warehouse_name": {source: "storage_location"},
	},
	3: {
		"sku": {source: "product_code"}, "product_name": {source: "description"},
		"category":        {source: "prod_type", values: map[string]string{"ELEC": "Electronics", "FOOD": "Food", "FURN": "Furniture", "KITCH": "Kitchenware", "STAT": "Stationery"}},
		"retail_price":    {source: "sale_price"},
		"stock_qty":       {source: "inventory_count"},
		"enabled":         {source: "is_active", values: map[string]string{"Y": "true", "N": "false"}},
		"stock_level":     {source: "availability_status", values: map[string]string{"avail": "in_stock", "disc": "discontinued", "low": "low_stock"}},
		"unit_of_measure": {source: "uom", values: map[string]string{"bx": "box", "jar": "jar", "pc": "piece", "pk": "pack", "set": "set"}},
		"permissions":     {source: "access_rights", values: map[string]string{"R": "view", "R+W": "view, edit", "R+W+D": "view, edit, delete"}},
		"supplier_id":     {source: "vendor_id"}, "supplier_name": {source: "vendor_name"},
	},
	4: {
		"id": {}, "parent_id": {}, "item_code": {source: "sku"}, "name": {source: "name"},
		"product_group": {}, "srp": {source: "price"}, "qty": {source: "stock"},
		"department_id": {}, "department_name": {}, "category_id": {}, "category_name": {},
		"store_id": {}, "store_name": {source: "store"},
	},
}

type tally struct {
	colsRight, colsWrong, colsMissed      int
	valuesRight, valuesWrong, valuesUnres int
	errors                                []string
}

func (s *tally) check(pair int, got []types.ColumnSchema) {
	for _, col := range got {
		w := expected[pair][col.TargetColumn]
		switch {
		case col.Column == w.source:
			s.colsRight++
		case col.Column == "":
			s.colsMissed++
			s.errors = append(s.errors, fmt.Sprintf("pair %d %s: unmatched, want %q", pair, col.TargetColumn, w.source))
			continue
		default:
			s.colsWrong++
			s.errors = append(s.errors, fmt.Sprintf("pair %d %s: got %q, want %q", pair, col.TargetColumn, col.Column, w.source))
			continue
		}
		for src, dst := range w.values {
			got, ok := col.ValuesMapping[src]
			switch {
			case !ok:
				s.valuesUnres++
				s.errors = append(s.errors, fmt.Sprintf("pair %d %s: %q unresolved, want %q", pair, col.TargetColumn, src, dst))
			case got != dst:
				s.valuesWrong++
				s.errors = append(s.errors, fmt.Sprintf("pair %d %s: %q -> %q, want %q", pair, col.TargetColumn, src, got, dst))
			default:
				s.valuesRight++
			}
		}
	}
}

func run(t *testing.T, opts Options) tally {
	var s tally
	for pair := 1; pair <= 4; pair++ {
		src, err := utils.ReadCSVRecords(fmt.Sprintf("../input/samples/source_sample_data_%d.csv", pair))
		if err != nil {
			t.Fatal(err)
		}
		tgt, err := utils.ReadCSVRecords(fmt.Sprintf("../input/samples/target_sample_data_%d.csv", pair))
		if err != nil {
			t.Fatal(err)
		}
		result, err := Generate(src, tgt, opts)
		if err != nil {
			t.Fatal(err)
		}
		s.check(pair, result.Source)
	}
	return s
}

func loadModel(t *testing.T) Embedder {
	m, err := embedding.Load("../models/all-MiniLM-L6-v2")
	if os.IsNotExist(err) {
		t.Skip("embedding model not present")
	}
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestSharedRecords(t *testing.T) {
	s := run(t, Options{})
	for _, e := range s.errors {
		t.Error(e)
	}
	t.Logf("columns: %d right; values: %d right", s.colsRight, s.valuesRight)
}

func TestSharedRecordsWithEmbeddings(t *testing.T) {
	s := run(t, Options{Embedder: loadModel(t)})
	for _, e := range s.errors {
		t.Error(e)
	}
}

// Measures how well steps 3-4 do when the samples share no records. Logged
// rather than asserted: it is a benchmark, not a guarantee.
func TestWithoutSharedRecords(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts Options
	}{
		{"rules only (steps 1+3)", Options{IgnoreSharedRecords: true}},
		{"rules + embeddings (steps 1+3+4)", Options{IgnoreSharedRecords: true, Embedder: loadModel(t)}},
	} {
		s := run(t, tc.opts)
		t.Logf("%s: columns %d right, %d wrong, %d unmatched; values %d right, %d wrong, %d unresolved",
			tc.name, s.colsRight, s.colsWrong, s.colsMissed, s.valuesRight, s.valuesWrong, s.valuesUnres)
		for _, e := range s.errors {
			t.Log("   ", e)
		}
	}
}
