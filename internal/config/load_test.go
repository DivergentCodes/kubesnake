package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromFile_Valid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	raw := []byte("{\n  \"e2e\": { \"beaconUrl\": \"http://example\" }\n}\n")
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfigFromFile(p)
	if err != nil {
		t.Fatalf("LoadConfigFromFile: %v", err)
	}
	if cfg == nil || cfg.E2E == nil {
		t.Fatalf("expected e2e config")
	}
	if got, want := cfg.E2E.BeaconURL, "http://example"; got != want {
		t.Fatalf("beaconUrl mismatch: got %q want %q", got, want)
	}
}

func TestLoadConfigFromFile_RejectsUnknownFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	raw := []byte(`{"e2e":{"beaconUrl":"http://example"},"extra":true}`)
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadConfigFromFile(p)
	if err == nil {
		t.Fatalf("expected error for unknown field")
	}
}

func TestLoadConfigFromFile_RejectsTrailingTokens(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	raw := []byte(`{"e2e":{"beaconUrl":"http://example"}} {"another":1}`)
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadConfigFromFile(p)
	if err == nil {
		t.Fatalf("expected error for trailing JSON tokens")
	}
}

func TestLoadEmbeddedConfigFromExecutable_NoConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "fakebin")
	if err := os.WriteFile(p, []byte("HELLO"), 0o755); err != nil {
		t.Fatalf("write fake bin: %v", err)
	}

	cfg, err := LoadEmbeddedConfigFromExecutable(p)
	if err != nil {
		t.Fatalf("LoadEmbeddedConfigFromExecutable: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config")
	}
}

func TestLoadEmbeddedConfigFromExecutable_Valid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "fakebin")
	if err := os.WriteFile(p, []byte("BIN"), 0o755); err != nil {
		t.Fatalf("write fake bin: %v", err)
	}

	// Embed a valid config payload (schema-valid; strict parser).
	embedded := []byte(`{"e2e":{"beaconUrl":"http://example"}}`)
	if err := EmbedConfigDataIntoExecutable(p, embedded); err != nil {
		t.Fatalf("EmbedConfigDataIntoExecutable: %v", err)
	}

	cfg, err := LoadEmbeddedConfigFromExecutable(p)
	if err != nil {
		t.Fatalf("LoadEmbeddedConfigFromExecutable: %v", err)
	}
	if cfg == nil || cfg.E2E == nil {
		t.Fatalf("expected e2e config")
	}
	if got, want := cfg.E2E.BeaconURL, "http://example"; got != want {
		t.Fatalf("beaconUrl mismatch: got %q want %q", got, want)
	}
}

func TestLoadEmbeddedConfigFromSelf_NoError(t *testing.T) {
	t.Parallel()

	// The test binary typically does not have embedded config. We only assert this path
	// is safe and does not error.
	if _, err := LoadEmbeddedConfigFromSelf(); err != nil {
		t.Fatalf("LoadEmbeddedConfigFromSelf: %v", err)
	}
}
