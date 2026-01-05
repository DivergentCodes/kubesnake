package config

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFooter_ReadFooter_RoundTrip(t *testing.T) {
	t.Parallel()

	var ft footer
	copy(ft.magic[:], []byte(footerMagic))
	ft.length = 123
	ft.crc32 = 456

	var buf bytes.Buffer
	if err := writeFooter(&buf, ft); err != nil {
		t.Fatalf("writeFooter: %v", err)
	}
	if got, want := buf.Len(), footerSize; got != want {
		t.Fatalf("footer size mismatch: got %d want %d", got, want)
	}

	rd := bytes.NewReader(buf.Bytes())
	out, err := readFooter(rd, int64(buf.Len()))
	if err != nil {
		t.Fatalf("readFooter: %v", err)
	}
	if out.length != ft.length || out.crc32 != ft.crc32 || string(out.magic[:]) != footerMagic {
		t.Fatalf("footer mismatch: got=%+v want=%+v", out, ft)
	}
}

func TestReadFooter_InvalidMagic(t *testing.T) {
	t.Parallel()

	// Footer-like bytes but with wrong magic.
	buf := make([]byte, footerSize)
	copy(buf[:16], []byte("NOT_A_REAL_MAGIC!")) // 16 bytes
	binary.LittleEndian.PutUint32(buf[16:20], 1)
	binary.LittleEndian.PutUint32(buf[20:24], 2)

	rd := bytes.NewReader(buf)
	_, err := readFooter(rd, int64(len(buf)))
	if err == nil {
		t.Fatalf("expected error")
	}
	if err != errNoEmbeddedConfig {
		t.Fatalf("expected errNoEmbeddedConfig, got %v", err)
	}
}

func TestExecutableBaseSizeAndMode_NoFooter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "fakebin")
	if err := os.WriteFile(p, []byte("HELLO"), 0o700); err != nil {
		t.Fatalf("write fake bin: %v", err)
	}

	baseSize, mode, err := executableBaseSizeAndMode(p)
	if err != nil {
		t.Fatalf("executableBaseSizeAndMode: %v", err)
	}
	if got, want := baseSize, int64(len("HELLO")); got != want {
		t.Fatalf("base size mismatch: got %d want %d", got, want)
	}
	if got, want := mode.Perm(), os.FileMode(0o700); got != want {
		t.Fatalf("mode mismatch: got %o want %o", got, want)
	}
}

func TestExecutableBaseSizeAndMode_WithEmbeddedConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "fakebin")

	base := []byte("BINPREFIX")
	cfg := []byte(`{"e2e":{"beaconUrl":"http://example"}}`)
	if err := os.WriteFile(p, base, 0o755); err != nil {
		t.Fatalf("write fake bin: %v", err)
	}

	// Append config + footer manually (simulates an embedded config).
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatalf("open for append: %v", err)
	}
	if _, err := f.Write(cfg); err != nil {
		_ = f.Close()
		t.Fatalf("write cfg: %v", err)
	}
	var ft footer
	copy(ft.magic[:], []byte(footerMagic))
	ft.length = uint32(len(cfg))
	ft.crc32 = crc32.ChecksumIEEE(cfg)
	if err := writeFooter(f, ft); err != nil {
		_ = f.Close()
		t.Fatalf("writeFooter: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	baseSize, _, err := executableBaseSizeAndMode(p)
	if err != nil {
		t.Fatalf("executableBaseSizeAndMode: %v", err)
	}
	if got, want := baseSize, int64(len(base)); got != want {
		t.Fatalf("base size mismatch: got %d want %d", got, want)
	}
}

func TestExecutableBaseSizeAndMode_InvalidEmbeddedLength(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p := filepath.Join(dir, "fakebin")

	// Construct a file that only contains a valid footer claiming a non-zero
	// length, which must underflow when computing the base size.
	buf := make([]byte, footerSize)
	copy(buf[:16], []byte(footerMagic))
	binary.LittleEndian.PutUint32(buf[16:20], 1234) // length
	binary.LittleEndian.PutUint32(buf[20:24], crc32.ChecksumIEEE([]byte("x")))

	if err := os.WriteFile(p, buf, 0o755); err != nil {
		t.Fatalf("write fake bin: %v", err)
	}

	_, _, err := executableBaseSizeAndMode(p)
	if err == nil {
		t.Fatalf("expected error")
	}
}
