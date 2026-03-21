package entities

import (
	"encoding/json"
	"testing"
)

func TestExpressionFieldEvaluate(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		value   any
		present bool
		want    bool
	}{
		{
			name:    "exists matches present field",
			raw:     `{"expr_kind":"field","select":"service.user","expr":{"op":"exists"}}`,
			value:   "1001",
			present: true,
			want:    true,
		},
		{
			name:    "eq matches bool",
			raw:     `{"expr_kind":"field","select":"service.read_only","expr":{"op":"eq","value":true}}`,
			value:   true,
			present: true,
			want:    true,
		},
		{
			name:    "contains matches list member",
			raw:     `{"expr_kind":"field","select":"service.cap_drop","expr":{"op":"contains","value":"ALL"}}`,
			value:   []string{"NET_RAW", "ALL"},
			present: true,
			want:    true,
		},
		{
			name:    "regex matches environment payload",
			raw:     `{"expr_kind":"field","select":"service.environment","expr":{"op":"regex","pattern":"(?i)password"}}`,
			value:   map[string]any{"APP_PASSWORD": "secret"},
			present: true,
			want:    true,
		},
		{
			name:    "all with not handles missing field",
			raw:     `{"expr_kind":"field","select":"service.healthcheck","expr":{"op":"all","args":[{"op":"not","arg":{"op":"exists"}},{"op":"not","arg":{"op":"eq","value":true}}]}}`,
			value:   nil,
			present: false,
			want:    true,
		},
		{
			name:    "in matches scalar",
			raw:     `{"expr_kind":"field","select":"service.restart","expr":{"op":"in","values":["no","always"]}}`,
			value:   "no",
			present: true,
			want:    true,
		},
		{
			name:    "field node resolves nested selector",
			raw:     `{"expr_kind":"field","select":"service","expr":{"op":"field","select":"service.secrets","arg":{"op":"not","arg":{"op":"exists"}}}}`,
			value:   map[string]any{"name": "app"},
			present: true,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var expr ExpressionField
			if err := json.Unmarshal([]byte(tt.raw), &expr); err != nil {
				t.Fatalf("unmarshal expression: %v", err)
			}
			resolver := FieldResolver(nil)
			if tt.name == "field node resolves nested selector" {
				resolver = func(path string) (any, bool) {
					if path == "service.secrets" {
						return nil, false
					}
					return nil, false
				}
			}
			if got := expr.Evaluate(tt.value, tt.present, resolver); got != tt.want {
				t.Fatalf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}
