package dockerfile

import (
	"context"
	"fmt"
	"strings"

	"github.com/bonus2k/contain-sentry/internal/engine"
	"github.com/bonus2k/contain-sentry/internal/entities"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

const targetDockerfile = "dockerfile"

type DockerfileDriver struct {
	df        *Dockerfile
	i         int
	dom       *DockerStageEval
	finalized bool
	eofSent   bool
}

func NewDockerfileDriver(df *Dockerfile, dom *DockerStageEval) *DockerfileDriver {
	if dom == nil {
		dom = &DockerStageEval{}
	}
	if df != nil {
		df.context = dom
	}
	return &DockerfileDriver{df: df, dom: dom}
}

func (d *DockerfileDriver) Target() string {
	return targetDockerfile
}

func (d *DockerfileDriver) DomainContext() any {
	return d.dom
}

func (d *DockerfileDriver) Next(ctx context.Context) (engine.Step, bool, error) {
	_ = ctx
	if d.df == nil {
		return engine.Step{}, false, nil
	}

	if d.i >= len(d.df.nodes) {
		if !d.finalized {
			d.dom.ensureFinalized()
			d.finalized = true
		}
		if !d.eofSent {
			d.eofSent = true
			return engine.Step{Target: targetDockerfile, Subject: "eof", Raw: ""}, true, nil
		}
		return engine.Step{}, false, nil
	}

	node := d.df.nodes[d.i]
	d.i++

	instruction, err := instructions.ParseInstruction(node)
	if err != nil {
		return engine.Step{}, false, err
	}

	return engine.Step{
		Target:   targetDockerfile,
		Subject:  dockerSubject(instruction),
		Raw:      dockerRaw(instruction),
		Location: nodeLocation(node),
		Command:  instruction,
	}, true, nil
}

func (d *DockerfileDriver) Transfer(ctx context.Context, step engine.Step) error {
	_ = ctx
	if d.dom == nil {
		return nil
	}

	location, _ := step.Location.(SourceRef)

	switch command := step.Command.(type) {
	case *instructions.Stage:
		stg := instructionStage(len(d.dom.Stages), command)
		d.dom.startStage(stg)
	case *instructions.UserCommand:
		if !d.dom.hasFinal {
			return nil
		}
		d.dom.Final.User = Tracked[AbsString]{
			Val:      AbsString{Kind: "literal", Known: command.User},
			Location: location,
		}
		d.dom.Final.HasUser = true
	case *instructions.WorkdirCommand:
		if !d.dom.hasFinal {
			return nil
		}
		d.dom.Final.Workdir = Tracked[AbsString]{
			Val:      AbsString{Kind: "literal", Known: command.Path},
			Location: location,
		}
	case *instructions.EnvCommand:
		if !d.dom.hasFinal {
			return nil
		}
		if d.dom.Final.Env == nil {
			d.dom.Final.Env = map[string]Tracked[AbsString]{}
		}
		for _, kv := range command.Env {
			d.dom.Final.Env[kv.Key] = Tracked[AbsString]{
				Val:      AbsString{Kind: "literal", Known: kv.Value},
				Location: location,
			}
		}
	case *instructions.ArgCommand:
		if !d.dom.hasFinal {
			return nil
		}
		if d.dom.Final.Args == nil {
			d.dom.Final.Args = map[string]Tracked[AbsString]{}
		}
		for _, kv := range command.Args {
			value := ""
			if kv.Value != nil {
				value = *kv.Value
			}
			d.dom.Final.Args[kv.Key] = Tracked[AbsString]{
				Val:      AbsString{Kind: "literal", Known: value},
				Location: location,
			}
		}
	case *instructions.ShellCommand:
		if !d.dom.hasFinal {
			return nil
		}
		d.dom.Final.Shell = Tracked[[]string]{Val: append([]string{}, command.Shell...), Location: location}
	case *instructions.EntrypointCommand:
		if !d.dom.hasFinal {
			return nil
		}
		d.dom.Final.Entrypoint = Tracked[[]string]{Val: append([]string{}, command.CmdLine...), Location: location}
	case *instructions.CmdCommand:
		if !d.dom.hasFinal {
			return nil
		}
		d.dom.Final.Cmd = Tracked[[]string]{Val: append([]string{}, command.CmdLine...), Location: location}
	case *instructions.CopyCommand:
		if !d.dom.hasFinal {
			return nil
		}
		if strings.TrimSpace(command.From) != "" {
			d.dom.Final.HasCopyFrom = true
		}
	case *instructions.HealthCheckCommand:
		if !d.dom.hasFinal {
			return nil
		}
		d.dom.Final.HasHealthcheck = true
	case *instructions.RunCommand:
		if !d.dom.hasFinal {
			return nil
		}
		if runLooksLikeBuildTooling(step.Raw) {
			d.dom.Final.HasBuildTooling = true
		}
	}

	return nil
}

type DockerfileRunner struct{}

func (r *DockerfileRunner) Target() string {
	return targetDockerfile
}

func (r *DockerfileRunner) Eval(ctx context.Context, dom any, rule entities.BaseRule, step engine.Step) []entities.Finding {
	_ = ctx

	if !strings.EqualFold(strings.TrimSpace(rule.Subject), strings.TrimSpace(step.Subject)) {
		return nil
	}
	if rule.Expression == nil {
		return nil
	}

	if expression, ok := rule.Expression.(*entities.ExpressionDockerfileConstraint); ok {
		return r.evalDockerfileConstraint(rule, step, dom, expression)
	}

	if !rule.Expression.MatchCommand(step.Subject, step.Command, step.Raw) {
		return nil
	}

	return []entities.Finding{buildFinding(rule, step)}
}

func (r *DockerfileRunner) evalDockerfileConstraint(rule entities.BaseRule, step engine.Step, dom any, expression *entities.ExpressionDockerfileConstraint) []entities.Finding {
	if step.Subject != "eof" {
		return nil
	}

	state, ok := dom.(*DockerStageEval)
	if !ok {
		return nil
	}

	if !matchesDockerfileConstraint(state, expression.Check) {
		return nil
	}

	return []entities.Finding{buildFinding(rule, step)}
}

func buildFinding(rule entities.BaseRule, step engine.Step) entities.Finding {
	finding := entities.Finding{
		CodeSample: step.Raw,
		Location:   step.Location,
	}
	if rule.Metadata != nil {
		finding.ID = rule.Metadata.ID
		finding.Name = rule.Metadata.Name
		finding.Severity = rule.Metadata.Severity
		finding.Confidence = rule.Metadata.Confidence
		finding.Description = rule.Metadata.Description
	}
	return finding
}

// matchesDockerfileConstraint evaluates aggregate constraints that depend on the
// full Dockerfile state, primarily final-stage and multi-stage properties.
func matchesDockerfileConstraint(dom *DockerStageEval, check string) bool {
	if dom == nil || len(dom.Stages) == 0 {
		return false
	}

	finalStage := dom.Stages[len(dom.Stages)-1]
	switch check {
	case "missing_user_final_stage":
		return !finalStage.HasUser
	case "missing_healthcheck_final_stage":
		return !finalStage.HasHealthcheck
	case "missing_copy_from_in_multistage":
		return len(dom.Stages) >= 2 && !finalStage.HasCopyFrom
	case "single_stage_with_build_tools":
		return len(dom.Stages) == 1 && finalStage.HasBuildTooling
	default:
		return false
	}
}

func runLooksLikeBuildTooling(raw string) bool {
	r := strings.ToLower(raw)
	buildHints := []string{
		"apk add build-base",
		"build-essential",
		"apt-get install gcc",
		"apt-get install g++",
		" apt install gcc",
		" apt install g++",
		" make ",
		"go build",
		"gradle build",
		"mvn package",
		"cargo build",
	}

	for _, hint := range buildHints {
		if strings.Contains(r, hint) {
			return true
		}
	}
	return false
}

func (df *Dockerfile) Validate(ctx context.Context, rules []entities.BaseRule) ([]entities.Finding, error) {
	dom := &DockerStageEval{}
	driver := NewDockerfileDriver(df, dom)
	eng := engine.New(rules, &DockerfileRunner{})
	return eng.Run(ctx, driver)
}

func dockerSubject(command any) string {
	switch command.(type) {
	case *instructions.Stage:
		return "from"
	case *instructions.RunCommand:
		return "run"
	case *instructions.UserCommand:
		return "user"
	case *instructions.EnvCommand:
		return "env"
	case *instructions.WorkdirCommand:
		return "workdir"
	case *instructions.ArgCommand:
		return "arg"
	case *instructions.ShellCommand:
		return "shell"
	case *instructions.EntrypointCommand:
		return "entrypoint"
	case *instructions.CmdCommand:
		return "cmd"
	case *instructions.CopyCommand:
		return "copy"
	case *instructions.AddCommand:
		return "add"
	case *instructions.HealthCheckCommand:
		return "healthcheck"
	case *instructions.ExposeCommand:
		return "expose"
	default:
		return "unknown"
	}
}

func dockerRaw(command any) string {
	type stringer interface{ String() string }

	switch command := command.(type) {
	case *instructions.Stage:
		if command.SourceCode != "" {
			return command.SourceCode
		}
		if command.OrigCmd != "" {
			return command.OrigCmd
		}
		return fmt.Sprintf("FROM %s", command.BaseName)
	case stringer:
		return command.String()
	default:
		return ""
	}
}

func nodeLocation(node *parser.Node) SourceRef {
	if node == nil {
		return SourceRef{}
	}
	return SourceRef{
		Start: Position{Line: node.StartLine, Character: 0},
		End:   Position{Line: node.EndLine, Character: 0},
	}
}
