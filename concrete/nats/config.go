// Package nats provides the NATS JetStream connection, subject routing, and event/queue subscribers shared by all Tupic
// services.
package nats

// Config carries NATS connection settings plus the service identity used for durable consumer naming and connection
// naming. The mapstructure tags let a service load it directly from JSON (alias it); AppSlug is injected by the service
// from its identity constant, not loaded, so it is skipped.
type Config struct {
	URL           string `mapstructure:"url"`
	Token         string `mapstructure:"token"`
	SubjectPrefix string `mapstructure:"subject_prefix"`
	AppSlug       string `mapstructure:"-"`
}
