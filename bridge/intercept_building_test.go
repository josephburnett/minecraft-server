package main

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// captureLogs runs fn while capturing slog output to a buffer and returns the output.
func captureLogs(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(old)
	fn()
	return buf.String()
}

// --- logInventoryTransaction ---

func TestLogInventoryTransaction_VerboseOff(t *testing.T) {
	gs := NewGameState() // verbose off by default
	pk := &packet.InventoryTransaction{
		TransactionData: &protocol.UseItemTransactionData{
			ActionType: protocol.UseItemActionClickBlock,
		},
	}
	output := captureLogs(t, func() {
		logInventoryTransaction(pk, gs)
	})
	if output != "" {
		t.Errorf("expected no output when verbose off, got: %s", output)
	}
}

func TestLogInventoryTransaction_UseItem_LearnsBlock(t *testing.T) {
	gs := NewGameState()
	gs.SetVerbosePacketLog(true)
	// Register item name
	gs.mu.Lock()
	gs.itemRegistry[5] = "minecraft:stone"
	gs.mu.Unlock()

	pk := &packet.InventoryTransaction{
		TransactionData: &protocol.UseItemTransactionData{
			ActionType:     protocol.UseItemActionClickBlock,
			BlockPosition:  protocol.BlockPos{10, 64, 20},
			BlockFace:      1,
			BlockRuntimeID: 42,
			HeldItem:       protocol.ItemInstance{Stack: protocol.ItemStack{ItemType: protocol.ItemType{NetworkID: 5}}},
		},
	}
	output := captureLogs(t, func() {
		logInventoryTransaction(pk, gs)
	})
	if !strings.Contains(output, "InventoryTransaction") {
		t.Errorf("expected log to contain InventoryTransaction, got: %s", output)
	}
	// Verify block was learned
	if got := gs.ResolveBlockName(42); got != "minecraft:stone" {
		t.Errorf("expected block 42 learned as minecraft:stone, got %q", got)
	}
}

func TestLogInventoryTransaction_UseItem_UnknownItem(t *testing.T) {
	gs := NewGameState()
	gs.SetVerbosePacketLog(true)

	pk := &packet.InventoryTransaction{
		TransactionData: &protocol.UseItemTransactionData{
			ActionType:     protocol.UseItemActionClickBlock,
			BlockRuntimeID: 99,
			HeldItem:       protocol.ItemInstance{Stack: protocol.ItemStack{ItemType: protocol.ItemType{NetworkID: 0}}},
		},
	}
	// Should not panic, should not learn block (item is "unknown:0")
	output := captureLogs(t, func() {
		logInventoryTransaction(pk, gs)
	})
	if output == "" {
		t.Error("expected log output for verbose UseItem")
	}
	// Block should NOT be learned because item is "unknown:0"
	if got := gs.ResolveBlockName(99); got != "rid:99" {
		t.Errorf("expected rid:99 (not learned), got %q", got)
	}
}

func TestLogInventoryTransaction_AllTypes(t *testing.T) {
	gs := NewGameState()
	gs.SetVerbosePacketLog(true)

	types := []protocol.InventoryTransactionData{
		&protocol.UseItemTransactionData{ActionType: protocol.UseItemActionClickAir},
		&protocol.UseItemOnEntityTransactionData{ActionType: protocol.UseItemOnEntityActionAttack},
		&protocol.NormalTransactionData{},
		&protocol.MismatchTransactionData{},
	}

	for i, td := range types {
		pk := &packet.InventoryTransaction{TransactionData: td}
		output := captureLogs(t, func() {
			logInventoryTransaction(pk, gs)
		})
		if output == "" {
			t.Errorf("type %d: expected log output, got empty", i)
		}
	}
}

// --- logPlayerAction ---

func TestLogPlayerAction_BuildingActions(t *testing.T) {
	gs := NewGameState()
	gs.SetVerbosePacketLog(true)

	buildingActions := []int32{
		protocol.PlayerActionStartBreak,
		protocol.PlayerActionAbortBreak,
		protocol.PlayerActionStopBreak,
		protocol.PlayerActionDropItem,
		protocol.PlayerActionCreativePlayerDestroyBlock,
		protocol.PlayerActionCrackBreak,
		protocol.PlayerActionStartBuildingBlock,
		protocol.PlayerActionPredictDestroyBlock,
		protocol.PlayerActionContinueDestroyBlock,
		protocol.PlayerActionStartItemUseOn,
		protocol.PlayerActionStopItemUseOn,
	}

	for _, action := range buildingActions {
		pk := &packet.PlayerAction{
			ActionType:    action,
			BlockPosition: protocol.BlockPos{1, 2, 3},
		}
		output := captureLogs(t, func() {
			logPlayerAction(pk, gs)
		})
		if output == "" {
			t.Errorf("action %d: expected log output for building action", action)
		}
	}
}

