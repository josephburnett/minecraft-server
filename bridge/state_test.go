package main

import (
	"fmt"
	"sync"
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestNewGameState(t *testing.T) {
	gs := NewGameState()
	if gs.Status() != StatusStarting {
		t.Errorf("expected status %q, got %q", StatusStarting, gs.Status())
	}
	if gs.VerbosePacketLog() {
		t.Error("expected verbosePacketLog to be false")
	}
	// Maps should be initialized (non-nil)
	// Inventory returns nil when empty, Players returns empty non-nil slice
	inv := gs.Inventory()
	if len(inv) != 0 {
		t.Errorf("expected empty inventory, got %d items", len(inv))
	}
	p := gs.Players()
	if len(p) != 0 {
		t.Errorf("expected empty players, got %d", len(p))
	}
}

func TestStatus(t *testing.T) {
	gs := NewGameState()
	statuses := []string{
		StatusStarting, StatusWaitingForClient, StatusConnectingToRealm,
		StatusConnected, StatusDisconnected,
	}
	for _, s := range statuses {
		gs.SetStatus(s)
		if got := gs.Status(); got != s {
			t.Errorf("expected status %q, got %q", s, got)
		}
	}
}

func TestIdentity(t *testing.T) {
	gs := NewGameState()
	gs.SetIdentity("Steve", "12345", 42)
	name, xuid := gs.Identity()
	if name != "Steve" {
		t.Errorf("expected name %q, got %q", "Steve", name)
	}
	if xuid != "12345" {
		t.Errorf("expected xuid %q, got %q", "12345", xuid)
	}
	if gs.EntityID() != 42 {
		t.Errorf("expected entityID 42, got %d", gs.EntityID())
	}
}

func TestPosition(t *testing.T) {
	gs := NewGameState()
	gs.UpdatePosition(1.0, 2.0, 3.0, 45.0, 90.0)
	gs.SetDimension(1)
	x, y, z, pitch, yaw, dim := gs.Position()
	if x != 1.0 || y != 2.0 || z != 3.0 {
		t.Errorf("expected position (1,2,3), got (%v,%v,%v)", x, y, z)
	}
	if pitch != 45.0 || yaw != 90.0 {
		t.Errorf("expected rotation (45,90), got (%v,%v)", pitch, yaw)
	}
	if dim != 1 {
		t.Errorf("expected dimension 1, got %d", dim)
	}
}

func TestInventory(t *testing.T) {
	gs := NewGameState()
	// Register an item name
	gs.mu.Lock()
	gs.itemRegistry[5] = "minecraft:stone"
	gs.mu.Unlock()

	items := []protocol.ItemInstance{
		{Stack: protocol.ItemStack{ItemType: protocol.ItemType{NetworkID: 5}, Count: 16}},
		{Stack: protocol.ItemStack{ItemType: protocol.ItemType{NetworkID: 0}, Count: 0}}, // empty, should be skipped
		{Stack: protocol.ItemStack{ItemType: protocol.ItemType{NetworkID: 99}, Count: 1}}, // unknown item
	}
	gs.SetInventory(0, items)

	inv := gs.Inventory()
	if len(inv) != 2 {
		t.Fatalf("expected 2 non-empty slots, got %d", len(inv))
	}

	// Check that we have the known and unknown items
	found := map[string]bool{}
	for _, slot := range inv {
		found[slot.Item] = true
	}
	if !found["minecraft:stone"] {
		t.Error("expected minecraft:stone in inventory")
	}
	if !found["unknown:99"] {
		t.Error("expected unknown:99 in inventory")
	}
}

func TestUpdateInventorySlot_NewWindow(t *testing.T) {
	gs := NewGameState()
	item := protocol.ItemInstance{Stack: protocol.ItemStack{ItemType: protocol.ItemType{NetworkID: 1}, Count: 5}}
	gs.UpdateInventorySlot(10, 3, item)

	gs.mu.RLock()
	defer gs.mu.RUnlock()
	slots, ok := gs.inventory[10]
	if !ok {
		t.Fatal("expected window 10 to exist")
	}
	if len(slots) < 4 {
		t.Fatalf("expected at least 4 slots, got %d", len(slots))
	}
	if slots[3].Stack.NetworkID != 1 {
		t.Errorf("expected NetworkID 1 at slot 3, got %d", slots[3].Stack.NetworkID)
	}
}

func TestUpdateInventorySlot_GrowsSlice(t *testing.T) {
	gs := NewGameState()
	// Start with a small inventory
	gs.SetInventory(0, []protocol.ItemInstance{
		{Stack: protocol.ItemStack{ItemType: protocol.ItemType{NetworkID: 1}, Count: 1}},
	})
	// Update a slot beyond current length
	item := protocol.ItemInstance{Stack: protocol.ItemStack{ItemType: protocol.ItemType{NetworkID: 2}, Count: 10}}
	gs.UpdateInventorySlot(0, 5, item)

	gs.mu.RLock()
	defer gs.mu.RUnlock()
	if len(gs.inventory[0]) < 6 {
		t.Fatalf("expected at least 6 slots, got %d", len(gs.inventory[0]))
	}
	if gs.inventory[0][5].Stack.NetworkID != 2 {
		t.Errorf("expected NetworkID 2 at slot 5, got %d", gs.inventory[0][5].Stack.NetworkID)
	}
}

func TestResolveItemName(t *testing.T) {
	gs := NewGameState()
	gs.mu.Lock()
	gs.itemRegistry[10] = "minecraft:dirt"
	gs.mu.Unlock()

	if got := gs.ResolveItemName(10); got != "minecraft:dirt" {
		t.Errorf("expected minecraft:dirt, got %q", got)
	}
	if got := gs.ResolveItemName(999); got != "unknown:999" {
		t.Errorf("expected unknown:999, got %q", got)
	}
}

func TestChatRingBuffer(t *testing.T) {
	gs := NewGameState()
	// Add 150 messages
	for i := 0; i < 150; i++ {
		gs.AppendChat(ChatMessage{Message: fmt.Sprintf("msg%d", i)})
	}
	history := gs.ChatHistory(0) // 0 means all
	if len(history) != 100 {
		t.Fatalf("expected 100 messages, got %d", len(history))
	}
	// First message should be msg50 (oldest kept)
	if history[0].Message != "msg50" {
		t.Errorf("expected first message %q, got %q", "msg50", history[0].Message)
	}
	// Last message should be msg149
	if history[99].Message != "msg149" {
		t.Errorf("expected last message %q, got %q", "msg149", history[99].Message)
	}
}

func TestChatHistory(t *testing.T) {
	gs := NewGameState()

	// Empty history
	if h := gs.ChatHistory(5); len(h) != 0 {
		t.Errorf("expected empty history, got %d", len(h))
	}

	// Add 10 messages
	for i := 0; i < 10; i++ {
		gs.AppendChat(ChatMessage{Message: fmt.Sprintf("msg%d", i)})
	}

	// n=0 returns all
	if h := gs.ChatHistory(0); len(h) != 10 {
		t.Errorf("expected 10 messages for n=0, got %d", len(h))
	}

	// n > len returns all
	if h := gs.ChatHistory(50); len(h) != 10 {
		t.Errorf("expected 10 messages for n=50, got %d", len(h))
	}

	// n < len returns last n
	h := gs.ChatHistory(3)
	if len(h) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(h))
	}
	if h[0].Message != "msg7" {
		t.Errorf("expected first of last 3 to be msg7, got %q", h[0].Message)
	}
}

