package configv3

import (
	"fmt"

	"github.com/specialfish9/confuso/v2"
)

type Config struct {
	Services []Service `confuso:"services"`
	Profiles []Profile `confuso:"profiles"`
}

type Profile struct {
	Name      string                   `confuso:"name"`
	Namespace confuso.Optional[string] `confuso:"namespace"`
	Services  []*ServiceOverlay        `confuso:"services"`
}

type ServiceOverlay struct {
	Ref       string                   `confuso:"ref"`
	Namespace confuso.Optional[string] `confuso:"namespace"`
	LPort     confuso.Optional[int]    `confuso:"lport"`
}

type Service struct {
	Name  string                   `confuso:"name"`
	Alias confuso.Optional[string] `confuso:"alias"`
	RPort int                      `confuso:"rport"`
	LPort confuso.Optional[int]    `confuso:"lport"`
}

func Load(path string) (*Config, error) {
	var config = Config{}

	if err := confuso.Do(path, &config); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	return &config, nil
}
