package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/katvixlab/contain-sentry/internal/entities"
)

type ruleSet struct {
	Rules []entities.BaseRule `json:"rules"`
}

func loadRules(path string) ([]entities.BaseRule, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rules file %q: %w", path, err)
	}

	var rules []entities.BaseRule
	arrayErr := json.Unmarshal(payload, &rules)
	if arrayErr == nil {
		return rules, nil
	}

	var wrapped ruleSet
	if err := json.Unmarshal(payload, &wrapped); err != nil {
		return nil, fmt.Errorf("unmarshal rules file %q: array_err=%v, object_err=%w", path, arrayErr, err)
	}
	return wrapped.Rules, nil
}
