package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidationError is returned when required config fields are missing after
// env-var expansion. Fields lists every missing field so the user sees all
// problems in one startup failure rather than one at a time.
type ValidationError struct {
	Fields []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("config validation failed — missing required fields: %s",
		strings.Join(e.Fields, ", "))
}

// Config holds the full Memora server configuration. It is loaded once at
// startup and passed explicitly through constructors — never stored as a global.
type Config struct {
	Database      DatabaseConfig      `yaml:"database"`
	Embedding     EmbeddingConfig     `yaml:"embedding"`
	ExtractionLLM ExtractionLLMConfig `yaml:"extraction_llm"`
	Server        ServerConfig        `yaml:"server"`
	Defaults      DefaultsConfig      `yaml:"defaults"`
}

// DatabaseConfig holds PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"sslmode"`
}

// DSN returns a PostgreSQL connection URL safe for any password value.
// Uses net/url encoding so special characters in passwords are handled
// correctly without manual libpq quoting rules.
func (d DatabaseConfig) DSN() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.User, d.Password),
		Host:   fmt.Sprintf("%s:%d", d.Host, d.Port),
		Path:   "/" + d.Name,
	}
	q := u.Query()
	q.Set("sslmode", d.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

// EmbeddingConfig configures the active embedding provider.
// Dimensions is a string because the sentinel value "auto" means the adapter
// should detect the dimension count at profile creation time. The embedding
// adapter (internal/adapter/embedding) interprets "auto" vs a numeric string.
type EmbeddingConfig struct {
	Provider        string  `yaml:"provider"`
	Model           string  `yaml:"model"`
	APIBaseURL      string  `yaml:"api_base_url"` // empty = adapter uses built-in default
	APIKey          string  `yaml:"api_key"`      // empty allowed for Ollama (local, no auth)
	Dimensions      string  `yaml:"dimensions"`   // "auto" or integer string e.g. "1536"
	DistanceMetric  string  `yaml:"distance_metric"`
	DedupeStrategy  string  `yaml:"dedupe_strategy"`
	DedupeThreshold float64 `yaml:"dedupe_threshold"`
}

// ExtractionLLMConfig configures the optional extraction LLM used when
// auto_extract is enabled. All fields are optional — extraction is a v1 bonus.
type ExtractionLLMConfig struct {
	Provider   string `yaml:"provider"`
	Model      string `yaml:"model"`
	APIBaseURL string `yaml:"api_base_url"`
	APIKey     string `yaml:"api_key"`
}

// ServerConfig holds MCP server identity and logging settings.
type ServerConfig struct {
	Name     string `yaml:"name"`
	Version  string `yaml:"version"`
	LogLevel string `yaml:"log_level"`
}

// DefaultsConfig holds fallback values applied when MCP tool calls omit
// optional parameters.
type DefaultsConfig struct {
	Project       string `yaml:"project"`
	IncludeGlobal bool   `yaml:"include_global"`
	RecallMode    string `yaml:"recall_mode"`
	MaxResults    int    `yaml:"max_results"`
}

// LoadConfig reads the YAML file at path, expands ${VAR} and $VAR references
// using os.ExpandEnv, unmarshals into Config, applies defaults for optional
// fields, and validates that all required fields are non-empty.
//
// Secrets must never be hardcoded in the YAML file. Use env-var references:
//
//	password: ${MEMORA_DB_PASSWORD}
//	api_key:  ${OPENAI_API_KEY}
//
// os.ExpandEnv silently replaces unset vars with ""; validate catches them.
func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %q: %w", path, err)
	}

	expanded := os.ExpandEnv(string(raw))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := applyDefaultsAndValidate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// applyDefaultsAndValidate fills in optional field defaults then checks
// required fields. Defaults are applied before validation so callers always
// receive a fully-populated Config on success.
func applyDefaultsAndValidate(cfg *Config) error {
	// Apply defaults for optional fields first.
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}
	if cfg.Embedding.Dimensions == "" {
		cfg.Embedding.Dimensions = "auto"
	}
	if cfg.Embedding.DistanceMetric == "" {
		cfg.Embedding.DistanceMetric = "cosine"
	}
	if cfg.Embedding.DedupeStrategy == "" {
		cfg.Embedding.DedupeStrategy = "skip"
	}
	if cfg.Embedding.DedupeThreshold == 0 {
		cfg.Embedding.DedupeThreshold = 0.92
	}
	if cfg.Server.LogLevel == "" {
		cfg.Server.LogLevel = "info"
	}
	if cfg.Defaults.MaxResults == 0 {
		cfg.Defaults.MaxResults = 10
	}
	if cfg.Defaults.RecallMode == "" {
		cfg.Defaults.RecallMode = "hybrid"
	}

	// Validate required fields.
	var missing []string

	if cfg.Database.Host == "" {
		missing = append(missing, "database.host")
	}
	if cfg.Database.Port == 0 {
		missing = append(missing, "database.port")
	}
	if cfg.Database.Name == "" {
		missing = append(missing, "database.name")
	}
	if cfg.Database.User == "" {
		missing = append(missing, "database.user")
	}
	if cfg.Database.Password == "" {
		missing = append(missing, "database.password (check env var)")
	}
	if cfg.Embedding.Provider == "" {
		missing = append(missing, "embedding.provider")
	}
	if cfg.Embedding.Model == "" {
		missing = append(missing, "embedding.model")
	}
	// api_key is required for all providers except Ollama, which runs locally
	// without authentication. The adapter (internal/adapter/embedding) enforces
	// this at construction time too, but catching it here gives a better error
	// at startup rather than on the first embedding call.
	if cfg.Embedding.Provider != "ollama" && cfg.Embedding.APIKey == "" {
		missing = append(missing, "embedding.api_key (not required for ollama)")
	}

	if len(missing) > 0 {
		return &ValidationError{Fields: missing}
	}
	return nil
}
