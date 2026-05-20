package config_test

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/kidboy-man/memora/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeConfig writes content to a temp file and returns its path.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

// minimalValidYAML returns a YAML string with all required fields set.
// Tests that need to omit or change specific fields start from this base.
const minimalValidYAML = `
database:
  host: localhost
  port: 5432
  name: memora
  user: memora
  password: ${TEST_DB_PASSWORD}
embedding:
  provider: openai
  model: text-embedding-3-small
  api_key: ${TEST_EMBED_KEY}
`

func TestLoadConfig_HappyPath(t *testing.T) {
	t.Setenv("TEST_DB_PASSWORD", "supersecret")
	t.Setenv("TEST_EMBED_KEY", "sk-testkey1234567890abcdef1234567890ab")

	cfg, err := config.LoadConfig(writeConfig(t, minimalValidYAML))
	require.NoError(t, err)

	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "memora", cfg.Database.Name)
	assert.Equal(t, "supersecret", cfg.Database.Password)
	assert.Equal(t, "sk-testkey1234567890abcdef1234567890ab", cfg.Embedding.APIKey)
}

func TestLoadConfig_EnvExpansionBothSyntaxes(t *testing.T) {
	t.Setenv("TEST_DB_PW", "mypassword")
	t.Setenv("TEST_DB_HOST", "db.internal")

	yaml := `
database:
  host: $TEST_DB_HOST
  port: 5432
  name: memora
  user: memora
  password: ${TEST_DB_PW}
embedding:
  provider: openai
  model: text-embedding-3-small
  api_key: sk-validkey1234567890abcdefabcdefabcdef
`
	cfg, err := config.LoadConfig(writeConfig(t, yaml))
	require.NoError(t, err)
	assert.Equal(t, "db.internal", cfg.Database.Host)
	assert.Equal(t, "mypassword", cfg.Database.Password)
}

func TestLoadConfig_DefaultsApplied(t *testing.T) {
	t.Setenv("TEST_DB_PASSWORD", "pw")
	t.Setenv("TEST_EMBED_KEY", "sk-key1234567890abcdefabcdefabcdef1234")

	cfg, err := config.LoadConfig(writeConfig(t, minimalValidYAML))
	require.NoError(t, err)

	// All optional fields should have defaults.
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, "auto", cfg.Embedding.Dimensions)
	assert.Equal(t, "cosine", cfg.Embedding.DistanceMetric)
	assert.Equal(t, "skip", cfg.Embedding.DedupeStrategy)
	assert.InDelta(t, 0.92, cfg.Embedding.DedupeThreshold, 0.001)
	assert.Equal(t, "info", cfg.Server.LogLevel)
	assert.Equal(t, 10, cfg.Defaults.MaxResults)
	assert.Equal(t, "hybrid", cfg.Defaults.RecallMode)
}

func TestLoadConfig_ExplicitValuesOverrideDefaults(t *testing.T) {
	t.Setenv("TEST_DB_PASSWORD", "pw")
	t.Setenv("TEST_EMBED_KEY", "sk-key1234567890abcdefabcdefabcdef1234")

	yaml := minimalValidYAML + `
  distance_metric: l2
  dedupe_strategy: warn
  dedupe_threshold: 0.85
  dimensions: "1536"
server:
  log_level: debug
defaults:
  max_results: 20
  recall_mode: semantic
`
	cfg, err := config.LoadConfig(writeConfig(t, yaml))
	require.NoError(t, err)

	assert.Equal(t, "l2", cfg.Embedding.DistanceMetric)
	assert.Equal(t, "warn", cfg.Embedding.DedupeStrategy)
	assert.InDelta(t, 0.85, cfg.Embedding.DedupeThreshold, 0.001)
	assert.Equal(t, "1536", cfg.Embedding.Dimensions)
	assert.Equal(t, "debug", cfg.Server.LogLevel)
	assert.Equal(t, 20, cfg.Defaults.MaxResults)
	assert.Equal(t, "semantic", cfg.Defaults.RecallMode)
}

func TestLoadConfig_ExplicitSSLModeNotOverridden(t *testing.T) {
	t.Setenv("TEST_DB_PASSWORD", "pw")
	t.Setenv("TEST_EMBED_KEY", "sk-key1234567890abcdefabcdefabcdef1234")

	yaml := `
database:
  host: localhost
  port: 5432
  name: memora
  user: memora
  password: ${TEST_DB_PASSWORD}
  sslmode: require
embedding:
  provider: openai
  model: text-embedding-3-small
  api_key: ${TEST_EMBED_KEY}
  distance_metric: l2
  dedupe_strategy: none
  dedupe_threshold: 0.5
  dimensions: "768"
server:
  log_level: warn
defaults:
  max_results: 5
  recall_mode: keyword
`
	cfg, err := config.LoadConfig(writeConfig(t, yaml))
	require.NoError(t, err)
	// Explicit values must not be overridden by defaults.
	assert.Equal(t, "require", cfg.Database.SSLMode)
	assert.Equal(t, "l2", cfg.Embedding.DistanceMetric)
	assert.Equal(t, "none", cfg.Embedding.DedupeStrategy)
	assert.InDelta(t, 0.5, cfg.Embedding.DedupeThreshold, 0.001)
	assert.Equal(t, "768", cfg.Embedding.Dimensions)
	assert.Equal(t, "warn", cfg.Server.LogLevel)
	assert.Equal(t, 5, cfg.Defaults.MaxResults)
	assert.Equal(t, "keyword", cfg.Defaults.RecallMode)
}

