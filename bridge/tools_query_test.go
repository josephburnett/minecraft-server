package main

import (
	"strings"
	"testing"
)

func TestRequireConnected(t *testing.T) {
	gs := NewGameState()

	// Not connected — should return error
	gs.SetStatus(StatusStarting)
	if err := requireConnected(gs); err == nil {
		t.Error("expected error when status is starting")
	} else if !strings.Contains(err.Error(), "starting") {
		t.Errorf("error should mention status, got: %v", err)
	}

	gs.SetStatus(StatusWaitingForClient)
	if err := requireConnected(gs); err == nil {
		t.Error("expected error when status is waiting_for_client")
	}

	gs.SetStatus(StatusConnectingToRealm)
	if err := requireConnected(gs); err == nil {
		t.Error("expected error when status is connecting_to_realm")
	}

	gs.SetStatus(StatusDisconnected)
	if err := requireConnected(gs); err == nil {
		t.Error("expected error when status is disconnected")
	}

	// Connected — should return nil
	gs.SetStatus(StatusConnected)
	if err := requireConnected(gs); err != nil {
		t.Errorf("expected nil error when connected, got: %v", err)
	}
}

func TestJsonResult(t *testing.T) {
	data := map[string]any{
		"name":  "test",
		"count": 42,
	}
	result, err := jsonResult(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
}

func TestJsonResultContent(t *testing.T) {
	data := map[string]string{"hello": "world"}
	result, err := jsonResult(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The result should have content
	if len(result.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
}

func TestDimensionName(t *testing.T) {
	tests := []struct {
		dim  int32
		want string
	}{
		{0, "overworld"},
		{1, "nether"},
		{2, "the_end"},
		{99, "unknown(99)"},
		{-1, "unknown(-1)"},
	}
	for _, tt := range tests {
		got := dimensionName(tt.dim)
		if got != tt.want {
			t.Errorf("dimensionName(%d) = %q, want %q", tt.dim, got, tt.want)
		}
	}
}

func TestGameModeName(t *testing.T) {
	tests := []struct {
		mode int32
		want string
	}{
		{0, "survival"},
		{1, "creative"},
		{2, "adventure"},
		{3, "spectator"},
		{99, "unknown(99)"},
		{-1, "unknown(-1)"},
	}
	for _, tt := range tests {
		got := gameModeName(tt.mode)
		if got != tt.want {
			t.Errorf("gameModeName(%d) = %q, want %q", tt.mode, got, tt.want)
		}
	}
}
