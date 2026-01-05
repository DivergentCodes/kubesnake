package config

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
)

// LoadConfigFromFile reads and parses a JSON config file.
func LoadConfigFromFile(cfgPath string) (*Config, error) {
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	return parseConfigJSON(raw)
}

// LoadEmbeddedConfigFromSelf loads the embedded config from the current executable.
// If no embedded config exists, it returns (nil, nil).
func LoadEmbeddedConfigFromSelf() (*Config, error) {
	exePath, err := selfExecutablePath()
	if err != nil {
		return nil, err
	}
	return LoadEmbeddedConfigFromExecutable(exePath)
}

// LoadEmbeddedConfigFromExecutable loads and parses an embedded config from a specific executable path.
// If no embedded config exists, it returns (nil, nil).
func LoadEmbeddedConfigFromExecutable(exePath string) (*Config, error) {
	raw, ok, err := loadEmbeddedConfigData(exePath)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	return parseConfigJSON(raw)
}

// loadEmbeddedConfigData reads an embedded config from the given executable path.
// If no embedded config exists, it returns (nil, false, nil).
func loadEmbeddedConfigData(exePath string) (raw []byte, ok bool, err error) {
	f, err := os.Open(exePath)
	if err != nil {
		return nil, false, fmt.Errorf("open executable: %w", err)
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, false, fmt.Errorf("stat executable: %w", err)
	}
	if st.Size() < footerSize {
		return nil, false, nil
	}

	// Footer exists?
	ft, err := readFooter(f, st.Size())
	if err != nil {
		if errors.Is(err, errNoEmbeddedConfig) {
			return nil, false, nil
		}
		return nil, false, err
	}

	// Empty config?
	if ft.length == 0 {
		// Treat empty config as "not present" to avoid surprising behavior.
		return nil, false, nil
	}

	// Config too large?
	if ft.length > maxConfigSize {
		return nil, false, fmt.Errorf("embedded config too large: %d bytes", ft.length)
	}

	// Config offset underflow?
	cfgOffset := st.Size() - footerSize - int64(ft.length)
	if cfgOffset < 0 {
		return nil, false, fmt.Errorf("embedded config offset underflow")
	}

	// Valid config offset?
	if _, err := f.Seek(cfgOffset, io.SeekStart); err != nil {
		return nil, false, fmt.Errorf("seek embedded config: %w", err)
	}

	// Read config bytes?
	raw = make([]byte, ft.length)
	if _, err := io.ReadFull(f, raw); err != nil {
		return nil, false, fmt.Errorf("read embedded config: %w", err)
	}

	// Config checksum mismatch?
	if got := crc32.ChecksumIEEE(raw); got != ft.crc32 {
		return nil, false, fmt.Errorf("embedded config checksum mismatch")
	}

	return raw, true, nil
}
