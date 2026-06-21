package worker_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tupicapp/go-modules/concrete/worker"
)

func registrar(applied *[]string) *worker.Registrar {
	r := worker.NewRegistrar()
	for _, name := range []string{"events", "tasks", "storage"} {
		r.Add(name, func() { *applied = append(*applied, name) })
	}
	return r
}

func TestActivate_EmptyFilterRunsAll(t *testing.T) {
	var applied []string
	require.NoError(t, registrar(&applied).Activate(worker.Filter{}))
	require.ElementsMatch(t, []string{"events", "tasks", "storage"}, applied)
}

func TestActivate_SelectedSubsetOnly(t *testing.T) {
	var applied []string
	err := registrar(&applied).Activate(worker.Filter{Subscriptions: []string{"storage", "tasks"}})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"storage", "tasks"}, applied)
}

func TestActivate_UnknownNameFailsFast(t *testing.T) {
	var applied []string
	err := registrar(&applied).Activate(worker.Filter{Subscriptions: []string{"events", "bogus"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown subscription \"bogus\"")
	require.Contains(t, err.Error(), "events, storage, tasks") // available, sorted
}
