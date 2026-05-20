package domain_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/kidboy-man/memora/internal/domain"
)

func TestSHA256Hash(t *testing.T) {
	t.Run("returns 32 bytes", func(t *testing.T) {
		h := domain.SHA256Hash("hello")
		if len(h) != 32 {
			t.Errorf("expected 32 bytes, got %d", len(h))
		}
	})

	t.Run("deterministic", func(t *testing.T) {
		h1 := domain.SHA256Hash("the sky is blue")
		h2 := domain.SHA256Hash("the sky is blue")
		if !bytes.Equal(h1, h2) {
			t.Error("same input produced different hashes")
		}
	})

	t.Run("different inputs produce different hashes", func(t *testing.T) {
		h1 := domain.SHA256Hash("foo")
		h2 := domain.SHA256Hash("bar")
		if bytes.Equal(h1, h2) {
			t.Error("different inputs produced the same hash")
		}
	})

	t.Run("known value", func(t *testing.T) {
		// echo -n "hello" | sha256sum → 2cf24db...
		want, _ := hex.DecodeString("2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824")
		got := domain.SHA256Hash("hello")
		if !bytes.Equal(got, want) {
			t.Errorf("SHA256Hash(\"hello\") = %x, want %x", got, want)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		// sha256 of empty string is well-known
		want, _ := hex.DecodeString("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
		got := domain.SHA256Hash("")
		if !bytes.Equal(got, want) {
			t.Errorf("SHA256Hash(\"\") = %x, want %x", got, want)
		}
	})

	t.Run("normalized vs unnormalized differ without normalization", func(t *testing.T) {
		// Callers must normalize before hashing — this test documents that
		// SHA256Hash itself does NOT normalize.
		h1 := domain.SHA256Hash("hello world")
		h2 := domain.SHA256Hash("Hello  World")
		if bytes.Equal(h1, h2) {
			t.Error("expected different hashes for non-normalized inputs")
		}
	})

	t.Run("normalized inputs produce same hash", func(t *testing.T) {
		n1 := domain.NormalizeContent("Hello  World")
		n2 := domain.NormalizeContent("hello world")
		h1 := domain.SHA256Hash(n1)
		h2 := domain.SHA256Hash(n2)
		if !bytes.Equal(h1, h2) {
			t.Error("normalized inputs should produce same hash")
		}
	})
}
