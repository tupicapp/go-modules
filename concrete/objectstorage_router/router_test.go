package objectstorage_router

import (
	"testing"

	"github.com/tupicapp/go-modules/contract/objectstorage"
)

func TestRegistererExactMatchTakesPrecedenceOverWildcard(t *testing.T) {
	r := NewRouter()
	var called string
	r.Register("ObjectCreated:*", func(_ objectstorage.Event) error {
		called = "wildcard"
		return nil
	})
	r.Register("ObjectCreated:Put", func(_ objectstorage.Event) error {
		called = "exact"
		return nil
	})

	_ = r.Handle(objectstorage.Event{EventName: "ObjectCreated:Put"})

	if called != "exact" {
		t.Fatalf("called = %q, want exact", called)
	}
}

func TestRegistererWildcardMatchesCategory(t *testing.T) {
	r := NewRouter()
	var called string
	r.Register("ObjectCreated:*", func(_ objectstorage.Event) error {
		called = "wildcard"
		return nil
	})

	_ = r.Handle(objectstorage.Event{EventName: "ObjectCreated:Copy"})

	if called != "wildcard" {
		t.Fatalf("called = %q, want wildcard", called)
	}
}

func TestRegistererNoMatchIsNoOp(t *testing.T) {
	r := NewRouter()
	r.Register("ObjectCreated:*", func(_ objectstorage.Event) error {
		t.Fatal("handler must not run")
		return nil
	})

	if err := r.Handle(objectstorage.Event{EventName: "ObjectRemoved:Delete"}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
}
