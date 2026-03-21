package model

import composetypes "github.com/compose-spec/compose-go/v2/types"

type Project struct {
	Name     string
	Files    []string
	Services map[string]*Service
	Secrets  composetypes.Secrets
	Networks composetypes.Networks
	Volumes  composetypes.Volumes
	Profiles []string
	Raw      map[string]any
	TopLevel map[string]bool
}

type Service struct {
	Name      string
	Config    composetypes.ServiceConfig
	Raw       map[string]any
	Present   map[string]bool
	Disabled  bool
	Project   *Project
	SourceRaw any
}

type Location struct {
	Files       []string `json:"files,omitempty"`
	ServiceName string   `json:"service_name,omitempty"`
	Path        string   `json:"path,omitempty"`
}

type Command struct {
	Service *Service
	Path    string
	Value   any
}

func (s *Service) HasField(name string) bool {
	if s == nil {
		return false
	}
	return s.Present[name]
}

func (s *Service) Snapshot() map[string]any {
	if s == nil {
		return nil
	}
	return map[string]any{
		"name":              s.Name,
		"profiles":          append([]string{}, s.Config.Profiles...),
		"user":              s.Config.User,
		"userns_mode":       s.Config.UserNSMode,
		"build":             s.Config.Build,
		"image":             s.Config.Image,
		"cap_add":           append([]string{}, s.Config.CapAdd...),
		"read_only":         s.Config.ReadOnly,
		"privileged":        s.Config.Privileged,
		"cap_drop":          append([]string{}, s.Config.CapDrop...),
		"security_opt":      append([]string{}, s.Config.SecurityOpt...),
		"network_mode":      s.Config.NetworkMode,
		"networks":          s.Config.Networks,
		"pid":               s.Config.Pid,
		"ipc":               s.Config.Ipc,
		"devices":           s.Config.Devices,
		"ports":             s.Config.Ports,
		"volumes":           s.Config.Volumes,
		"environment":       s.Config.Environment,
		"secrets":           s.Config.Secrets,
		"healthcheck":       s.Config.HealthCheck,
		"depends_on":        s.Config.DependsOn,
		"restart":           s.Config.Restart,
		"logging":           s.Config.Logging,
		"init":              s.Config.Init,
		"stop_grace_period": s.Config.StopGracePeriod,
		"stop_signal":       s.Config.StopSignal,
		"deploy":            s.Config.Deploy,
		"resource_limits":   resourceLimitsSnapshot(s.Config.Deploy),
		"disabled":          s.Disabled,
		"present":           s.Present,
		"raw":               s.Raw,
		"top_secrets":       s.Project.Secrets,
		"top_networks":      s.Project.Networks,
		"top_volumes":       s.Project.Volumes,
		"project_name":      s.Project.Name,
		"project_files":     append([]string{}, s.Project.Files...),
	}
}

func resourceLimitsSnapshot(deploy *composetypes.DeployConfig) any {
	if deploy == nil {
		return nil
	}
	return deploy.Resources
}
