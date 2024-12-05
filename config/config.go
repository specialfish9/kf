package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
)

type Config struct {
	Profiles []*Profile `yaml:"profiles"`
}

type Service struct {
	Name       string `yaml:"name"`
	LocalPort  int    `yaml:"lport"`
	RemotePort int    `yaml:"rport"`
	Namespace  string `yaml:"namespace"`
}

type Profile struct {
	Name     string     `yaml:"name"`
	Services []*Service `yaml:"services"`
}

func DefaultPath() string {
	return filepath.Join(homedir.HomeDir(), ".config/kf.yaml")
}

func Read(path string) (*Config, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("config: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error: %v", err)
	}

	if err = validate(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func validate(c *Config) error {
	for _, profile := range c.Profiles {
		if profile.Name == "" {
			return fmt.Errorf("config: missing name on a profile")
		}
		for _, service := range profile.Services {
			if service.Name == "" {
				return fmt.Errorf("config: missing name on a service for profile %s", profile.Name)
			}
			if service.Namespace == "" {
				service.Namespace = "dev"
			}
			if service.LocalPort == 0 {
				return fmt.Errorf("config: missing local on service %s for profile %s", service.Name, profile.Name)
			}
			if service.RemotePort == 0 {
				service.RemotePort = 80
			}
		}
	}

	return nil
}
