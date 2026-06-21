package gorm_outbox

// Config carries the relay's publishing identity.
// SubjectPrefix prefixes every published subject; Source identifies the publishing service in the event
// envelope (typically the app slug).
type Config struct {
	SubjectPrefix string
	Source        string
}
