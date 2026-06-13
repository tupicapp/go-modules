// Package nats provides the NATS JetStream connection, subject routing, and event/queue subscribers shared by all Tupic
// services.
package nats

// Config carries NATS connection settings plus the service identity used for durable consumer naming and connection
// naming.
type Config struct {
	URL           string
	Token         string
	SubjectPrefix string
	AppSlug       string
}
