package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rules.yaml")

	configContent := `
miniflux_url: "https://miniflux.example.com"
interval: 300

rules:
  - name: "Test Rule"
    feed: "Test Feed"
    author: "Test Author"
    content: "#test"
    action: "read"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.MinifluxURL != "https://miniflux.example.com" {
		t.Errorf("Expected MinifluxURL 'https://miniflux.example.com', got '%s'", config.MinifluxURL)
	}

	if config.Interval != 300 {
		t.Errorf("Expected Interval 300, got %d", config.Interval)
	}

	if len(config.Rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(config.Rules))
	}

	rule := config.Rules[0]
	if rule.Name != "Test Rule" {
		t.Errorf("Expected rule name 'Test Rule', got '%s'", rule.Name)
	}
	if rule.Feed != "Test Feed" {
		t.Errorf("Expected feed 'Test Feed', got '%s'", rule.Feed)
	}
	if rule.Author != "Test Author" {
		t.Errorf("Expected author 'Test Author', got '%s'", rule.Author)
	}
	if rule.Content != "#test" {
		t.Errorf("Expected content '#test', got '%s'", rule.Content)
	}
	if rule.Action != "read" {
		t.Errorf("Expected action 'read', got '%s'", rule.Action)
	}
}

func TestLoadConfigReplaceRule(t *testing.T) {
	os.Unsetenv("MINIFLUX_URL")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rules.yaml")

	configContent := `
miniflux_url: "https://miniflux.example.com"
rules:
  - name: "Normalize title"
    title: "PR"
    action: "replace"
    replace_from: "[PR] "
    replace_to: ""
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	rule := config.Rules[0]
	if rule.Action != "replace" {
		t.Errorf("Expected action 'replace', got '%s'", rule.Action)
	}
	if rule.ReplaceFrom != "[PR] " {
		t.Errorf("Expected replace_from '[PR] ', got '%s'", rule.ReplaceFrom)
	}
	if rule.normalizedReplaceField() != replaceFieldTitle {
		t.Errorf("Expected default replace field '%s', got '%s'", replaceFieldTitle, rule.normalizedReplaceField())
	}
}

func TestLoadConfigMissingURL(t *testing.T) {
	os.Unsetenv("MINIFLUX_URL")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rules.yaml")

	configContent := `
interval: 300
rules:
  - name: "Test Rule"
    action: "read"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for missing miniflux_url")
	}
}

func TestLoadConfigURLFromEnv(t *testing.T) {
	os.Setenv("MINIFLUX_URL", "https://env.example.com")
	defer os.Unsetenv("MINIFLUX_URL")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rules.yaml")

	configContent := `
interval: 300
rules:
  - name: "Test Rule"
    action: "read"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.MinifluxURL != "https://env.example.com" {
		t.Errorf("Expected MinifluxURL from env, got '%s'", config.MinifluxURL)
	}
}

func TestLoadConfigInvalidAction(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rules.yaml")

	configContent := `
miniflux_url: "https://miniflux.example.com"
rules:
  - name: "Test Rule"
    action: "invalid"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid action")
	}
}

func TestLoadConfigReplaceRuleMissingFrom(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rules.yaml")

	configContent := `
miniflux_url: "https://miniflux.example.com"
rules:
  - name: "Normalize title"
    action: "replace"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for missing replace_from")
	}
}

func TestLoadConfigReplaceRuleInvalidField(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rules.yaml")

	configContent := `
miniflux_url: "https://miniflux.example.com"
rules:
  - name: "Normalize title"
    action: "replace"
    replace_field: "summary"
    replace_from: "[PR] "
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid replace_field")
	}
}

func TestLoadConfigRejectsReplaceFieldsForReadAction(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "rules.yaml")

	configContent := `
miniflux_url: "https://miniflux.example.com"
rules:
  - name: "Mark read"
    action: "read"
    replace_from: "[PR] "
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error when replace_* fields are used with action 'read'")
	}
}

func TestGetAPIKey(t *testing.T) {
	// Test with MINIFLUX_API_KEY
	os.Setenv("MINIFLUX_API_KEY", "test-api-key")
	defer os.Unsetenv("MINIFLUX_API_KEY")

	apiKey, err := GetAPIKey()
	if err != nil {
		t.Fatalf("Failed to get API key: %v", err)
	}
	if apiKey != "test-api-key" {
		t.Errorf("Expected 'test-api-key', got '%s'", apiKey)
	}
}

func TestGetAPIKeyFromFile(t *testing.T) {
	// Clear direct env var
	os.Unsetenv("MINIFLUX_API_KEY")

	// Create a temporary key file
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "api-key")
	if err := os.WriteFile(keyPath, []byte("file-api-key\n"), 0o644); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	os.Setenv("MINIFLUX_API_KEY_FILE", keyPath)
	defer os.Unsetenv("MINIFLUX_API_KEY_FILE")

	apiKey, err := GetAPIKey()
	if err != nil {
		t.Fatalf("Failed to get API key from file: %v", err)
	}
	if apiKey != "file-api-key" {
		t.Errorf("Expected 'file-api-key', got '%s'", apiKey)
	}
}

func TestGetAPIKeyMissing(t *testing.T) {
	os.Unsetenv("MINIFLUX_API_KEY")
	os.Unsetenv("MINIFLUX_API_KEY_FILE")

	_, err := GetAPIKey()
	if err == nil {
		t.Error("Expected error when no API key is configured")
	}
}
