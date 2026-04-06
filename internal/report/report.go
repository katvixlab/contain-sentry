package report

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/katvixlab/contain-sentry/internal/entities"
)

type Report struct {
	Findings []ReportFinding `json:"findings"`
	Summary  ReportSummary   `json:"summary"`
}

type ReportFinding struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Severity    string `json:"severity,omitempty"`
	Description string `json:"description,omitempty"`
	Mitigation  string `json:"mitigation,omitempty"`
	Reference   string `json:"reference,omitempty"`
	CodeSample  string `json:"code_sample,omitempty"`
	Location    any    `json:"location,omitempty"`
	Target      string `json:"target,omitempty"`
	Subject     string `json:"subject,omitempty"`
}

type ReportSummary struct {
	Total      int            `json:"total"`
	BySeverity map[string]int `json:"by_severity,omitempty"`
}

func Build(findings []entities.Finding) Report {
	reportFindings := make([]ReportFinding, 0, len(findings))
	bySeverity := map[string]int{}
	for _, finding := range findings {
		reportFindings = append(reportFindings, ReportFinding{
			ID:          finding.ID,
			Name:        finding.Name,
			Severity:    finding.Severity,
			Description: finding.Description,
			Mitigation:  finding.Mitigation,
			Reference:   finding.Reference,
			CodeSample:  finding.CodeSample,
			Location:    finding.Location,
			Target:      finding.Target,
			Subject:     finding.Subject,
		})
		severity := strings.ToLower(strings.TrimSpace(finding.Severity))
		if severity == "" {
			severity = "unknown"
		}
		bySeverity[severity]++
	}

	return Report{
		Findings: reportFindings,
		Summary: ReportSummary{
			Total:      len(reportFindings),
			BySeverity: bySeverity,
		},
	}
}

func MarshalJSON(report Report) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

func WriteJSON(path string, report Report) error {
	payload, err := MarshalJSON(report)
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}
