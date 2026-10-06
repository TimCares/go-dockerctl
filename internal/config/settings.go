package config

// Settings are values passed when invoking a command.
//
// ProjectPath, ConfigPath, and Env are required to load the file.
// The log fields override [LoggingConfig] only when non-nil, which is how an
// explicit flag or environment variable is separated from a flag default.
type Settings struct {
	ProjectPath string  `yaml:"project_path"`
	ConfigPath  string  `yaml:"config_path"`
	Env         string  `yaml:"env"`
	LogLevel    *string `yaml:"log_level,omitempty"`
	LogFormat   *string `yaml:"log_format,omitempty"`
	LogFile     *string `yaml:"log_file,omitempty"`
}
