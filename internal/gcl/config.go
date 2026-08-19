package gcl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// fileConfig is the shape of the optional JSON config file used to
// configure self-hosted forge instances (see configPath).
type fileConfig struct {
	GitHubHosts map[string]string `json:"github_hosts"`
	GitLabHosts []string          `json:"gitlab_hosts"`
}

// configPath returns the location of the config file: GCL_CONFIG if set,
// otherwise <user config dir>/gcl/config.json.
func configPath() string {
	if path := os.Getenv("GCL_CONFIG"); path != "" {
		return path
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gcl", "config.json")
}

// loadFileConfig reads and parses the config file. A missing file is not
// an error; an unparsable one is reported on stderr and otherwise ignored.
func loadFileConfig() fileConfig {
	path := configPath()
	if path == "" {
		return fileConfig{}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fileConfig{}
	}

	var cfg fileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "gcl: ignoring invalid config file %s: %v\n", path, err)
		return fileConfig{}
	}
	return cfg
}
