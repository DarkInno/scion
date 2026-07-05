package registry

import (
	"bytes"
	"testing"
)

func TestCanonicalFileBytesNormalizesTextLineEndings(t *testing.T) {
	data := CanonicalFileBytes("registry/cache/README.md", []byte("a\r\nb\rc\n"))
	if string(data) != "a\nb\nc\n" {
		t.Fatalf("unexpected canonical text: %q", data)
	}
}

func TestCanonicalFileBytesLeavesBinaryExtensionsUnchanged(t *testing.T) {
	in := []byte{0x89, 'P', 'N', 'G', '\r', '\n'}
	out := CanonicalFileBytes("registry/assets/logo.png", in)
	if !bytes.Equal(out, in) {
		t.Fatalf("binary data changed: %v", out)
	}
}
