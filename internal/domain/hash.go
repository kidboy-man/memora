package domain

import "crypto/sha256"

// SHA256Hash returns the raw 32-byte SHA-256 digest of the normalized content.
//
// Callers must normalize content with NormalizeContent before hashing so that
// semantically identical content always produces the same hash regardless of
// whitespace or Unicode form.
//
// The result is stored as BYTEA in PostgreSQL. Do not hex-encode it — pgx
// handles []byte ↔ BYTEA natively without conversion overhead.
func SHA256Hash(normalizedContent string) []byte {
	h := sha256.Sum256([]byte(normalizedContent))
	return h[:]
}
