package compose

import (
	"context"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/katvixlab/contain-sentry/internal/compose/model"
	"github.com/katvixlab/contain-sentry/internal/engine"
	"github.com/katvixlab/contain-sentry/internal/entities"
)

func TestComposeRunnerEval(t *testing.T) {
	project := &model.Project{
		Name:     "demo",
		Files:    []string{"compose.yaml"},
		Services: map[string]*model.Service{},
	}
	service := &model.Service{
		Name: "app",
		Config: types.ServiceConfig{
			Name:       "app",
			Privileged: true,
		},
		Present: map[string]bool{"privileged": true, "service": true, "name": true},
		Project: project,
	}
	project.Services["app"] = service

	rule := entities.BaseRule{
		Target:  "compose",
		Phase:   "post",
		Subject: "privileged",
		Metadata: &entities.Metadata{
			ID: "CP003",
		},
		Expression: &entities.ExpressionField{
			ExprKind: "field",
			Select:   "service.privileged",
			Expr: &entities.FieldExprNode{
				Op:    "eq",
				Value: true,
			},
		},
	}

	step := engine.Step{
		Target:  "compose",
		Subject: "privileged",
		Service: "app",
		Raw:     "true",
	}

	findings := (&ComposeRunner{}).Eval(context.Background(), project, rule, step)
	if len(findings) != 1 {
		t.Fatalf("Eval() findings = %d, want 1", len(findings))
	}
	if findings[0].ID != "CP003" {
		t.Fatalf("finding ID = %q, want %q", findings[0].ID, "CP003")
	}
}

func TestComposeRunnerEvalSubjectMismatch(t *testing.T) {
	project := &model.Project{
		Services: map[string]*model.Service{},
	}

	rule := entities.BaseRule{
		Target:  "compose",
		Subject: "user",
		Expression: &entities.ExpressionField{
			ExprKind: "field",
			Select:   "service.user",
			Expr:     &entities.FieldExprNode{Op: "exists"},
		},
	}

	step := engine.Step{Target: "compose", Subject: "image", Service: "app"}
	findings := (&ComposeRunner{}).Eval(context.Background(), project, rule, step)
	if len(findings) != 0 {
		t.Fatalf("Eval() findings = %d, want 0", len(findings))
	}
}

func TestComposeServiceSummaryStepUsesShortRaw(t *testing.T) {
	project := &model.Project{
		Files:    []string{"compose.yaml"},
		Services: map[string]*model.Service{},
	}
	service := &model.Service{
		Name: "app",
		Config: types.ServiceConfig{
			Name: "app",
		},
		Present: map[string]bool{"service": true, "name": true},
		Project: project,
	}
	project.Services["app"] = service

	steps := composeSteps(project)
	for _, step := range steps {
		if step.Subject != "service" {
			continue
		}
		if step.Raw != "services.app" {
			t.Fatalf("service step raw = %q, want services.app", step.Raw)
		}
		return
	}
	t.Fatalf("service step not found")
}
