package entities

import (
	"encoding/json"
	"fmt"
)

type BaseRule struct {
	Target     string     `json:"target"`
	Phase      string     `json:"phase"`
	Subject    string     `json:"subject"`
	Expression Expression `json:"expression"`
	Metadata   *Metadata  `json:"metadata,omitempty"`
}

type baseRuleAlias struct {
	Target     string          `json:"target"`
	Phase      string          `json:"phase"`
	Subject    string          `json:"subject"`
	Expression json.RawMessage `json:"expression"`
	Metadata   *Metadata       `json:"metadata,omitempty"`
}

func (r *BaseRule) UnmarshalJSON(data []byte) error {
	var aux baseRuleAlias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	expr, err := UnmarshalExpression(aux.Expression)
	if err != nil {
		return fmt.Errorf("unmarshal rule expression: %w", err)
	}

	r.Target = aux.Target
	r.Phase = aux.Phase
	r.Subject = aux.Subject
	r.Expression = expr
	r.Metadata = aux.Metadata
	return nil
}

type EqualType int

const (
	Eq EqualType = iota
	Ne
	Gt
	Lt
	In
	Nin
)

type MatchType int

const (
	AnyMatch MatchType = iota
	AllMatch
)

type Metadata struct {
	ID            string   `json:"id,omitempty"`
	Name          string   `json:"name,omitempty"`
	Description   string   `json:"description,omitempty"`
	Type          string   `json:"type,omitempty"`
	Severity      string   `json:"severity,omitempty"`
	Confidence    string   `json:"confidence,omitempty"`
	CWEs          []string `json:"cwes,omitempty"`
	CVEs          []string `json:"cves,omitempty"`
	Mitigation    string   `json:"mitigation,omitempty"`
	Reference     string   `json:"reference,omitempty"`
	SafeExample   string   `json:"safe_example,omitempty"`
	UnsafeExample string   `json:"unsafe_example,omitempty"`
}

type Finding struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Severity    string `json:"severity,omitempty"`
	CodeSample  string `json:"code_sample,omitempty"`
	Confidence  string `json:"confidence,omitempty"`
	Description string `json:"description,omitempty"`
	Location    any    `json:"location,omitempty"`
}
