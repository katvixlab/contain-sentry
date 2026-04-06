package engine

import "github.com/katvixlab/contain-sentry/internal/entities"

func BuildFinding(rule entities.BaseRule, step Step) entities.Finding {
	finding := entities.Finding{
		CodeSample: step.Raw,
		Location:   step.Location,
		Target:     step.Target,
		Subject:    step.Subject,
	}
	if rule.Metadata != nil {
		finding.ID = rule.Metadata.ID
		finding.Name = rule.Metadata.Name
		finding.Severity = rule.Metadata.Severity
		finding.Description = rule.Metadata.Description
		finding.Mitigation = rule.Metadata.Mitigation
		finding.Reference = rule.Metadata.Reference
	}
	return finding
}
