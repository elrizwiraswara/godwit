package utils

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	types "github.com/elrizwiraswara/godwit/types"
)

// CompositeSeparator matches the separators inside grouped values such as
// "R+W+D" or "view, edit, delete".
var CompositeSeparator = regexp.MustCompile(`\s*[+,;|/]\s*`)

// SplitComposite splits a grouped value into its parts: "R+W" -> [R W].
func SplitComposite(value string) []string {
	return CompositeSeparator.Split(strings.TrimSpace(value), -1)
}

// ParseCSV reads CSV leniently, as exported by spreadsheets and POS systems.
func ParseCSV(r io.Reader) ([][]string, error) {
	br := bufio.NewReader(r)
	// Skip the UTF-8 byte order mark Excel adds, or it ends up in the first header
	if bom, err := br.Peek(3); err == nil && bytes.Equal(bom, []byte{0xEF, 0xBB, 0xBF}) {
		br.Discard(3)
	}

	reader := csv.NewReader(br)
	reader.LazyQuotes = true       // Allow lazy quotes
	reader.TrimLeadingSpace = true // Trim spaces after delimiters

	return reader.ReadAll()
}

func ReadCSVRecords(path string) ([][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	records, err := ParseCSV(file)
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV must have at least header and one data row")
	}

	return records, nil
}

func WriteCSV(path string, records [][]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	return writer.WriteAll(records)
}

func SaveJSON(path string, data any) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func LoadSchemaJSON(path string) ([]types.ColumnSchema, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var schema []types.ColumnSchema
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&schema); err != nil {
		return nil, err
	}

	return schema, nil
}
