package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/rubensantoniorosa2704/schemaping-worker/pkg/types"
)

type file struct {
	Monitors []types.Monitor `yaml:"monitors"`
}

// Load reads a YAML config file and returns validated monitors with defaults applied.
func Load(path string) ([]types.Monitor, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read file: %w", err)
	}

	var f file
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("config: parse yaml: %w", err)
	}

	for i := range f.Monitors {
		m := &f.Monitors[i]

		if m.Name == "" {
			return nil, fmt.Errorf("config: monitor[%d]: name is required", i)
		}
		if m.URL == "" {
			return nil, fmt.Errorf("config: monitor %q: url is required", m.Name)
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

	return f.Monitors, nil
}
