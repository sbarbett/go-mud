package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"
)

// Mob represents a mobile entity in the game
type Mob struct {
	ID               int      `yaml:"id"`
	Keywords         []string `yaml:"keywords"`
	ShortDescription string   `yaml:"short_description"` // Used when the mob performs an action
	LongDescription  string   `yaml:"long_description"`  // Displayed when the mob is in a room
	Description      string   `yaml:"description"`       // Displayed when a player looks at the mob
	Race             string   `yaml:"race"`
	Level            int      `yaml:"level"`
	Toughness        string   `yaml:"toughness"`

	// Derived stats
	HP    int
	MaxHP int

	// Current room
	Room *Room
}

// MobReset represents a mob spawn configuration
type MobReset struct {
	MobVnum  int    `yaml:"mob_vnum"`
	RoomVnum int    `yaml:"room_vnum"`
	Limit    int    `yaml:"limit"`
	MaxWorld int    `yaml:"max_world"`
	Comment  string `yaml:"comment"`
}

// MobInstance represents an actual mob in the game world
type MobInstance struct {
	*Mob
	InstanceID int // Unique identifier for this specific instance
}

// Global variables for mob management
var (
	mobRegistry       = make(map[int]*Mob)           // Maps mob ID to mob template
	mobInstances      = make(map[int]*MobInstance)   // Maps instance ID to mob instance
	worldMobCounts    = make(map[int]int)            // Maps mob ID to count of instances in world
	roomMobs          = make(map[int][]*MobInstance) // Maps room ID to mobs in that room
	mobMutex          sync.RWMutex                   // Mutex for thread-safe mob operations
	nextMobInstanceID = 1                            // Counter for generating unique instance IDs
)

// Toughness multipliers for HP calculation
var toughnessMultipliers = map[string]float64{
	"easy":   0.8,
	"medium": 1.0,
	"hard":   1.2,
	"savage": 1.5,
	"boss":   2.0,
	"god":    5.0,
}

// RegisterMob adds a mob template to the registry
func RegisterMob(mob *Mob) {
	// Trim any extra whitespace from descriptions
	mob.ShortDescription = strings.TrimSpace(mob.ShortDescription)
	mob.LongDescription = strings.TrimSpace(mob.LongDescription)
	mob.Description = strings.TrimSpace(mob.Description)

	fmt.Printf("Registering mob [%d]: %s\nLong Description: %s\n", mob.ID, mob.ShortDescription, mob.LongDescription)
	mobMutex.Lock()
	defer mobMutex.Unlock()

	// Calculate base stats based on level and toughness
	calculateMobStats(mob)

	mobRegistry[mob.ID] = mob
	log.Printf("Registered mob [%d]: %s", mob.ID, mob.ShortDescription)
}

// calculateMobStats sets the derived stats for a mob based on level and toughness
func calculateMobStats(mob *Mob) {
	// Default to medium if toughness is invalid
	multiplier, exists := toughnessMultipliers[strings.ToLower(mob.Toughness)]
	if !exists {
		multiplier = toughnessMultipliers["medium"]
	}

	// Base HP formula: (level * 10) * toughness_multiplier
	baseHP := float64(mob.Level * 10)
	mob.MaxHP = int(baseHP * multiplier)
	mob.HP = mob.MaxHP
}

// SpawnMob creates a new instance of a mob in the specified room
func SpawnMob(mobID int, room *Room) (*MobInstance, error) {
	mobMutex.Lock()
	defer mobMutex.Unlock()

	// Get the mob template
	mobTemplate := mobRegistry[mobID]
	if mobTemplate == nil {
		return nil, fmt.Errorf("mob ID %d not found in registry", mobID)
	}

	fmt.Printf("Spawning mob [%d]: %s\nLong Description: %s\n", mobID, mobTemplate.ShortDescription, mobTemplate.LongDescription)

	// Check if mob template exists
	if worldMobCounts[mobID] >= GetMobMaxWorld(mobID) {
		return nil, fmt.Errorf("world limit reached for mob ID %d", mobID)
	}

	// Check room limit
	roomLimit := GetMobRoomLimit(mobID, room.ID)
	roomCount := 0
	for _, instance := range roomMobs[room.ID] {
		if instance.ID == mobID {
			roomCount++
		}
	}
	if roomCount >= roomLimit {
		return nil, fmt.Errorf("room limit reached for mob ID %d in room %d", mobID, room.ID)
	}

	// Create a new instance
	instance := &MobInstance{
		Mob: &Mob{
			ID:               mobTemplate.ID,
			Keywords:         mobTemplate.Keywords,
			ShortDescription: mobTemplate.ShortDescription,
			LongDescription:  strings.TrimSpace(mobTemplate.LongDescription),
			Description:      strings.TrimSpace(mobTemplate.Description),
			Race:             mobTemplate.Race,
			Level:            mobTemplate.Level,
			Toughness:        mobTemplate.Toughness,
			MaxHP:            mobTemplate.MaxHP,
			HP:               mobTemplate.MaxHP,
			Room:             room,
		},
		InstanceID: nextMobInstanceID,
	}
	nextMobInstanceID++

	// Add to tracking maps
	mobInstances[instance.InstanceID] = instance
	worldMobCounts[mobID]++

	// Add to room
	if roomMobs[room.ID] == nil {
		roomMobs[room.ID] = make([]*MobInstance, 0)
	}
	roomMobs[room.ID] = append(roomMobs[room.ID], instance)

	log.Printf("Spawned mob [%d] instance %d in room %d", mobID, instance.InstanceID, room.ID)
	return instance, nil
}

