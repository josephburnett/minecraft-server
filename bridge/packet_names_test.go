package main

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestPlayerActionName(t *testing.T) {
	tests := []struct {
		action int32
		want   string
	}{
		{protocol.PlayerActionStartBreak, "StartBreak"},
		{protocol.PlayerActionAbortBreak, "AbortBreak"},
		{protocol.PlayerActionStopBreak, "StopBreak"},
		{protocol.PlayerActionDropItem, "DropItem"},
		{protocol.PlayerActionCreativePlayerDestroyBlock, "CreativeDestroyBlock"},
		{protocol.PlayerActionCrackBreak, "CrackBreak"},
		{protocol.PlayerActionStartBuildingBlock, "StartBuildingBlock"},
		{protocol.PlayerActionPredictDestroyBlock, "PredictDestroyBlock"},
		{protocol.PlayerActionContinueDestroyBlock, "ContinueDestroyBlock"},
		{protocol.PlayerActionStartItemUseOn, "StartItemUseOn"},
		{protocol.PlayerActionStopItemUseOn, "StopItemUseOn"},
		{9999, "Action(9999)"},
	}
	for _, tt := range tests {
		got := playerActionName(tt.action)
		if got != tt.want {
			t.Errorf("playerActionName(%d) = %q, want %q", tt.action, got, tt.want)
		}
	}
}

func TestUseItemActionName(t *testing.T) {
	tests := []struct {
		action uint32
		want   string
	}{
		{protocol.UseItemActionClickBlock, "ClickBlock"},
		{protocol.UseItemActionClickAir, "ClickAir"},
		{protocol.UseItemActionBreakBlock, "BreakBlock"},
		{9999, "UseItemAction(9999)"},
	}
	for _, tt := range tests {
		got := useItemActionName(tt.action)
		if got != tt.want {
			t.Errorf("useItemActionName(%d) = %q, want %q", tt.action, got, tt.want)
		}
	}
}

func TestBlockFaceName(t *testing.T) {
	tests := []struct {
		face int32
		want string
	}{
		{0, "Down"},
		{1, "Up"},
		{2, "North"},
		{3, "South"},
		{4, "West"},
		{5, "East"},
		{99, "Face(99)"},
	}
	for _, tt := range tests {
		got := blockFaceName(tt.face)
		if got != tt.want {
			t.Errorf("blockFaceName(%d) = %q, want %q", tt.face, got, tt.want)
		}
	}
}

func TestFormatBlockPos(t *testing.T) {
	tests := []struct {
		pos  protocol.BlockPos
		want string
	}{
		{protocol.BlockPos{100, 64, -200}, "[100, 64, -200]"},
		{protocol.BlockPos{0, 0, 0}, "[0, 0, 0]"},
		{protocol.BlockPos{-1, -2, -3}, "[-1, -2, -3]"},
	}
	for _, tt := range tests {
		got := formatBlockPos(tt.pos)
		if got != tt.want {
			t.Errorf("formatBlockPos(%v) = %q, want %q", tt.pos, got, tt.want)
		}
	}
}

func TestFormatVec3(t *testing.T) {
	tests := []struct {
		v    mgl32.Vec3
		want string
	}{
		{mgl32.Vec3{100.5, 64.0, -200.3}, "(100.5, 64.0, -200.3)"},
		{mgl32.Vec3{0, 0, 0}, "(0.0, 0.0, 0.0)"},
	}
	for _, tt := range tests {
		got := formatVec3(tt.v)
		if got != tt.want {
			t.Errorf("formatVec3(%v) = %q, want %q", tt.v, got, tt.want)
		}
	}
}