func TestPlayers(t *testing.T) {
	gs := NewGameState()
	gs.AddPlayer("x1", "Alice")
	gs.AddPlayer("x2", "Bob")
	gs.AddPlayer("x3", "Charlie")

	players := gs.Players()
	if len(players) != 3 {
		t.Fatalf("expected 3 players, got %d", len(players))
	}

	gs.RemovePlayer("x2")
	players = gs.Players()
	if len(players) != 2 {
		t.Fatalf("expected 2 players after removal, got %d", len(players))
	}
	for _, p := range players {
		if p.Username == "Bob" {
			t.Error("Bob should have been removed")
		}
	}
}

func TestHealth(t *testing.T) {
	gs := NewGameState()
	gs.SetHealth(15.5)

	_, _, _, health, _ := gs.WorldInfo()
	if health != 15.5 {
		t.Errorf("expected health 15.5, got %v", health)
	}
	// Also check attributes map
	gs.mu.RLock()
	if gs.attributes["health"] != 15.5 {
		t.Errorf("expected attributes[health]=15.5, got %v", gs.attributes["health"])
	}
	gs.mu.RUnlock()
}

func TestSetAttribute(t *testing.T) {
	gs := NewGameState()

	// minecraft:health should update the health field
	gs.SetAttribute("minecraft:health", 18.0)
	_, _, _, health, _ := gs.WorldInfo()
	if health != 18.0 {
		t.Errorf("expected health 18.0, got %v", health)
	}

	// Other attributes should only update the map
	gs.SetAttribute("minecraft:movement", 0.1)
	gs.mu.RLock()
	if gs.attributes["minecraft:movement"] != 0.1 {
		t.Errorf("expected movement 0.1, got %v", gs.attributes["minecraft:movement"])
	}
	gs.mu.RUnlock()
}

