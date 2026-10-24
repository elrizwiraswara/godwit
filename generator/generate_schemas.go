package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	config "github.com/elrizwiraswara/godwit/config"
	embedding "github.com/elrizwiraswara/godwit/embedding"
	infer "github.com/elrizwiraswara/godwit/infer"
	utils "github.com/elrizwiraswara/godwit/utils"
)

func main() {
	// Usage: go run generator\generate_schemas.go

	var targetSampleDataPath, sourceSampleDataPath, schemaName string

	// Ask for input interactively
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Please enter the source sample CSV path: ")
	sourceSampleDataPath, _ = reader.ReadString('\n')
	sourceSampleDataPath = strings.TrimSpace(sourceSampleDataPath)

	fmt.Print("Please enter the target sample CSV path: ")
	targetSampleDataPath, _ = reader.ReadString('\n')
	targetSampleDataPath = strings.TrimSpace(targetSampleDataPath)

	fmt.Print("Please enter a name for the schemas: ")
	schemaName, _ = reader.ReadString('\n')
	schemaName = strings.TrimSpace(schemaName)

	source, err := utils.ReadCSVRecords(sourceSampleDataPath)
	if err != nil {
		log.Fatalf("Error reading source sample: %v", err)
	}
	target, err := utils.ReadCSVRecords(targetSampleDataPath)
	if err != nil {
		log.Fatalf("Error reading target sample: %v", err)
	}

	var opts infer.Options
	model, err := embedding.Load(config.EMBEDDING_MODEL_DIR)
	if err != nil {
		fmt.Printf("Embedding model not loaded (%v), continuing with rules only\n", err)
	} else {
		opts.Embedder = model
	}

	fmt.Println("Generating schemas...")
	result, err := infer.Generate(source, target, opts)
	if err != nil {
		log.Fatalf("Error generating schemas: %v", err)
	}
	printReview(result)

	// Save target schema
	targetSchemaFile := fmt.Sprintf("output/schemas/target_schema_%s.json", schemaName)
	if err := utils.SaveJSON(targetSchemaFile, result.Target); err != nil {
		log.Fatalf("Error saving target schema: %v", err)
	}
	fmt.Printf("✓ %s generated successfully\n", targetSchemaFile)

	// Save source schema
	sourceSchemaFile := fmt.Sprintf("output/schemas/source_schema_%s.json", schemaName)
	if err := utils.SaveJSON(sourceSchemaFile, result.Source); err != nil {
		log.Fatalf("Error saving source schema: %v", err)
	}
	fmt.Printf("✓ %s generated successfully\n", sourceSchemaFile)
}

func printReview(result infer.Result) {
	fmt.Println("\n" + strings.Repeat("-", 80))
	fmt.Println("SCHEMA REVIEW")
	fmt.Println(strings.Repeat("-", 80))

	if result.SharedKey != "" {
		fmt.Printf("Shared records: %s\n\n", result.SharedKey)
	} else {
		fmt.Print("No shared records found, matched by names and values only\n\n")
	}

	for _, col := range result.Source {
		source := col.Column
		if source == "" {
			source = "(none)"
		}
		fmt.Printf("%-22s <- %-22s %s\n", col.TargetColumn, source, col.Evidence)
	}

	if len(result.UnusedSource) > 0 {
		fmt.Printf("\nSource columns not used: %s\n", strings.Join(result.UnusedSource, ", "))
	}
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println("Review lines marked CHECK or UNRESOLVED before converting.")
	fmt.Println(strings.Repeat("-", 80))
}
