package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Rule defines a single filtering rule for entries
type Rule struct {
	Name    string `yaml:"name"`
	Feed    string `yaml:"feed"`    // regex pattern for feed title
	Author  string `yaml:"author"`  // regex pattern for author
	Title   string `yaml:"title"`   // regex pattern for entry title
	Content string `yaml:"content"` // regex pattern for entry content
	Action  string `yaml:"action"`  // "read" or "remove"
}

// Config holds the application configuration
type Config struct {
	MinifluxURL string `yaml:"miniflux_url"`
	Interval    int    `yaml:"interval"` // seconds between runs (0 = run once)
	Rules       []Rule `yaml:"rules"`
}

// LoadConfig reads and parses the YAML configuration file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if envURL := os.Getenv("MINIFLUX_URL"); envURL != "" {
		config.MinifluxURL = envURL
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.MinifluxURL == "" {
		return fmt.Errorf("miniflux_url is required")
	}

	if c.Interval < 0 {
		return fmt.Errorf("interval must be >= 0")
	}

	for i, rule := range c.Rules {
		if rule.Name == "" {
			return fmt.Errorf("rule %d: name is required", i)
		}

		action := strings.ToLower(rule.Action)
		if action != "read" && action != "remove" {
			return fmt.Errorf("rule %d (%s): action must be 'read' or 'remove'", i, rule.Name)
		}
	}

	return nil
}

// GetAPIKey retrieves the Miniflux API key from environment variables
// It first checks MINIFLUX_API_KEY, then falls back to reading from MINIFLUX_API_KEY_FILE
func GetAPIKey() (string, error) {
	// Try direct environment variable first
	if apiKey := os.Getenv("MINIFLUX_API_KEY"); apiKey != "" {
		return apiKey, nil
	}

	// Try reading from file
	if keyFile := os.Getenv("MINIFLUX_API_KEY_FILE"); keyFile != "" {
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return "", fmt.Errorf("failed to read API key file: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	}

	return "", fmt.Errorf("MINIFLUX_API_KEY or MINIFLUX_API_KEY_FILE environment variable is required")
}