func TestLogPlayerAction_NonBuildingAction(t *testing.T) {
	gs := NewGameState()
	gs.SetVerbosePacketLog(true)

	pk := &packet.PlayerAction{
		ActionType: protocol.PlayerActionJump,
	}
	output := captureLogs(t, func() {
		logPlayerAction(pk, gs)
	})
	if output != "" {
		t.Errorf("expected no output for non-building action, got: %s", output)
	}
}

// --- logMobEquipment ---

func TestLogMobEquipment(t *testing.T) {
	gs := NewGameState()

	pk := &packet.MobEquipment{
		NewItem:    protocol.ItemInstance{Stack: protocol.ItemStack{ItemType: protocol.ItemType{NetworkID: 1}}},
		HotBarSlot: 0,
		WindowID:   0,
	}

	// Verbose off
	output := captureLogs(t, func() {
		logMobEquipment(pk, gs)
	})
	if output != "" {
		t.Errorf("expected no output when verbose off, got: %s", output)
	}

	// Verbose on
	gs.SetVerbosePacketLog(true)
	output = captureLogs(t, func() {
		logMobEquipment(pk, gs)
	})
	if output == "" {
		t.Error("expected log output when verbose on")
	}
}

// --- logPlayerAuthInputBuilding ---

func TestLogPlayerAuthInput_ItemInteraction(t *testing.T) {
	gs := NewGameState()
	gs.SetVerbosePacketLog(true)

	inputData := protocol.NewBitset(packet.PlayerAuthInputBitsetSize)
	inputData.Set(packet.InputFlagPerformItemInteraction)

	pk := &packet.PlayerAuthInput{
		InputData: inputData,
		ItemInteractionData: protocol.UseItemTransactionData{
			ActionType:    protocol.UseItemActionClickBlock,
			BlockPosition: protocol.BlockPos{5, 10, 15},
		},
	}
	output := captureLogs(t, func() {
		logPlayerAuthInputBuilding(pk, gs)
	})
	if !strings.Contains(output, "ItemInteraction") {
		t.Errorf("expected log to contain ItemInteraction, got: %s", output)
	}
}

func TestLogPlayerAuthInput_BlockActions(t *testing.T) {
	gs := NewGameState()
	gs.SetVerbosePacketLog(true)

	inputData := protocol.NewBitset(packet.PlayerAuthInputBitsetSize)
	inputData.Set(packet.InputFlagPerformBlockActions)

	pk := &packet.PlayerAuthInput{
		InputData: inputData,
		BlockActions: []protocol.PlayerBlockAction{
			{Action: protocol.PlayerActionStartBreak, BlockPos: protocol.BlockPos{1, 2, 3}, Face: 1},
		},
	}
	output := captureLogs(t, func() {
		logPlayerAuthInputBuilding(pk, gs)
	})
	if !strings.Contains(output, "BlockAction") {
		t.Errorf("expected log to contain BlockAction, got: %s", output)
	}
}

func TestLogPlayerAuthInput_NoFlags(t *testing.T) {
	gs := NewGameState()
	gs.SetVerbosePacketLog(true)

	inputData := protocol.NewBitset(packet.PlayerAuthInputBitsetSize)
	pk := &packet.PlayerAuthInput{
		InputData: inputData,
	}
	output := captureLogs(t, func() {
		logPlayerAuthInputBuilding(pk, gs)
	})
	if output != "" {
		t.Errorf("expected no output with no flags set, got: %s", output)
	}
}

func TestLogPlayerAuthInput_VerboseOff(t *testing.T) {
	gs := NewGameState() // verbose off

	inputData := protocol.NewBitset(packet.PlayerAuthInputBitsetSize)
	inputData.Set(packet.InputFlagPerformItemInteraction)

	pk := &packet.PlayerAuthInput{
		InputData: inputData,
	}
	output := captureLogs(t, func() {
		logPlayerAuthInputBuilding(pk, gs)
	})
	if output != "" {
		t.Errorf("expected no output when verbose off, got: %s", output)
	}
}

// --- logUpdateBlock ---

