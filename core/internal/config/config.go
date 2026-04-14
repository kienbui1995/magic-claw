// Package config loads MagiC server configuration from YAML files.
// Environment variables override YAML values (env takes precedence).
package config

import (
	"os"

	"go.yaml.in/yaml/v2"
)

// Config is the top-level server configuration.
type Config struct {
	Port     string    `yaml:"port"`
	APIKey   string    `yaml:"api_key"`
	Store    StoreConf `yaml:"store"`
	LLM      LLMConf   `yaml:"llm"`
	CORS     string    `yaml:"cors_origin"`
	TrustedProxy bool  `yaml:"trusted_proxy"`
}

// StoreConf configures the storage backend.
type StoreConf struct {
	Driver      string `yaml:"driver"` // memory, sqlite, postgres
	SQLitePath  string `yaml:"sqlite_path"`
	PostgresURL string `yaml:"postgres_url"`
}

// LLMConf configures LLM providers.
type LLMConf struct {
	OpenAI    OpenAIConf    `yaml:"openai"`
	Anthropic AnthropicConf `yaml:"anthropic"`
	Ollama    OllamaConf    `yaml:"ollama"`
}

type OpenAIConf struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
}

type AnthropicConf struct {
	APIKey string `yaml:"api_key"`
}

type OllamaConf struct {
	URL string `yaml:"url"`
}

// Load reads config from a YAML file, then overlays environment variables.
func Load(path string) (*Config, error) {
	cfg := &Config{Port: "8080"}

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// Env vars override YAML
	envOverride(&cfg.Port, "MAGIC_PORT")
	envOverride(&cfg.APIKey, "MAGIC_API_KEY")
	envOverride(&cfg.Store.PostgresURL, "MAGIC_POSTGRES_URL")
	envOverride(&cfg.Store.SQLitePath, "MAGIC_STORE")
	envOverride(&cfg.LLM.OpenAI.APIKey, "OPENAI_API_KEY")
	envOverride(&cfg.LLM.OpenAI.BaseURL, "OPENAI_BASE_URL")
	envOverride(&cfg.LLM.Anthropic.APIKey, "ANTHROPIC_API_KEY")
	envOverride(&cfg.LLM.Ollama.URL, "OLLAMA_URL")
	envOverride(&cfg.CORS, "MAGIC_CORS_ORIGIN")
	if os.Getenv("MAGIC_TRUSTED_PROXY") == "true" {
		cfg.TrustedProxy = true
	}

	// Auto-detect store driver
	if cfg.Store.Driver == "" {
		switch {
		case cfg.Store.PostgresURL != "":
			cfg.Store.Driver = "postgres"
		case cfg.Store.SQLitePath != "":
			cfg.Store.Driver = "sqlite"
		default:
			cfg.Store.Driver = "memory"
		}
	}

	return cfg, nil
}

func envOverride(target *string, key string) {
	if v := os.Getenv(key); v != "" {
		*target = v
	}
}
