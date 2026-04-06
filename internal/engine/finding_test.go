package engine

import (
	"testing"

	"github.com/katvixlab/contain-sentry/internal/entities"
)

func TestBuildFindingEnrichesMetadata(t *testing.T) {
	rule := entities.BaseRule{
		Metadata: &entities.Metadata{
			ID:          "CP999",
			Name:        "example",
			Severity:    "warn",
			Description: "desc",
			Mitigation:  "fix it",
			Reference:   "docs",
		},
	}
	step := Step{
		Target:   "compose",
		Subject:  "service",
		Raw:      "services.app",
		Location: "loc",
	}

	finding := BuildFinding(rule, step)
	if finding.ID != "CP999" || finding.Name != "example" || finding.Severity != "warn" {
		t.Fatalf("unexpected basic metadata: %+v", finding)
	}
	if finding.Description != "desc" || finding.Mitigation != "fix it" || finding.Reference != "docs" {
		t.Fatalf("unexpected enriched metadata: %+v", finding)
	}
	if finding.Target != "compose" || finding.Subject != "service" {
		t.Fatalf("unexpected target/subject: %+v", finding)
	}
}
