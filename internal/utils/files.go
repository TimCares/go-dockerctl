package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

const (
	SecretFilePerm os.FileMode = 0o600
	// User can read + write, others can only read.
	UserFilePerm  os.FileMode = 0o644
	UserDirPerm   os.FileMode = 0o755
	SecretDirPerm os.FileMode = 0o700
)

// configVersion is decoded before [Config],
// so an incompatible file is rejected before its schema is parsed.
type configVersion struct {
	APIVersion uint `yaml:"apiVersion"`
}

// CheckMajorAPIVersionMatch finds the first occurrence of "apiVersion" in "rawBody"
// (must be at the beginning of a line) and checks whether the version matches
// with apiVersion.
func CheckMajorAPIVersionMatch(rawBody []byte, apiVersion uint) (*uint, error) {
	var cfg configVersion
	if err := yaml.Unmarshal(rawBody, &cfg); err != nil {
		return nil, fmt.Errorf("parsing file, missing apiVersion: %w", err)
	}

	if cfg.APIVersion == apiVersion {
		return &cfg.APIVersion, nil
	}
	if cfg.APIVersion > apiVersion {
		return nil, fmt.Errorf("apiVersion too high, found %d, expected %d", cfg.APIVersion, apiVersion)
	}
	// Here: cfg.APIVersion < apiVersion. Calling code should apply migrations.
	return &cfg.APIVersion, nil
}

func FilterFilesExist(filePaths []string) ([]string, error) {
	existingFilePaths := make([]string, 0, len(filePaths))
	for _, filePath := range filePaths {
		_, err := os.Stat(filePath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return []string{}, err
			}
			// Error, but means the file does not exist. Therefore: Skip the file.
		} else {
			// No error, file exsits.
			existingFilePaths = append(existingFilePaths, filePath)
		}
	}
	return existingFilePaths, nil
}

func AtomicWrite(filePath string, data []byte, perm os.FileMode) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(filePath), filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("creating temporary file for atomic write of %s: %w", filePath, err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if err := tmpFile.Chmod(perm); err != nil {
		return fmt.Errorf("atomic write of %s: %w", filePath, err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("atomic write of %s: %w", filePath, err)
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("atomic write of %s: %w", filePath, err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("atomic write of %s: %w", filePath, err)
	}

	return os.Rename(tmpFile.Name(), filePath)
}