func TestLoadConfig_OllamaNoAPIKeyRequired(t *testing.T) {
	t.Setenv("TEST_DB_PASSWORD", "pw")

	yaml := `
database:
  host: localhost
  port: 5432
  name: memora
  user: memora
  password: ${TEST_DB_PASSWORD}
embedding:
  provider: ollama
  model: nomic-embed-text
`
	cfg, err := config.LoadConfig(writeConfig(t, yaml))
	require.NoError(t, err)
	assert.Equal(t, "ollama", cfg.Embedding.Provider)
	assert.Empty(t, cfg.Embedding.APIKey)
}

func TestLoadConfig_MissingEnvVar_ValidationError(t *testing.T) {
	// Ensure TEST_DB_PASSWORD is unset → expands to "" → validation failure.
	t.Setenv("TEST_DB_PASSWORD", "")
	t.Setenv("TEST_EMBED_KEY", "sk-key1234567890abcdefabcdefabcdef1234")

	_, err := config.LoadConfig(writeConfig(t, minimalValidYAML))
	require.Error(t, err)

	var ve *config.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Contains(t, err.Error(), "database.password")
}

func TestLoadConfig_MultipleFieldsMissing(t *testing.T) {
	// All env vars unset → multiple required fields missing.
	t.Setenv("TEST_DB_PASSWORD", "")
	t.Setenv("TEST_EMBED_KEY", "")

	_, err := config.LoadConfig(writeConfig(t, minimalValidYAML))
	require.Error(t, err)

	var ve *config.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Contains(t, err.Error(), "database.password")
	assert.Contains(t, err.Error(), "embedding.api_key")
}

func TestLoadConfig_NonOllamaWithoutAPIKey_ValidationError(t *testing.T) {
	t.Setenv("TEST_DB_PASSWORD", "pw")

	yaml := `
database:
  host: localhost
  port: 5432
  name: memora
  user: memora
  password: ${TEST_DB_PASSWORD}
embedding:
  provider: openrouter
  model: openai/text-embedding-3-small
`
	_, err := config.LoadConfig(writeConfig(t, yaml))
	require.Error(t, err)

	var ve *config.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Contains(t, err.Error(), "embedding.api_key")
}

func TestLoadConfig_MissingDatabaseFields(t *testing.T) {
	t.Setenv("TEST_EMBED_KEY", "sk-key1234567890abcdefabcdefabcdef1234")

	yaml := `
database: {}
embedding:
  provider: openai
  model: text-embedding-3-small
  api_key: ${TEST_EMBED_KEY}
`
	_, err := config.LoadConfig(writeConfig(t, yaml))
	require.Error(t, err)

	var ve *config.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Contains(t, err.Error(), "database.host")
	assert.Contains(t, err.Error(), "database.port")
	assert.Contains(t, err.Error(), "database.name")
	assert.Contains(t, err.Error(), "database.user")
	assert.Contains(t, err.Error(), "database.password")
}

func TestLoadConfig_MissingEmbeddingFields(t *testing.T) {
	t.Setenv("TEST_DB_PASSWORD", "pw")

	yaml := `
database:
  host: localhost
  port: 5432
  name: memora
  user: memora
  password: ${TEST_DB_PASSWORD}
embedding: {}
`
	_, err := config.LoadConfig(writeConfig(t, yaml))
	require.Error(t, err)

	var ve *config.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Contains(t, err.Error(), "embedding.provider")
	assert.Contains(t, err.Error(), "embedding.model")
	// provider="" is not "ollama", so api_key also required
	assert.Contains(t, err.Error(), "embedding.api_key")
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := config.LoadConfig("/nonexistent/path/config.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read config file")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	path := writeConfig(t, "database: [invalid: yaml: {")
	_, err := config.LoadConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse config")
}

func TestDatabaseConfig_DSN(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.DatabaseConfig
		contains []string
	}{
		{
			name: "standard connection",
			cfg: config.DatabaseConfig{
				Host: "localhost", Port: 5432, Name: "memora",
				User: "memora", Password: "secret", SSLMode: "disable",
			},
			contains: []string{"postgres://", "localhost:5432", "/memora", "sslmode=disable"},
		},
		{
			name: "password with special characters",
			cfg: config.DatabaseConfig{
				Host: "db.host", Port: 5432, Name: "mydb",
				User: "user", Password: "p@ss w=rd&more", SSLMode: "require",
			},
			// net/url percent-encodes the password; DSN must be parseable
			contains: []string{"postgres://", "db.host:5432", "/mydb", "sslmode=require"},
		},
		{
			name: "sslmode require",
			cfg: config.DatabaseConfig{
				Host: "prod.db", Port: 5432, Name: "app",
				User: "app", Password: "pw", SSLMode: "verify-full",
			},
			contains: []string{"sslmode=verify-full"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := tt.cfg.DSN()
			for _, sub := range tt.contains {
				assert.Contains(t, dsn, sub)
			}
		})
	}
}

func TestDatabaseConfig_DSN_SpecialCharPassword_Parseable(t *testing.T) {
	// Verify the DSN with a special-char password round-trips correctly.
	// net/url.Parse should succeed — proof the encoding is valid.
	cfg := config.DatabaseConfig{
		Host: "localhost", Port: 5432, Name: "db",
		User: "user", Password: "p@ss=w rd/back\\slash", SSLMode: "disable",
	}
	dsn := cfg.DSN()
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	pw, ok := u.User.Password()
	assert.True(t, ok)
	assert.Equal(t, "p@ss=w rd/back\\slash", pw)
}
