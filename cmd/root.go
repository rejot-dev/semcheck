package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"rejot.dev/semcheck/internal/config"
	"rejot.dev/semcheck/internal/providers"
)

var (
	configPath = flag.String("config", "semcheck.yaml", "path to configuration file")
	showHelp   = flag.Bool("help", false, "show help message")
	showVer    = flag.Bool("version", false, "show version")
)

const version = "0.1.0"

func Execute() error {
	flag.Parse()

	if *showHelp {
		showUsage()
		return nil
	}

	if *showVer {
		fmt.Printf("semcheck version %s\n", version)
		return nil
	}

	files := flag.Args()
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no files specified\n")
		showUsage()
		return fmt.Errorf("no files specified")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return err
	}

	fmt.Printf("Processing %d files with config: %s\n", len(files), *configPath)
	fmt.Printf("Provider: %s\n", cfg.Provider)
	for _, file := range files {
		fmt.Printf("  - %s\n", file)
	}

	// Test AI client
	if err := interimAIClientTest(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "AI client test failed: %v\n", err)
		return err
	}

	return nil
}

func interimAIClientTest(cfg *config.Config) error {
	fmt.Println("\n--- Testing AI Client ---")

	// Convert config to provider config
	providerConfig := &providers.Config{
		Provider:   cfg.Provider,
		Model:      cfg.Model,
		APIKey:     cfg.APIKey,
		BaseURL:    cfg.BaseURL,
		Timeout:    time.Duration(cfg.Timeout) * time.Second,
		MaxRetries: cfg.MaxRetries,
	}

	var client providers.Client
	var err error

	switch cfg.Provider {
	case "openai":
		client, err = providers.NewOpenAIClient(providerConfig)
	default:
		return fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}

	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	fmt.Printf("Created %s client with model: %s\n", client.Name(), cfg.Model)

	// Test request
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &providers.Request{
		Prompt:      "What is the answer to the ultimate question of life, the universe, and everything?",
		MaxTokens:   20,
		Temperature: 0.1,
	}

	fmt.Println("Sending test request...")
	resp, err := client.Complete(ctx, req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}

	fmt.Printf("✓ AI Response: %s\n", resp.Content)
	fmt.Printf("✓ Tokens used: %d prompt + %d completion = %d total\n",
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)

	return nil
}

func showUsage() {
	fmt.Printf("Usage: %s [options] <files...>\n\n", os.Args[0])
	fmt.Printf("Semcheck is a tool for semantic checking of code implementations against specifications using LLMs.\n\n")
	fmt.Println("Arguments:")
	fmt.Println("  <files...> - Files to check, either specifications or implementation files. Semcheck will use rules to determine which it is.")
	fmt.Println("Options:")
	flag.PrintDefaults()

}
