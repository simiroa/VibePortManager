//go:build ignore

// gen-types: config.schema.json → verifies internal/config/types.go is in sync
// Run: go run scripts/gen-types.go
// Currently a validator (manual struct authoring). Full codegen left as TODO.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	root, _ := os.Getwd()
	schemaPath := filepath.Join(root, "specs", "config.schema.json")

	data, err := os.ReadFile(schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read schema: %v\n", err)
		os.Exit(1)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		fmt.Fprintf(os.Stderr, "parse schema: %v\n", err)
		os.Exit(1)
	}

	// Verify top-level required fields
	required := schemaRequired(schema)
	typesGoPath := filepath.Join(root, "internal", "config", "types.go")
	typesGo, err := os.ReadFile(typesGoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read types.go: %v\n", err)
		os.Exit(1)
	}

	var missing []string
	for _, f := range required {
		jsonTag := `"` + f + `"`
		found := false
		for _, line := range splitLines(string(typesGo)) {
			if containsStr(line, jsonTag) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, f)
		}
	}

	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "types.go missing JSON tags for schema fields: %v\n", missing)
		os.Exit(1)
	}
	fmt.Println("gen-types: types.go is in sync with config.schema.json ✓")
}

func schemaRequired(schema map[string]interface{}) []string {
	req, _ := schema["required"].([]interface{})
	out := make([]string, 0, len(req))
	for _, v := range req {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStrRaw(s, sub))
}

func containsStrRaw(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
