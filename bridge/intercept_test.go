package main

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestIntercept_PlayerAuthInput(t *testing.T) {
	gs := NewGameState()
	pk := &packet.PlayerAuthInput{
		Position: mgl32.Vec3{100, 65, 200},
		Pitch:    30.0,
		Yaw:      90.0,
		InputData: protocol.NewBitset(packet.PlayerAuthInputBitsetSize),
	}
	interceptClientPacket(pk, gs)

	x, y, z, pitch, yaw, _ := gs.Position()
	if x != 100 || y != 65 || z != 200 {
		t.Errorf("expected position (100,65,200), got (%v,%v,%v)", x, y, z)
	}
	if pitch != 30.0 || yaw != 90.0 {
		t.Errorf("expected rotation (30,90), got (%v,%v)", pitch, yaw)
	}
}

func TestIntercept_OutgoingChat(t *testing.T) {
	gs := NewGameState()
	pk := &packet.Text{
		TextType:   packet.TextTypeChat,
		SourceName: "Player1",
		Message:    "hello world",
	}
	interceptClientPacket(pk, gs)

	history := gs.ChatHistory(0)
	if len(history) != 1 {
		t.Fatalf("expected 1 chat message, got %d", len(history))
	}
	if history[0].Type != "outgoing" {
		t.Errorf("expected type outgoing, got %q", history[0].Type)
	}
	if history[0].Source != "Player1" {
		t.Errorf("expected source Player1, got %q", history[0].Source)
	}
	if history[0].Message != "hello world" {
		t.Errorf("expected message 'hello world', got %q", history[0].Message)
	}
}

func TestIntercept_OutgoingNonChat(t *testing.T) {
	gs := NewGameState()
	pk := &packet.Text{
		TextType:   packet.TextTypeSystem,
		SourceName: "Server",
		Message:    "system msg",
	}
	interceptClientPacket(pk, gs)

	history := gs.ChatHistory(0)
	if len(history) != 0 {
		t.Errorf("expected no chat messages for non-chat text, got %d", len(history))
	}
}

func TestIntercept_MovePlayer_OurEntity(t *testing.T) {
	gs := NewGameState()
	gs.SetIdentity("Steve", "123", 42)

	pk := &packet.MovePlayer{
		EntityRuntimeID: 42,
		Position:        mgl32.Vec3{50, 70, 80},
		Pitch:           15.0,
		Yaw:             45.0,
	}
	interceptServerPacket(pk, gs)

	x, y, z, pitch, yaw, _ := gs.Position()
	if x != 50 || y != 70 || z != 80 {
		t.Errorf("expected (50,70,80), got (%v,%v,%v)", x, y, z)
	}
	if pitch != 15.0 || yaw != 45.0 {
		t.Errorf("expected (15,45), got (%v,%v)", pitch, yaw)
	}
}

func TestIntercept_MovePlayer_OtherEntity(t *testing.T) {
	gs := NewGameState()
	gs.SetIdentity("Steve", "123", 42)
	gs.UpdatePosition(1, 2, 3, 4, 5)

	pk := &packet.MovePlayer{
		EntityRuntimeID: 99, // different entity
		Position:        mgl32.Vec3{50, 70, 80},
		Pitch:           15.0,
		Yaw:             45.0,
	}
	interceptServerPacket(pk, gs)

	x, y, z, _, _, _ := gs.Position()
	if x != 1 || y != 2 || z != 3 {
		t.Errorf("position should be unchanged, got (%v,%v,%v)", x, y, z)
	}
}

func TestIntercept_ChangeDimension(t *testing.T) {
	gs := NewGameState()
	pk := &packet.ChangeDimension{
		Dimension: 1,
		Position:  mgl32.Vec3{0, 0, 0},
	}
	interceptServerPacket(pk, gs)

	_, _, _, _, _, dim := gs.Position()
	if dim != 1 {
		t.Errorf("expected dimension 1, got %d", dim)
	}
}

func TestIntercept_InventoryContent(t *testing.T) {
	gs := NewGameState()
	items := []protocol.ItemInstance{
		{Stack: protocol.ItemStack{ItemType: protocol.ItemType{NetworkID: 5}, Count: 10}},
		{Stack: protocol.ItemStack{ItemType: protocol.ItemType{NetworkID: 6}, Count: 20}},
	}
	pk := &packet.InventoryContent{
		WindowID: 0,
		Content:  items,
	}
	interceptServerPacket(pk, gs)

	gs.mu.RLock()
	defer gs.mu.RUnlock()
	if len(gs.inventory[0]) != 2 {
		t.Errorf("expected 2 items, got %d", len(gs.inventory[0]))
	}
}

func TestIntercept_InventorySlot(t *testing.T) {
	gs := NewGameState()
	item := protocol.ItemInstance{Stack: protocol.ItemStack{ItemType: protocol.ItemType{NetworkID: 7}, Count: 5}}
	pk := &packet.InventorySlot{
		WindowID: 0,
		Slot:     3,
		NewItem:  item,
	}
	interceptServerPacket(pk, gs)

	gs.mu.RLock()
	defer gs.mu.RUnlock()
	if gs.inventory[0][3].Stack.NetworkID != 7 {
		t.Errorf("expected NetworkID 7 at slot 3, got %d", gs.inventory[0][3].Stack.NetworkID)
	}
}

