package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/rubensantoniorosa2704/schemaping-worker/pkg/types"
)

type file struct {
	Monitors []types.Monitor       `yaml:"monitors"`
	Webhooks []types.WebhookConfig `yaml:"webhooks"`
}

// Config holds the fully-loaded and validated configuration.
type Config struct {
	Monitors []types.Monitor
	Webhooks []types.WebhookConfig
}

// Load reads a YAML config file, expands ${ENV_VAR} references, and returns
// validated monitors with defaults applied alongside any webhook configs.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("config: read file: %w", err)
	}

	// Expand environment variables before YAML parsing so that ${TOKEN} in
	// any field (URL, chat_id, headers…) is resolved at load time.
	expanded := os.ExpandEnv(string(data))

	var f file
	if err := yaml.Unmarshal([]byte(expanded), &f); err != nil {
		return Config{}, fmt.Errorf("config: parse yaml: %w", err)
	}

	for i := range f.Monitors {
		m := &f.Monitors[i]

		if m.Name == "" {
			return Config{}, fmt.Errorf("config: monitor[%d]: name is required", i)
		}
		if m.URL == "" {
			return Config{}, fmt.Errorf("config: monitor %q: url is required", m.Name)
		}

		if m.Method == "" {
			m.Method = "GET"
		}
		if m.ExpectedStatus == 0 {
			m.ExpectedStatus = 200
		}
		if m.Timeout == 0 {
			m.Timeout = 10 * time.Second
		}
		if m.Interval == 0 {
			m.Interval = time.Minute
		}
	}

	return Config{Monitors: f.Monitors, Webhooks: f.Webhooks}, nil
}
