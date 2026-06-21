package worker

import (
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
	contract "github.com/tupicapp/go-modules/contract/worker"
)

// Registrar collects named subscriptions and activates a selected subset at startup. It implements
// contract/worker.Registry: a service's interface layer Adds its subscriptions during wiring, and the
// worker composition calls Activate with the --subscriptions filter.
type Registrar struct {
	subscriptions []subscription
}

type subscription struct {
	name  string
	apply func()
}

// NewRegistrar returns an empty Registrar.
func NewRegistrar() *Registrar { return &Registrar{} }

// Add registers a named subscription; apply runs its route registrations when the subscription activates.
func (r *Registrar) Add(name string, apply func()) {
	r.subscriptions = append(r.subscriptions, subscription{name: name, apply: apply})
}

// Filter selects which subscriptions to activate. An empty Subscriptions activates all.
type Filter struct {
	Subscriptions []string
}

// Activate applies the selected subscriptions onto their routers. An empty selection activates every
// subscription; an unknown name is a fatal misconfiguration so a typo fails fast instead of silently
// subscribing to nothing.
func (r *Registrar) Activate(filter Filter) error {
	byName := make(map[string]subscription, len(r.subscriptions))
	for _, s := range r.subscriptions {
		byName[s.name] = s
	}

	if len(filter.Subscriptions) == 0 {
		for _, s := range r.subscriptions {
			s.apply()
		}
		return nil
	}

	for _, name := range filter.Subscriptions {
		s, ok := byName[name]
		if !ok {
			return errors.Errorf("unknown subscription %q (available: %s)", name, r.available())
		}
		s.apply()
	}
	return nil
}

func (r *Registrar) available() string {
	names := make([]string, 0, len(r.subscriptions))
	for _, s := range r.subscriptions {
		names = append(names, s.name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

var _ contract.Registry = (*Registrar)(nil)
