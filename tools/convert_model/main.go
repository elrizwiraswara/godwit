package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	embedding "github.com/elrizwiraswara/godwit/embedding"
)

func main() {
	// Usage: go run tools/convert_model/main.go <source model.safetensors> <output model.safetensors>
	// Converts the float32 weights of a Hugging Face BERT embedding model to
	// float16 (half the size) and drops tensors that inference does not use.

	if len(os.Args) != 3 {
		log.Fatal("usage: go run tools/convert_model/main.go <source.safetensors> <output.safetensors>")
	}

	keep := func(name string) bool {
		return !strings.HasPrefix(name, "pooler.") && name != "embeddings.position_ids"
	}
	if err := embedding.ConvertToF16(os.Args[1], os.Args[2], keep); err != nil {
		log.Fatalf("Error converting model: %v", err)
	}

	fmt.Printf("✓ Converted %s -> %s\n", os.Args[1], os.Args[2])
}
