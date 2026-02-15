package entities

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"mvdan.cc/sh/v3/syntax"
)

type Expression interface {
	Kind() string
	Match(input string) bool
	MatchCommand(subject string, command any, raw string) bool
}

type expressionProbe struct {
	ExprKind string `json:"expr_kind"`
	Kind     string `json:"kind"`
}

func UnmarshalExpression(raw json.RawMessage) (Expression, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var probe expressionProbe
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}

	kind := probe.ExprKind
	if kind == "" {
		kind = probe.Kind
	}

	switch strings.ToLower(kind) {
	case "regex":
		var expr ExpressionRegex
		if err := json.Unmarshal(raw, &expr); err != nil {
			return nil, err
		}
		return &expr, nil
	case "user_id_compare":
		var expr ExpressionUserIDCompare
		if err := json.Unmarshal(raw, &expr); err != nil {
			return nil, err
		}
		return &expr, nil
	case "dockerfile_constraint":
		var expr ExpressionDockerfileConstraint
		if err := json.Unmarshal(raw, &expr); err != nil {
			return nil, err
		}
		return &expr, nil
	case "dsl":
		var expr ExpressionDSL
		if err := json.Unmarshal(raw, &expr); err != nil {
			return nil, err
		}
		return &expr, nil
	case "":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown expression kind %q", kind)
	}
}

type ExpressionRegex struct {
	ExprKind    string    `json:"expr_kind,omitempty"`
	KindAlias   string    `json:"kind,omitempty"`
	Type        MatchType `json:"type,omitempty"`
	Expressions []string  `json:"expressions,omitempty"`

	compiled []*regexp.Regexp
}

func (r *ExpressionRegex) UnmarshalJSON(data []byte) error {
	type alias ExpressionRegex
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	kind := aux.ExprKind
	if kind == "" {
		kind = aux.KindAlias
	}
	if kind == "" {
		kind = "regex"
	}
	if strings.ToLower(kind) != "regex" {
		return fmt.Errorf("invalid regex expression kind %q", kind)
	}

	compiled := make([]*regexp.Regexp, 0, len(aux.Expressions))
	for _, expression := range aux.Expressions {
		rgx, err := regexp.Compile(expression)
		if err != nil {
			return fmt.Errorf("compile regex %q: %w", expression, err)
		}
		compiled = append(compiled, rgx)
	}

	*r = ExpressionRegex(aux)
	r.ExprKind = "regex"
	r.compiled = compiled
	return nil
}

func (r *ExpressionRegex) Kind() string {
	return "regex"
}

func (r *ExpressionRegex) Match(command string) bool {
	r.ensureCompiled()

	switch r.Type {
	case AnyMatch:
		return r.runAnyMatch(command)
	case AllMatch:
		return r.runAllMatch(command)
	default:
		return r.runAnyMatch(command)
	}
}

func (r *ExpressionRegex) ensureCompiled() {
	if len(r.compiled) > 0 || len(r.Expressions) == 0 {
		return
	}

	compiled := make([]*regexp.Regexp, 0, len(r.Expressions))
	for _, expression := range r.Expressions {
		rgx, err := regexp.Compile(expression)
		if err != nil {
			continue
		}
		compiled = append(compiled, rgx)
	}
	r.compiled = compiled
}

func (r *ExpressionRegex) runAnyMatch(command string) bool {
	for _, expression := range r.compiled {
		if expression.FindString(command) != "" {
			return true
		}
	}
	return false
}

func (r *ExpressionRegex) runAllMatch(command string) bool {
	for _, expression := range r.compiled {
		if expression.FindString(command) == "" {
			return false
		}
	}
	return len(r.compiled) > 0
}

func (r *ExpressionRegex) MatchCommand(subject string, command any, raw string) bool {
	normalizedSubject := strings.ToLower(strings.TrimSpace(subject))
	switch normalizedSubject {
	case "env":
		if env, ok := command.(*instructions.EnvCommand); ok {
			return r.matchEnvPairs(env, raw)
		}
	case "arg":
		if arg, ok := command.(*instructions.ArgCommand); ok {
			return r.matchArgPairs(arg, raw)
		}
	}
	return r.Match(raw)
}

