package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/katvixlab/contain-sentry/internal/compose/model"
	"github.com/katvixlab/contain-sentry/internal/engine"
	"github.com/katvixlab/contain-sentry/internal/entities"
)

var composeSubjects = []string{
	"service",
	"build",
	"image",
	"user",
	"userns_mode",
	"cap_add",
	"read_only",
	"privileged",
	"cap_drop",
	"security_opt",
	"network_mode",
	"networks",
	"pid",
	"ipc",
	"devices",
	"ports",
	"volumes",
	"environment",
	"secrets",
	"healthcheck",
	"depends_on",
	"restart",
	"profiles",
	"logging",
	"init",
	"stop_grace_period",
	"stop_signal",
	"resource_limits",
}

type ComposeDriver struct {
	project *model.Project
	steps   []engine.Step
	index   int
}

func NewComposeDriver(project *model.Project) *ComposeDriver {
	return &ComposeDriver{
		project: project,
		steps:   composeSteps(project),
	}
}

func (d *ComposeDriver) Target() string {
	return targetCompose
}

func (d *ComposeDriver) DomainContext() any {
	return d.project
}

func (d *ComposeDriver) Next(ctx context.Context) (engine.Step, bool, error) {
	_ = ctx
	if d.index >= len(d.steps) {
		return engine.Step{}, false, nil
	}
	step := d.steps[d.index]
	d.index++
	return step, true, nil
}

func (d *ComposeDriver) Transfer(ctx context.Context, step engine.Step) error {
	_ = ctx
	_ = step
	return nil
}

type ComposeRunner struct{}

func (r *ComposeRunner) Target() string {
	return targetCompose
}

func (r *ComposeRunner) Eval(ctx context.Context, dom any, rule entities.BaseRule, step engine.Step) []entities.Finding {
	_ = ctx

	if !strings.EqualFold(strings.TrimSpace(rule.Subject), strings.TrimSpace(step.Subject)) {
		return nil
	}

	fieldExpr, ok := rule.Expression.(*entities.ExpressionField)
	if !ok || fieldExpr == nil {
		return nil
	}

	project, ok := dom.(*model.Project)
	if !ok || project == nil {
		return nil
	}

	service := project.Services[step.Service]
	value, present := composeSelect(project, service, fieldExpr.Select)
	if !fieldExpr.Evaluate(value, present, func(path string) (any, bool) {
		return composeSelect(project, service, path)
	}) {
		return nil
	}

	return []entities.Finding{engine.BuildFinding(rule, step)}
}

