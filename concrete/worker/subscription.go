package worker

import (
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
)

// SubscriptionTag is the fx value-group tag a service tags its Subscriptions with so its
// activation wiring can collect them. It lives here (not in fx wiring) so the producing and
// consuming sides agree on the group name; the activation invoke itself is owned by the service.
const SubscriptionTag = `group:"messaging_subscriptions"`

// Subscription is a named bundle of message-handler registrations. A service splits its
// registrations into named subscriptions so a worker can run a subset
// (`work --subscriptions=a,b`); with none selected it runs all. Grouping lets each
// consumer be deployed and scaled as its own worker.
type Subscription struct {
	Name  string
	Apply func()
}

// Filter selects which subscriptions to activate. An empty Subscriptions activates all.
type Filter struct {
	Subscriptions []string
}

// Activate applies the selected subscriptions onto their routers. An empty selection
// activates every subscription; an unknown name is a fatal misconfiguration so a typo
// fails fast instead of silently subscribing to nothing.
func Activate(filter Filter, subscriptions []Subscription) error {
	byName := make(map[string]Subscription, len(subscriptions))
	for _, s := range subscriptions {
		byName[s.Name] = s
	}

	if len(filter.Subscriptions) == 0 {
		for _, s := range subscriptions {
			s.Apply()
		}
		return nil
	}

	for _, name := range filter.Subscriptions {
		s, ok := byName[name]
		if !ok {
			return errors.Errorf("unknown subscription %q (available: %s)", name, available(subscriptions))
		}
		s.Apply()
	}
	return nil
}

func available(subscriptions []Subscription) string {
	names := make([]string, 0, len(subscriptions))
	for _, s := range subscriptions {
		names = append(names, s.Name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
