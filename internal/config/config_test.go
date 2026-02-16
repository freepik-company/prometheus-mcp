package config

import (
	"os"
	"testing"
)

func TestEnvVarExpansion(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		envVars  map[string]string
		wantPass string
		wantErr  bool
	}{
		{
			name: "expand ${VAR} syntax",
			yaml: `
prometheus:
  url: "http://localhost:9090"
  auth:
    type: "basic"
    username: "admin"
    password: "${TEST_PASSWORD}"
`,
			envVars: map[string]string{
				"TEST_PASSWORD": "secret123",
			},
			wantPass: "secret123",
			wantErr:  false,
		},
		{
			name: "expand $VAR syntax",
			yaml: `
prometheus:
  url: "http://localhost:9090"
  auth:
    type: "basic"
    username: "admin"
    password: "$TEST_PASSWORD"
`,
			envVars: map[string]string{
				"TEST_PASSWORD": "secret456",
			},
			wantPass: "secret456",
			wantErr:  false,
		},
		{
			name: "no expansion for plain text",
			yaml: `
prometheus:
  url: "http://localhost:9090"
  auth:
    type: "basic"
    username: "admin"
    password: "plain-password"
`,
			envVars:  map[string]string{},
			wantPass: "plain-password",
			wantErr:  false,
		},
		{
			name: "empty when env var not set",
			yaml: `
prometheus:
  url: "http://localhost:9090"
  auth:
    type: "basic"
    username: "admin"
    password: "${NONEXISTENT_VAR}"
`,
			envVars:  map[string]string{},
			wantPass: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// Parse YAML
			config, err := Unmarshal([]byte(tt.yaml))
			if err != nil {
				t.Fatalf("failed to unmarshal yaml: %v", err)
			}

			// Expand environment variables (simulating what ReadFile does)
			expandedYAML := os.ExpandEnv(tt.yaml)
			config, err = Unmarshal([]byte(expandedYAML))

			if (err != nil) != tt.wantErr {
				t.Errorf("got error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && config.Prometheus.Auth.Password != tt.wantPass {
				t.Errorf("got password = %q, want %q", config.Prometheus.Auth.Password, tt.wantPass)
			}
		})
	}
}

func TestPMMEnvVarExpansion(t *testing.T) {
	yaml := `
pmm:
  url: "http://pmm:8080"
  auth:
    type: "basic"
    username: "admin"
    password: "${PMM_PASSWORD}"
`

	// Set environment variable
	os.Setenv("PMM_PASSWORD", "pmm-secret")
	defer os.Unsetenv("PMM_PASSWORD")

	// Expand and parse
	expandedYAML := os.ExpandEnv(yaml)
	config, err := Unmarshal([]byte(expandedYAML))
	if err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	if config.PMM.Auth.Password != "pmm-secret" {
		t.Errorf("got PMM password = %q, want %q", config.PMM.Auth.Password, "pmm-secret")
	}
}
