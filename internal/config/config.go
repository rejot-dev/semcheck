package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version      string `yaml:"version"`
	Provider     string `yaml:"provider"`
	Model        string `yaml:"model"`
	APIKey       string `yaml:"api_key"`
	BaseURL      string `yaml:"base_url,omitempty"`
	Timeout      int    `yaml:"timeout"`
	MaxRetries   int    `yaml:"max_retries"`
	FailOnIssues bool   `yaml:"fail_on_issues"`
	Rules        []Rule `yaml:"rules"`
}

type Rule struct {
	Name                string      `yaml:"name"`
	Description         string      `yaml:"description"`
	Enabled             bool        `yaml:"enabled"`
	Files               FilePattern `yaml:"files"`
	Specs               []Spec      `yaml:"specs"`
	Prompt              string      `yaml:"prompt,omitempty"`
	Severity            string      `yaml:"severity"`
	ConfidenceThreshold float64     `yaml:"confidence_threshold"`
}

type FilePattern struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude,omitempty"`
}

type Spec struct {
	Path        string `yaml:"path"`
	Type        string `yaml:"type"`
	Description string `yaml:"description,omitempty"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	data = []byte(os.ExpandEnv(string(data)))

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
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

	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if c.Provider != "openai" && c.Provider != "anthropic" && c.Provider != "local" {
		return fmt.Errorf("unsupported provider: %s", c.Provider)
	}

	if c.Model == "" {
		return fmt.Errorf("model is required")
	}

	if c.Provider != "local" && c.APIKey == "" {
		return fmt.Errorf("api_key is required for provider: %s", c.Provider)
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

		if rule.Description == "" {
			return fmt.Errorf("rule description is required for rule: %s", rule.Name)
		}
		if len(rule.Files.Include) == 0 {
			return fmt.Errorf("at least one include pattern is required for rule: %s", rule.Name)
		}
		if len(rule.Specs) == 0 {
			return fmt.Errorf("at least one spec is required for rule: %s", rule.Name)
		}

		// Set default confidence threshold if not provided
		if rule.ConfidenceThreshold == 0 {
			c.Rules[i].ConfidenceThreshold = 0.8
		}

		if c.Rules[i].ConfidenceThreshold < 0 || c.Rules[i].ConfidenceThreshold > 1 {
			return fmt.Errorf("confidence_threshold must be between 0 and 1 for rule: %s", rule.Name)
		}
		for _, spec := range rule.Specs {
			if spec.Path == "" {
				return fmt.Errorf("spec path is required for rule: %s", rule.Name)
			}
			if spec.Type == "" {
				return fmt.Errorf("spec type is required for rule: %s", rule.Name)
			}
		}
	}

	// Set defaults
	if c.Timeout == 0 {
		c.Timeout = 30
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}

	return nil
}
