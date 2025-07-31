package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/charmbracelet/lipgloss"
	"github.com/rejot-dev/semcheck/internal/providers"
)

var configTemplate = `# Semcheck configuration file
# This file configures semantic checking of code implementations against specifications.

version: "1.0"

# AI Provider configuration
provider: "{{ .Provider }}"
model: "{{ .Model }}"
{{- if ne .APIKeyVar "" }}
api_key: "${{ "{" }}{{ .APIKeyVar }}{{ "}" }}"
{{- end }}
temperature: 0.1
{{- if eq .Provider "ollama" }}
base_url: "http://localhost:11434"
{{- end }}

# Rules define which files to check and their specifications
# You can also link specification files directly from source code using comments:
# 	// semcheck:file(./path/to/spec.md)
rules:
  - name: "example-rule"
    description: "Example rule - edit or remove this to match your project"
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
    fail_on: "error"         # error, warning, or notice
`

type ConfigData struct {
	Provider  string
	Model     string
	APIKeyVar string
}

func runInit() error {
	// Create styled title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(0, 2).
		MarginBottom(1)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		MarginBottom(1)

	fmt.Println(titleStyle.Render("📋 Semcheck Configuration Setup"))
	fmt.Println(subtitleStyle.Render("Will setup your semcheck.yaml configuration file."))

	reader := bufio.NewReader(os.Stdin)

	// 1. Ask for config filename
	configFile := promptForInput(reader, "Config filename", "semcheck.yaml")

	// Check if file already exists
	if _, err := os.Stat(configFile); err == nil {
		warningStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Bold(true)

		fmt.Printf("%s File '%s' already exists. Overwrite? (y/N): ",
			warningStyle.Render("⚠️"), configFile)
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

	// Success message
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")).
		Bold(true).
		MarginTop(1)

	fmt.Println(successStyle.Render(fmt.Sprintf("✅ Configuration file '%s' created successfully!", configFile)))

	// Next steps styling
	nextStepsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true).
		MarginTop(1)

	stepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		MarginLeft(3)

	codeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")).
		Background(lipgloss.Color("0")).
		Padding(0, 1)

	noteStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")).
		Bold(true)

	if providerDefaults.ApiKeyVar != "" {
		fmt.Println(noteStyle.Render(fmt.Sprintf("📝 Don't forget to set your %s environment variable.", providerDefaults.ApiKeyVar)))
		fmt.Println(nextStepsStyle.Render("🎯 Next steps:"))
		fmt.Println(stepStyle.Render(fmt.Sprintf("1. Set your API key: %s",
			codeStyle.Render("export "+providerDefaults.ApiKeyVar+"='your-api-key-here'"))))
		fmt.Println(stepStyle.Render(fmt.Sprintf("2. Edit the rules in '%s' to match your project", configFile)))
		fmt.Println(stepStyle.Render(fmt.Sprintf("3. Run: %s", codeStyle.Render("semcheck <files>"))))
	} else {
		fmt.Println(nextStepsStyle.Render("🎯 Next steps:"))
		if string(provider) == "ollama" {
			fmt.Println(stepStyle.Render(fmt.Sprintf("1. Make sure Ollama is running: %s",
				codeStyle.Render("ollama serve"))))
			fmt.Println(stepStyle.Render(fmt.Sprintf("2. Pull a model: %s",
				codeStyle.Render("ollama pull llama3.2"))))
		}
		fmt.Println(stepStyle.Render(fmt.Sprintf("3. Edit the rules in '%s' to match your project", configFile)))
		fmt.Println(stepStyle.Render(fmt.Sprintf("4. Run: %s", codeStyle.Render("semcheck <files>"))))
	}

	return nil
}

func promptForInput(reader *bufio.Reader, prompt, defaultValue string) string {
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")).
		Bold(true)

	defaultStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Italic(true)

	if defaultValue != "" {
		fmt.Printf("%s %s: ",
			promptStyle.Render(prompt),
			defaultStyle.Render("(default: "+defaultValue+")"))
	} else {
		fmt.Printf("%s: ", promptStyle.Render(prompt))
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
