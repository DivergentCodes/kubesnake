package config

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
)

// TestReadEmbeddedConfig_NoConfig tests the ReadEmbeddedConfig return value
// when no embedded config is present.
func TestReadFromExecutable_NoConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "fakebin")
	if err := os.WriteFile(p, []byte("HELLO"), 0o755); err != nil {
		t.Fatalf("write fake bin: %v", err)
	}

	raw, ok, err := loadEmbeddedConfigData(p)
	if err != nil {
		t.Fatalf("ReadFromExecutable: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false")
	}
	if raw != nil {
		t.Fatalf("expected nil raw")
	}
}

// TestEmbedIntoExecutable_ReadBack tests that EmbedIntoExecutable can
// read back the embedded config.
func TestEmbedIntoExecutable_ReadBack(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "fakebin")
	orig := []byte("ORIGINAL_BINARY_BYTES")
	if err := os.WriteFile(p, orig, 0o755); err != nil {
		t.Fatalf("write fake bin: %v", err)
	}

	cfg := []byte(`{"a":1,"b":"two"}`)
	if err := EmbedConfigDataIntoExecutable(p, cfg); err != nil {
		t.Fatalf("EmbedIntoExecutable: %v", err)
	}

	raw, ok, err := loadEmbeddedConfigData(p)
	if err != nil {
		t.Fatalf("ReadFromExecutable: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if !bytes.Equal(raw, cfg) {
		t.Fatalf("config mismatch: got %q want %q", string(raw), string(cfg))
	}

	after, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read rewritten bin: %v", err)
	}
	if !bytes.HasPrefix(after, orig) {
		t.Fatalf("expected rewritten bin to retain original prefix")
	}
}

func TestEmbedConfigFileIntoExecutable_RejectsUnknownFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	exe := filepath.Join(dir, "fakebin")
	if err := os.WriteFile(exe, []byte("BIN"), 0o755); err != nil {
		t.Fatalf("write fake bin: %v", err)
	}

	cfgPath := filepath.Join(dir, "config.json")
	// Unknown top-level field "extra" should be rejected by strict parser.
	if err := os.WriteFile(cfgPath, []byte(`{"e2e":{"beaconUrl":"http://example"},"extra":true}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := EmbedConfigFileIntoExecutable(exe, cfgPath); err == nil {
		t.Fatalf("expected error")
	}
}

func TestEmbedConfigFileIntoExecutable_EmbedsFileBytes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	exe := filepath.Join(dir, "fakebin")
	orig := []byte("BIN")
	if err := os.WriteFile(exe, orig, 0o755); err != nil {
		t.Fatalf("write fake bin: %v", err)
	}

	cfgPath := filepath.Join(dir, "config.json")
	raw := []byte("{\n  \"e2e\": { \"beaconUrl\": \"http://example\" }\n}\n")
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := EmbedConfigFileIntoExecutable(exe, cfgPath); err != nil {
		t.Fatalf("EmbedConfigFileIntoExecutable: %v", err)
	}

	got, ok, err := loadEmbeddedConfigData(exe)
	if err != nil {
		t.Fatalf("loadEmbeddedConfigData: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("embedded bytes mismatch: got %q want %q", string(got), string(raw))
	}
}

// TestEmbedIntoExecutable_OverwritesExisting tests that EmbedIntoExecutable can
// overwrite an existing embedded config.
func TestEmbedIntoExecutable_OverwritesExisting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "fakebin")
	orig := []byte("BIN")
	if err := os.WriteFile(p, orig, 0o700); err != nil {
		t.Fatalf("write fake bin: %v", err)
	}

	if err := EmbedConfigDataIntoExecutable(p, []byte(`{"one":1}`)); err != nil {
		t.Fatalf("EmbedIntoExecutable first: %v", err)
	}
	if err := EmbedConfigDataIntoExecutable(p, []byte(`{"two":2}`)); err != nil {
		t.Fatalf("EmbedIntoExecutable second: %v", err)
	}

	raw, ok, err := loadEmbeddedConfigData(p)
	if err != nil {
		t.Fatalf("ReadFromExecutable: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if string(raw) != `{"two":2}` {
		t.Fatalf("expected second config, got %q", string(raw))
	}

	st, err := os.Stat(p)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got, want := st.Mode().Perm(), os.FileMode(0o700); got != want {
		t.Fatalf("mode changed: got %o want %o", got, want)
	}
}

// TestReadFromExecutable_ChecksumMismatch tests that ReadEmbeddedConfig returns
// an error when the embedded config checksum mismatch.
func TestReadFromExecutable_ChecksumMismatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "fakebin")
	if err := os.WriteFile(p, []byte("BIN"), 0o755); err != nil {
		t.Fatalf("write fake bin: %v", err)
	}

	cfg := []byte(`{"a":"b"}`)
	if err := EmbedConfigDataIntoExecutable(p, cfg); err != nil {
		t.Fatalf("EmbedIntoExecutable: %v", err)
	}

	// Corrupt the embedded config bytes in-place (flip the last byte of config).
	all, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read bin: %v", err)
	}
	if len(all) < footerSize+1 {
		t.Fatalf("unexpected size")
	}
	all[len(all)-footerSize-1] ^= 0xFF
	if err := os.WriteFile(p, all, 0o755); err != nil {
		t.Fatalf("rewrite bin: %v", err)
	}

	_, _, err = loadEmbeddedConfigData(p)
	if err == nil {
		t.Fatalf("expected checksum error")
	}
}

// TestReadFromExecutable_InvalidLengthUnderflow tests that ReadEmbeddedConfig
// returns an error when the embedded config length underflows.
func TestReadFromExecutable_InvalidLengthUnderflow(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "fakebin")

	// Construct a file that only contains a valid footer claiming a non-zero
	// length, which must underflow when computing the config offset.
	buf := make([]byte, footerSize)
	copy(buf[:16], []byte(footerMagic))
	binary.LittleEndian.PutUint32(buf[16:20], 1234) // length
	binary.LittleEndian.PutUint32(buf[20:24], crc32.ChecksumIEEE([]byte("x")))

	if err := os.WriteFile(p, buf, 0o755); err != nil {
		t.Fatalf("write fake bin: %v", err)
	}

	_, _, err := loadEmbeddedConfigData(p)
	if err == nil {
		t.Fatalf("expected error")
	}
}

// TestEmbedIntoExecutable_RejectsEmptyAndTooLarge tests that EmbedIntoExecutable
// returns an error when the config is empty or too large.
func TestEmbedIntoExecutable_RejectsEmptyAndTooLarge(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "fakebin")
	if err := os.WriteFile(p, []byte("BIN"), 0o755); err != nil {
		t.Fatalf("write fake bin: %v", err)
	}

	if err := EmbedConfigDataIntoExecutable(p, nil); err == nil {
		t.Fatalf("expected error for empty config")
	}

	tooBig := make([]byte, maxConfigSize+1)
	if err := EmbedConfigDataIntoExecutable(p, tooBig); err == nil {
		t.Fatalf("expected error for too large config")
	}
}
