package config

import env "github.com/caarlos0/env/v11"

// ApplicationSettings defines the configuration options for the Contain Sentry.
// Configuration can be loaded from a YAML file or environment variables.
type ApplicationSettings struct {
	// Logger configuration
	Logger Config `yaml:"logger" env:"-"`

	DockerfilePath string `yaml:"dockerfile" env:"DOCKERFILE_PATH" envDefault:"Dockerfile"`
	RulesPath      string `yaml:"rules" env:"RULES_PATH" envDefault:"dockerfile-rules.json"`
}

func LoadApplicationSettings() (*ApplicationSettings, error) {
	cfg := &ApplicationSettings{
		Logger: NewDefaultConfig(),
	}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
