package cli

import (
	"testing"

	"github.com/rejot-dev/semcheck/internal/config"
	"github.com/rejot-dev/semcheck/internal/providers"
)

func TestGenerateConfig(t *testing.T) {
	// Test that generateConfig works correctly with provider defaults
	allProviders := providers.GetAllProviders()

	for _, provider := range allProviders {
		t.Run(string(provider), func(t *testing.T) {
			defaults := providers.GetProviderDefaults(provider)

			configStr, err := generateConfig(provider, defaults.Model, defaults.ApiKeyVar)
			if err != nil {
				t.Errorf("generateConfig() with provider defaults failed: %v", err)
				return
			}

			config, err := config.ParseFromBytes([]byte(configStr))
			if err != nil {
				t.Errorf("generateConfig() with provider defaults failed: %v", err)
				return
			}

			if config.Provider != string(provider) {
				t.Errorf("generateConfig() provider mismatch: got %s, want %s", config.Provider, provider)
			}
			if config.Model != defaults.Model {
				t.Errorf("generateConfig() model mismatch: got %s, want %s", config.Model, defaults.Model)
			}
		})
	}
}
