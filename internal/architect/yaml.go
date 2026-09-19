package architect

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxYAMLDepth = 50

// unmarshalSafe rejects the two YAML features most likely to make an agent's
// interpretation differ from a reviewer’s: aliases and duplicate keys.
func unmarshalSafe(data []byte, target any) error {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return err
	}
	if len(node.Content) == 0 {
		return fmt.Errorf("YAML document is empty")
	}
	if err := validateYAMLNode(node.Content[0], 0); err != nil {
		return err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func validateYAMLNode(node *yaml.Node, depth int) error {
	if depth > maxYAMLDepth {
		return fmt.Errorf("YAML nesting exceeds %d", maxYAMLDepth)
	}
	if node.Kind == yaml.AliasNode {
		return fmt.Errorf("YAML aliases are not allowed")
	}
	if node.Kind == yaml.MappingNode {
		seen := make(map[string]struct{}, len(node.Content)/2)
		for index := 0; index < len(node.Content); index += 2 {
			key := node.Content[index]
			if key.Kind != yaml.ScalarNode {
				return fmt.Errorf("YAML mapping keys must be scalar values")
			}
			if _, exists := seen[key.Value]; exists {
				return fmt.Errorf("duplicate YAML key %q", key.Value)
			}
			seen[key.Value] = struct{}{}
			if err := validateYAMLNode(key, depth+1); err != nil {
				return err
			}
			if err := validateYAMLNode(node.Content[index+1], depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	for _, child := range node.Content {
		if err := validateYAMLNode(child, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func marshalYAML(value any) ([]byte, error) {
	data, err := yaml.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal YAML: %w", err)
	}
	return data, nil
}

func extractFrontmatter(content string) (string, string, error) {
	lines := strings.SplitAfter(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", content, nil
	}
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			return strings.Join(lines[1:index], ""), strings.Join(lines[index+1:], ""), nil
		}
	}
	return "", "", fmt.Errorf("frontmatter is not terminated")
}

func frontmatter(content string, target any) (string, error) {
	front, body, err := extractFrontmatter(content)
	if err != nil {
		return "", err
	}
	if front == "" {
		return "", fmt.Errorf("frontmatter is required")
	}
	if err := unmarshalSafe([]byte(front), target); err != nil {
		return "", fmt.Errorf("parse frontmatter: %w", err)
	}
	return body, nil
}