func (r *ExpressionRegex) matchEnvPairs(env *instructions.EnvCommand, raw string) bool {
	if env == nil {
		return r.Match(raw)
	}
	for _, kv := range env.Env {
		if r.Match(kv.Key) || r.Match(kv.Value) || r.Match(kv.Key+"="+kv.Value) {
			return true
		}
	}
	return r.Match(raw)
}

func (r *ExpressionRegex) matchArgPairs(arg *instructions.ArgCommand, raw string) bool {
	if arg == nil {
		return r.Match(raw)
	}
	for _, kv := range arg.Args {
		if r.Match(kv.Key) {
			return true
		}
		if kv.Value != nil {
			if r.Match(*kv.Value) || r.Match(kv.Key+"="+*kv.Value) {
				return true
			}
		}
	}
	return r.Match(raw)
}

type ExpressionUserIDCompare struct {
	ExprKind  string `json:"expr_kind,omitempty"`
	KindAlias string `json:"kind,omitempty"`
	Operator  string `json:"operator"`
	Value     int    `json:"value"`
}

func (e *ExpressionUserIDCompare) Kind() string {
	return "user_id_compare"
}

func (e *ExpressionUserIDCompare) Match(input string) bool {
	uid, ok := extractUserID(input)
	if !ok {
		return false
	}
	return compareInt(uid, e.Operator, e.Value)
}

func (e *ExpressionUserIDCompare) MatchCommand(subject string, _ any, raw string) bool {
	if !strings.EqualFold(strings.TrimSpace(subject), "user") {
		return false
	}
	return e.Match(raw)
}

type ExpressionDockerfileConstraint struct {
	ExprKind  string `json:"expr_kind,omitempty"`
	KindAlias string `json:"kind,omitempty"`
	Check     string `json:"check"`
}

func (e *ExpressionDockerfileConstraint) Kind() string {
	return "dockerfile_constraint"
}

func (e *ExpressionDockerfileConstraint) Match(_ string) bool {
	return false
}

func (e *ExpressionDockerfileConstraint) MatchCommand(_ string, _ any, _ string) bool {
	return false
}

type ExpressionDSL struct {
	ExprKind  string    `json:"expr_kind,omitempty"`
	KindAlias string    `json:"kind,omitempty"`
	Select    string    `json:"select"`
	Expr      *ExprNode `json:"expr"`
}

func (e *ExpressionDSL) Kind() string {
	return "dsl"
}

func (e *ExpressionDSL) Match(_ string) bool {
	return false
}

func (e *ExpressionDSL) MatchCommand(subject string, command any, raw string) bool {
	if !strings.EqualFold(strings.TrimSpace(subject), "run") {
		return false
	}
	if e.Expr == nil {
		return false
	}

	facts := buildRunFacts(command, raw)
	ctx := evalContext{facts: facts}
	switch strings.ToLower(strings.TrimSpace(e.Select)) {
	case "run.script", "run.mounts":
		return e.Expr.eval(ctx)
	default:
		return false
	}
}

type ExprNode struct {
	Op string `json:"op"`

	Args  []*ExprNode `json:"args,omitempty"`
	Arg   *ExprNode   `json:"arg,omitempty"`
	Where *ExprNode   `json:"where,omitempty"`
	Left  *ExprNode   `json:"left,omitempty"`
	Right *ExprNode   `json:"right,omitempty"`

	Name *Matcher  `json:"name,omitempty"`
	Call *CallArgs `json:"-"`

	MountType string   `json:"type,omitempty"`
	Target    *Matcher `json:"target,omitempty"`
	Has       []string `json:"has,omitempty"`
	Missing   []string `json:"missing,omitempty"`
	ID        *Matcher `json:"id,omitempty"`
	Sharing   *Matcher `json:"sharing,omitempty"`
}