func TestInitFromGameData(t *testing.T) {
	gs := NewGameState()
	gd := minecraft.GameData{
		WorldName:      "TestWorld",
		Time:           12345,
		Dimension:      0,
		WorldSpawn:     protocol.BlockPos{0, 64, 0},
		PlayerPosition: mgl32.Vec3{10, 65, 20},
		Pitch:          30.0,
		Yaw:            90.0,
		PlayerGameMode: 1,
		Items: []protocol.ItemEntry{
			{RuntimeID: 5, Name: "minecraft:stone"},
			{RuntimeID: 10, Name: "minecraft:dirt"},
		},
	}
	gs.InitFromGameData(gd)

	worldName, worldTime, gameMode, health, spawnPos := gs.WorldInfo()
	if worldName != "TestWorld" {
		t.Errorf("expected worldName %q, got %q", "TestWorld", worldName)
	}
	if worldTime != 12345 {
		t.Errorf("expected worldTime 12345, got %d", worldTime)
	}
	if gameMode != 1 {
		t.Errorf("expected gameMode 1, got %d", gameMode)
	}
	if health != 20 {
		t.Errorf("expected health 20, got %v", health)
	}
	if spawnPos.X() != 0 || spawnPos.Y() != 64 || spawnPos.Z() != 0 {
		t.Errorf("expected spawn (0,64,0), got (%d,%d,%d)", spawnPos.X(), spawnPos.Y(), spawnPos.Z())
	}

	x, y, z, pitch, yaw, dim := gs.Position()
	if x != 10 || y != 65 || z != 20 {
		t.Errorf("expected position (10,65,20), got (%v,%v,%v)", x, y, z)
	}
	if pitch != 30.0 || yaw != 90.0 {
		t.Errorf("expected rotation (30,90), got (%v,%v)", pitch, yaw)
	}
	if dim != 0 {
		t.Errorf("expected dimension 0, got %d", dim)
	}

	// Item registry should be populated
	if got := gs.ResolveItemName(5); got != "minecraft:stone" {
		t.Errorf("expected minecraft:stone, got %q", got)
	}
	if got := gs.ResolveItemName(10); got != "minecraft:dirt" {
		t.Errorf("expected minecraft:dirt, got %q", got)
	}
}

