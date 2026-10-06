package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, body string) (dir, path string) {
	t.Helper()
	dir = t.TempDir()
	path = filepath.Join(dir, "dockerctl.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, path
}

func TestLoadKeepsDefaultsForOmittedFields(t *testing.T) {
	dir, path := writeConfig(t, `
apiVersion: 1
name: demo
envs:
  - dev
serviceGroups:
  - name: web
`)

	cfg, err := Load(context.Background(), Settings{
		ProjectPath: dir,
		ConfigPath:  path,
		Env:         "dev",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !cfg.ManageSOPSIdentities {
		t.Fatal("expected manageSopsIdentities to stay at its default")
	}
	if cfg.Observability.Logging.MessageKey != "msg" {
		t.Fatalf("message key: got %q", cfg.Observability.Logging.MessageKey)
	}
	if cfg.Observability.Logging.LogLevel != "info" {
		t.Fatalf("log level: got %q", cfg.Observability.Logging.LogLevel)
	}
	if cfg.Observability.Logging.LogFormat != "console" {
		t.Fatalf("log format: got %q", cfg.Observability.Logging.LogFormat)
	}
	if cfg.Observability.Logging.LogFile != DefaultLogFile() {
		t.Fatalf("log file: got %q", cfg.Observability.Logging.LogFile)
	}
	if cfg.Observability.Otel.Enabled {
		t.Fatal("expected otel to stay disabled")
	}
	if cfg.ServiceGroups[0].DockerComposeFile != DockerComposeDefaultFileName {
		t.Fatalf("compose file: got %q", cfg.ServiceGroups[0].DockerComposeFile)
	}
	wantPath := filepath.Join(dir, ServiceGroupsDefaultDirName, "web")
	if cfg.ServiceGroups[0].Path != wantPath {
		t.Fatalf("service group path: got %q, want %q", cfg.ServiceGroups[0].Path, wantPath)
	}
}

func TestLoadFileOverridesDefaults(t *testing.T) {
	dir, path := writeConfig(t, `
apiVersion: 1
name: demo
manageSopsIdentities: false
envs:
  - dev
serviceGroups:
  - name: web
observability:
  logging:
    logLevel: DEBUG
    logFormat: json
    logFile: none
`)

	cfg, err := Load(context.Background(), Settings{
		ProjectPath: dir,
		ConfigPath:  path,
		Env:         "dev",
	})
	if err != nil {
		t.Fatal(err)
	}

	if cfg.ManageSOPSIdentities {
		t.Fatal("expected manageSopsIdentities from the file")
	}
	if cfg.Observability.Logging.LogLevel != "debug" {
		t.Fatalf("log level: got %q", cfg.Observability.Logging.LogLevel)
	}
	if cfg.Observability.Logging.LogFormat != "json" {
		t.Fatalf("log format: got %q", cfg.Observability.Logging.LogFormat)
	}
	if cfg.Observability.Logging.LogFile != "none" {
		t.Fatalf("log file: got %q", cfg.Observability.Logging.LogFile)
	}
	if cfg.Observability.Logging.MessageKey != "msg" {
		t.Fatalf("message key: got %q", cfg.Observability.Logging.MessageKey)
	}
}

func TestLoadExplicitSettingsOverrideFile(t *testing.T) {
	dir, path := writeConfig(t, `
apiVersion: 1
name: demo
envs:
  - dev
serviceGroups:
  - name: web
observability:
  logging:
    logLevel: debug
`)
	level := "error"
	cfg, err := Load(context.Background(), Settings{
		ProjectPath: dir,
		ConfigPath:  path,
		Env:         "dev",
		LogLevel:    &level,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Observability.Logging.LogLevel != "error" {
		t.Fatalf("log level: got %q", cfg.Observability.Logging.LogLevel)
	}
}

func TestLoadEmptyObservabilityStringUsesDefault(t *testing.T) {
	dir, path := writeConfig(t, `
apiVersion: 1
name: demo
envs:
  - dev
serviceGroups:
  - name: web
observability:
  logging:
    logLevel: ""
    logFormat: ""
`)

	cfg, err := Load(context.Background(), Settings{
		ProjectPath: dir,
		ConfigPath:  path,
		Env:         "dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Observability.Logging.LogLevel != "info" {
		t.Fatalf("log level: got %q", cfg.Observability.Logging.LogLevel)
	}
	if cfg.Observability.Logging.LogFormat != "console" {
		t.Fatalf("log format: got %q", cfg.Observability.Logging.LogFormat)
	}
}

func TestLoadOtelValidation(t *testing.T) {
	dir, path := writeConfig(t, `
apiVersion: 1
name: demo
envs:
  - dev
serviceGroups:
  - name: web
observability:
  otel:
    enabled: true
    endpoint: localhost:4317
    secretFile: otel.token
`)

	cfg, err := Load(context.Background(), Settings{
		ProjectPath: dir,
		ConfigPath:  path,
		Env:         "dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	wantSecret := filepath.Join(dir, "otel.token")
	if cfg.Observability.Otel.SecretFile != wantSecret {
		t.Fatalf("secret file: got %q, want %q", cfg.Observability.Otel.SecretFile, wantSecret)
	}

	_, path = writeConfig(t, `
apiVersion: 1
name: demo
envs:
  - dev
serviceGroups:
  - name: web
observability:
  otel:
    enabled: true
`)
	_, err = Load(context.Background(), Settings{
		ProjectPath: dir,
		ConfigPath:  path,
		Env:         "dev",
	})
	if err == nil || !strings.Contains(err.Error(), "endpoint") {
		t.Fatalf("expected missing otel endpoint to fail validation, got %v", err)
	}
}
