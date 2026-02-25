package config

import "spark/internal/conf"

// ApplyEnvOverrides is a no-op. All configuration is read from config.yaml
// and hot-reloaded via ConfigWatcher. Environment variable overrides are
// intentionally not supported to keep configuration in a single source.
func ApplyEnvOverrides(bc *conf.Bootstrap) {}
