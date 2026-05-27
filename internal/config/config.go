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
	Qwen      QwenConfig      `toml:"qwen"`
	Moonshot  MoonshotConfig  `toml:"moonshot"`
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

type QwenConfig struct {
	APIKey string `toml:"api_key"`
	Model  string `toml:"model"`
}

type MoonshotConfig struct {
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
		Qwen: QwenConfig{
			Model: "qwen-turbo",
		},
		Moonshot: MoonshotConfig{
			Model: "moonshot-v1-8k",
		},
	}
}

func Load() (Config, error) {
	cfg := DefaultConfig()
	path := ConfigPath()
	// Decode file if it exists (file values override defaults)
	if _, err := os.Stat(path); err == nil {
		if _, err := toml.DecodeFile(path, &cfg); err != nil {
			return cfg, err
		}
	}
	// Env vars take highest priority — applied after file decode
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		cfg.OpenAI.APIKey = v
	}
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		cfg.Anthropic.APIKey = v
	}
	if v := os.Getenv("GEMINI_API_KEY"); v != "" {
		cfg.Gemini.APIKey = v
	}
	if v := os.Getenv("QWEN_API_KEY"); v != "" {
		cfg.Qwen.APIKey = v
	}
	if v := os.Getenv("MOONSHOT_API_KEY"); v != "" {
		cfg.Moonshot.APIKey = v
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
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, ".git-explain", "config.toml")
}
