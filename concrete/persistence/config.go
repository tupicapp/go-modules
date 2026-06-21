// Package persistence wires the database connector, migrator, and unit of work shared by all platform services.
package persistence

import "github.com/tupicapp/go-modules/concrete/persistence/config"

// PostgresConfig holds PostgreSQL connection settings.
type PostgresConfig = config.PostgresConfig

// Config selects the persistence driver and carries its settings.
type Config = config.Config