type exprNodeAlias struct {
	Op string `json:"op"`

	Args  json.RawMessage `json:"args,omitempty"`
	Arg   json.RawMessage `json:"arg,omitempty"`
	Where json.RawMessage `json:"where,omitempty"`
	Left  json.RawMessage `json:"left,omitempty"`
	Right json.RawMessage `json:"right,omitempty"`

	Name *Matcher  `json:"name,omitempty"`
	Call *CallArgs `json:"args_match,omitempty"`

	MountType string   `json:"type,omitempty"`
	Target    *Matcher `json:"target,omitempty"`
	Has       []string `json:"has,omitempty"`
	Missing   []string `json:"missing,omitempty"`
	ID        *Matcher `json:"id,omitempty"`
	Sharing   *Matcher `json:"sharing,omitempty"`
}

func (n *ExprNode) UnmarshalJSON(data []byte) error {
	var aux exprNodeAlias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	n.Op = strings.ToLower(strings.TrimSpace(aux.Op))
	n.Name = aux.Name
	n.Call = aux.Call
	n.MountType = strings.ToLower(strings.TrimSpace(aux.MountType))
	n.Target = aux.Target
	n.Has = toLowerSlice(aux.Has)
	n.Missing = toLowerSlice(aux.Missing)
	n.ID = aux.ID
	n.Sharing = aux.Sharing

	switch n.Op {
	case "all", "any":
		var children []json.RawMessage
		if err := json.Unmarshal(aux.Args, &children); err != nil {
			return fmt.Errorf("op %s requires args array: %w", n.Op, err)
		}
		n.Args = make([]*ExprNode, 0, len(children))
		for _, raw := range children {
			child := &ExprNode{}
			if err := json.Unmarshal(raw, child); err != nil {
				return err
			}
			n.Args = append(n.Args, child)
		}
	case "not":
		if len(aux.Arg) == 0 {
			return fmt.Errorf("op not requires arg")
		}
		n.Arg = &ExprNode{}
		if err := json.Unmarshal(aux.Arg, n.Arg); err != nil {
			return err
		}
	case "exists":
		if len(aux.Where) == 0 {
			return fmt.Errorf("op exists requires where")
		}
		n.Where = &ExprNode{}
		if err := json.Unmarshal(aux.Where, n.Where); err != nil {
			return err
		}
	case "pipe":
		if len(aux.Left) == 0 || len(aux.Right) == 0 {
			return fmt.Errorf("op pipe requires left and right")
		}
		n.Left = &ExprNode{}
		n.Right = &ExprNode{}
		if err := json.Unmarshal(aux.Left, n.Left); err != nil {
			return err
		}
		if err := json.Unmarshal(aux.Right, n.Right); err != nil {
			return err
		}
	case "call":
		if len(aux.Args) != 0 {
			callArgs := &CallArgs{}
			if err := json.Unmarshal(aux.Args, callArgs); err != nil {
				return fmt.Errorf("op call requires args object: %w", err)
			}
			n.Call = callArgs
		}
	case "mount":
		return nil
	default:
		return fmt.Errorf("unknown node op %q", n.Op)
	}

	return nil
}

type CallArgs struct {
	Any []*Matcher `json:"any,omitempty"`
	All []*Matcher `json:"all,omitempty"`
}

type Matcher struct {
	Op      string   `json:"op"`
	Value   string   `json:"value,omitempty"`
	Values  []string `json:"values,omitempty"`
	Pattern string   `json:"pattern,omitempty"`

	compiled *regexp.Regexp
}

func (m *Matcher) UnmarshalJSON(data []byte) error {
	type alias Matcher
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	m.Op = strings.ToLower(strings.TrimSpace(aux.Op))
	m.Value = strings.ToLower(aux.Value)
	m.Pattern = aux.Pattern
	m.Values = toLowerSlice(aux.Values)

	if m.Op == "regex" {
		rgx, err := regexp.Compile(m.Pattern)
		if err != nil {
			return fmt.Errorf("compile regex pattern %q: %w", m.Pattern, err)
		}
		m.compiled = rgx
	}
	return nil
}