func TestIntercept_IncomingChat(t *testing.T) {
	gs := NewGameState()
	pk := &packet.Text{
		TextType:   packet.TextTypeChat,
		SourceName: "OtherPlayer",
		Message:    "hey there",
	}
	interceptServerPacket(pk, gs)

	history := gs.ChatHistory(0)
	if len(history) != 1 {
		t.Fatalf("expected 1 message, got %d", len(history))
	}
	if history[0].Type != "incoming" {
		t.Errorf("expected incoming, got %q", history[0].Type)
	}
}

func TestIntercept_PlayerListAddRemove(t *testing.T) {
	gs := NewGameState()

	// Add players
	addPk := &packet.PlayerList{
		ActionType: packet.PlayerListActionAdd,
		Entries: []protocol.PlayerListEntry{
			{XUID: "x1", Username: "Alice"},
			{XUID: "x2", Username: "Bob"},
		},
	}
	interceptServerPacket(addPk, gs)

	players := gs.Players()
	if len(players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(players))
	}

	// Remove one
	removePk := &packet.PlayerList{
		ActionType: packet.PlayerListActionRemove,
		Entries: []protocol.PlayerListEntry{
			{XUID: "x1"},
		},
	}
	interceptServerPacket(removePk, gs)

	players = gs.Players()
	if len(players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(players))
	}
	if players[0].Username != "Bob" {
		t.Errorf("expected Bob to remain, got %q", players[0].Username)
	}
}

func TestIntercept_SetTime(t *testing.T) {
	gs := NewGameState()
	pk := &packet.SetTime{Time: 12345}
	interceptServerPacket(pk, gs)

	_, worldTime, _, _, _ := gs.WorldInfo()
	if worldTime != 12345 {
		t.Errorf("expected time 12345, got %d", worldTime)
	}
}

func TestIntercept_UpdateAttributes(t *testing.T) {
	gs := NewGameState()
	gs.SetIdentity("Steve", "123", 42)

	pk := &packet.UpdateAttributes{
		EntityRuntimeID: 42,
		Attributes: []protocol.Attribute{
			{AttributeValue: protocol.AttributeValue{Name: "minecraft:health", Value: 15.0}},
		},
	}
	interceptServerPacket(pk, gs)

	_, _, _, health, _ := gs.WorldInfo()
	if health != 15.0 {
		t.Errorf("expected health 15.0, got %v", health)
	}
}

func TestIntercept_UpdateAttributes_OtherEntity(t *testing.T) {
	gs := NewGameState()
	gs.SetIdentity("Steve", "123", 42)
	gs.SetHealth(20)

	pk := &packet.UpdateAttributes{
		EntityRuntimeID: 99, // different entity
		Attributes: []protocol.Attribute{
			{AttributeValue: protocol.AttributeValue{Name: "minecraft:health", Value: 5.0}},
		},
	}
	interceptServerPacket(pk, gs)

	_, _, _, health, _ := gs.WorldInfo()
	if health != 20 {
		t.Errorf("expected health unchanged at 20, got %v", health)
	}
}

func TestIntercept_SetHealth(t *testing.T) {
	gs := NewGameState()
	pk := &packet.SetHealth{Health: 18}
	interceptServerPacket(pk, gs)

	_, _, _, health, _ := gs.WorldInfo()
	if health != 18 {
		t.Errorf("expected health 18, got %v", health)
	}
}

func TestIntercept_AddActor(t *testing.T) {
	gs := NewGameState()
	pk := &packet.AddActor{
		EntityRuntimeID: 200,
		EntityType:      "minecraft:zombie",
		Position:        mgl32.Vec3{10, 20, 30},
	}
	interceptServerPacket(pk, gs)

	gs.mu.RLock()
	defer gs.mu.RUnlock()
	e, ok := gs.entities[200]
	if !ok {
		t.Fatal("expected entity 200 to be tracked")
	}
	if e.Type != "minecraft:zombie" {
		t.Errorf("expected type minecraft:zombie, got %q", e.Type)
	}
}

func TestIntercept_AddPlayer(t *testing.T) {
	gs := NewGameState()
	pk := &packet.AddPlayer{
		EntityRuntimeID: 300,
		Username:        "OtherPlayer",
		Position:        mgl32.Vec3{5, 10, 15},
	}
	interceptServerPacket(pk, gs)

	gs.mu.RLock()
	defer gs.mu.RUnlock()
	e, ok := gs.entities[300]
	if !ok {
		t.Fatal("expected entity 300 to be tracked")
	}
	if e.Type != "OtherPlayer" {
		t.Errorf("expected type OtherPlayer, got %q", e.Type)
	}
}

func TestIntercept_RemoveActor(t *testing.T) {
	gs := NewGameState()
	gs.AddEntity(500, "minecraft:creeper", mgl32.Vec3{0, 0, 0})

	pk := &packet.RemoveActor{EntityUniqueID: 500}
	interceptServerPacket(pk, gs)

	gs.mu.RLock()
	defer gs.mu.RUnlock()
	if _, ok := gs.entities[500]; ok {
		t.Error("expected entity 500 to be removed")
	}
}

func TestIntercept_MoveActorDelta(t *testing.T) {
	gs := NewGameState()
	gs.AddEntity(600, "minecraft:pig", mgl32.Vec3{0, 0, 0})

	pk := &packet.MoveActorDelta{
		EntityRuntimeID: 600,
		Position:        mgl32.Vec3{100, 200, 300},
	}
	interceptServerPacket(pk, gs)

	gs.mu.RLock()
	defer gs.mu.RUnlock()
	e := gs.entities[600]
	if e.Position.X() != 100 || e.Position.Y() != 200 || e.Position.Z() != 300 {
		t.Errorf("expected position (100,200,300), got %v", e.Position)
	}
}
