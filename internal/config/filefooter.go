package config

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	footerMagic   = "KUBESNAKECFGv1\000\000" // 16 bytes (includes trailing NULs)
	footerSize    = 16 + 4 + 4
	maxConfigSize = 256 << 10 // 256 KiB safety limit to prevent writing oversized configs
)

var (
	errNoEmbeddedConfig = errors.New("no embedded config")
)

type footer struct {
	magic  [16]byte
	length uint32
	crc32  uint32
}

// executableBaseSizeAndMode returns the size of the base executable and the mode of the original executable.
func executableBaseSizeAndMode(exePath string) (baseSize int64, mode os.FileMode, err error) {
	f, err := os.Open(exePath)
	if err != nil {
		return 0, 0, fmt.Errorf("open executable: %w", err)
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return 0, 0, fmt.Errorf("stat executable: %w", err)
	}

	mode = st.Mode()
	size := st.Size()
	if size < footerSize {
		return size, mode, nil
	}

	ft, err := readFooter(f, size)
	if err != nil {
		if errors.Is(err, errNoEmbeddedConfig) {
			return size, mode, nil
		}
		return 0, 0, err
	}

	baseSize = size - footerSize - int64(ft.length)
	if baseSize < 0 {
		return 0, 0, fmt.Errorf("invalid embedded config size")
	}
	return baseSize, mode, nil
}

// readFooter reads the embedded footer from the given file.
func readFooter(r io.ReadSeeker, fileSize int64) (footer, error) {
	var ft footer
	if fileSize < footerSize {
		return ft, errNoEmbeddedConfig
	}
	if _, err := r.Seek(fileSize-footerSize, io.SeekStart); err != nil {
		return ft, fmt.Errorf("seek footer: %w", err)
	}

	buf := make([]byte, footerSize)
	if _, err := io.ReadFull(r, buf); err != nil {
		return ft, fmt.Errorf("read footer: %w", err)
	}

	copy(ft.magic[:], buf[:16])
	if string(ft.magic[:]) != footerMagic {
		return footer{}, errNoEmbeddedConfig
	}

	ft.length = binary.LittleEndian.Uint32(buf[16:20])
	ft.crc32 = binary.LittleEndian.Uint32(buf[20:24])
	return ft, nil
}

// writeFooter writes the embedded footer to the given writer.
func writeFooter(w io.Writer, ft footer) error {
	buf := make([]byte, footerSize)
	copy(buf[:16], ft.magic[:])
	binary.LittleEndian.PutUint32(buf[16:20], ft.length)
	binary.LittleEndian.PutUint32(buf[20:24], ft.crc32)
	if _, err := w.Write(buf); err != nil {
		return fmt.Errorf("write footer: %w", err)
	}
	return nil
}
