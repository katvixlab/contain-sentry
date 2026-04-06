package compose

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/katvixlab/contain-sentry/internal/entities"
)

func TestComposeFixtures(t *testing.T) {
	rules := loadComposeRules(t)
	testdataDir := filepath.Join("testdata")

	tests := []struct {
		name string
		file string
		want []string
	}{
		{name: "insecure root service", file: "insecure-root-service.compose.yaml", want: []string{"CP002"}},
		{name: "privileged service", file: "privileged-service.compose.yaml", want: []string{"CP003"}},
		{name: "host network", file: "host-network.compose.yaml", want: []string{"CP005"}},
		{name: "missing read_only", file: "missing-read-only.compose.yaml", want: []string{"CP004"}},
		{name: "missing healthcheck", file: "missing-healthcheck.compose.yaml", want: []string{"CP013"}},
		{name: "secret in environment", file: "secret-in-environment.compose.yaml", want: []string{"CP012", "CP022"}},
		{name: "userns host", file: "userns-host.compose.yaml", want: []string{"CP017"}},
		{name: "cap add sys admin", file: "cap-add-sys-admin.compose.yaml", want: []string{"CP018", "CP019"}},
		{name: "docker sock mount", file: "docker-sock-mount.compose.yaml", want: []string{"CP020"}},
		{name: "proc mount", file: "proc-mount.compose.yaml", want: []string{"CP021"}},
		{name: "public port", file: "public-port.compose.yaml", want: []string{"CP024"}},
		{name: "debug service without profiles", file: "debug-service-no-profiles.compose.yaml", want: []string{"CP025"}},
		{name: "build without image", file: "build-no-image.compose.yaml", want: []string{"CP026"}},
		{name: "service without explicit networks", file: "no-networks.compose.yaml", want: []string{"CP028"}},
		{name: "service without logging", file: "no-logging.compose.yaml", want: []string{"CP029"}},
		{name: "service without init", file: "no-init.compose.yaml", want: []string{"CP030"}},
		{name: "service without resource limits", file: "no-resource-limits.compose.yaml", want: []string{"CP023"}},
		{name: "service without stop controls", file: "no-stop-controls.compose.yaml", want: []string{"CP031"}},
		{name: "secure service", file: "secure-service.compose.yaml", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := NewProject(context.Background(), []string{filepath.Join(testdataDir, tt.file)})
			if err != nil {
				t.Fatalf("NewProject() error = %v", err)
			}

			findings, err := project.Validate(context.Background(), rules)
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}

			got := findingIDs(findings)
			sort.Strings(got)
			sort.Strings(tt.want)
			if len(got) != len(tt.want) {
				t.Fatalf("finding count = %d, want %d; got=%v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("findings[%d] = %q, want %q; all=%v", i, got[i], tt.want[i], got)
				}
			}
		})
	}
}

func loadComposeRules(t *testing.T) []entities.BaseRule {
	t.Helper()
	var rules []entities.BaseRule
	readJSON(t, filepath.Join("..", "..", "compose-rules.json"), &rules)
	return rules
}

func readJSON(t *testing.T, path string, target any) {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("Unmarshal(%q): %v", path, err)
	}
}

func findingIDs(findings []entities.Finding) []string {
	ids := make([]string, 0, len(findings))
	for _, finding := range findings {
		ids = append(ids, finding.ID)
	}
	return ids
}
