// Package config loads layered JSON configuration files with environment variable overrides into a service-defined
// config struct.
package config

import (
	"log"
	"os"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/spf13/viper"
)

// Load reads the given JSON config files in order (the first is required, the rest are optional overrides), applies
// automatic environment variable overrides (dots become underscores), and unmarshals into target. The service performs
// its own defaulting and validation afterward.
func Load(target any, paths ...string) error {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.SetConfigType("json")

	for i, p := range paths {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			if i == 0 {
				return errors.Wrapf(err, "config file not found: %s", p)
			}
			log.Printf("config: skipping missing optional file %s\n", p)
			continue
		}

		v.SetConfigFile(p)

		if i == 0 {
			if err := v.ReadInConfig(); err != nil {
				return errors.Wrapf(err, "cannot read config: %s", p)
			}
		} else {
			if err := v.MergeInConfig(); err != nil {
				return errors.Wrapf(err, "cannot merge config: %s", p)
			}
		}

		log.Printf("config: loaded %s\n", p)
	}

	if err := v.Unmarshal(target); err != nil {
		return errors.Wrapf(err, "cannot unmarshal config")
	}

	return nil
}
