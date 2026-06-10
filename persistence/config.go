// Package persistence wires the database connector, migrator, and unit of
// work shared by all Tupic services.
package persistence

import "github.com/tupic/common-go/persistence/config"

// PostgresConfig holds PostgreSQL connection settings.
type PostgresConfig = config.PostgresConfig

// Config selects the persistence driver and carries its settings.
type Config = config.Config
