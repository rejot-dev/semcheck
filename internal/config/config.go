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
	FailOnIssues *bool  `yaml:"fail_on_issues,omitempty"`
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
	Path string `yaml:"path"`
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

		// Set default severity if not provided
		if rule.Severity == "" {
			c.Rules[i].Severity = "error"
		}

		if c.Rules[i].ConfidenceThreshold < 0 || c.Rules[i].ConfidenceThreshold > 1 {
			return fmt.Errorf("confidence_threshold must be between 0 and 1 for rule: %s", rule.Name)
		}

		// Validate severity values
		if c.Rules[i].Severity != "error" && c.Rules[i].Severity != "warning" && c.Rules[i].Severity != "info" {
			return fmt.Errorf("severity must be 'error', 'warning', or 'info' for rule: %s", rule.Name)
		}
		for _, spec := range rule.Specs {
			if spec.Path == "" {
				return fmt.Errorf("spec path is required for rule: %s", rule.Name)
			}
			// Validate that spec file exists and is readable
			if _, err := os.Stat(spec.Path); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("spec file does not exist: %s for rule: %s", spec.Path, rule.Name)
				}
				return fmt.Errorf("spec file is not readable: %s for rule: %s (%v)", spec.Path, rule.Name, err)
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
	if c.FailOnIssues == nil {
		defaultFailOnIssues := true
		c.FailOnIssues = &defaultFailOnIssues
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