func composeSteps(project *model.Project) []engine.Step {
	if project == nil {
		return nil
	}

	serviceNames := make([]string, 0, len(project.Services))
	for name := range project.Services {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	steps := make([]engine.Step, 0, len(serviceNames)*(len(composeSubjects)+1))
	for _, name := range serviceNames {
		service := project.Services[name]
		for _, subject := range composeSubjects {
			value, raw, present, path := subjectValue(service, subject)
			steps = append(steps, engine.Step{
				Target:   targetCompose,
				Subject:  subject,
				Path:     path,
				Service:  name,
				Raw:      raw,
				Value:    value,
				Present:  present,
				Location: model.Location{Files: append([]string{}, project.Files...), ServiceName: name, Path: path},
				Command:  &model.Command{Service: service, Path: path, Value: value},
			})
		}
		steps = append(steps, engine.Step{
			Target:   targetCompose,
			Subject:  "eof",
			Path:     "services." + name,
			Service:  name,
			Raw:      "services." + name,
			Value:    service.Snapshot(),
			Present:  true,
			Location: model.Location{Files: append([]string{}, project.Files...), ServiceName: name, Path: "services." + name},
			Command:  &model.Command{Service: service, Path: "services." + name, Value: service.Snapshot()},
		})
	}

	return steps
}

func subjectValue(service *model.Service, subject string) (any, string, bool, string) {
	base := "services." + service.Name
	switch subject {
	case "service":
		value := service.Snapshot()
		return value, base, true, base
	case "build":
		return service.Config.Build, stringifyComposeValue(service.Config.Build), service.HasField("build"), base + ".build"
	case "image":
		return service.Config.Image, stringifyComposeValue(service.Config.Image), service.HasField("image"), base + ".image"
	case "user":
		return service.Config.User, stringifyComposeValue(service.Config.User), service.HasField("user"), base + ".user"
	case "userns_mode":
		return service.Config.UserNSMode, stringifyComposeValue(service.Config.UserNSMode), service.HasField("userns_mode"), base + ".userns_mode"
	case "cap_add":
		return service.Config.CapAdd, stringifyComposeValue(service.Config.CapAdd), service.HasField("cap_add"), base + ".cap_add"
	case "read_only":
		return service.Config.ReadOnly, stringifyComposeValue(service.Config.ReadOnly), service.HasField("read_only"), base + ".read_only"
	case "privileged":
		return service.Config.Privileged, stringifyComposeValue(service.Config.Privileged), service.HasField("privileged"), base + ".privileged"
	case "cap_drop":
		return service.Config.CapDrop, stringifyComposeValue(service.Config.CapDrop), service.HasField("cap_drop"), base + ".cap_drop"
	case "security_opt":
		return service.Config.SecurityOpt, stringifyComposeValue(service.Config.SecurityOpt), service.HasField("security_opt"), base + ".security_opt"
	case "network_mode":
		return service.Config.NetworkMode, stringifyComposeValue(service.Config.NetworkMode), service.HasField("network_mode"), base + ".network_mode"
	case "networks":
		return service.Config.Networks, stringifyComposeValue(service.Config.Networks), service.HasField("networks"), base + ".networks"
	case "pid":
		return service.Config.Pid, stringifyComposeValue(service.Config.Pid), service.HasField("pid"), base + ".pid"
	case "ipc":
		return service.Config.Ipc, stringifyComposeValue(service.Config.Ipc), service.HasField("ipc"), base + ".ipc"
	case "devices":
		return service.Config.Devices, stringifyComposeValue(service.Config.Devices), service.HasField("devices"), base + ".devices"
	case "ports":
		return service.Config.Ports, stringifyComposeValue(service.Config.Ports), service.HasField("ports"), base + ".ports"
	case "volumes":
		return service.Config.Volumes, stringifyComposeValue(service.Config.Volumes), service.HasField("volumes"), base + ".volumes"
	case "environment":
		return service.Config.Environment, stringifyComposeValue(service.Config.Environment), service.HasField("environment"), base + ".environment"
	case "secrets":
		return service.Config.Secrets, stringifyComposeValue(service.Config.Secrets), service.HasField("secrets"), base + ".secrets"
	case "healthcheck":
		return service.Config.HealthCheck, stringifyComposeValue(service.Config.HealthCheck), service.HasField("healthcheck"), base + ".healthcheck"
	case "depends_on":
		return service.Config.DependsOn, stringifyComposeValue(service.Config.DependsOn), service.HasField("depends_on"), base + ".depends_on"
	case "restart":
		return service.Config.Restart, stringifyComposeValue(service.Config.Restart), service.HasField("restart"), base + ".restart"
	case "profiles":
		return service.Config.Profiles, stringifyComposeValue(service.Config.Profiles), service.HasField("profiles"), base + ".profiles"
	case "logging":
		return service.Config.Logging, stringifyComposeValue(service.Config.Logging), service.HasField("logging"), base + ".logging"
	case "init":
		return service.Config.Init, stringifyComposeValue(service.Config.Init), service.HasField("init"), base + ".init"
	case "stop_grace_period":
		return service.Config.StopGracePeriod, stringifyComposeValue(service.Config.StopGracePeriod), service.HasField("stop_grace_period"), base + ".stop_grace_period"
	case "stop_signal":
		return service.Config.StopSignal, stringifyComposeValue(service.Config.StopSignal), service.HasField("stop_signal"), base + ".stop_signal"
	case "resource_limits":
		if service.Config.Deploy == nil {
			return nil, "", service.HasField("resource_limits"), base + ".deploy.resources"
		}
		return service.Config.Deploy.Resources, stringifyComposeValue(service.Config.Deploy.Resources), service.HasField("resource_limits"), base + ".deploy.resources"
	default:
		return nil, "", false, base
	}
}

func composeSelect(project *model.Project, service *model.Service, selectPath string) (any, bool) {
	normalized := strings.ToLower(strings.TrimSpace(selectPath))
	if strings.HasPrefix(normalized, "service.") || normalized == "service" {
		if service == nil {
			return nil, false
		}
	}

	switch normalized {
	case "service":
		return service.Snapshot(), true
	case "service.name":
		return service.Name, true
	case "service.user":
		return service.Config.User, service.HasField("user")
	case "service.userns_mode":
		return service.Config.UserNSMode, service.HasField("userns_mode")
	case "service.read_only":
		return service.Config.ReadOnly, service.HasField("read_only")
	case "service.privileged":
		return service.Config.Privileged, service.HasField("privileged")
	case "service.cap_add":
		return service.Config.CapAdd, service.HasField("cap_add")
	case "service.cap_drop":
		return service.Config.CapDrop, service.HasField("cap_drop")
	case "service.security_opt":
		return service.Config.SecurityOpt, service.HasField("security_opt")
	case "service.network_mode":
		return service.Config.NetworkMode, service.HasField("network_mode")
	case "service.networks":
		return service.Config.Networks, service.HasField("networks")
	case "service.pid":
		return service.Config.Pid, service.HasField("pid")
	case "service.ipc":
		return service.Config.Ipc, service.HasField("ipc")
	case "service.devices":
		return service.Config.Devices, service.HasField("devices")
	case "service.environment":
		return service.Config.Environment, service.HasField("environment")
	case "service.secrets":
		return service.Config.Secrets, service.HasField("secrets")
	case "service.healthcheck":
		return service.Config.HealthCheck, service.HasField("healthcheck")
	case "service.depends_on":
		return service.Config.DependsOn, service.HasField("depends_on")
	case "service.restart":
		return service.Config.Restart, service.HasField("restart")
	case "service.profiles":
		return service.Config.Profiles, service.HasField("profiles")
	case "service.ports":
		return service.Config.Ports, service.HasField("ports")
	case "service.volumes":
		return service.Config.Volumes, service.HasField("volumes")
	case "service.image":
		return service.Config.Image, service.HasField("image")
	case "service.build":
		return service.Config.Build, service.HasField("build")
	case "service.logging":
		return service.Config.Logging, service.HasField("logging")
	case "service.init":
		return service.Config.Init, service.HasField("init")
	case "service.stop_grace_period":
		return service.Config.StopGracePeriod, service.HasField("stop_grace_period")
	case "service.stop_signal":
		return service.Config.StopSignal, service.HasField("stop_signal")
	case "service.resource_limits":
		if service.Config.Deploy == nil {
			return nil, service.HasField("resource_limits")
		}
		return service.Config.Deploy.Resources, service.HasField("resource_limits")
	case "compose.project_name":
		if project == nil {
			return "", false
		}
		return project.Name, project.TopLevel["name"]
	case "compose.secrets":
		if project == nil {
			return nil, false
		}
		return project.Secrets, project.TopLevel["secrets"]
	case "compose.networks":
		if project == nil {
			return nil, false
		}
		return project.Networks, project.TopLevel["networks"]
	case "compose.volumes":
		if project == nil {
			return nil, false
		}
		return project.Volumes, project.TopLevel["volumes"]
	case "compose.profiles":
		if project == nil {
			return nil, false
		}
		return project.Profiles, project.TopLevel["profiles"] || len(project.Profiles) > 0
	default:
		return nil, false
	}
}

func stringifyComposeValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(payload)
}
