package nats

import "testing"

func TestDurableNameFor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		appSlug string
		subject string
		want    string
	}{
		{
			name:    "dotted subject becomes hyphenated and app-scoped",
			appSlug: "insights",
			subject: "insights.events.asset.validated",
			want:    "insights-insights-events-asset-validated",
		},
		{
			name:    "single token wildcard maps to wild",
			appSlug: "svc",
			subject: "events.*",
			want:    "svc-events-wild",
		},
		{
			name:    "multi token wildcard maps to all",
			appSlug: "svc",
			subject: "events.>",
			want:    "svc-events-all",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := durableNameFor(tc.appSlug, tc.subject); got != tc.want {
				t.Fatalf("durableNameFor(%q, %q) = %q, want %q", tc.appSlug, tc.subject, got, tc.want)
			}
		})
	}
}

// Subjects differing only by '.' vs '-' collapse to the same durable name, so they must never both be registered. This
// documents that constraint.
func TestDurableNameFor_DotAndHyphenCollide(t *testing.T) {
	t.Parallel()
	if durableNameFor("svc", "foo.bar") != durableNameFor("svc", "foo-bar") {
		t.Fatal("expected '.' and '-' subjects to produce the same durable name")
	}
}