// GetMobsInRoom returns all mobs in a specific room
func GetMobsInRoom(roomID int) []*MobInstance {
	mobMutex.RLock()
	defer mobMutex.RUnlock()

	// Debug logging
	if mobs, exists := roomMobs[roomID]; exists && len(mobs) > 0 {
		fmt.Printf("DEBUG: Room %d has %d mobs:\n", roomID, len(mobs))
		for _, mob := range mobs {
			if mob != nil {
				fmt.Printf("DEBUG: - %s (ID: %d)\n", mob.ShortDescription, mob.ID)
			}
		}
	}

	return roomMobs[roomID]
}

// FindMobInRoom looks for a mob in the room by keyword
func FindMobInRoom(roomID int, keyword string) *MobInstance {
	mobMutex.RLock()
	defer mobMutex.RUnlock()

	keyword = strings.ToLower(keyword)
	for _, mob := range roomMobs[roomID] {
		for _, k := range mob.Keywords {
			if strings.ToLower(k) == keyword {
				return mob
			}
		}
	}
	return nil
}

// GetMobRoomLimit returns the limit for a specific mob in a specific room
func GetMobRoomLimit(mobID, roomID int) int {
	// This would normally check the reset data
	// For now, return a default of 5 or the value from resets
	for _, reset := range mobResets {
		if reset.MobVnum == mobID && reset.RoomVnum == roomID {
			return reset.Limit
		}
	}
	return 5 // Default limit
}

// GetMobMaxWorld returns the maximum number of instances of a mob allowed in the world
func GetMobMaxWorld(mobID int) int {
	// This would normally check the reset data
	// For now, return a default of 20 or the value from resets
	for _, reset := range mobResets {
		if reset.MobVnum == mobID {
			return reset.MaxWorld
		}
	}
	return 20 // Default max world
}

// Global variable to store mob resets
var mobResets []MobReset

// ProcessMobResets spawns mobs according to the reset configuration
func ProcessMobResets() {
	log.Println("Processing mob resets...")

	for _, reset := range mobResets {
		room, err := GetRoom(reset.RoomVnum)
		if err != nil {
			log.Printf("Error getting room %d for mob reset: %v", reset.RoomVnum, err)
			continue
		}

		// Determine how many to spawn (random up to limit)
		count := rand.Intn(reset.Limit) + 1
		if count > reset.Limit {
			count = reset.Limit
		}

		// Spawn the mobs
		for i := 0; i < count; i++ {
			_, err := SpawnMob(reset.MobVnum, room)
			if err != nil {
				log.Printf("Error spawning mob %d in room %d: %v", reset.MobVnum, reset.RoomVnum, err)
				break
			}
		}
	}

	log.Println("Mob resets completed")
}

// MoveMob moves a mob from one room to another
func MoveMob(mob *MobInstance, direction string) error {
	if mob.Room == nil {
		return fmt.Errorf("mob is not in a room")
	}

	// Check if the exit exists
	exit, exists := mob.Room.Exits[direction]
	if !exists {
		return fmt.Errorf("no exit in that direction")
	}

	// Get the destination room
	var destRoomID int
	switch exitID := exit.ID.(type) {
	case int:
		destRoomID = exitID
	case string:
		// Handle cross-area movement (simplified)
		parts := strings.Split(exitID, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid exit format")
		}
		var err error
		destRoomID, err = strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("invalid room ID in exit")
		}
	default:
		return fmt.Errorf("unknown exit type")
	}

	destRoom, err := GetRoom(destRoomID)
	if err != nil {
		return err
	}

	// Remove from current room
	mobMutex.Lock()
	defer mobMutex.Unlock()

	oldRoom := mob.Room

	// Find and remove from old room's mob list
	for i, m := range roomMobs[oldRoom.ID] {
		if m.InstanceID == mob.InstanceID {
			// Remove by swapping with last element and truncating
			roomMobs[oldRoom.ID][i] = roomMobs[oldRoom.ID][len(roomMobs[oldRoom.ID])-1]
			roomMobs[oldRoom.ID] = roomMobs[oldRoom.ID][:len(roomMobs[oldRoom.ID])-1]
			break
		}
	}

	// Add to new room
	if roomMobs[destRoom.ID] == nil {
		roomMobs[destRoom.ID] = make([]*MobInstance, 0)
	}
	roomMobs[destRoom.ID] = append(roomMobs[destRoom.ID], mob)
	mob.Room = destRoom

	// Notify players in the old room about departure
	for _, p := range activePlayers {
		if p.Room == oldRoom {
			p.Conn.Write([]byte(fmt.Sprintf("%s leaves %s.\r\n", mob.ShortDescription, direction)))
		}
	}

	// Notify players in the new room about arrival
	for _, p := range activePlayers {
		if p.Room == destRoom {
			p.Conn.Write([]byte(fmt.Sprintf("%s arrives from the %s.\r\n",
				mob.ShortDescription, getOppositeDirection(direction))))
		}
	}

	return nil
}

// getOppositeDirection returns the opposite of a given direction
func getOppositeDirection(dir string) string {
	switch dir {
	case "north":
		return "south"
	case "south":
		return "north"
	case "east":
		return "west"
	case "west":
		return "east"
	case "up":
		return "down"
	case "down":
		return "up"
	default:
		return "somewhere"
	}
}
