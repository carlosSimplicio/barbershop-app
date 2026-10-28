package establishment

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"

	_ "golang.org/x/image/webp"
)

const (
	MaxLogoBytes     = 2 * 1024 * 1024
	MaxLogoDimension = 4096
)

var (
	ErrInvalidLogo    = errors.New("invalid establishment logo")
	ErrLogoTooLarge   = errors.New("establishment logo exceeds the size limit")
	ErrLogoDimensions = errors.New("establishment logo dimensions exceed the limit")
)

type ValidatedLogo struct {
	content   []byte
	MediaType string
	SizeBytes int64
	Width     int
	Height    int
	SHA256    [sha256.Size]byte
}

func (logo ValidatedLogo) ContentReader() io.Reader {
	return bytes.NewReader(logo.content)
}

// ValidateLogo checks the image bytes and returns metadata derived from the content.
func ValidateLogo(reader io.Reader) (ValidatedLogo, error) {
	if reader == nil {
		return ValidatedLogo{}, ErrInvalidLogo
	}

	content, err := io.ReadAll(io.LimitReader(reader, MaxLogoBytes+1))
	if err != nil {
		return ValidatedLogo{}, ErrInvalidLogo
	}
	if len(content) > MaxLogoBytes {
		return ValidatedLogo{}, ErrLogoTooLarge
	}

	config, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		return ValidatedLogo{}, ErrInvalidLogo
	}
	mediaType, ok := supportedLogoMediaTypes[format]
	if !ok {
		return ValidatedLogo{}, ErrInvalidLogo
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > MaxLogoDimension || config.Height > MaxLogoDimension {
		return ValidatedLogo{}, ErrLogoDimensions
	}

	decoded, decodedFormat, err := image.Decode(bytes.NewReader(content))
	if err != nil || decodedFormat != format || decoded.Bounds().Dx() != config.Width || decoded.Bounds().Dy() != config.Height {
		return ValidatedLogo{}, ErrInvalidLogo
	}

	return ValidatedLogo{
		content:   content,
		MediaType: mediaType,
		SizeBytes: int64(len(content)),
		Width:     config.Width,
		Height:    config.Height,
		SHA256:    sha256.Sum256(content),
	}, nil
}

var supportedLogoMediaTypes = map[string]string{
	"png":  "image/png",
	"jpeg": "image/jpeg",
	"webp": "image/webp",
}

func isSupportedLogoMediaType(mediaType string) bool {
	for _, supportedMediaType := range supportedLogoMediaTypes {
		if mediaType == supportedMediaType {
			return true
		}
	}
	return false
}
