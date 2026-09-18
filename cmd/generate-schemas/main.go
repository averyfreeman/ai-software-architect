package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/averyfreeman/adr-repo-governance/internal/architect"
)

func main() {
	output := flag.String("output", "shared/schemas/generated", "schema output directory")
	check := flag.Bool("check", false, "fail when generated schemas differ from checked-in files")
	flag.Parse()
	if err := run(*output, *check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(output string, check bool) error {
	schemas := map[string]map[string]any{
		"architecture-contract.schema.json":        architect.ContractJSONSchema(),
		"architecture-artifact-bundle.schema.json": architect.BundleJSONSchema(),
	}
	for name, schema := range schemas {
		data, err := architect.SchemaJSON(schema)
		if err != nil {
			return err
		}
		path := filepath.Join(output, name)
		if check {
			existing, readErr := os.ReadFile(path)
			if readErr != nil {
				return fmt.Errorf("%s is missing: %w", path, readErr)
			}
			if string(existing) != string(data) {
				return fmt.Errorf("generated schema differs: %s", path)
			}
			continue
		}
		if err := os.MkdirAll(output, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}
