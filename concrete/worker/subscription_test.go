package worker_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tupicapp/go-modules/concrete/worker"
)

func subscriptions(applied *[]string) []worker.Subscription {
	mk := func(name string) worker.Subscription {
		return worker.Subscription{Name: name, Apply: func() { *applied = append(*applied, name) }}
	}
	return []worker.Subscription{mk("events"), mk("tasks"), mk("storage")}
}

func TestActivate_EmptyFilterRunsAll(t *testing.T) {
	var applied []string
	require.NoError(t, worker.Activate(worker.Filter{}, subscriptions(&applied)))
	require.ElementsMatch(t, []string{"events", "tasks", "storage"}, applied)
}

func TestActivate_SelectedSubsetOnly(t *testing.T) {
	var applied []string
	err := worker.Activate(worker.Filter{Subscriptions: []string{"storage", "tasks"}}, subscriptions(&applied))
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"storage", "tasks"}, applied)
}

func TestActivate_UnknownNameFailsFast(t *testing.T) {
	var applied []string
	err := worker.Activate(worker.Filter{Subscriptions: []string{"events", "bogus"}}, subscriptions(&applied))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown subscription \"bogus\"")
	require.Contains(t, err.Error(), "events, storage, tasks") // available, sorted
}