func TestWorldInfo(t *testing.T) {
	gs := NewGameState()
	gs.mu.Lock()
	gs.worldName = "MyWorld"
	gs.worldTime = 6000
	gs.gameMode = 2
	gs.health = 19.5
	gs.spawnPos = protocol.BlockPos{100, 70, -50}
	gs.mu.Unlock()

	worldName, worldTime, gameMode, health, spawnPos := gs.WorldInfo()
	if worldName != "MyWorld" {
		t.Errorf("expected %q, got %q", "MyWorld", worldName)
	}
	if worldTime != 6000 {
		t.Errorf("expected 6000, got %d", worldTime)
	}
	if gameMode != 2 {
		t.Errorf("expected 2, got %d", gameMode)
	}
	if health != 19.5 {
		t.Errorf("expected 19.5, got %v", health)
	}
	if spawnPos.X() != 100 || spawnPos.Y() != 70 || spawnPos.Z() != -50 {
		t.Errorf("expected spawn (100,70,-50), got (%d,%d,%d)", spawnPos.X(), spawnPos.Y(), spawnPos.Z())
	}
}

func TestEntities(t *testing.T) {
	gs := NewGameState()

	gs.AddEntity(100, "minecraft:zombie", mgl32.Vec3{10, 20, 30})
	gs.AddEntity(101, "minecraft:skeleton", mgl32.Vec3{40, 50, 60})

	gs.mu.RLock()
	if len(gs.entities) != 2 {
		t.Errorf("expected 2 entities, got %d", len(gs.entities))
	}
	gs.mu.RUnlock()

	// Update position
	gs.UpdateEntityPosition(100, mgl32.Vec3{11, 21, 31})
	gs.mu.RLock()
	e := gs.entities[100]
	gs.mu.RUnlock()
	if e.Position.X() != 11 || e.Position.Y() != 21 || e.Position.Z() != 31 {
		t.Errorf("expected updated position (11,21,31), got %v", e.Position)
	}

	// Remove
	gs.RemoveEntity(100)
	gs.mu.RLock()
	if len(gs.entities) != 1 {
		t.Errorf("expected 1 entity after removal, got %d", len(gs.entities))
	}
	gs.mu.RUnlock()

	// Update non-existent entity (should not panic)
	gs.UpdateEntityPosition(999, mgl32.Vec3{0, 0, 0})
}

func TestVerbosePacketLog(t *testing.T) {
	gs := NewGameState()
	if gs.VerbosePacketLog() {
		t.Error("expected false initially")
	}
	gs.SetVerbosePacketLog(true)
	if !gs.VerbosePacketLog() {
		t.Error("expected true after setting")
	}
	gs.SetVerbosePacketLog(false)
	if gs.VerbosePacketLog() {
		t.Error("expected false after unsetting")
	}
}

func TestBlockRegistry(t *testing.T) {
	gs := NewGameState()

	gs.LearnBlock(100, "minecraft:stone")
	if got := gs.ResolveBlockName(100); got != "minecraft:stone" {
		t.Errorf("expected minecraft:stone, got %q", got)
	}
	if got := gs.ResolveBlockName(999); got != "rid:999" {
		t.Errorf("expected rid:999, got %q", got)
	}
}

func TestConcurrency(t *testing.T) {
	gs := NewGameState()
	var wg sync.WaitGroup
	const goroutines = 50

	wg.Add(goroutines * 4)

	// Position writers
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			gs.UpdatePosition(float32(i), float32(i), float32(i), 0, 0)
		}(i)
	}

	// Position readers
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			gs.Position()
		}()
	}

	// Chat writers
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			gs.AppendChat(ChatMessage{Message: fmt.Sprintf("msg%d", i)})
		}(i)
	}

	// Inventory writers
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			gs.UpdateInventorySlot(0, i, protocol.ItemInstance{
				Stack: protocol.ItemStack{ItemType: protocol.ItemType{NetworkID: int32(i)}, Count: 1},
			})
		}(i)
	}

	wg.Wait()
	// If we get here without deadlock or panic, concurrency is OK.
	// The -race flag will catch data races.
}
