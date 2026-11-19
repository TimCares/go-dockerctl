package project

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/TimCares/go-dockerctl/internal/utils"
	"github.com/TimCares/go-see"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ProjectMetadata struct {
	Id uuid.UUID `json:"id"`
}

func ReadProjectMetaFile(projectPath string) (*ProjectMetadata, error) {
	path := MetaFile(projectPath)

	projectsMetaFileBody, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil // File does not exist, which we do not treat as an unexpected error.
	}

	if err != nil {
		return nil, fmt.Errorf("reading project meta file %s: %w", path, err)
	}

	var projectMetadata ProjectMetadata
	if err := json.Unmarshal(projectsMetaFileBody, &projectMetadata); err != nil {
		return nil, fmt.Errorf("parsing project meta file %s: %w", path, err)
	}
	return &projectMetadata, nil
}

func WriteProjectMetaFile(projectPath string, metadata *ProjectMetadata) error {
	path := MetaFile(projectPath)

	data, err := json.MarshalIndent(metadata, "", "  ") // Indent by 2.
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), utils.UserDirPerm); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, utils.UserFilePerm); err != nil {
		return err
	}

	return nil
}

// Add .dockerctl/project.json to .gitignore if a git repo exists in the project or any parent.
//
// Log but do not surface errors.
// We do not use an extra git package, normals ops are sufficient.
func MaybeAddProjectMetaFileToGitignore(ctx context.Context, projectPath string) {
	gitRepoPath, err := tryFindGitRepoInParents(projectPath)
	if err != nil {
		see.L(ctx).Warn("Error when search for git repo in project dir and any parent.")
		return
	}
	if gitRepoPath == "" {
		return
	}
	gitignoreFilePath := filepath.Join(gitRepoPath, ".gitignore")
	gitignoreFile, err := os.OpenFile(gitignoreFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, utils.UserFilePerm)
	if err != nil {
		return
	}
	defer gitignoreFile.Close()
	gitignoreContent, err := io.ReadAll(gitignoreFile)
	if err != nil {
		return
	}

	if gitignoreAlreadyContainsIgnore(string(gitignoreContent)) {
		return
	}
	// Else:

	projectPathAbs, err := filepath.Abs(projectPath)
	if err != nil {
		return
	}
	gitRepoPathAbs, err := filepath.Abs(gitRepoPath)
	if err != nil {
		return
	}

	pathToMetafile := MetaFile(projectPathAbs)
	pathToMetafileRel, err := filepath.Rel(gitRepoPathAbs, pathToMetafile)
	if err != nil {
		return
	}
	addString := fmt.Sprintf("\n# Dockerctl project.json (Added automatically by dockerctl)\n%s\n", pathToMetafileRel)
	gitignoreFile.WriteString(addString)

	see.L(ctx).Info("Added dockerctl metafile to .gitignore at: %s. Reason: Should not be commited.", zap.String("pathToGitRepo", gitRepoPathAbs))

}

func gitignoreAlreadyContainsIgnore(gitignoreContent string) bool {
	projectMetaFileIdentifier := MetaFile("")
	for ignoreFilePath := range strings.SplitSeq(gitignoreContent, "\n") {
		if strings.HasPrefix(ignoreFilePath, "#") {
			continue // Is comment line.
		}
		ignoreFilePathAbs, err := filepath.Abs(ignoreFilePath)
		if err != nil {
			continue
		}
		if strings.Contains(ignoreFilePathAbs, projectMetaFileIdentifier) {
			return true
		}
	}

	return false
}

func tryFindGitRepoInParents(startPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dirPath := startPath
	for {
		if isGitRepo(dirPath) {
			return dirPath, nil
		}

		if dirPath == home {
			break
		}

		parent := filepath.Dir(dirPath)
		if parent == dirPath {
			break // Reached filesystem root.
		}

		dirPath = parent
	}

	return "", nil
}

func isGitRepo(dirPath string) bool {
	_, err := os.Stat(filepath.Join(dirPath, ".git"))
	return err == nil
}
