package config

import (
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
)

// EmbedConfigFileIntoSelf embeds the JSON config at cfgPath into the current executable,
// returning the resolved executable path.
func EmbedConfigFileIntoSelf(cfgPath string) (string, error) {
	exePath, err := selfExecutablePath()
	if err != nil {
		return "", err
	}
	if err := EmbedConfigFileIntoExecutable(exePath, cfgPath); err != nil {
		return "", err
	}
	return exePath, nil
}

// EmbedConfigFileIntoExecutable reads cfgPath, validates it as JSON, and embeds
// the config data into exePath.
func EmbedConfigFileIntoExecutable(exePath, cfgPath string) error {
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	// Validate config using the same strict parser used for loading.
	if _, err := parseConfigJSON(raw); err != nil {
		return fmt.Errorf("invalid config in %s: %w", cfgPath, err)
	}

	return EmbedConfigDataIntoExecutable(exePath, raw)
}

// EmbedConfigDataIntoExecutable embeds config bytes into the given executable path.
// The executable is rewritten to a temporary file in the same directory and atomically
// renamed into place.
func EmbedConfigDataIntoExecutable(exePath string, config []byte) error {
	if len(config) == 0 {
		return fmt.Errorf("config is empty")
	}
	if len(config) > maxConfigSize {
		return fmt.Errorf("config too large: %d bytes (max %d)", len(config), maxConfigSize)
	}

	// Determine how many bytes from the original executable should be kept.
	baseSize, mode, err := executableBaseSizeAndMode(exePath)
	if err != nil {
		return err
	}

	src, err := os.Open(exePath)
	if err != nil {
		return fmt.Errorf("open executable: %w", err)
	}
	defer src.Close()

	// Create a temporary file to write the new executable to.
	dir := filepath.Dir(exePath)
	tmp, err := os.CreateTemp(dir, filepath.Base(exePath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	// Set the mode of the temporary file to the mode of the original executable.
	if err := tmp.Chmod(mode); err != nil {
		return fmt.Errorf("chmod temp executable: %w", err)
	}

	// Copy the original executable bytes to the temporary file.
	if _, err := io.CopyN(tmp, src, baseSize); err != nil {
		return fmt.Errorf("copy executable bytes: %w", err)
	}

	// Write the embedded config to the temporary file.
	if _, err := tmp.Write(config); err != nil {
		return fmt.Errorf("write embedded config: %w", err)
	}

	// Write the footer to the temporary file.
	ft := footer{
		length: uint32(len(config)),
		crc32:  crc32.ChecksumIEEE(config),
	}
	copy(ft.magic[:], []byte(footerMagic))

	// Write the footer to the temporary file.
	if err := writeFooter(tmp, ft); err != nil {
		return err
	}

	// Sync the temporary file to disk.
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temp executable: %w", err)
	}

	// Close the temporary file.
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp executable: %w", err)
	}

	// Replace the original executable with the temporary file.
	if err := os.Rename(tmpPath, exePath); err != nil {
		return fmt.Errorf("replace executable: %w", err)
	}

	// Best-effort: ensure the final file is executable (rename preserves mode,
	// but some filesystems can be surprising).
	_ = os.Chmod(exePath, mode)

	return nil
}
