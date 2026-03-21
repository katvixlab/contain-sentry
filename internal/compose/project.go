package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/compose-spec/compose-go/v2/loader"
	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/katvixlab/contain-sentry/internal/compose/model"
	"github.com/katvixlab/contain-sentry/internal/engine"
	"github.com/katvixlab/contain-sentry/internal/entities"
)

const targetCompose = "compose"

type Project struct {
	Model *model.Project
}

func NewProject(ctx context.Context, files []string) (*Project, error) {
	details, normalizedFiles, err := composeConfigDetails(files)
	if err != nil {
		return nil, err
	}

	projectName := loader.NormalizeProjectName(filepath.Base(details.WorkingDir))
	project, err := loader.LoadWithContext(ctx, details, func(options *loader.Options) {
		options.SetProjectName(projectName, true)
	})
	if err != nil {
		return nil, fmt.Errorf("load compose project: %w", err)
	}

	rawModel, err := loader.LoadModelWithContext(ctx, details, func(options *loader.Options) {
		options.SetProjectName(projectName, true)
	})
	if err != nil {
		return nil, fmt.Errorf("load compose model: %w", err)
	}

	return &Project{
		Model: buildProjectModel(project, rawModel, normalizedFiles),
	}, nil
}

func (p *Project) Validate(ctx context.Context, rules []entities.BaseRule) ([]entities.Finding, error) {
	driver := NewComposeDriver(p.Model)
	eng := engine.New(rules, &ComposeRunner{})
	return eng.Run(ctx, driver)
}

func composeConfigDetails(files []string) (composetypes.ConfigDetails, []string, error) {
	if len(files) == 0 {
		return composetypes.ConfigDetails{}, nil, fmt.Errorf("no compose files provided")
	}

	normalized := make([]string, 0, len(files))
	configFiles := make([]composetypes.ConfigFile, 0, len(files))
	for _, file := range files {
		if file == "" {
			continue
		}
		abs, err := filepath.Abs(file)
		if err != nil {
			return composetypes.ConfigDetails{}, nil, fmt.Errorf("resolve compose path %q: %w", file, err)
		}
		normalized = append(normalized, abs)
		configFiles = append(configFiles, composetypes.ConfigFile{Filename: abs})
	}
	if len(configFiles) == 0 {
		return composetypes.ConfigDetails{}, nil, fmt.Errorf("no compose files provided")
	}

	workingDir := filepath.Dir(normalized[0])
	return composetypes.ConfigDetails{
		WorkingDir:  workingDir,
		ConfigFiles: configFiles,
		Environment: composeEnvironment(),
	}, normalized, nil
}

func composeEnvironment() composetypes.Mapping {
	env := composetypes.Mapping{}
	for _, pair := range os.Environ() {
		key, value, ok := splitEnv(pair)
		if !ok {
			continue
		}
		env[key] = value
	}
	return env
}

func splitEnv(pair string) (string, string, bool) {
	for i := 0; i < len(pair); i++ {
		if pair[i] == '=' {
			return pair[:i], pair[i+1:], true
		}
	}
	return "", "", false
}

func buildProjectModel(project *composetypes.Project, raw map[string]any, files []string) *model.Project {
	pm := &model.Project{
		Name:     project.Name,
		Files:    append([]string{}, files...),
		Services: map[string]*model.Service{},
		Secrets:  project.Secrets,
		Networks: project.Networks,
		Volumes:  project.Volumes,
		Raw:      raw,
		TopLevel: map[string]bool{},
	}

	for _, key := range []string{"services", "secrets", "networks", "volumes", "profiles", "name"} {
		_, ok := raw[key]
		pm.TopLevel[key] = ok
	}

	rawServices := nestedMap(raw, "services")
	serviceNames := map[string]struct{}{}
	for name := range project.Services {
		serviceNames[name] = struct{}{}
	}
	for name := range project.DisabledServices {
		serviceNames[name] = struct{}{}
	}
	for name := range rawServices {
		serviceNames[name] = struct{}{}
	}

	profiles := map[string]struct{}{}
	for name := range serviceNames {
		serviceCfg, disabled := lookupService(project, name)
		serviceRaw := nestedMap(rawServices, name)
		svc := &model.Service{
			Name:     name,
			Config:   serviceCfg,
			Raw:      serviceRaw,
			Present:  servicePresence(serviceRaw),
			Disabled: disabled,
			Project:  pm,
		}
		pm.Services[name] = svc
		for _, profile := range serviceCfg.Profiles {
			if profile != "" {
				profiles[profile] = struct{}{}
			}
		}
	}

	for _, profile := range project.Profiles {
		if profile != "" {
			profiles[profile] = struct{}{}
		}
	}
	pm.Profiles = sortedKeys(profiles)
	return pm
}

func lookupService(project *composetypes.Project, name string) (composetypes.ServiceConfig, bool) {
	if service, ok := project.Services[name]; ok {
		return service, false
	}
	if service, ok := project.DisabledServices[name]; ok {
		return service, true
	}
	return composetypes.ServiceConfig{Name: name}, false
}

func servicePresence(raw map[string]any) map[string]bool {
	present := map[string]bool{}
	for _, field := range []string{
		"profiles", "build", "image", "user", "read_only", "privileged",
		"userns_mode", "cap_add", "cap_drop", "security_opt", "network_mode",
		"networks", "pid", "ipc", "devices", "ports", "volumes", "environment",
		"secrets", "healthcheck", "depends_on", "restart", "logging", "init",
		"stop_grace_period", "stop_signal", "deploy",
	} {
		_, ok := raw[field]
		present[field] = ok
	}
	if deployRaw := nestedMap(raw, "deploy"); deployRaw != nil {
		_, limitsOk := nestedMap(deployRaw, "resources")["limits"]
		_, reservationsOk := nestedMap(deployRaw, "resources")["reservations"]
		present["resource_limits"] = limitsOk || reservationsOk
	} else {
		present["resource_limits"] = false
	}
	present["service"] = true
	present["name"] = true
	return present
}

func nestedMap(input map[string]any, key string) map[string]any {
	if input == nil {
		return nil
	}
	value, ok := input[key]
	if !ok {
		return nil
	}
	mapping, ok := value.(map[string]any)
	if ok {
		return mapping
	}
	generic, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}
	result := make(map[string]any, len(generic))
	for k, v := range generic {
		result[k] = v
	}
	return result
}

func sortedKeys(items map[string]struct{}) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