func TestLogUpdateBlock(t *testing.T) {
	gs := NewGameState()
	gs.SetVerbosePacketLog(true)
	gs.LearnBlock(100, "minecraft:dirt")

	pk := &packet.UpdateBlock{
		Position:          protocol.BlockPos{10, 64, 20},
		NewBlockRuntimeID: 100,
		Flags:             1,
		Layer:             0,
	}
	output := captureLogs(t, func() {
		logUpdateBlock(pk, gs)
	})
	if !strings.Contains(output, "UpdateBlock") {
		t.Errorf("expected log to contain UpdateBlock, got: %s", output)
	}
	if !strings.Contains(output, "minecraft:dirt") {
		t.Errorf("expected log to contain block name, got: %s", output)
	}
}

func TestLogUpdateBlock_VerboseOff(t *testing.T) {
	gs := NewGameState()
	pk := &packet.UpdateBlock{Position: protocol.BlockPos{0, 0, 0}}
	output := captureLogs(t, func() {
		logUpdateBlock(pk, gs)
	})
	if output != "" {
		t.Errorf("expected no output when verbose off, got: %s", output)
	}
}

// --- logLevelEvent ---

func TestLogLevelEvent_BuildingEvents(t *testing.T) {
	gs := NewGameState()
	gs.SetVerbosePacketLog(true)

	events := []int32{
		packet.LevelEventStartBlockCracking,
		packet.LevelEventStopBlockCracking,
		packet.LevelEventUpdateBlockCracking,
		packet.LevelEventParticlesDestroyBlock,
	}
	for _, eventType := range events {
		pk := &packet.LevelEvent{
			EventType: eventType,
			Position:  mgl32.Vec3{1, 2, 3},
		}
		output := captureLogs(t, func() {
			logLevelEvent(pk, gs)
		})
		if output == "" {
			t.Errorf("event %d: expected log output for building event", eventType)
		}
	}
}

func TestLogLevelEvent_NonBuildingEvent(t *testing.T) {
	gs := NewGameState()
	gs.SetVerbosePacketLog(true)

	pk := &packet.LevelEvent{
		EventType: 9999, // not a building event
		Position:  mgl32.Vec3{1, 2, 3},
	}
	output := captureLogs(t, func() {
		logLevelEvent(pk, gs)
	})
	if output != "" {
		t.Errorf("expected no output for non-building event, got: %s", output)
	}
}

// --- logItemStackResponse ---

func TestLogItemStackResponse(t *testing.T) {
	gs := NewGameState()
	gs.SetVerbosePacketLog(true)

	pk := &packet.ItemStackResponse{
		Responses: []protocol.ItemStackResponse{
			{Status: 0, RequestID: 1},
			{Status: 0, RequestID: 2},
		},
	}
	output := captureLogs(t, func() {
		logItemStackResponse(pk, gs)
	})
	if !strings.Contains(output, "ItemStackResponse") {
		t.Errorf("expected log to contain ItemStackResponse, got: %s", output)
	}
}

func TestLogItemStackResponse_VerboseOff(t *testing.T) {
	gs := NewGameState()
	pk := &packet.ItemStackResponse{
		Responses: []protocol.ItemStackResponse{{Status: 0}},
	}
	output := captureLogs(t, func() {
		logItemStackResponse(pk, gs)
	})
	if output != "" {
		t.Errorf("expected no output when verbose off, got: %s", output)
	}
}

// --- logContainerOpen / logContainerClose ---

func TestLogContainerOpen(t *testing.T) {
	gs := NewGameState()

	pk := &packet.ContainerOpen{
		WindowID:          1,
		ContainerType:     0,
		ContainerPosition: protocol.BlockPos{5, 6, 7},
	}

	// Verbose off
	output := captureLogs(t, func() {
		logContainerOpen(pk, gs)
	})
	if output != "" {
		t.Errorf("expected no output when verbose off, got: %s", output)
	}

	// Verbose on
	gs.SetVerbosePacketLog(true)
	output = captureLogs(t, func() {
		logContainerOpen(pk, gs)
	})
	if !strings.Contains(output, "ContainerOpen") {
		t.Errorf("expected log to contain ContainerOpen, got: %s", output)
	}
}

func TestLogContainerClose(t *testing.T) {
	gs := NewGameState()

	pk := &packet.ContainerClose{
		WindowID:      1,
		ContainerType: 0,
		ServerSide:    false,
	}

	// Verbose off
	output := captureLogs(t, func() {
		logContainerClose(pk, gs)
	})
	if output != "" {
		t.Errorf("expected no output when verbose off, got: %s", output)
	}

	// Verbose on
	gs.SetVerbosePacketLog(true)
	output = captureLogs(t, func() {
		logContainerClose(pk, gs)
	})
	if !strings.Contains(output, "ContainerClose") {
		t.Errorf("expected log to contain ContainerClose, got: %s", output)
	}
}
