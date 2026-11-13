package sops

import (
	"fmt"

	"go.yaml.in/yaml/v3"
)

type stringList []string

func (s *stringList) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var value string
		if err := node.Decode(&value); err != nil {
			return err
		}
		*s = []string{value}

	case yaml.SequenceNode:
		var values []string
		if err := node.Decode(&values); err != nil {
			return err
		}
		*s = values

	default:
		return fmt.Errorf("expected string or list of strings")
	}

	return nil
}

type CreationRule struct {
	PathRegex string     `yaml:"path_regex"`
	Age       stringList `yaml:"age"`

	Extra map[string]interface{} `yaml:",inline"`
}

type Config struct {
	CreationRules []CreationRule `yaml:"creation_rules"`

	Extra map[string]interface{} `yaml:",inline"`
}
