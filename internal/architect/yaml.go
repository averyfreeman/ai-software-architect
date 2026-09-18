package architect

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

func decodeStrictYAML(content string, target any) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("YAML document is empty")
	}
	var document yaml.Node
	decoder := yaml.NewDecoder(strings.NewReader(content))
	if err := decoder.Decode(&document); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}
	if err := rejectUnsafeYAML(&document, 0); err != nil {
		return err
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple YAML documents are not allowed")
		}
		return fmt.Errorf("invalid YAML: %w", err)
	}
	known := yaml.NewDecoder(strings.NewReader(content))
	known.KnownFields(true)
	if err := known.Decode(target); err != nil {
		return fmt.Errorf("invalid YAML shape: %w", err)
	}
	return nil
}

func rejectUnsafeYAML(node *yaml.Node, depth int) error {
	if depth > MaxNestingDepth {
		return fmt.Errorf("YAML nesting exceeds %d", MaxNestingDepth)
	}
	if node == nil {
		return nil
	}
	if node.Kind == yaml.AliasNode {
		return fmt.Errorf("YAML aliases are not allowed")
	}
	if node.Kind == yaml.MappingNode {
		keys := map[string]bool{}
		for index := 0; index+1 < len(node.Content); index += 2 {
			key := node.Content[index]
			if key.Kind != yaml.ScalarNode {
				return fmt.Errorf("YAML mapping keys must be scalar")
			}
			if keys[key.Value] {
				return fmt.Errorf("duplicate YAML mapping key %q", key.Value)
			}
			keys[key.Value] = true
		}
	}
	for _, child := range node.Content {
		if err := rejectUnsafeYAML(child, depth+1); err != nil {
			return err
		}
	}
	return nil
}
