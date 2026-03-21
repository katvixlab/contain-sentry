package entities

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type ExpressionField struct {
	ExprKind  string         `json:"expr_kind,omitempty"`
	KindAlias string         `json:"kind,omitempty"`
	Select    string         `json:"select"`
	Expr      *FieldExprNode `json:"expr"`
}

func (e *ExpressionField) Kind() string {
	return "field"
}

func (e *ExpressionField) Match(_ string) bool {
	return false
}

func (e *ExpressionField) MatchCommand(_ string, _ any, _ string) bool {
	return false
}

func (e *ExpressionField) Evaluate(value any, present bool, resolver FieldResolver) bool {
	if e == nil || e.Expr == nil {
		return false
	}
	return e.Expr.Eval(FieldInput{Value: value, Present: present, Resolve: resolver})
}

type FieldResolver func(path string) (any, bool)

type FieldInput struct {
	Value   any
	Present bool
	Resolve FieldResolver
}

type FieldExprNode struct {
	Op string `json:"op"`

	Args []*FieldExprNode `json:"args,omitempty"`
	Arg  *FieldExprNode   `json:"arg,omitempty"`

	Select string `json:"select,omitempty"`

	Value  any   `json:"value,omitempty"`
	Values []any `json:"values,omitempty"`

	Pattern string `json:"pattern,omitempty"`

	compiled *regexp.Regexp
}

type fieldExprNodeAlias struct {
	Op string `json:"op"`

	Args []json.RawMessage `json:"args,omitempty"`
	Arg  json.RawMessage   `json:"arg,omitempty"`

	Select string `json:"select,omitempty"`

	Value  any   `json:"value,omitempty"`
	Values []any `json:"values,omitempty"`

	Pattern string `json:"pattern,omitempty"`
}

func (n *FieldExprNode) UnmarshalJSON(data []byte) error {
	var aux fieldExprNodeAlias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	n.Op = strings.ToLower(strings.TrimSpace(aux.Op))
	n.Select = strings.TrimSpace(aux.Select)
	n.Value = aux.Value
	n.Values = aux.Values
	n.Pattern = aux.Pattern

	switch n.Op {
	case "all", "any":
		n.Args = make([]*FieldExprNode, 0, len(aux.Args))
		for _, raw := range aux.Args {
			child := &FieldExprNode{}
			if err := json.Unmarshal(raw, child); err != nil {
				return err
			}
			n.Args = append(n.Args, child)
		}
	case "not":
		if len(aux.Arg) == 0 {
			return fmt.Errorf("op not requires arg")
		}
		n.Arg = &FieldExprNode{}
		if err := json.Unmarshal(aux.Arg, n.Arg); err != nil {
			return err
		}
	case "field":
		if n.Select == "" {
			return fmt.Errorf("op field requires select")
		}
		if len(aux.Arg) == 0 {
			return fmt.Errorf("op field requires arg")
		}
		n.Arg = &FieldExprNode{}
		if err := json.Unmarshal(aux.Arg, n.Arg); err != nil {
			return err
		}
	case "regex":
		rgx, err := regexp.Compile(n.Pattern)
		if err != nil {
			return fmt.Errorf("compile regex pattern %q: %w", n.Pattern, err)
		}
		n.compiled = rgx
	case "exists", "eq", "ne", "contains", "in":
	default:
		return fmt.Errorf("unknown field node op %q", n.Op)
	}

	return nil
}

func (n *FieldExprNode) Eval(input FieldInput) bool {
	if n == nil {
		return false
	}

	switch n.Op {
	case "exists":
		return input.Present
	case "eq":
		return compareFieldValue(input.Value, n.Value)
	case "ne":
		return !compareFieldValue(input.Value, n.Value)
	case "contains":
		return containsFieldValue(input.Value, n.Value)
	case "in":
		for _, candidate := range n.Values {
			if compareFieldValue(input.Value, candidate) {
				return true
			}
		}
		return false
	case "regex":
		for _, item := range stringifyCandidates(input.Value) {
			if n.ensureCompiled().MatchString(item) {
				return true
			}
		}
		return false
	case "all":
		for _, child := range n.Args {
			if !child.Eval(input) {
				return false
			}
		}
		return len(n.Args) > 0
	case "any":
		for _, child := range n.Args {
			if child.Eval(input) {
				return true
			}
		}
		return false
	case "not":
		if n.Arg == nil {
			return false
		}
		return !n.Arg.Eval(input)
	case "field":
		if n.Arg == nil || input.Resolve == nil {
			return false
		}
		value, present := input.Resolve(n.Select)
		return n.Arg.Eval(FieldInput{
			Value:   value,
			Present: present,
			Resolve: input.Resolve,
		})
	default:
		return false
	}
}

