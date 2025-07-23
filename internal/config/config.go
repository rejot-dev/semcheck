package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Version      string   `yaml:"version"`
	Provider     string   `yaml:"provider"`
	Model        string   `yaml:"model"`
	APIKey       string   `yaml:"api_key"`
	BaseURL      string   `yaml:"base_url,omitempty"`
	Timeout      int      `yaml:"timeout"`
	MaxTokens    int      `yaml:"max_tokens"`
	Temperature  *float64 `yaml:"temperature,omitempty"`
	FailOnIssues *bool    `yaml:"fail_on_issues,omitempty"`
	Rules        []Rule   `yaml:"rules"`
}

type Rule struct {
	Name                string      `yaml:"name"`
	Description         string      `yaml:"description"`
	Enabled             bool        `yaml:"enabled"`
	Files               FilePattern `yaml:"files"`
	Specs               []Spec      `yaml:"specs"`
	Prompt              string      `yaml:"prompt,omitempty"`
	FailOn              string      `yaml:"fail_on"`
	ConfidenceThreshold *float64    `yaml:"confidence_threshold,omitempty"` // Deprecated field
}

type FilePattern struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude,omitempty"`
}

type Spec struct {
	Path string `yaml:"path"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	data = []byte(os.ExpandEnv(string(data)))

	config, err := ParseFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func ParseFromBytes(data []byte) (*Config, error) {

	var config Config
	if err := yaml.UnmarshalWithOptions(data, &config, yaml.Strict()); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func (c *Config) validate() error {
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}
	if c.Version != "1.0" {
		return fmt.Errorf("unsupported version: %s", c.Version)
	}

	// If correct provider is passed is checked on client instantiation
	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}

	if c.Model == "" {
		return fmt.Errorf("model is required")
	}

	// API key is optional for Ollama (local provider)
	if c.Provider != "ollama" && c.APIKey == "" {
		return fmt.Errorf("api_key is required for provider %s", c.Provider)
	}

	if len(c.Rules) == 0 {
		return fmt.Errorf("at least one rule is required")
	}

	ruleNames := make(map[string]bool)
	for i, rule := range c.Rules {
		if rule.Name == "" {
			return fmt.Errorf("rule name is required")
		}
		if ruleNames[rule.Name] {
			return fmt.Errorf("duplicate rule name: %s", rule.Name)
		}
		ruleNames[rule.Name] = true

		if len(rule.Files.Include) == 0 {
			return fmt.Errorf("at least one include pattern is required for rule: %s", rule.Name)
		}
		if len(rule.Specs) == 0 {
			return fmt.Errorf("at least one spec is required for rule: %s", rule.Name)
		}

		if rule.ConfidenceThreshold != nil {
			fmt.Printf("Warning: confidence_threshold field is deprecated and will be ignored for rule '%s'. Please remove this field from your configuration.\n", rule.Name)
		}

		// Set default fail_on if not provided
		if rule.FailOn == "" {
			c.Rules[i].FailOn = "error"
		}

		// Validate fail_on values
		if c.Rules[i].FailOn != "error" && c.Rules[i].FailOn != "warning" && c.Rules[i].FailOn != "notice" {
			return fmt.Errorf("fail_on must be 'error', 'warning', or 'notice' for rule: %s", rule.Name)
		}
		for _, spec := range rule.Specs {
			if spec.Path == "" {
				return fmt.Errorf("specification path is required for rule: %s", rule.Name)
			}

			// Check if path looks like a URL
			if strings.Contains(spec.Path, "://") {
				// Try to parse as URL
				parsedURL, err := url.Parse(spec.Path)
				if err != nil {
					return fmt.Errorf("invalid URL format in specification path: %s for rule: %s (%v)", spec.Path, rule.Name, err)
				}
				// Only allow HTTP/HTTPS URLs
				if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
					return fmt.Errorf("only HTTP/HTTPS URLs are supported for specification path: %s for rule: %s", spec.Path, rule.Name)
				}
			} else {
				// Validate that local spec file exists and is readable
				if _, err := os.Stat(spec.Path); err != nil {
					if os.IsNotExist(err) {
						return fmt.Errorf("specification file does not exist: %s for rule: %s", spec.Path, rule.Name)
					}
					return fmt.Errorf("specification file is not readable: %s for rule: %s (%v)", spec.Path, rule.Name, err)
				}
			}
		}
	}

	// Set defaults
	if c.Timeout == 0 {
		c.Timeout = 30
	}

	if c.MaxTokens == 0 {
		c.MaxTokens = 3000
	}

	if c.Temperature == nil {
		defaultTemperature := 0.1
		c.Temperature = &defaultTemperature
	}
	if c.FailOnIssues == nil {
		defaultFailOnIssues := true
		c.FailOnIssues = &defaultFailOnIssues
	}

	// Validate timeout range
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be positive number, got: %d", c.Timeout)
	}

	// Validate temperature range (0.0 is allowed for deterministic output)
	if *c.Temperature < 0 || *c.Temperature > 1 {
		return fmt.Errorf("temperature must be between 0.0 and 1.0, got: %f", *c.Temperature)
	}

	return nil
}

// maskAPIKey masks the API key for secure display
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 11 {
		return "[MASKED]"
	}
	return apiKey[:7] + "[MASKED]" + apiKey[len(apiKey)-4:]
}

func (c *Config) PrintAsYAML() error {
	// Create a copy of the config with masked API key
	configCopy := *c
	configCopy.APIKey = maskAPIKey(c.APIKey)

	yamlData, err := yaml.Marshal(&configCopy)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Add a newline to the end of the YAML string
	fmt.Println(string(yamlData))
	return nil
}
