package config

import (
	"flag"
	"fmt"
	"io"
	"strings"

	env "github.com/caarlos0/env/v11"
)

// ApplicationSettings defines the configuration options for the Contain Sentry.
// Configuration can be loaded from a YAML file or environment variables.
type ApplicationSettings struct {
	// Logger configuration
	Logger Config `yaml:"logger" env:"-"`

	DockerfilePath string   `yaml:"dockerfile" env:"DOCKERFILE_PATH" envDefault:"Dockerfile"`
	ComposeFiles   []string `yaml:"compose_files" env:"COMPOSE_FILES" envSeparator:"," envDefault:"compose.yaml"`
	ReportJSONPath string   `yaml:"report_json" env:"REPORT_JSON"`
	Target         string   `yaml:"target" env:"TARGET" envDefault:"dockerfile"`
	RulesPath      string   `yaml:"rules" env:"RULES_PATH" envDefault:"dockerfile-rules.json"`
}

func LoadApplicationSettings(args []string, stdout io.Writer, stderr io.Writer) (*ApplicationSettings, bool, error) {
	cfg := &ApplicationSettings{
		Logger: NewDefaultConfig(),
	}

	if err := env.Parse(cfg); err != nil {
		return nil, false, err
	}

	helpRequested, err := applyCLIFlags(cfg, args, stdout, stderr)
	if err != nil {
		return nil, false, err
	}

	return cfg, helpRequested, nil
}

func applyCLIFlags(cfg *ApplicationSettings, args []string, stdout io.Writer, stderr io.Writer) (bool, error) {
	if cfg == nil {
		return false, fmt.Errorf("application settings are nil")
	}

	fs := flag.NewFlagSet("containsentry", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		writeHelp(stdout, fs)
	}

	target := fs.String("target", cfg.Target, "analysis target: dockerfile or compose")
	dockerfilePath := fs.String("dockerfile", cfg.DockerfilePath, "path to Dockerfile")
	composeFiles := fs.String("compose-files", strings.Join(cfg.ComposeFiles, ","), "comma-separated compose files")
	rulesPath := fs.String("rules", cfg.RulesPath, "path to rules JSON file")
	reportJSONPath := fs.String("report-json", cfg.ReportJSONPath, "write findings report to JSON file")
	help := fs.Bool("help", false, "show help")
	fs.BoolVar(help, "h", false, "show help")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			writeHelp(stdout, fs)
			return true, nil
		}
		return false, err
	}

	if *help {
		writeHelp(stdout, fs)
		return true, nil
	}

	cfg.Target = strings.TrimSpace(*target)
	cfg.DockerfilePath = strings.TrimSpace(*dockerfilePath)
	cfg.RulesPath = strings.TrimSpace(*rulesPath)
	cfg.ReportJSONPath = strings.TrimSpace(*reportJSONPath)
	cfg.ComposeFiles = splitCommaSeparated(*composeFiles)

	return false, nil
}

func splitCommaSeparated(input string) []string {
	parts := strings.Split(input, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		items = append(items, part)
	}
	return items
}

func writeHelp(output io.Writer, fs *flag.FlagSet) {
	if output == nil {
		return
	}
	_, _ = fmt.Fprintln(output, "ContainSentry")
	_, _ = fmt.Fprintln(output, "")
	_, _ = fmt.Fprintln(output, "Usage:")
	_, _ = fmt.Fprintln(output, "  containsentry [flags]")
	_, _ = fmt.Fprintln(output, "")
	_, _ = fmt.Fprintln(output, "Flags:")
	fs.SetOutput(output)
	fs.PrintDefaults()
	_, _ = fmt.Fprintln(output, "")
	_, _ = fmt.Fprintln(output, "Environment variables:")
	_, _ = fmt.Fprintln(output, "  TARGET")
	_, _ = fmt.Fprintln(output, "  DOCKERFILE_PATH")
	_, _ = fmt.Fprintln(output, "  COMPOSE_FILES")
	_, _ = fmt.Fprintln(output, "  RULES_PATH")
	_, _ = fmt.Fprintln(output, "  REPORT_JSON")
}
