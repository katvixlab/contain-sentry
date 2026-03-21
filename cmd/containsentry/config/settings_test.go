package config

import (
	"bytes"
	"testing"
)

func TestLoadApplicationSettingsCLIOverrides(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cfg, help, err := LoadApplicationSettings([]string{
		"--target", "compose",
		"--compose-files", "compose.yaml,compose.prod.yaml",
		"--rules", "compose-rules.json",
		"--report-json", "out.json",
	}, stdout, stderr)
	if err != nil {
		t.Fatalf("LoadApplicationSettings() error = %v", err)
	}
	if help {
		t.Fatalf("help = true, want false")
	}
	if cfg.Target != "compose" {
		t.Fatalf("Target = %q, want compose", cfg.Target)
	}
	if len(cfg.ComposeFiles) != 2 {
		t.Fatalf("ComposeFiles len = %d, want 2", len(cfg.ComposeFiles))
	}
	if cfg.ComposeFiles[0] != "compose.yaml" || cfg.ComposeFiles[1] != "compose.prod.yaml" {
		t.Fatalf("ComposeFiles = %v", cfg.ComposeFiles)
	}
	if cfg.RulesPath != "compose-rules.json" {
		t.Fatalf("RulesPath = %q, want compose-rules.json", cfg.RulesPath)
	}
	if cfg.ReportJSONPath != "out.json" {
		t.Fatalf("ReportJSONPath = %q, want out.json", cfg.ReportJSONPath)
	}
}

func TestLoadApplicationSettingsHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	_, help, err := LoadApplicationSettings([]string{"--help"}, stdout, stderr)
	if err != nil {
		t.Fatalf("LoadApplicationSettings() error = %v", err)
	}
	if !help {
		t.Fatalf("help = false, want true")
	}
	if stdout.Len() == 0 {
		t.Fatalf("stdout is empty, want help output")
	}
}
