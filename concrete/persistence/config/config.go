// Package config holds the persistence configuration types in a leaf package so the connector and migrator can import
// them without cycles.
package config

// PostgresConfig holds PostgreSQL connection settings. Services embed it in their config structs; the
// mapstructure/validate tags live here so every service shares the same config schema.
type PostgresConfig struct {
	Host          string `mapstructure:"host"           validate:"required"`
	Port          int    `mapstructure:"port"           validate:"required"`
	Name          string `mapstructure:"name"           validate:"required"`
	Username      string `mapstructure:"username"       validate:"required"`
	Password      string `mapstructure:"password"       validate:"required"`
	Timeout       int    `mapstructure:"timeout"        validate:"required"`
	SSLMode       string `mapstructure:"ssl_mode"       validate:"required,oneof=disable require verify-ca verify-full"`
	MigrationPath string `mapstructure:"migration_path"`
}

// Config selects the persistence driver and nests each driver's own settings. The mapstructure/validate tags let a
// service load it directly (alias it). Test is injected by the service from its environment, not loaded; it enables the
// smaller test connection pool and skips re-pinging a pool already in use (suite-level transactions hold a connection).
type Config struct {
	Driver   string         `mapstructure:"driver"   validate:"required,oneof=postgres"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	Test     bool           `mapstructure:"-"`
}
