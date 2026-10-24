// Package convert applies migration schemas to CSV records.
package convert

import (
	"fmt"
	"sort"
	"strings"

	types "github.com/elrizwiraswara/godwit/types"
	utils "github.com/elrizwiraswara/godwit/utils"
)

// Unmapped is a value copied unchanged because its column has mappings but
// none for this value (for example a status missing from the sample).
type Unmapped struct {
	Column string `json:"column"`
	Value  string `json:"value"`
	Rows   int    `json:"rows"`
}

type Result struct {
	Records  [][]string
	Unmapped []Unmapped
}

// Convert turns source records (header first) into target records.
func Convert(records [][]string, sourceSchema, targetSchema []types.ColumnSchema) (Result, error) {
	if len(records) < 1 {
		return Result{}, fmt.Errorf("CSV has no data")
	}

	// Build source column index map
	sourceColIndex := make(map[string]int)
	for i, colName := range records[0] {
		sourceColIndex[strings.TrimSpace(colName)] = i
	}

	// Count values copied unchanged in columns that have mappings
	type key struct{ column, value string }
	unmappedCount := make(map[key]int)

	// Create output header from target schema
	outputHeader := make([]string, len(targetSchema))
	for i, col := range targetSchema {
		outputHeader[i] = col.Column
	}
	output := [][]string{outputHeader}

	// Convert each data row
	for _, sourceRow := range records[1:] {
		outputRow := make([]string, len(targetSchema))

		for i, targetCol := range targetSchema {
			value := ""

			// Find corresponding source column in schema
			for _, sourceCol := range sourceSchema {
				if sourceCol.TargetColumn == targetCol.Column {
					// Get value from source row
					if colIdx, exists := sourceColIndex[sourceCol.Column]; exists && colIdx < len(sourceRow) {
						sourceValue := strings.TrimSpace(sourceRow[colIdx])

						if sourceValue != "" {
							// Convert value if mapping exists
							convertedValue, mapped := Value(sourceValue, sourceCol)
							value = convertedValue
							if !mapped {
								unmappedCount[key{sourceCol.Column, sourceValue}]++
							}
						}
					}
					break
				}
			}

			outputRow[i] = value
		}

		output = append(output, outputRow)
	}

	unmapped := make([]Unmapped, 0, len(unmappedCount))
	for k, count := range unmappedCount {
		unmapped = append(unmapped, Unmapped{Column: k.column, Value: k.value, Rows: count})
	}
	sort.Slice(unmapped, func(i, j int) bool {
		if unmapped[i].Column != unmapped[j].Column {
			return unmapped[i].Column < unmapped[j].Column
		}
		return unmapped[i].Value < unmapped[j].Value
	})

	return Result{Records: output, Unmapped: unmapped}, nil
}

// Value converts one value, reporting false when the column has mappings but
// none fits the value.
func Value(value string, sourceCol types.ColumnSchema) (string, bool) {
	// Columns without mappings are copied as they are
	if sourceCol.ValuesMapping == nil && sourceCol.TokenMapping == nil {
		return value, true
	}

	// If there's a values mapping, apply it
	if mappedValue, exists := sourceCol.ValuesMapping[value]; exists {
		return mappedValue, true
	}

	// Grouped values not seen in the sample ("R+D") convert part by part
	if sourceCol.TokenMapping != nil {
		parts := utils.SplitComposite(value)
		for i, part := range parts {
			mappedPart, exists := sourceCol.TokenMapping[part]
			if !exists {
				return value, false
			}
			parts[i] = mappedPart
		}
		return strings.Join(parts, sourceCol.TokenSeparator), true
	}

	// Return original value if no mapping found
	return value, false
}
