package repository

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func UpdateYAMLValue(filePath, route, newValue string) error {
	if route == "" {
		return fmt.Errorf("yaml route must not be empty")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read yaml file %q: %w", filePath, err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("failed to parse yaml: %w", err)
	}

	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return fmt.Errorf("unexpected yaml structure: document is empty")
	}

	keys := strings.Split(route, ".")
	if err := setScalarAtPath(root.Content[0], keys, newValue); err != nil {
		return err
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		_ = enc.Close()
		return fmt.Errorf("failed to encode yaml: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("failed to close yaml encoder: %w", err)
	}

	if err := os.WriteFile(filePath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write yaml file %q: %w", filePath, err)
	}
	return nil
}

func setScalarAtPath(node *yaml.Node, keys []string, newValue string) error {
	current := node
	for i, key := range keys {
		if current.Kind != yaml.MappingNode {
			return fmt.Errorf("yaml path %q: expected mapping at segment %q", strings.Join(keys, "."), key)
		}

		found := false
		for j := 0; j < len(current.Content); j += 2 {
			keyNode := current.Content[j]
			valNode := current.Content[j+1]
			if keyNode.Value != key {
				continue
			}
			found = true
			if i == len(keys)-1 {
				if valNode.Kind != yaml.ScalarNode {
					return fmt.Errorf("yaml path %q: target is not a scalar", strings.Join(keys, "."))
				}
				valNode.Value = newValue
				valNode.Tag = "!!str"
				valNode.Style = 0
				return nil
			}
			current = valNode
			break
		}
		if !found {
			return fmt.Errorf("yaml path %q: key %q not found", strings.Join(keys, "."), key)
		}
	}
	return nil
}