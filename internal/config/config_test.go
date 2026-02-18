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
backends:
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
backends:
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
backends:
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
backends:
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
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			config, err := Unmarshal([]byte(tt.yaml))
			if err != nil {
				t.Fatalf("failed to unmarshal yaml: %v", err)
			}

			expandedYAML := os.ExpandEnv(tt.yaml)
			config, err = Unmarshal([]byte(expandedYAML))

			if (err != nil) != tt.wantErr {
				t.Errorf("got error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				backend, ok := config.Backends["prometheus"]
				if !ok {
					t.Fatal("expected 'prometheus' backend to exist")
				}
				if backend.Auth.Password != tt.wantPass {
					t.Errorf("got password = %q, want %q", backend.Auth.Password, tt.wantPass)
				}
			}
		})
	}
}

func TestBackendEnvVarExpansion(t *testing.T) {
	yaml := `
backends:
  pmm:
    url: "http://pmm:8080"
    auth:
      type: "basic"
      username: "admin"
      password: "${PMM_PASSWORD}"
`

	os.Setenv("PMM_PASSWORD", "pmm-secret")
	defer os.Unsetenv("PMM_PASSWORD")

	expandedYAML := os.ExpandEnv(yaml)
	config, err := Unmarshal([]byte(expandedYAML))
	if err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	backend, ok := config.Backends["pmm"]
	if !ok {
		t.Fatal("expected 'pmm' backend to exist")
	}
	if backend.Auth.Password != "pmm-secret" {
		t.Errorf("got PMM password = %q, want %q", backend.Auth.Password, "pmm-secret")
	}
}

func TestMultipleBackends(t *testing.T) {
	yaml := `
backends:
  prometheus:
    url: "http://localhost:9090"
    org_id: "tenant-1"
    available_orgs:
      - tenant-1
      - tenant-2
  pmm:
    url: "http://pmm:8080"
    auth:
      type: "basic"
      username: "admin"
      password: "secret"
  thanos:
    url: "http://thanos:9090"
    auth:
      type: "token"
      token: "my-token"
`

	config, err := Unmarshal([]byte(yaml))
	if err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	if len(config.Backends) != 3 {
		t.Errorf("expected 3 backends, got %d", len(config.Backends))
	}

	prom := config.Backends["prometheus"]
	if prom.URL != "http://localhost:9090" {
		t.Errorf("prometheus URL = %q, want %q", prom.URL, "http://localhost:9090")
	}
	if prom.OrgID != "tenant-1" {
		t.Errorf("prometheus OrgID = %q, want %q", prom.OrgID, "tenant-1")
	}
	if len(prom.AvailableOrgs) != 2 {
		t.Errorf("prometheus AvailableOrgs count = %d, want 2", len(prom.AvailableOrgs))
	}

	pmm := config.Backends["pmm"]
	if pmm.Auth.Type != "basic" {
		t.Errorf("pmm auth type = %q, want %q", pmm.Auth.Type, "basic")
	}

	thanos := config.Backends["thanos"]
	if thanos.Auth.Type != "token" {
		t.Errorf("thanos auth type = %q, want %q", thanos.Auth.Type, "token")
	}
	if thanos.Auth.Token != "my-token" {
		t.Errorf("thanos token = %q, want %q", thanos.Auth.Token, "my-token")
	}
}
