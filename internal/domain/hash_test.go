package domain_test

import (
	"encoding/hex"
	"testing"

	"github.com/kidboy-man/memora/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSHA256Hash(t *testing.T) {
	t.Run("returns 32 bytes", func(t *testing.T) {
		assert.Len(t, domain.SHA256Hash("hello"), 32)
	})

	t.Run("deterministic", func(t *testing.T) {
		assert.Equal(t, domain.SHA256Hash("the sky is blue"), domain.SHA256Hash("the sky is blue"))
	})

	t.Run("different inputs produce different hashes", func(t *testing.T) {
		assert.NotEqual(t, domain.SHA256Hash("foo"), domain.SHA256Hash("bar"))
	})

	t.Run("known value", func(t *testing.T) {
		// echo -n "hello" | sha256sum
		want, err := hex.DecodeString("2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824")
		require.NoError(t, err)
		assert.Equal(t, want, domain.SHA256Hash("hello"))
	})

	t.Run("empty string", func(t *testing.T) {
		want, err := hex.DecodeString("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
		require.NoError(t, err)
		assert.Equal(t, want, domain.SHA256Hash(""))
	})

	t.Run("unnormalized inputs hash differently", func(t *testing.T) {
		// SHA256Hash does NOT normalize — callers must normalize first.
		assert.NotEqual(t, domain.SHA256Hash("hello world"), domain.SHA256Hash("Hello  World"))
	})

	t.Run("normalized inputs hash identically", func(t *testing.T) {
		n1 := domain.NormalizeContent("Hello  World")
		n2 := domain.NormalizeContent("hello world")
		assert.Equal(t, domain.SHA256Hash(n1), domain.SHA256Hash(n2))
	})
}