func (n *FieldExprNode) ensureCompiled() *regexp.Regexp {
	if n.compiled != nil {
		return n.compiled
	}
	rgx, err := regexp.Compile(n.Pattern)
	if err != nil {
		return regexp.MustCompile("$^")
	}
	n.compiled = rgx
	return n.compiled
}

func compareFieldValue(actual any, expected any) bool {
	actual = derefValue(actual)
	expected = derefValue(expected)
	switch av := actual.(type) {
	case string:
		return strings.EqualFold(strings.TrimSpace(av), stringifyScalar(expected))
	case bool:
		ev, ok := boolValue(expected)
		return ok && av == ev
	case int:
		ev, ok := intValue(expected)
		return ok && av == ev
	case int64:
		ev, ok := int64Value(expected)
		return ok && av == ev
	case float64:
		ev, ok := float64Value(expected)
		return ok && av == ev
	case fmt.Stringer:
		return strings.EqualFold(strings.TrimSpace(av.String()), stringifyScalar(expected))
	default:
		return strings.EqualFold(strings.TrimSpace(stringifyScalar(actual)), stringifyScalar(expected))
	}
}

func containsFieldValue(actual any, expected any) bool {
	actual = derefValue(actual)
	expected = derefValue(expected)
	switch av := actual.(type) {
	case string:
		target := stringifyScalar(expected)
		return target != "" && strings.Contains(strings.ToLower(av), target)
	}

	if items, ok := listCandidates(actual); ok {
		for _, item := range items {
			if compareFieldValue(item, expected) {
				return true
			}
		}
	}

	if mapping, ok := actual.(map[string]any); ok {
		key := stringifyScalar(expected)
		for k, v := range mapping {
			if strings.EqualFold(strings.TrimSpace(k), key) || compareFieldValue(v, expected) {
				return true
			}
		}
	}

	return false
}

func listCandidates(value any) ([]any, bool) {
	value = derefValue(value)
	switch typed := value.(type) {
	case []string:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return items, true
	case []any:
		return typed, true
	default:
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, false
		}
		var decoded []any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, false
		}
		return decoded, true
	}
}

func stringifyCandidates(value any) []string {
	value = derefValue(value)
	if value == nil {
		return nil
	}

	switch typed := value.(type) {
	case string:
		return []string{typed}
	case []string:
		return typed
	}

	raw, err := json.Marshal(value)
	if err == nil {
		return []string{string(raw)}
	}
	return []string{fmt.Sprintf("%v", value)}
}

func stringifyScalar(value any) string {
	value = derefValue(value)
	switch typed := value.(type) {
	case string:
		return strings.ToLower(strings.TrimSpace(typed))
	case nil:
		return ""
	default:
		return strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", value)))
	}
}

func boolValue(value any) (bool, bool) {
	value = derefValue(value)
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		v, err := strconv.ParseBool(strings.TrimSpace(typed))
		return v, err == nil
	default:
		return false, false
	}
}

func intValue(value any) (int, bool) {
	value = derefValue(value)
	switch typed := value.(type) {
	case int:
		return typed, true
	case float64:
		return int(typed), true
	case string:
		v, err := strconv.Atoi(strings.TrimSpace(typed))
		return v, err == nil
	default:
		return 0, false
	}
}

func int64Value(value any) (int64, bool) {
	value = derefValue(value)
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case string:
		v, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return v, err == nil
	default:
		return 0, false
	}
}

func float64Value(value any) (float64, bool) {
	value = derefValue(value)
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case string:
		v, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return v, err == nil
	default:
		return 0, false
	}
}

func derefValue(value any) any {
	if value == nil {
		return nil
	}
	rv := reflect.ValueOf(value)
	for rv.IsValid() && rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return nil
	}
	return rv.Interface()
}
