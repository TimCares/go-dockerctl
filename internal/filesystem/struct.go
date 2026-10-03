// Package filesystem describes the expected layout of a dockerctl project as a tree of
// [Node] values and validates it against the disk.
package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
)

// Node is one entry in the expected project layout.
type Node interface {
	// Validate returns an error if the entry at path does not match the node.
	Validate(path string) error
}

// File expects a regular file.
type File struct {
	// Secret requires that only the owner can read and write: mode 0600 on Unix,
	// see [checkSecretAccess] for Windows.
	Secret bool
}

// Dir expects a directory containing at least the given Children, keyed by name.
// Entries not listed in Children are allowed.
type Dir struct {
	// Secret requires that only the owner can read, write, and enter: mode 0700 on Unix,
	// see [checkSecretAccess] for Windows. It is not inherited by Children.
	Secret   bool
	Children map[string]Node
}

// Optional accepts a missing entry, but validates Node if the entry exists.
type Optional struct {
	Node Node
}

// AtLeastOne treats its path as a glob pattern and requires at least one match.
// Every match must satisfy Node.
type AtLeastOne struct {
	Node Node
}

const (
	secretFilePerm os.FileMode = 0o600
	secretDirPerm  os.FileMode = 0o700
)

func stat(path, kind string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("missing %s: %s", kind, path)
	}
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	return info, nil
}

// Validate implements [Node].
func (f File) Validate(path string) error {
	info, err := stat(path, "file")
	if err != nil {
		return err
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("expected file: %s", path)
	}

	if f.Secret {
		return checkSecretAccess(path, "file", info, secretFilePerm)
	}
	return nil
}

// Validate implements [Node]. It stops at the first invalid child.
func (d Dir) Validate(path string) error {
	info, err := stat(path, "directory")
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("expected directory: %s", path)
	}

	if d.Secret {
		if err := checkSecretAccess(path, "directory", info, secretDirPerm); err != nil {
			return err
		}
	}

	for name, child := range d.Children {
		if err := child.Validate(filepath.Join(path, name)); err != nil {
			return err
		}
	}

	return nil
}

// Validate implements [Node].
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

// Validate implements [Node], with pattern used as a glob.
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
