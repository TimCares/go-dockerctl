package appdirs

import (
	"os"
	"path/filepath"
	"runtime"
)

// configDir is the per-user base dir for config: ~/Library/Application Support on macOS,
// %AppData% on Windows, and $XDG_CONFIG_HOME or ~/.config otherwise.
func configDir() (string, error) {
	return os.UserConfigDir()
}

// stateDir is the per-user base dir for data that can be rebuilt: ~/Library/Application Support
// on macOS, %LocalAppData% on Windows, and $XDG_STATE_HOME or ~/.local/state otherwise.
func stateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support"), nil
	case "windows":
		return localAppData(home), nil
	default:
		return xdgDir("XDG_STATE_HOME", filepath.Join(home, ".local", "state")), nil
	}
}

// logDir is the per-user base dir for logs: ~/Library/Logs on macOS,
// %LocalAppData% on Windows, and $XDG_STATE_HOME or ~/.local/state otherwise.
func logDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Logs"), nil
	case "windows":
		return localAppData(home), nil
	default:
		return xdgDir("XDG_STATE_HOME", filepath.Join(home, ".local", "state")), nil
	}
}

func localAppData(home string) string {
	if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
		return dir
	}
	return filepath.Join(home, "AppData", "Local")
}

// xdgDir follows the XDG spec, which requires an absolute path and ignores a relative one.
func xdgDir(envVar string, fallback string) string {
	if dir := os.Getenv(envVar); filepath.IsAbs(dir) {
		return dir
	}
	return fallback
}
