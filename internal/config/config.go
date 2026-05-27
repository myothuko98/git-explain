package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Provider  string          `toml:"provider"`
	Ollama    OllamaConfig    `toml:"ollama"`
	OpenAI    OpenAIConfig    `toml:"openai"`
	Anthropic AnthropicConfig `toml:"anthropic"`
	Gemini    GeminiConfig    `toml:"gemini"`
}

type OllamaConfig struct {
	URL   string `toml:"url"`
	Model string `toml:"model"`
}

type OpenAIConfig struct {
	APIKey string `toml:"api_key"`
	Model  string `toml:"model"`
}

type AnthropicConfig struct {
	APIKey string `toml:"api_key"`
	Model  string `toml:"model"`
}

type GeminiConfig struct {
	APIKey string `toml:"api_key"`
	Model  string `toml:"model"`
}

func DefaultConfig() Config {
	return Config{
		Provider: "auto",
		Ollama: OllamaConfig{
			URL:   "http://localhost:11434",
			Model: "llama3.2",
		},
		OpenAI: OpenAIConfig{
			Model: "gpt-4o-mini",
		},
		Anthropic: AnthropicConfig{
			Model: "claude-haiku-4-5",
		},
		Gemini: GeminiConfig{
			Model: "gemini-2.0-flash",
		},
	}
}

func Load() (Config, error) {
	cfg := DefaultConfig()
	path := ConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, err
	}
	// Env var overrides
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		cfg.OpenAI.APIKey = v
	}
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		cfg.Anthropic.APIKey = v
	}
	if v := os.Getenv("GEMINI_API_KEY"); v != "" {
		cfg.Gemini.APIKey = v
	}
	return cfg, nil
}

func Save(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(ConfigPath()), 0700); err != nil {
		return err
	}
	f, err := os.Create(ConfigPath())
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".git-explain", "config.toml")
}
