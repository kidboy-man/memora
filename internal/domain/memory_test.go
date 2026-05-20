package domain_test

import (
	"errors"
	"testing"

	"github.com/kidboy-man/memora/internal/domain"
)

func TestMemoryValidate(t *testing.T) {
	validBase := func() domain.Memory {
		return domain.Memory{
			Content:    "the sky is blue",
			Type:       domain.TypeFact,
			Scope:      domain.ScopeGlobal,
			Source:     "claude-code",
			Confidence: 1.0,
		}
	}

	tests := []struct {
		name        string
		mutate      func(*domain.Memory)
		wantErr     bool
		errContains []string // all substrings must appear in error message
	}{
		{
			name:    "valid global fact",
			mutate:  func(_ *domain.Memory) {},
			wantErr: false,
		},
		{
			name: "valid project decision",
			mutate: func(m *domain.Memory) {
				m.Type = domain.TypeDecision
				m.Scope = domain.ScopeProject
				m.Project = "memora"
			},
			wantErr: false,
		},
		{
			name: "valid preference confidence 0.01",
			mutate: func(m *domain.Memory) {
				m.Type = domain.TypePreference
				m.Confidence = 0.01
			},
			wantErr: false,
		},
		{
			name: "empty content",
			mutate: func(m *domain.Memory) {
				m.Content = ""
			},
			wantErr:     true,
			errContains: []string{"content"},
		},
		{
			name: "whitespace-only content",
			mutate: func(m *domain.Memory) {
				m.Content = "   \t\n  "
			},
			wantErr:     true,
			errContains: []string{"content"},
		},
		{
			name: "invalid type",
			mutate: func(m *domain.Memory) {
				m.Type = "opinion"
			},
			wantErr:     true,
			errContains: []string{"invalid type"},
		},
		{
			name: "invalid scope",
			mutate: func(m *domain.Memory) {
				m.Scope = "team"
			},
			wantErr:     true,
			errContains: []string{"invalid scope"},
		},
		{
			name: "project scope without project",
			mutate: func(m *domain.Memory) {
				m.Scope = domain.ScopeProject
				m.Project = ""
			},
			wantErr:     true,
			errContains: []string{"project is required"},
		},
		{
			name: "project scope with whitespace-only project",
			mutate: func(m *domain.Memory) {
				m.Scope = domain.ScopeProject
				m.Project = "   "
			},
			wantErr:     true,
			errContains: []string{"project is required"},
		},
		{
			name: "confidence zero (excluded from range)",
			mutate: func(m *domain.Memory) {
				m.Confidence = 0.0
			},
			wantErr:     true,
			errContains: []string{"confidence", "out of range"},
		},
		{
			name: "confidence negative",
			mutate: func(m *domain.Memory) {
				m.Confidence = -0.1
			},
			wantErr:     true,
			errContains: []string{"confidence", "out of range"},
		},
		{
			name: "confidence above 1",
			mutate: func(m *domain.Memory) {
				m.Confidence = 1.001
			},
			wantErr:     true,
			errContains: []string{"confidence", "out of range"},
		},
		{
			name: "empty source",
			mutate: func(m *domain.Memory) {
				m.Source = ""
			},
			wantErr:     true,
			errContains: []string{"source is required"},
		},
		{
			name: "whitespace-only source",
			mutate: func(m *domain.Memory) {
				m.Source = "  "
			},
			wantErr:     true,
			errContains: []string{"source is required"},
		},
		{
			name: "multiple errors collected together",
			mutate: func(m *domain.Memory) {
				m.Content = ""
				m.Type = "bad"
				m.Source = ""
				m.Confidence = 0.0
			},
			wantErr:     true,
			errContains: []string{"content", "invalid type", "source", "confidence"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := validBase()
			tt.mutate(&m)

			err := m.Validate()

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr {
				var ve *domain.ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("expected *ValidationError, got %T: %v", err, err)
				}
				msg := err.Error()
				for _, sub := range tt.errContains {
					if !containsSubstring(msg, sub) {
						t.Errorf("error %q missing %q", msg, sub)
					}
				}
			}
		})
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsAt(s, sub))
}

func containsAt(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
