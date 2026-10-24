package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	convert "github.com/elrizwiraswara/godwit/convert"
	utils "github.com/elrizwiraswara/godwit/utils"
)

func main() {
	// Usage: go run converter\convert_csv.go

	var sourceDataPath, sourceSchemaPath, targetSchemaPath, schemaName string

	// Ask for input interactively
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Please enter the source data CSV path: ")
	sourceDataPath, _ = reader.ReadString('\n')
	sourceDataPath = strings.TrimSpace(sourceDataPath)

	fmt.Print("Please enter the source schema JSON path: ")
	sourceSchemaPath, _ = reader.ReadString('\n')
	sourceSchemaPath = strings.TrimSpace(sourceSchemaPath)

	fmt.Print("Please enter the target schema JSON path: ")
	targetSchemaPath, _ = reader.ReadString('\n')
	targetSchemaPath = strings.TrimSpace(targetSchemaPath)

	fmt.Print("Please enter a name for the output file: ")
	schemaName, _ = reader.ReadString('\n')
	schemaName = strings.TrimSpace(schemaName)

	// Load schemas
	sourceSchema, err := utils.LoadSchemaJSON(sourceSchemaPath)
	if err != nil {
		log.Fatalf("Error loading source schema: %v", err)
	}

	targetSchema, err := utils.LoadSchemaJSON(targetSchemaPath)
	if err != nil {
		log.Fatalf("Error loading target schema: %v", err)
	}

	// Read CSV data
	records, err := utils.ReadCSVRecords(sourceDataPath)
	if err != nil {
		log.Fatalf("Error reading CSV data: %v", err)
	}

	// Convert CSV data
	fmt.Println("Converting CSV data...")
	result, err := convert.Convert(records, sourceSchema, targetSchema)
	if err != nil {
		log.Fatalf("Error converting data: %v", err)
	}

	// Write output CSV
	csvFile := fmt.Sprintf("output/converted_%s.csv", schemaName)
	if err := utils.WriteCSV(csvFile, result.Records); err != nil {
		log.Fatalf("Error writing output CSV: %v", err)
	}

	fmt.Printf("✓ Successfully converted %d rows to %s\n", len(result.Records)-1, csvFile)

	if len(result.Unmapped) > 0 {
		fmt.Println("⚠ These values have no mapping in the schema and were copied unchanged:")
		for _, u := range result.Unmapped {
			rows := "rows"
			if u.Rows == 1 {
				rows = "row"
			}
			fmt.Printf("  %s: %q (%d %s)\n", u.Column, u.Value, u.Rows, rows)
		}
	}
}
