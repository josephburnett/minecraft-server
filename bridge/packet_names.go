package main

import (
	"fmt"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// playerActionName returns a readable name for building-relevant player actions.
func playerActionName(action int32) string {
	switch action {
	case protocol.PlayerActionStartBreak:
		return "StartBreak"
	case protocol.PlayerActionAbortBreak:
		return "AbortBreak"
	case protocol.PlayerActionStopBreak:
		return "StopBreak"
	case protocol.PlayerActionDropItem:
		return "DropItem"
	case protocol.PlayerActionCreativePlayerDestroyBlock:
		return "CreativeDestroyBlock"
	case protocol.PlayerActionCrackBreak:
		return "CrackBreak"
	case protocol.PlayerActionStartBuildingBlock:
		return "StartBuildingBlock"
	case protocol.PlayerActionPredictDestroyBlock:
		return "PredictDestroyBlock"
	case protocol.PlayerActionContinueDestroyBlock:
		return "ContinueDestroyBlock"
	case protocol.PlayerActionStartItemUseOn:
		return "StartItemUseOn"
	case protocol.PlayerActionStopItemUseOn:
		return "StopItemUseOn"
	default:
		return fmt.Sprintf("Action(%d)", action)
	}
}

// useItemActionName returns a readable name for UseItem action types.
func useItemActionName(action uint32) string {
	switch action {
	case protocol.UseItemActionClickBlock:
		return "ClickBlock"
	case protocol.UseItemActionClickAir:
		return "ClickAir"
	case protocol.UseItemActionBreakBlock:
		return "BreakBlock"
	default:
		return fmt.Sprintf("UseItemAction(%d)", action)
	}
}

// blockFaceName returns a readable name for a block face.
func blockFaceName(face int32) string {
	switch face {
	case 0:
		return "Down"
	case 1:
		return "Up"
	case 2:
		return "North"
	case 3:
		return "South"
	case 4:
		return "West"
	case 5:
		return "East"
	default:
		return fmt.Sprintf("Face(%d)", face)
	}
}

// formatBlockPos formats a BlockPos as "[x, y, z]".
func formatBlockPos(pos protocol.BlockPos) string {
	return fmt.Sprintf("[%d, %d, %d]", pos[0], pos[1], pos[2])
}

// formatVec3 formats a Vec3 as "(x, y, z)".
func formatVec3(v mgl32.Vec3) string {
	return fmt.Sprintf("(%.1f, %.1f, %.1f)", v.X(), v.Y(), v.Z())
}
