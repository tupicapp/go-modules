// Package worker defines the port a service's interface layer uses to declare named worker
// subscriptions. concrete/worker implements it; the worker composition activates a selected subset.
package worker

// Registry collects named subscriptions during wiring. Each subscription's apply callback registers its
// subject→handler routes; the worker activates a selected subset (`work --subscriptions=a,b`) at startup,
// so grouping lets each consumer be deployed and scaled as its own worker.
type Registry interface {
	// Add registers a named subscription; apply runs its route registrations when the subscription activates.
	Add(name string, apply func())
}
