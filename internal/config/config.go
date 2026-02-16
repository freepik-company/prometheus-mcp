package config

import (
	"os"
	"prometheus-mcp/api"

	"gopkg.in/yaml.v3"
)

func Marshal(config api.Configuration) ([]byte, error) {
	return yaml.Marshal(config)
}

func Unmarshal(bytes []byte) (api.Configuration, error) {
	var config api.Configuration
	err := yaml.Unmarshal(bytes, &config)
	return config, err
}

// ReadFile reads and parses a configuration file, expanding environment variables.
// Supports ${VAR} and $VAR syntax for environment variable expansion.
func ReadFile(filepath string) (api.Configuration, error) {
	fileBytes, err := os.ReadFile(filepath)
	if err != nil {
		return api.Configuration{}, err
	}

	expandedContent := os.ExpandEnv(string(fileBytes))
	return Unmarshal([]byte(expandedContent))
}
