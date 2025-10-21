package config

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/util/homedir"
	"kf/internal/utils"
	"os"
	"path/filepath"
)

const DefaultEnv = "dev"

type Config struct {
	Profiles   []*Profile `yaml:"profiles" validate:"required"`
	Services   []*Service `yaml:"services" validate:"required"`
	ServiceMap map[string]*Service
}

type Profile struct {
	Name      string            `yaml:"name" validate:"required"`
	Namespace string            `yaml:"namespace"` //default: dev
	Services  []*ServiceOverlay `yaml:"services" validate:"required,min=1"`
}

type ServiceOverlay struct {
	Ref        string `yaml:"ref" validate:"required"` //alias reference in services
	LocalPort  int    `yaml:"lport"`                   //overrides the local port
	RemotePort int    `yaml:"rport"`                   //overrides the remote port
	Service    *Service
}

type Service struct {
	Name       string `yaml:"name" validate:"required"`
	Alias      string `yaml:"alias"` //alias reference in profiles; default = name
	LocalPort  int    `yaml:"lport" validate:"required"`
	RemotePort int    `yaml:"rport"` //default: local port
}

func (cfg *Config) PrintList() {
	fmt.Println("Profiles:")
	for _, profile := range cfg.Profiles {
		fmt.Printf("  - %s\n", profile.Name)
	}
	fmt.Println("Services:")
	for _, service := range cfg.Services {
		fmt.Printf("  - %s\n", service.Alias)
	}
}

func (cfg *Config) GetProfile(profile string) *Profile {
	for _, p := range cfg.Profiles {
		if p.Name == profile {
			return p
		}
	}
	return nil
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

	v := validator.New(validator.WithRequiredStructEnabled())
	if err := v.Struct(c); err != nil {
		return err
	}

	for _, service := range c.Services {
		if service.Alias == "" {
			service.Alias = service.Name
		}
		if service.RemotePort == 0 {
			service.RemotePort = service.LocalPort
		}
	}

	//filling service map
	c.ServiceMap = utils.MapFromSlice(c.Services, func(s *Service) string { return s.Alias })

	//filling service in profiles
	for _, profile := range c.Profiles {
		if profile.Namespace == "" {
			profile.Namespace = DefaultEnv
		}
		for _, overlay := range profile.Services {
			if service, ok := c.ServiceMap[overlay.Ref]; ok {
				overlay.Service = service
			} else {
				return fmt.Errorf("config: service %s not found", overlay.Ref)
			}
		}
	}

	return nil
}
