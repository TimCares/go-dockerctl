package utils

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v4"
)

// UniqueStringList is a list of strings that rejects duplicates when decoded from YAML.
type UniqueStringList []string

// UnmarshalYAML decodes a YAML sequence and fails if any value appears more than once.
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

func FindExcluded[T comparable](subset, superset []T) []T {
	excluded := []T{}
	set := make(map[T]struct{}, len(superset))

	for _, item := range superset {
		set[item] = struct{}{}
	}

	for _, item := range subset {
		if _, ok := set[item]; !ok {
			excluded = append(excluded, item)
		}
	}

	return excluded
}

func IsSubset[T comparable](subset, superset []T) bool {
	excluded := FindExcluded(subset, superset)
	return len(excluded) == 0
}

func Hostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return ""
	}
	return hostname
}
