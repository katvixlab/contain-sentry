package engine

import (
	"context"
	"strings"

	"github.com/bonus2k/contain-sentry/internal/entities"
)

type Step struct {
	Target   string
	Subject  string
	Raw      string
	Location any
	Command  any
}

type Driver interface {
	Target() string
	Next(ctx context.Context) (Step, bool, error)
	Transfer(ctx context.Context, step Step) error
	DomainContext() any
}

type Runner interface {
	Target() string
	Eval(ctx context.Context, dom any, rule entities.BaseRule, step Step) []entities.Finding
}

type Engine struct {
	rules   []entities.BaseRule
	runners map[string]Runner
}

func New(rules []entities.BaseRule, runners ...Runner) *Engine {
	runnerMap := make(map[string]Runner, len(runners))
	for _, runner := range runners {
		runnerMap[strings.ToLower(runner.Target())] = runner
	}

	return &Engine{
		rules:   rules,
		runners: runnerMap,
	}
}

func (e *Engine) Run(ctx context.Context, driver Driver) ([]entities.Finding, error) {
	runner, ok := e.runners[strings.ToLower(driver.Target())]
	if !ok {
		return nil, nil
	}

	var findings []entities.Finding
	for {
		step, hasNext, err := driver.Next(ctx)
		if err != nil {
			return findings, err
		}
		if !hasNext {
			break
		}

		findings = append(findings, e.evalPhase(ctx, runner, driver, step, "pre")...)
		if err := driver.Transfer(ctx, step); err != nil {
			return findings, err
		}
		findings = append(findings, e.evalPhase(ctx, runner, driver, step, "post")...)
	}

	return findings, nil
}

func (e *Engine) evalPhase(ctx context.Context, runner Runner, driver Driver, step Step, phase string) []entities.Finding {
	var findings []entities.Finding
	for _, rule := range e.rules {
		if !sameString(rule.Target, step.Target) {
			continue
		}

		if !matchPhase(rule.Phase, phase) {
			continue
		}

		findings = append(findings, runner.Eval(ctx, driver.DomainContext(), rule, step)...)
	}
	return findings
}

func matchPhase(rulePhase, actual string) bool {
	if sameString(rulePhase, actual) {
		return true
	}

	if strings.TrimSpace(rulePhase) == "" {
		return actual == "post"
	}

	return false
}

func sameString(x, y string) bool {
	return strings.EqualFold(strings.TrimSpace(x), strings.TrimSpace(y))
}
