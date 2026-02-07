package main

import (
	"log/slog"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// --- Client → Server ---

func logInventoryTransaction(p *packet.InventoryTransaction, state *GameState) {
	if !state.VerbosePacketLog() {
		return
	}
	switch td := p.TransactionData.(type) {
	case *protocol.UseItemTransactionData:
		actionName := useItemActionName(td.ActionType)
		itemName := state.ResolveItemName(td.HeldItem.Stack.NetworkID)
		slog.Info("pkt", "dir", "C→S", "pkt", "InventoryTransaction", "type", "UseItem",
			"action", actionName,
			"pos", formatBlockPos(td.BlockPosition),
			"face", blockFaceName(td.BlockFace),
			"rid", td.BlockRuntimeID,
			"item", itemName,
			"slot", td.HotBarSlot,
		)
		// Learn block mapping when placing a block
		if td.ActionType == protocol.UseItemActionClickBlock && itemName != "unknown:0" {
			state.LearnBlock(td.BlockRuntimeID, itemName)
		}
	case *protocol.UseItemOnEntityTransactionData:
		actionName := "Interact"
		if td.ActionType == protocol.UseItemOnEntityActionAttack {
			actionName = "Attack"
		}
		slog.Info("pkt", "dir", "C→S", "pkt", "InventoryTransaction", "type", "UseItemOnEntity",
			"action", actionName,
			"target", td.TargetEntityRuntimeID,
			"item", state.ResolveItemName(td.HeldItem.Stack.NetworkID),
		)
	case *protocol.NormalTransactionData:
		slog.Info("pkt", "dir", "C→S", "pkt", "InventoryTransaction", "type", "Normal",
			"actions", len(p.Actions),
		)
	case *protocol.MismatchTransactionData:
		slog.Warn("pkt", "dir", "C→S", "pkt", "InventoryTransaction", "type", "Mismatch (inventory desync)")
	}
}

func logPlayerAction(p *packet.PlayerAction, state *GameState) {
	if !state.VerbosePacketLog() {
		return
	}
	// Only log building-relevant actions
	switch p.ActionType {
	case protocol.PlayerActionStartBreak,
		protocol.PlayerActionAbortBreak,
		protocol.PlayerActionStopBreak,
		protocol.PlayerActionDropItem,
		protocol.PlayerActionCreativePlayerDestroyBlock,
		protocol.PlayerActionCrackBreak,
		protocol.PlayerActionStartBuildingBlock,
		protocol.PlayerActionPredictDestroyBlock,
		protocol.PlayerActionContinueDestroyBlock,
		protocol.PlayerActionStartItemUseOn,
		protocol.PlayerActionStopItemUseOn:
		slog.Info("pkt", "dir", "C→S", "pkt", "PlayerAction",
			"action", playerActionName(p.ActionType),
			"pos", formatBlockPos(p.BlockPosition),
			"resultPos", formatBlockPos(p.ResultPosition),
			"face", blockFaceName(p.BlockFace),
		)
	}
}

func logMobEquipment(p *packet.MobEquipment, state *GameState) {
	if !state.VerbosePacketLog() {
		return
	}
	slog.Info("pkt", "dir", "C→S", "pkt", "MobEquipment",
		"item", state.ResolveItemName(p.NewItem.Stack.NetworkID),
		"slot", p.HotBarSlot,
		"window", p.WindowID,
	)
}

func logPlayerAuthInputBuilding(p *packet.PlayerAuthInput, state *GameState) {
	if !state.VerbosePacketLog() {
		return
	}
	if p.InputData.Load(packet.InputFlagPerformItemInteraction) {
		td := p.ItemInteractionData
		slog.Info("pkt", "dir", "C→S", "pkt", "PlayerAuthInput", "flag", "ItemInteraction",
			"action", useItemActionName(td.ActionType),
			"pos", formatBlockPos(td.BlockPosition),
			"face", blockFaceName(td.BlockFace),
			"rid", td.BlockRuntimeID,
			"item", state.ResolveItemName(td.HeldItem.Stack.NetworkID),
		)
	}
	if p.InputData.Load(packet.InputFlagPerformBlockActions) {
		for _, ba := range p.BlockActions {
			slog.Info("pkt", "dir", "C→S", "pkt", "PlayerAuthInput", "flag", "BlockAction",
				"action", playerActionName(ba.Action),
				"pos", formatBlockPos(ba.BlockPos),
				"face", blockFaceName(ba.Face),
			)
		}
	}
}

// --- Server → Client ---

func logUpdateBlock(p *packet.UpdateBlock, state *GameState) {
	if !state.VerbosePacketLog() {
		return
	}
	slog.Info("pkt", "dir", "S→C", "pkt", "UpdateBlock",
		"pos", formatBlockPos(p.Position),
		"rid", p.NewBlockRuntimeID,
		"name", state.ResolveBlockName(p.NewBlockRuntimeID),
		"flags", p.Flags,
		"layer", p.Layer,
	)
}

func logLevelEvent(p *packet.LevelEvent, state *GameState) {
	if !state.VerbosePacketLog() {
		return
	}
	var eventName string
	switch p.EventType {
	case packet.LevelEventStartBlockCracking:
		eventName = "StartBlockCracking"
	case packet.LevelEventStopBlockCracking:
		eventName = "StopBlockCracking"
	case packet.LevelEventUpdateBlockCracking:
		eventName = "UpdateBlockCracking"
	case packet.LevelEventParticlesDestroyBlock:
		eventName = "ParticlesDestroyBlock"
	default:
		return // not a building event
	}
	slog.Info("pkt", "dir", "S→C", "pkt", "LevelEvent",
		"event", eventName,
		"pos", formatVec3(p.Position),
		"data", p.EventData,
	)
}

func logItemStackResponse(p *packet.ItemStackResponse, state *GameState) {
	if !state.VerbosePacketLog() {
		return
	}
	for _, resp := range p.Responses {
		slog.Info("pkt", "dir", "S→C", "pkt", "ItemStackResponse",
			"status", resp.Status,
			"requestID", resp.RequestID,
		)
	}
}

func logContainerOpen(p *packet.ContainerOpen, state *GameState) {
	if !state.VerbosePacketLog() {
		return
	}
	slog.Info("pkt", "dir", "S→C", "pkt", "ContainerOpen",
		"window", p.WindowID,
		"type", p.ContainerType,
		"pos", formatBlockPos(p.ContainerPosition),
	)
}

func logContainerClose(p *packet.ContainerClose, state *GameState) {
	if !state.VerbosePacketLog() {
		return
	}
	slog.Info("pkt", "dir", "S→C", "pkt", "ContainerClose",
		"window", p.WindowID,
		"type", p.ContainerType,
		"serverSide", p.ServerSide,
	)
}
