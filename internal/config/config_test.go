package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig_Providers(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Provider != "auto" {
		t.Fatalf("default provider = %q, want 'auto'", cfg.Provider)
	}
	if cfg.Ollama.URL == "" {
		t.Fatal("Ollama URL should have a default")
	}
	if cfg.Ollama.Model == "" || cfg.OpenAI.Model == "" || cfg.Anthropic.Model == "" ||
		cfg.Gemini.Model == "" || cfg.Qwen.Model == "" || cfg.Moonshot.Model == "" {
		t.Fatal("all provider models should have defaults")
	}
}

func TestLoad_Defaults_WhenNoFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// Clear API key env vars so defaults are clean.
	for _, k := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GEMINI_API_KEY", "QWEN_API_KEY", "MOONSHOT_API_KEY"} {
		t.Setenv(k, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Provider != "auto" {
		t.Fatalf("want 'auto', got %q", cfg.Provider)
	}
	if cfg.Ollama.Model != "llama3.2" {
		t.Fatalf("want 'llama3.2', got %q", cfg.Ollama.Model)
	}
}

func TestLoad_EnvVarOverride(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "sk-test-key")
	t.Setenv("QWEN_API_KEY", "qwen-test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OpenAI.APIKey != "sk-test-key" {
		t.Fatalf("OPENAI_API_KEY not applied, got %q", cfg.OpenAI.APIKey)
	}
	if cfg.Qwen.APIKey != "qwen-test" {
		t.Fatalf("QWEN_API_KEY not applied, got %q", cfg.Qwen.APIKey)
	}
}

func TestLoad_EnvVarOverride_NoConfigFile(t *testing.T) {
	// Regression: env vars were skipped when config file didn't exist.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("ANTHROPIC_API_KEY", "ant-key")

	// Ensure no config file exists.
	cfgPath := filepath.Join(tmp, ".git-explain", "config.toml")
	_ = os.Remove(cfgPath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Anthropic.APIKey != "ant-key" {
		t.Fatalf("env var not applied without config file, got %q", cfg.Anthropic.APIKey)
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// Ensure env vars don't interfere with loaded values.
	for _, k := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GEMINI_API_KEY", "QWEN_API_KEY", "MOONSHOT_API_KEY"} {
		t.Setenv(k, "")
	}

	cfg := DefaultConfig()
	cfg.Provider = "openai"
	cfg.OpenAI.Model = "gpt-4o"
	cfg.Moonshot.Model = "moonshot-v1-32k"

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	if loaded.Provider != "openai" {
		t.Fatalf("provider: got %q, want 'openai'", loaded.Provider)
	}
	if loaded.OpenAI.Model != "gpt-4o" {
		t.Fatalf("OpenAI model: got %q, want 'gpt-4o'", loaded.OpenAI.Model)
	}
	if loaded.Moonshot.Model != "moonshot-v1-32k" {
		t.Fatalf("Moonshot model: got %q, want 'moonshot-v1-32k'", loaded.Moonshot.Model)
	}
}

func TestConfigPath_UsesHome(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	p := ConfigPath()
	if p != filepath.Join(tmp, ".git-explain", "config.toml") {
		t.Fatalf("unexpected config path: %q", p)
	}
}
