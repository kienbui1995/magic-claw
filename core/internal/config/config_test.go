package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8080" {
		t.Errorf("default port = %s, want 8080", cfg.Port)
	}
	if cfg.Store.Driver != "memory" {
		t.Errorf("default driver = %s, want memory", cfg.Store.Driver)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("MAGIC_PORT", "9090")
	t.Setenv("OPENAI_API_KEY", "sk-test")
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "9090" {
		t.Errorf("port = %s, want 9090", cfg.Port)
	}
	if cfg.LLM.OpenAI.APIKey != "sk-test" {
		t.Errorf("openai key = %s", cfg.LLM.OpenAI.APIKey)
	}
}

func TestLoad_YAMLFile(t *testing.T) {
	f, _ := os.CreateTemp("", "magic-*.yaml")
	f.WriteString("port: \"3000\"\nllm:\n  openai:\n    api_key: sk-yaml\n")
	f.Close()
	defer os.Remove(f.Name())

	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "3000" {
		t.Errorf("port = %s, want 3000", cfg.Port)
	}
	if cfg.LLM.OpenAI.APIKey != "sk-yaml" {
		t.Errorf("openai key = %s, want sk-yaml", cfg.LLM.OpenAI.APIKey)
	}
}

func TestLoad_AutoDetectDriver(t *testing.T) {
	t.Setenv("MAGIC_POSTGRES_URL", "postgres://localhost/magic")
	cfg, _ := Load("")
	if cfg.Store.Driver != "postgres" {
		t.Errorf("driver = %s, want postgres", cfg.Store.Driver)
	}
}
