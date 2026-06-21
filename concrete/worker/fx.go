package worker

import "go.uber.org/fx"

// SubscriptionTag is the fx value-group tag a service tags its Subscriptions with so the
// activation module can collect them. Must match the struct tag in activateParams.
const SubscriptionTag = `group:"messaging_subscriptions"`

// SubscriptionActivationModule collects every tagged Subscription plus the optional worker
// Filter and activates the selection. Services provide their subscriptions with
// fx.ResultTags(worker.SubscriptionTag) and include this module; the worker supplies
// a Filter via fx.Supply (absent → all subscriptions run).
var SubscriptionActivationModule = fx.Module("messaging.subscriptions", fx.Invoke(activate))

type activateParams struct {
	fx.In
	Filter        Filter         `optional:"true"`
	Subscriptions []Subscription `group:"messaging_subscriptions"`
}

func activate(p activateParams) error {
	return Activate(p.Filter, p.Subscriptions)
}
