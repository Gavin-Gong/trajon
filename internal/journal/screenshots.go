package journal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

const MaxImageBytes = 20 << 20
const MaxImagePixels = 24_000_000
const MaxScreenshots = 20

var ErrImage = errors.New("unsupported or oversized screenshot")

func validScreenshotFile(file string) bool {
	if len(file) != 68 || filepath.Ext(file) != ".png" {
		return false
	}
	_, err := hex.DecodeString(file[:64])
	return err == nil
}

func ReadImageFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > MaxImageBytes {
		return nil, ErrImage
	}
	b, err := io.ReadAll(io.LimitReader(f, MaxImageBytes+1))
	if len(b) > MaxImageBytes {
		return nil, ErrImage
	}
	return b, err
}

// ImportImage validates dimensions before decoding, strips metadata by encoding
// a single PNG frame, and commits a content-addressed local copy before the
// journal can reference it. It never retains an external file path.
func ImportImage(dir, name string, data []byte) (Screenshot, []byte, error) {
	if len(data) == 0 || len(data) > MaxImageBytes {
		return Screenshot{}, nil, ErrImage
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > MaxImagePixels {
		return Screenshot{}, nil, ErrImage
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Screenshot{}, nil, ErrImage
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		return Screenshot{}, nil, err
	}
	if encoded.Len() > MaxImageBytes {
		return Screenshot{}, nil, ErrImage
	}
	sum := sha256.Sum256(encoded.Bytes())
	file := hex.EncodeToString(sum[:]) + ".png"
	if err := atomicWrite(filepath.Join(dir, "images", file), encoded.Bytes()); err != nil {
		return Screenshot{}, nil, err
	}
	return Screenshot{File: file, Name: filepath.Base(name)}, encoded.Bytes(), nil
}

func LoadImage(dir, file string) ([]byte, error) {
	if !validScreenshotFile(file) {
		return nil, ErrImage
	}
	return ReadImageFile(filepath.Join(dir, "images", file))
}
