package main

import (
	"log/slog"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// interceptClientPacket processes a packet from the client heading to the server.
// It updates state but never modifies the packet.
func interceptClientPacket(pk packet.Packet, state *GameState) {
	switch p := pk.(type) {
	case *packet.PlayerAuthInput:
		state.UpdatePosition(
			p.Position.X(), p.Position.Y(), p.Position.Z(),
			p.Pitch, p.Yaw,
		)
	case *packet.Text:
		if p.TextType == packet.TextTypeChat {
			state.AppendChat(ChatMessage{
				Time:    time.Now(),
				Source:  p.SourceName,
				Message: p.Message,
				Type:    "outgoing",
			})
		}
	}
}

// interceptServerPacket processes a packet from the server heading to the client.
// It updates state but never modifies the packet.
func interceptServerPacket(pk packet.Packet, state *GameState) {
	switch p := pk.(type) {
	case *packet.MovePlayer:
		if p.EntityRuntimeID == state.EntityID() {
			state.UpdatePosition(
				p.Position.X(), p.Position.Y(), p.Position.Z(),
				p.Pitch, p.Yaw,
			)
		}

	case *packet.ChangeDimension:
		state.SetDimension(p.Dimension)
		slog.Debug("dimension changed", "dimension", p.Dimension)

	case *packet.InventoryContent:
		state.SetInventory(byte(p.WindowID), p.Content)

	case *packet.InventorySlot:
		state.UpdateInventorySlot(byte(p.WindowID), int(p.Slot), p.NewItem)

	case *packet.Text:
		state.AppendChat(ChatMessage{
			Time:    time.Now(),
			Source:  p.SourceName,
			Message: p.Message,
			Type:    "incoming",
		})

	case *packet.PlayerList:
		if p.ActionType == packet.PlayerListActionAdd {
			for _, entry := range p.Entries {
				state.AddPlayer(entry.XUID, entry.Username)
			}
		} else if p.ActionType == packet.PlayerListActionRemove {
			for _, entry := range p.Entries {
				state.RemovePlayer(entry.XUID)
			}
		}

	case *packet.SetTime:
		state.SetWorldTime(int64(p.Time))

	case *packet.UpdateAttributes:
		if p.EntityRuntimeID == state.EntityID() {
			for _, attr := range p.Attributes {
				state.SetAttribute(attr.Name, attr.Value)
			}
		}

	case *packet.SetHealth:
		state.SetHealth(float32(p.Health))

	case *packet.AddActor:
		state.AddEntity(p.EntityRuntimeID, p.EntityType, p.Position)

	case *packet.AddPlayer:
		state.AddEntity(p.EntityRuntimeID, p.Username, p.Position)

	case *packet.RemoveActor:
		// EntityUniqueID is int64; our entity map uses uint64 runtime IDs.
		// Servers typically use the same value for both.
		state.RemoveEntity(uint64(p.EntityUniqueID))

	case *packet.MoveActorDelta:
		state.UpdateEntityPosition(p.EntityRuntimeID, p.Position)
	}
}
