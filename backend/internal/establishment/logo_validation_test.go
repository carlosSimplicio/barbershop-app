package establishment

import (
	"bytes"
	"crypto/sha256"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateLogoAcceptsSupportedImageFormats(t *testing.T) {
	webpBytes, err := os.ReadFile(filepath.Join("testdata", "valid-logo.webp"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		content   []byte
		mediaType string
		width     int
		height    int
	}{
		{name: "png", content: encodeLogoPNG(t, image.Rect(0, 0, 2, 3)), mediaType: "image/png", width: 2, height: 3},
		{name: "jpeg", content: encodeLogoJPEG(t), mediaType: "image/jpeg", width: 3, height: 2},
		{name: "webp", content: webpBytes, mediaType: "image/webp", width: 150, height: 100},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ValidateLogo(bytes.NewReader(test.content))
			if err != nil {
				t.Fatalf("ValidateLogo() error = %v", err)
			}
			if got.MediaType != test.mediaType {
				t.Errorf("MediaType = %q, want %q", got.MediaType, test.mediaType)
			}
			validatedContent, err := io.ReadAll(got.ContentReader())
			if err != nil {
				t.Fatalf("read validated content: %v", err)
			}
			if !bytes.Equal(validatedContent, test.content) {
				t.Error("validated content differs from input")
			}
			if got.SizeBytes != int64(len(test.content)) {
				t.Errorf("SizeBytes = %d, want %d", got.SizeBytes, len(test.content))
			}
			if got.Width != test.width || got.Height != test.height {
				t.Errorf("dimensions = %dx%d, want %dx%d", got.Width, got.Height, test.width, test.height)
			}
			if got.SHA256 != sha256.Sum256(test.content) {
				t.Error("SHA256 does not match input bytes")
			}
		})
	}
}

func TestValidateLogoEnforcesByteLimitBeforeDecoding(t *testing.T) {
	content := encodeLogoPNG(t, image.Rect(0, 0, 1, 1))
	exactLimit := make([]byte, MaxLogoBytes)
	copy(exactLimit, content)
	if _, err := ValidateLogo(bytes.NewReader(exactLimit)); err != nil {
		t.Fatalf("ValidateLogo() rejected a valid image at the byte limit: %v", err)
	}

	reader := &countingReader{reader: bytes.NewReader(make([]byte, MaxLogoBytes*2))}
	if _, err := ValidateLogo(reader); err != ErrLogoTooLarge {
		t.Fatalf("ValidateLogo() error = %v, want ErrLogoTooLarge", err)
	}
	if reader.readBytes != MaxLogoBytes+1 {
		t.Errorf("read %d bytes from oversized input, want bounded read of %d", reader.readBytes, MaxLogoBytes+1)
	}
}

func TestValidateLogoRejectsUnsupportedAndMalformedContent(t *testing.T) {
	validPNG := encodeLogoPNG(t, image.Rect(0, 0, 2, 2))
	truncated := validPNG[:len(validPNG)-8]
	tests := []struct {
		name    string
		content []byte
	}{
		{name: "empty", content: nil},
		{name: "svg", content: []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)},
		{name: "gif", content: encodeLogoGIF(t)},
		{name: "truncated image", content: truncated},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ValidateLogo(bytes.NewReader(test.content)); err != ErrInvalidLogo {
				t.Fatalf("ValidateLogo() error = %v, want ErrInvalidLogo", err)
			}
		})
	}
}

func TestValidateLogoRejectsDimensionsAboveLimit(t *testing.T) {
	for _, bounds := range []image.Rectangle{
		image.Rect(0, 0, MaxLogoDimension+1, 1),
		image.Rect(0, 0, 1, MaxLogoDimension+1),
	} {
		t.Run(bounds.String(), func(t *testing.T) {
			content := encodeLogoPNG(t, bounds)
			if _, err := ValidateLogo(bytes.NewReader(content)); err != ErrLogoDimensions {
				t.Fatalf("ValidateLogo() error = %v, want ErrLogoDimensions", err)
			}
		})
	}
}

func encodeLogoPNG(t *testing.T, bounds image.Rectangle) []byte {
	t.Helper()
	var content bytes.Buffer
	if err := png.Encode(&content, image.NewRGBA(bounds)); err != nil {
		t.Fatal(err)
	}
	return content.Bytes()
}

func encodeLogoJPEG(t *testing.T) []byte {
	t.Helper()
	var content bytes.Buffer
	if err := jpeg.Encode(&content, image.NewRGBA(image.Rect(0, 0, 3, 2)), nil); err != nil {
		t.Fatal(err)
	}
	return content.Bytes()
}

func encodeLogoGIF(t *testing.T) []byte {
	t.Helper()
	var content bytes.Buffer
	if err := gif.Encode(&content, image.NewRGBA(image.Rect(0, 0, 1, 1)), nil); err != nil {
		t.Fatal(err)
	}
	return content.Bytes()
}

type countingReader struct {
	reader    *bytes.Reader
	readBytes int
}

func (reader *countingReader) Read(buffer []byte) (int, error) {
	n, err := reader.reader.Read(buffer)
	reader.readBytes += n
	return n, err
}
