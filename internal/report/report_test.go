package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/katvixlab/contain-sentry/internal/entities"
)

func TestBuildSummary(t *testing.T) {
	report := Build([]entities.Finding{
		{ID: "1", Severity: "fail"},
		{ID: "2", Severity: "warn"},
		{ID: "3", Severity: "fail"},
	})
	if report.Summary.Total != 3 {
		t.Fatalf("Total = %d, want 3", report.Summary.Total)
	}
	if report.Summary.BySeverity["fail"] != 2 || report.Summary.BySeverity["warn"] != 1 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
}

func TestMarshalAndWriteJSON(t *testing.T) {
	report := Build([]entities.Finding{{
		ID:          "CP001",
		Name:        "name",
		Severity:    "warn",
		Description: "desc",
		Mitigation:  "fix",
		Reference:   "docs",
		Target:      "compose",
		Subject:     "service",
	}})

	payload, err := MarshalJSON(report)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	var decoded Report
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded.Summary.Total != 1 || len(decoded.Findings) != 1 {
		t.Fatalf("unexpected decoded report: %+v", decoded)
	}

	path := filepath.Join(t.TempDir(), "report.json")
	if err := WriteJSON(path, report); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("report file not written: %v", err)
	}
}
