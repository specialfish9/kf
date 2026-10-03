package configv3

func (c *Config) GetProfile(name string) *Profile {
	for _, profile := range c.Profiles {
		if profile.Name == name {
			return &profile
		}
	}
	return nil
}

func (c *Config) GetService(nameOrAlias string) *Service {
	for _, service := range c.Services {
		if service.Name == nameOrAlias || service.Alias.Or("") == nameOrAlias {
			return &service
		}
	}

	return nil
}
