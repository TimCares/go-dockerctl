package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
)

type Node interface {
	Validate(path string) error
}

type File struct{}

type Dir map[string]Node

type Optional struct {
	Node Node
}

type AtLeastOne struct {
	Node Node
}

func (File) Validate(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing file: %s", path)
		}
		return fmt.Errorf("stat %s: %w", path, err)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("expected file: %s", path)
	}

	return nil
}

func (d Dir) Validate(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing directory: %s", path)
		}
		return fmt.Errorf("stat %s: %w", path, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("expected directory: %s", path)
	}

	for name, child := range d {
		childPath := filepath.Join(path, name)

		if err := child.Validate(childPath); err != nil {
			return err
		}
	}

	return nil
}

func (o Optional) Validate(path string) error {
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	return o.Node.Validate(path)
}

func (a AtLeastOne) Validate(pattern string) error {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern %q: %w", pattern, err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("expected at least one match for %s", pattern)
	}

	for _, match := range matches {
		if err := a.Node.Validate(match); err != nil {
			return err
		}
	}

	return nil
}
