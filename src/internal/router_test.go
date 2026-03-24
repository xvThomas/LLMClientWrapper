package internal

import (
	"testing"
)

func TestRouter_GetRegisteredModel(t *testing.T) {
	r := NewRouter()
	client := &stubClient{}
	r.Register("sonnet-4.6", client)

	got, err := r.Get("sonnet-4.6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != client {
		t.Error("expected registered client to be returned")
	}
}

func TestRouter_GetUnknownModelReturnsError(t *testing.T) {
	r := NewRouter()
	_, err := r.Get("unknown-model")
	if err == nil {
		t.Error("expected error for unknown model, got nil")
	}
}

func TestRouter_RegisterOverwritesPreviousClient(t *testing.T) {
	r := NewRouter()
	first := &stubClient{}
	second := &stubClient{}
	r.Register("sonnet-4.6", first)
	r.Register("sonnet-4.6", second)

	got, err := r.Get("sonnet-4.6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != second {
		t.Error("expected second client after overwrite")
	}
}
