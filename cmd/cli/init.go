package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/rejot-dev/semcheck/internal/providers"
)

var configTemplate = `# Semcheck configuration file
# This file configures semantic checking of code implementations against specifications.

version: "1.0"

# AI Provider configuration
provider: "{{ .Provider }}"
model: "{{ .Model }}"
api_key: "${{ "{" }}{{ .APIKeyVar }}{{ "}" }}"

# Rules define which files to check and their specifications
rules:
  - name: "example-rule"
    description: "Example rule - edit this to match your project"
    enabled: true
    files:
      include:
        - "**/*.go"          # Include all Go files
        - "**/*.py"          # Include all Python files
    # Files from .gitignore are excluded automatically
    exclude:
        - "**/*_test.go"     # Exclude test files
        - "**/vendor/**"     # Exclude vendor directory
    specs:
      - path: "SPEC.md"    # Specification file(s)
    severity: "error"        # error, warning, or notice
`

type ConfigData struct {
	Provider  string
	Model     string
	APIKeyVar string
}

func runInit() error {
	fmt.Println("🚀 Welcome to semcheck configuration setup!")
	fmt.Println("This will create a semcheck.yaml configuration file for you.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// 1. Ask for config filename
	configFile := promptForInput(reader, "Config filename", "semcheck.yaml")

	// Check if file already exists
	if _, err := os.Stat(configFile); err == nil {
		fmt.Printf("⚠️  File '%s' already exists. Overwrite? (y/N): ", configFile)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			return fmt.Errorf("not overwriting existing config file: %s", configFile)
		}
	}

	// 2. Ask for AI provider
	allProviders := providers.GetAllProviders()
	providerStrings := []string{}
	for _, provider := range allProviders {
		providerStrings = append(providerStrings, string(provider))
	}

	providerInput := promptForInput(reader, "AI Provider ["+strings.Join(providerStrings, ", ")+"]", string(providers.ProviderOpenAI))
	provider, err := providers.ToProvider(providerInput)
	if err != nil {
		return err
	}

	// 3. Ask for model with provider-specific defaults
	providerDefaults := providers.GetProviderDefaults(provider)

	model := promptForInput(reader, "Model", providerDefaults.Model)

	// Generate the configuration
	config, err := generateConfig(provider, model, providerDefaults.ApiKeyVar)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	// Write the configuration file
	err = os.WriteFile(configFile, []byte(config), 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("\n✅ Configuration file '%s' created successfully!\n", configFile)
	fmt.Printf("📝 Don't forget to set your %s environment variable.\n", providerDefaults.ApiKeyVar)
	fmt.Println("\n🎯 Next steps:")
	fmt.Printf("   1. Set your API key: export %s='your-api-key-here'\n", providerDefaults.ApiKeyVar)
	fmt.Printf("   2. Edit the rules in '%s' to match your project\n", configFile)
	fmt.Printf("   3. Run: semcheck <files>\n")

	return nil
}

func promptForInput(reader *bufio.Reader, prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s (default: %s): ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" && defaultValue != "" {
		return defaultValue
	}
	return input
}

func generateConfig(provider providers.Provider, model, apiKeyVar string) (string, error) {
	tmpl, err := template.New("config").Parse(configTemplate)
	if err != nil {
		return "", err
	}

	data := ConfigData{
		Provider:  string(provider),
		Model:     model,
		APIKeyVar: apiKeyVar,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
