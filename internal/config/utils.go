package config

import (
	"fmt"

	"go.yaml.in/yaml/v4"
)

type UniqueStringList []string

func (s *UniqueStringList) UnmarshalYAML(node *yaml.Node) error {
	var values []string

	if err := node.Decode(&values); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(values))

	for _, v := range values {
		if _, exists := seen[v]; exists {
			return fmt.Errorf("duplicate value %q", v)
		}
		seen[v] = struct{}{}
	}

	*s = values
	return nil
}

func isSubset[T comparable](subset, superset []T) bool {
	counts := make(map[T]int)

	for _, v := range superset {
		counts[v]++
	}

	for _, v := range subset {
		if counts[v] == 0 {
			return false
		}
		counts[v]--
	}

	return true
}
