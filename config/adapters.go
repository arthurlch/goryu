package config

// I thought that a frameworkAdapter to provides methods to convert the new config structure would be nice
type FrameworkAdapter struct {
	config *Config
}

func NewFrameworkAdapter(config *Config) *FrameworkAdapter {
	return &FrameworkAdapter{
		config: config,
	}
}

func (a *FrameworkAdapter) ToGoryuConfig() interface{} {
	// Gotta return an interface because I don't want to import goryu package here
	// to avoid circular dependencies. The caller should type assert this. Had some issues here due to circular imports,
	// wish earth was flat like my imports
	
	return struct {
		AppName               string
		ServerHeader          string
		StrictRouting         bool
		CaseSensitive         bool
		DisableStartupMessage bool
		RedirectTrailingSlash *bool
		EnableHEADFallback    *bool
	}{
		AppName:               a.config.App.Name,
		ServerHeader:          a.config.Framework.ServerHeader,
		StrictRouting:         a.config.Framework.StrictRouting,
		CaseSensitive:         a.config.Framework.CaseSensitive,
		DisableStartupMessage: a.config.Framework.DisableStartupMessage,
		RedirectTrailingSlash: &a.config.Framework.RedirectTrailingSlash,
		EnableHEADFallback:    &a.config.Framework.EnableHEADFallback,
	}
}

func (a *FrameworkAdapter) GetServerConfig() ServerConfig {
	return a.config.Server
}

func (a *FrameworkAdapter) GetDatabaseConfig() DatabaseConfig {
	return a.config.Database
}

func (a *FrameworkAdapter) GetAppConfig() AppConfig {
	return a.config.App
}

func (a *FrameworkAdapter) GetFrameworkConfig() FrameworkConfig {
	return a.config.Framework
}

// Legacy c**** that could be useful for some tests

func ConvertLegacyToNew(legacy *LegacyConfig) *Config {
	return legacy.ToNewConfig()
}

func ConvertNewToLegacy(config *Config) *LegacyConfig {
	legacy := &LegacyConfig{
		App: LegacyAppConfig{
			Name:              config.App.Name,
			Version:           config.App.Version,
			ServerHeader:      config.Framework.ServerHeader,
			StrictRouting:     config.Framework.StrictRouting,
			CaseSensitive:     config.Framework.CaseSensitive,
			DisableStartupMsg: config.Framework.DisableStartupMessage,
		},
		Server: LegacyServerConfig{
			Host:            config.Server.Host,
			Port:            config.Server.Port,
			ReadTimeout:     config.Server.ReadTimeout,
			WriteTimeout:    config.Server.WriteTimeout,
			ShutdownTimeout: config.Server.ShutdownTimeout,
		},
		Custom: config.App.Custom,
	}
	
	if legacy.Custom == nil {
		legacy.Custom = make(map[string]interface{})
	}
	
	legacy.Custom["database"] = map[string]interface{}{
		"driver":   config.Database.Driver,
		"host":     config.Database.Host,
		"port":     config.Database.Port,
		"database": config.Database.Database,
		"username": config.Database.Username,
		"password": config.Database.Password,
		"path":     config.Database.Path,
		"sslmode":  config.Database.SSLMode,
	}
	
	return legacy
}