package dockerfile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/katvixlab/contain-sentry/internal/entities"
)

func TestDockerfileFixtures(t *testing.T) {
	rules := loadDockerfileRules(t)
	testdataDir := filepath.Join("testdata")

	tests := []struct {
		name string
		file string
		want []string
	}{
		{name: "latest tag", file: "latest-tag.Dockerfile", want: []string{"DF001", "DF002"}},
		{name: "missing user", file: "missing-user.Dockerfile", want: []string{"DF005"}},
		{name: "root user", file: "root-user.Dockerfile", want: []string{"DF006", "DF007"}},
		{name: "secret env and arg", file: "secret-env-arg.Dockerfile", want: []string{"DF008", "DF009"}},
		{name: "curl pipe shell", file: "curl-pipe-shell.Dockerfile", want: []string{"DF012", "DF013"}},
		{name: "single stage build tooling", file: "single-stage-build-tools.Dockerfile", want: []string{"DF004", "DF019"}},
		{name: "secure dockerfile", file: "secure.Dockerfile", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df, err := NewDockerfile(context.Background(), filepath.Join(testdataDir, tt.file))
			if err != nil {
				t.Fatalf("NewDockerfile() error = %v", err)
			}

			findings, err := df.Validate(context.Background(), rules)
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

func loadDockerfileRules(t *testing.T) []entities.BaseRule {
	t.Helper()
	var rules []entities.BaseRule
	readDockerfileJSON(t, filepath.Join("..", "..", "dockerfile-rules.json"), &rules)
	return rules
}

func readDockerfileJSON(t *testing.T, path string, target any) {
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
