package fileupload

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestStorageSafeNameRejectsTraversal(t *testing.T) {
	for _, name := range []string{"", "../x", "a/b", `a\b`, "x\x00y", "x\r\ny"} {
		if err := safeName(name); !errors.Is(err, ErrInvalidName) {
			t.Fatalf("safeName(%q) = %v, want ErrInvalidName", name, err)
		}
	}
}

func TestStorageMemoryCopiesData(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage("/files")
	data := []byte("hello")
	url, err := storage.Save(ctx, "hello.txt", data)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if url != "/files/hello.txt" {
		t.Fatalf("url = %q", url)
	}
	data[0] = 'x'
	got, err := storage.Get(ctx, "hello.txt")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, []byte("hello")) {
		t.Fatalf("stored data mutated: %q", got)
	}
	got[0] = 'x'
	again, _ := storage.Get(ctx, "hello.txt")
	if !bytes.Equal(again, []byte("hello")) {
		t.Fatalf("Get should return a copy: %q", again)
	}
	if !storage.Exists(ctx, "hello.txt") {
		t.Fatal("Exists should report saved file")
	}
	if err := storage.Delete(ctx, "hello.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if storage.Exists(ctx, "hello.txt") {
		t.Fatal("file should not exist after delete")
	}
}

func TestMemoryStorageEvictsLeastRecentlyUsed(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage("/files")
	storage.MaxFiles = 2

	if _, err := storage.Save(ctx, "a.txt", []byte("a")); err != nil {
		t.Fatalf("Save a: %v", err)
	}
	if _, err := storage.Save(ctx, "b.txt", []byte("b")); err != nil {
		t.Fatalf("Save b: %v", err)
	}
	if _, err := storage.Get(ctx, "a.txt"); err != nil {
		t.Fatalf("Get a: %v", err)
	}
	if _, err := storage.Save(ctx, "c.txt", []byte("c")); err != nil {
		t.Fatalf("Save c: %v", err)
	}

	if !storage.Exists(ctx, "a.txt") {
		t.Fatal("recently used file should still exist")
	}
	if storage.Exists(ctx, "b.txt") {
		t.Fatal("least recently used file should be evicted")
	}
	if !storage.Exists(ctx, "c.txt") {
		t.Fatal("new file should exist")
	}
	if _, err := storage.Get(ctx, "b.txt"); !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("Get evicted file = %v, want ErrFileNotFound", err)
	}
}
