//go:build !windows

package filesystem

import (
	"fmt"
	"os"
)

// checkSecretAccess requires the permission bits of path to equal perm exactly.
func checkSecretAccess(path, kind string, info os.FileInfo, perm os.FileMode) error {
	if got := info.Mode().Perm(); got != perm {
		return fmt.Errorf("secret %s %s must have permissions %o, got %o", kind, path, perm, got)
	}
	return nil
}
