package nats

import "strings"

// durableNameFor derives a stable JetStream durable consumer name from a logical subject, scoped to the app slug so
// multiple services can each hold their own independent consumer on the same stream without name conflicts.
//
// The stream itself (e.g. "insights_events") is owned by the publishing service. This function names the consumer the
// subscribing service creates on it:
//
//	subject  "insights.events.asset.validated"
//	consumer "<app-slug>-insights-events-asset-validated"
//
// Subjects that differ only by '.' vs '-' produce the same slug and must not both be registered (e.g. "foo.bar" and
// "foo-bar" would collide).
func durableNameFor(appSlug, subject string) string {
	slug := strings.NewReplacer(".", "-", "*", "wild", ">", "all").Replace(subject)
	return appSlug + "-" + slug
}