func (m *Matcher) Match(input string) bool {
	if m == nil {
		return true
	}
	value := strings.ToLower(input)

	switch m.Op {
	case "eq":
		return value == m.Value
	case "contains":
		return strings.Contains(value, m.Value)
	case "in":
		for _, item := range m.Values {
			if value == item {
				return true
			}
		}
		return false
	case "regex":
		if m.compiled == nil {
			rgx, err := regexp.Compile(m.Pattern)
			if err != nil {
				return false
			}
			m.compiled = rgx
		}
		return m.compiled.MatchString(input)
	default:
		return false
	}
}

type evalContext struct {
	facts       RunFacts
	call        *CallFact
	mount       *MountSpec
	pipe        *PipeFact
	selectScope string
}

func (n *ExprNode) eval(ctx evalContext) bool {
	if n == nil {
		return false
	}

	switch n.Op {
	case "all":
		for _, child := range n.Args {
			if !child.eval(ctx) {
				return false
			}
		}
		return len(n.Args) > 0
	case "any":
		for _, child := range n.Args {
			if child.eval(ctx) {
				return true
			}
		}
		return false
	case "not":
		if n.Arg == nil {
			return false
		}
		return !n.Arg.eval(ctx)
	case "exists":
		if n.Where == nil {
			return false
		}
		switch n.Where.Op {
		case "call":
			for i := range ctx.facts.Calls {
				call := ctx.facts.Calls[i]
				child := ctx
				child.call = &call
				if n.Where.eval(child) {
					return true
				}
			}
			return false
		case "pipe":
			for i := range ctx.facts.Pipes {
				pipe := ctx.facts.Pipes[i]
				child := ctx
				child.pipe = &pipe
				if n.Where.eval(child) {
					return true
				}
			}
			return false
		case "mount":
			for i := range ctx.facts.Mounts {
				mount := ctx.facts.Mounts[i]
				child := ctx
				child.mount = &mount
				if n.Where.eval(child) {
					return true
				}
			}
			return false
		default:
			return n.Where.eval(ctx)
		}
	case "call":
		if ctx.call != nil {
			return n.matchCall(*ctx.call)
		}
		for _, call := range ctx.facts.Calls {
			if n.matchCall(call) {
				return true
			}
		}
		return false
	case "pipe":
		if ctx.pipe != nil {
			left := ctx
			left.call = &ctx.pipe.First
			right := ctx
			right.call = &ctx.pipe.Last
			return n.Left != nil && n.Right != nil && n.Left.eval(left) && n.Right.eval(right)
		}
		for _, pipe := range ctx.facts.Pipes {
			left := ctx
			right := ctx
			left.call = &pipe.First
			right.call = &pipe.Last
			if n.Left != nil && n.Right != nil && n.Left.eval(left) && n.Right.eval(right) {
				return true
			}
		}
		return false
	case "mount":
		if ctx.mount != nil {
			return n.matchMount(*ctx.mount)
		}
		for _, mount := range ctx.facts.Mounts {
			if n.matchMount(mount) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (n *ExprNode) matchCall(call CallFact) bool {
	if n.Name != nil && !n.Name.Match(call.Name) {
		return false
	}
	if n.Call == nil {
		return true
	}

	if len(n.Call.Any) > 0 {
		matched := false
		for _, matcher := range n.Call.Any {
			for _, arg := range call.Args {
				if matcher.Match(arg) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			return false
		}
	}

	for _, matcher := range n.Call.All {
		matched := false
		for _, arg := range call.Args {
			if matcher.Match(arg) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

func (n *ExprNode) matchMount(mount MountSpec) bool {
	if n.MountType != "" && !strings.EqualFold(n.MountType, mount.Type) {
		return false
	}
	if n.Target != nil && !n.Target.Match(mount.Target) {
		return false
	}
	if n.ID != nil && !n.ID.Match(mount.ID) {
		return false
	}
	if n.Sharing != nil && !n.Sharing.Match(mount.Sharing) {
		return false
	}
	for _, key := range n.Has {
		if _, ok := mount.Options[key]; !ok {
			return false
		}
	}
	for _, key := range n.Missing {
		if _, ok := mount.Options[key]; ok {
			return false
		}
	}
	return true
}

type RunFacts struct {
	Calls  []CallFact
	Pipes  []PipeFact
	Mounts []MountSpec
}

type CallFact struct {
	Name string
	Args []string
}

type PipeFact struct {
	First CallFact
	Last  CallFact
}

type MountSpec struct {
	Type    string
	Target  string
	ID      string
	Sharing string
	Raw     string
	Options map[string]string
}

func buildRunFacts(command any, raw string) RunFacts {
	facts := RunFacts{}
	facts.Mounts = parseRunMounts(command, raw)

	script := extractRunScript(command, raw)
	if strings.TrimSpace(script) == "" {
		return facts
	}

	calls, pipes := collectScriptFacts(script)
	facts.Calls = append(facts.Calls, calls...)
	facts.Pipes = append(facts.Pipes, pipes...)
	return facts
}

func collectScriptFacts(script string) ([]CallFact, []PipeFact) {
	parsed, err := syntax.NewParser().Parse(strings.NewReader(script), "dockerfile_run")
	if err != nil {
		return nil, nil
	}

	calls := make([]CallFact, 0)
	pipes := make([]PipeFact, 0)
	syntax.Walk(parsed, func(node syntax.Node) bool {
		switch current := node.(type) {
		case *syntax.CallExpr:
			if call, ok := callFromExpr(current); ok {
				calls = append(calls, call)
			}
			if nestedScript, ok := nestedShellScript(current); ok {
				nestedCalls, nestedPipes := collectScriptFacts(nestedScript)
				calls = append(calls, nestedCalls...)
				pipes = append(pipes, nestedPipes...)
			}
		case *syntax.BinaryCmd:
			if current.Op != syntax.Pipe {
				return true
			}
			leftCalls := collectCallsFromNode(current.X)
			rightCalls := collectCallsFromNode(current.Y)
			if len(leftCalls) == 0 || len(rightCalls) == 0 {
				return true
			}
			pipes = append(pipes, PipeFact{First: leftCalls[0], Last: rightCalls[len(rightCalls)-1]})
		}
		return true
	})

	return calls, pipes
}

func collectCallsFromNode(node syntax.Node) []CallFact {
	if node == nil {
		return nil
	}
	calls := make([]CallFact, 0)
	syntax.Walk(node, func(current syntax.Node) bool {
		if callExpr, ok := current.(*syntax.CallExpr); ok {
			if call, ok := callFromExpr(callExpr); ok {
				calls = append(calls, call)
			}
		}
		return true
	})
	return calls
}

func callFromExpr(callExpr *syntax.CallExpr) (CallFact, bool) {
	if callExpr == nil || len(callExpr.Args) == 0 {
		return CallFact{}, false
	}

	name := wordToString(callExpr.Args[0])
	if strings.TrimSpace(name) == "" {
		return CallFact{}, false
	}

	args := make([]string, 0, len(callExpr.Args)-1)
	for i := 1; i < len(callExpr.Args); i++ {
		arg := strings.TrimSpace(wordToString(callExpr.Args[i]))
		if arg == "" {
			continue
		}
		args = append(args, strings.ToLower(arg))
	}

	return CallFact{Name: strings.ToLower(name), Args: args}, true
}

func wordToString(word *syntax.Word) string {
	if word == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := syntax.NewPrinter().Print(&buf, word); err == nil {
		return strings.TrimSpace(buf.String())
	}
	return ""
}

func nestedShellScript(callExpr *syntax.CallExpr) (string, bool) {
	if callExpr == nil || len(callExpr.Args) < 3 {
		return "", false
	}

	name := strings.ToLower(strings.TrimSpace(wordToString(callExpr.Args[0])))
	if name != "sh" && name != "bash" {
		return "", false
	}
	flag := strings.TrimSpace(wordToString(callExpr.Args[1]))
	if flag != "-c" {
		return "", false
	}

	script := strings.TrimSpace(wordToString(callExpr.Args[2]))
	if len(script) >= 2 {
		if (script[0] == '\'' && script[len(script)-1] == '\'') || (script[0] == '"' && script[len(script)-1] == '"') {
			script = script[1 : len(script)-1]
		}
	}
	if script == "" {
		return "", false
	}
	return script, true
}

func extractRunScript(command any, raw string) string {
	if run, ok := command.(*instructions.RunCommand); ok {
		if len(run.CmdLine) > 0 {
			return strings.Join(run.CmdLine, " ")
		}
	}

	trimmed := strings.TrimSpace(raw)
	if len(trimmed) >= 4 && strings.EqualFold(trimmed[:4], "RUN ") {
		return strings.TrimSpace(trimmed[4:])
	}
	return trimmed
}

func parseRunMounts(command any, raw string) []MountSpec {
	flags := make([]string, 0)
	if run, ok := command.(*instructions.RunCommand); ok {
		flags = append(flags, run.FlagsUsed...)
	}

	// Buildkit may expose FlagsUsed as ["mount"] without payload,
	// so always parse raw RUN text as source of truth for mount specs.
	matcher := regexp.MustCompile(`--mount=([^\s]+)`)
	matches := matcher.FindAllStringSubmatch(raw, -1)
	for _, match := range matches {
		if len(match) > 1 {
			flags = append(flags, "--mount="+match[1])
		}
	}

	seen := map[string]struct{}{}
	mounts := make([]MountSpec, 0)
	for _, flag := range flags {
		if !strings.HasPrefix(flag, "--mount=") {
			continue
		}
		rawValue := strings.TrimPrefix(flag, "--mount=")
		if rawValue == "" {
			continue
		}
		if _, ok := seen[rawValue]; ok {
			continue
		}
		seen[rawValue] = struct{}{}
		options := map[string]string{}
		for _, part := range strings.Split(rawValue, ",") {
			item := strings.TrimSpace(part)
			if item == "" {
				continue
			}
			chunks := strings.SplitN(item, "=", 2)
			key := strings.ToLower(strings.TrimSpace(chunks[0]))
			value := ""
			if len(chunks) == 2 {
				value = strings.ToLower(strings.TrimSpace(chunks[1]))
			}
			options[key] = value
		}

		mounts = append(mounts, MountSpec{
			Type:    options["type"],
			Target:  options["target"],
			ID:      options["id"],
			Sharing: options["sharing"],
			Raw:     rawValue,
			Options: options,
		})
	}

	return mounts
}

func extractUserID(raw string) (int, bool) {
	clean := strings.TrimSpace(raw)
	if len(clean) == 0 {
		return 0, false
	}

	upper := strings.ToUpper(clean)
	if strings.HasPrefix(upper, "USER ") {
		clean = strings.TrimSpace(clean[4:])
	}

	parts := strings.Fields(clean)
	if len(parts) == 0 {
		return 0, false
	}

	principal := strings.Split(parts[0], ":")[0]
	if strings.EqualFold(principal, "root") {
		return 0, true
	}

	uid, err := strconv.Atoi(principal)
	if err != nil {
		return 0, false
	}
	return uid, true
}

func compareInt(left int, op string, right int) bool {
	switch strings.TrimSpace(op) {
	case ">":
		return left > right
	case ">=":
		return left >= right
	case "<":
		return left < right
	case "<=":
		return left <= right
	case "==", "=":
		return left == right
	case "!=":
		return left != right
	default:
		return false
	}
}

func toLowerSlice(input []string) []string {
	result := make([]string, 0, len(input))
	for _, item := range input {
		result = append(result, strings.ToLower(strings.TrimSpace(item)))
	}
	return result
}
