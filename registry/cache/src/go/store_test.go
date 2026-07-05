package cache

import (
	"strings"
	"testing"
	"time"
)

func TestStoreValidateKeyRejectsUnsafeInput(t *testing.T) {
	tests := map[string]error{
		"":                                  ErrEmptyKey,
		strings.Repeat("a", maxKeyLength+1): ErrKeyTooLong,
		"bad\r\nkey":                        ErrInvalidKey,
		"bad\x00key":                        ErrInvalidKey,
	}
	for key, want := range tests {
		if got := validateKey(key); got != want {
			t.Fatalf("validateKey(%q) = %v, want %v", key, got, want)
		}
	}
	if err := validateKey("safe-key"); err != nil {
		t.Fatalf("safe key rejected: %v", err)
	}
}

func TestStoreEntryAndLRUList(t *testing.T) {
	if (Entry[string]{Value: "ok"}).Expired() {
		t.Fatal("entry without expiration should not expire")
	}
	expired := Entry[string]{Expiration: time.Now().Add(-time.Second).UnixNano()}
	if !expired.Expired() {
		t.Fatal("past expiration should be expired")
	}

	l := newLRUList[string]()
	a := &lruNode[string]{key: "a"}
	b := &lruNode[string]{key: "b"}
	l.pushFront(a)
	l.pushFront(b)
	if l.back() != a {
		t.Fatal("least-recently-used node should be a")
	}
	l.moveFront(a)
	if l.back() != b {
		t.Fatal("moveFront should make b the tail")
	}
	l.remove(a)
	if l.head != b || l.tail != b {
		t.Fatal("remove should leave b as only node")
	}
}
