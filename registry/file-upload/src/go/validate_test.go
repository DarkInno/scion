package fileupload

import (
	"errors"
	"testing"
)

func TestValidateContentDetectsMagicBytes(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	ft, err := ValidateContent(png, []string{"image/png"})
	if err != nil {
		t.Fatalf("ValidateContent: %v", err)
	}
	if ft.MIME != "image/png" || ft.Extension != ".png" {
		t.Fatalf("unexpected type: %+v", ft)
	}

	if _, err := ValidateContent([]byte("%PDF-1.7"), []string{"image/png"}); !errors.Is(err, ErrTypeNotAllowed) {
		t.Fatalf("PDF should be rejected by whitelist: %v", err)
	}
	if _, err := ValidateContent(nil, []string{"image/png"}); !errors.Is(err, ErrEmptyFile) {
		t.Fatalf("empty file = %v", err)
	}
	if _, err := ValidateContent([]byte("plain"), []string{"image/png"}); !errors.Is(err, ErrUnknownType) {
		t.Fatalf("unknown file = %v", err)
	}
}

func TestValidateSanitizeFilename(t *testing.T) {
	got := SanitizeFilename("../bad\r\nname\x00.png")
	if got != "badname.png" {
		t.Fatalf("sanitized name = %q", got)
	}
}
