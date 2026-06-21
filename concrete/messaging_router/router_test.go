package messaging_router

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/tupicapp/go-modules/concrete/noop_logger"
	"github.com/tupicapp/go-modules/contract/messaging"
	"github.com/tupicapp/go-modules/kernel/apperror"
)

func newTestRouter() *Router {
	return NewRouter(noop_logger.NewNoop())
}

func msg(payload string) messaging.Message {
	return messaging.Message{Version: "1", Payload: json.RawMessage(payload)}
}

func TestRouterDispatchesToRegisteredHandler(t *testing.T) {
	r := newTestRouter()

	var got json.RawMessage
	r.Register("test.subject", func(_ context.Context, m messaging.Message) error {
		got = m.Payload
		return nil
	})

	if err := r.Handle(context.Background(), "test.subject", msg(`{"key":"value"}`)); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if string(got) != `{"key":"value"}` {
		t.Fatalf("handler received %s, want %s", got, `{"key":"value"}`)
	}
}

func TestRouterUnknownSubjectIsSkipped(t *testing.T) {
	r := newTestRouter()

	if err := r.Handle(context.Background(), "unknown.subject", msg(`{}`)); err != nil {
		t.Fatalf("Handle() expected nil for unknown subject, got %v", err)
	}
}

func TestRouterHandlerErrorPropagates(t *testing.T) {
	r := newTestRouter()
	appErr := apperror.NotFound("not found")

	r.Register("test.subject", func(_ context.Context, _ messaging.Message) error {
		return appErr
	})

	err := r.Handle(context.Background(), "test.subject", msg(`{}`))
	if !apperror.IsAppError(err) {
		t.Fatalf("expected AppError to propagate, got %T: %v", err, err)
	}
}

func TestRouterInfraErrorPropagates(t *testing.T) {
	r := newTestRouter()

	r.Register("test.subject", func(_ context.Context, _ messaging.Message) error {
		return errors.New("db connection refused")
	})

	err := r.Handle(context.Background(), "test.subject", msg(`{}`))
	if apperror.IsAppError(err) {
		t.Fatal("infra error should not be wrapped as AppError")
	}
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRouterPanicsOnDuplicateRegistration(t *testing.T) {
	r := newTestRouter()
	r.Register("test.dup", func(_ context.Context, _ messaging.Message) error { return nil })

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	r.Register("test.dup", func(_ context.Context, _ messaging.Message) error { return nil })
}
