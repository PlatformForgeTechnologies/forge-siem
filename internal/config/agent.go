package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type AgentFileConfig struct {
	Server struct {
		Host       string `yaml:"host"`
		Port       int    `yaml:"port"`
		CACert     string `yaml:"ca_cert"`
		ClientCert string `yaml:"client_cert"`
		ClientKey  string `yaml:"client_key"`
	} `yaml:"server"`
	LogCollection struct {
		Paths   []string `yaml:"paths"`
		Outputs struct {
			SIEM struct {
				Enabled bool `yaml:"enabled"`
			} `yaml:"siem"`
			Loki struct {
				Enabled   bool              `yaml:"enabled"`
				PushURL   string            `yaml:"push_url"`
				TenantID  string            `yaml:"tenant_id"`
				Labels    map[string]string `yaml:"labels"`
				OnlyPaths []string          `yaml:"only_paths"`
			} `yaml:"loki"`
		} `yaml:"outputs"`
	} `yaml:"log_collection"`
	FIM struct {
		Paths    []string `yaml:"paths"`
		Schedule string   `yaml:"schedule"`
	} `yaml:"fim"`
	Response struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"response"`
}

func LoadAgentFile(path string) (AgentFileConfig, error) {
	var cfg AgentFileConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read agent config: %w", err)
	}
	expanded := os.ExpandEnv(string(data))
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return cfg, fmt.Errorf("parse agent config: %w", err)
	}
	return cfg, nil
}
