/*
 * mob.go
 *
 * This file implements the mobile entity (mob) system for the MUD.
 * It defines data structures and functions for creating, managing, and
 * interacting with NPCs in the game world. The file handles mob spawning,
 * movement, combat, and reset mechanics. It includes functionality for
 * tracking mob instances, finding mobs in rooms, and processing mob resets
 * to maintain the game world's population.
 */

package main

import (
	"fmt"
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
	Wandering        bool     `yaml:"wandering"` // Whether this mob wanders around
	HomeArea         string   // The area this mob belongs to and should stay within

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

	//fmt.Printf("Registering mob [%d]: %s\nLong Description: %s\n", mob.ID, mob.ShortDescription, mob.LongDescription)
	mobMutex.Lock()
	defer mobMutex.Unlock()

	// Calculate base stats based on level and toughness
	calculateMobStats(mob)

	mobRegistry[mob.ID] = mob
	//log.Printf("Registered mob [%d]: %s", mob.ID, mob.ShortDescription)
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

	//fmt.Printf("Spawning mob [%d]: %s\nLong Description: %s\n", mobID, mobTemplate.ShortDescription, mobTemplate.LongDescription)

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
			Wandering:        mobTemplate.Wandering,
			HomeArea:         room.Area,
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

	//log.Printf("Spawned mob [%d] instance %d in room %d", mobID, instance.InstanceID, room.ID)
	return instance, nil
}

// GetMobsInRoom returns all mobs in a specific room
func GetMobsInRoom(roomID int) []*MobInstance {
	mobMutex.RLock()
	defer mobMutex.RUnlock()

	// Debug logging
	// if mobs, exists := roomMobs[roomID]; exists && len(mobs) > 0 {
	// 	fmt.Printf("DEBUG: Room %d has %d mobs:\n", roomID, len(mobs))
	// 	for _, mob := range mobs {
	// 		if mob != nil {
	// 			fmt.Printf("DEBUG: - %s (ID: %d)\n", mob.ShortDescription, mob.ID)
	// 		}
	// 	}
	// }

	return roomMobs[roomID]
}

// FindMobInRoom finds a mob in a room by name or keyword
func FindMobInRoom(roomID int, searchTerm string) *MobInstance {
	// First, check if the search term has a numeric prefix
	if mob := FindMobInRoomByNumericPrefix(roomID, searchTerm); mob != nil {
		return mob
	}

	mobMutex.RLock()
	defer mobMutex.RUnlock()

	mobs := roomMobs[roomID]
	if len(mobs) == 0 {
		return nil
	}

	searchTerm = strings.ToLower(searchTerm)

	// Group mobs by keyword/short description for potential numbering
	mobGroups := make(map[string][]*MobInstance)

	// First pass: group mobs by keywords and short descriptions
	for _, mob := range mobs {
		// Check keywords
		for _, keyword := range mob.Keywords {
			lowerKeyword := strings.ToLower(keyword)
			mobGroups[lowerKeyword] = append(mobGroups[lowerKeyword], mob)
		}

		// Also group by short description
		shortDesc := strings.ToLower(mob.ShortDescription)
		mobGroups[shortDesc] = append(mobGroups[shortDesc], mob)
	}

	// Second pass: check if the search term matches any group
	for groupKey, group := range mobGroups {
		if groupKey == searchTerm {
			// If there's a direct match, return the first mob in the group
			return group[0]
		}
	}

	// Third pass: check for partial matches in short descriptions
	for _, mob := range mobs {
		if strings.Contains(strings.ToLower(mob.ShortDescription), searchTerm) {
			return mob
		}
	}

	return nil
}

// FindMobInRoomByNumericPrefix finds a mob in a room by numeric prefix and keyword
// Format: "2.cityguard" would find the second cityguard in the room
func FindMobInRoomByNumericPrefix(roomID int, searchTerm string) *MobInstance {
	mobMutex.RLock()
	defer mobMutex.RUnlock()

	mobs := roomMobs[roomID]
	if len(mobs) == 0 {
		return nil
	}

	// Check if the search term has a numeric prefix
	parts := strings.SplitN(searchTerm, ".", 2)
	if len(parts) != 2 {
		// No numeric prefix, use the standard FindMobInRoom function
		return nil
	}

	// Parse the numeric prefix
	index, err := strconv.Atoi(parts[0])
	if err != nil || index < 1 {
		// Invalid numeric prefix
		return nil
	}

	// Get the actual search term (without the numeric prefix)
	keyword := strings.ToLower(parts[1])

	// Create a slice to hold matching mobs
	var matchingMobs []*MobInstance

	// First, try to match by exact keyword
	for _, mob := range mobs {
		for _, mobKeyword := range mob.Keywords {
			if strings.ToLower(mobKeyword) == keyword {
				matchingMobs = append(matchingMobs, mob)
				break
			}
		}
	}

	// If no exact keyword matches, try partial matches in short description
	if len(matchingMobs) == 0 {
		for _, mob := range mobs {
			if strings.Contains(strings.ToLower(mob.ShortDescription), keyword) {
				matchingMobs = append(matchingMobs, mob)
			}
		}
	}

	// Check if the index is valid for the matching mobs
	if index <= len(matchingMobs) {
		return matchingMobs[index-1] // Convert to 0-based index
	}

	// Index is out of range or no matching mobs found
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
	//log.Println("Processing mob resets...")

	// Lock the mob mutex to prevent race conditions
	mobMutex.Lock()
	defer mobMutex.Unlock()

	// Group resets by mob ID to handle world limits properly
	mobResetsByID := make(map[int][]MobReset)
	for _, reset := range mobResets {
		mobResetsByID[reset.MobVnum] = append(mobResetsByID[reset.MobVnum], reset)
	}

	// Process resets by mob ID
	for mobID, resets := range mobResetsByID {
		// Check if the mob exists in the registry
		if mobRegistry[mobID] == nil {
			//log.Printf("Skipping resets for mob %d: not found in registry", mobID)
			continue
		}

		// Get current world count and max world limit
		currentWorldCount := worldMobCounts[mobID]
		maxWorld := 0
		for _, reset := range resets {
			if reset.MaxWorld > maxWorld {
				maxWorld = reset.MaxWorld
			}
		}

		// If we've already reached the world limit, skip all resets for this mob
		if currentWorldCount >= maxWorld {
			//log.Printf("Skipping all resets for mob %d: world limit of %d reached", mobID, maxWorld)
			continue
		}

		// Calculate how many more we can spawn based on world limit
		remainingAllowed := maxWorld - currentWorldCount

		// Shuffle the resets to avoid predictable spawn patterns
		rng.Shuffle(len(resets), func(i, j int) {
			resets[i], resets[j] = resets[j], resets[i]
		})

		// For mobs with multiple spawn points (like janitors), distribute them evenly
		// rather than filling up one room before moving to the next
		if len(resets) > 1 && remainingAllowed > 1 {
			// First pass: try to spawn one mob per reset location until we reach the limit
			for i := 0; i < len(resets) && remainingAllowed > 0; i++ {
				reset := resets[i]
				room, err := GetRoom(reset.RoomVnum)
				if err != nil {
					//log.Printf("Error getting room %d for mob reset: %v", reset.RoomVnum, err)
					continue
				}

				// Check if this room already has this type of mob
				roomHasMob := false
				for _, instance := range roomMobs[room.ID] {
					if instance.ID == mobID {
						roomHasMob = true
						break
					}
				}

				// If room doesn't have this mob yet and room limit allows, spawn one
				if !roomHasMob && reset.Limit > 0 {
					// Get the mob template
					mobTemplate := mobRegistry[mobID]

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
							Wandering:        mobTemplate.Wandering,
							HomeArea:         room.Area,
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

					remainingAllowed--
				}
			}
		}

		// Second pass: traditional processing for any remaining mobs to spawn
		if remainingAllowed > 0 {
			// Process each reset until we reach the world limit
			for _, reset := range resets {
				if remainingAllowed <= 0 {
					break
				}

				room, err := GetRoom(reset.RoomVnum)
				if err != nil {
					//log.Printf("Error getting room %d for mob reset: %v", reset.RoomVnum, err)
					continue
				}

				// Check room limit
				roomLimit := reset.Limit
				roomCount := 0
				for _, instance := range roomMobs[room.ID] {
					if instance.ID == mobID {
						roomCount++
					}
				}

				// Skip if room is already at or over limit
				if roomCount >= roomLimit {
					continue
				}

				// Calculate how many more can be spawned in this room
				roomRemaining := roomLimit - roomCount
				if roomRemaining > remainingAllowed {
					roomRemaining = remainingAllowed
				}

				// Spawn the mobs
				for i := 0; i < roomRemaining; i++ {
					// Get the mob template
					mobTemplate := mobRegistry[mobID]

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
							Wandering:        mobTemplate.Wandering,
							HomeArea:         room.Area,
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

					remainingAllowed--
					if remainingAllowed <= 0 {
						break
					}
				}
			}
		}
	}

	//log.Println("Mob resets completed")
}

// MoveMob moves a mob from one room to another
func MoveMob(mob *MobInstance, direction string) error {
	if mob.Room == nil {
		return fmt.Errorf("mob is not in a room")
	}

	// Check if the mob is in combat - if so, prevent movement
	if IsMobInCombat(mob) {
		return fmt.Errorf("mob cannot move while in combat")
	}

	// Check if the exit exists
	exit, exists := mob.Room.Exits[direction]
	if !exists {
		return fmt.Errorf("no exit in that direction")
	}

	// Check if there's a closed door blocking the way
	if exit.Door != nil && exit.Door.Closed {
		return fmt.Errorf("the %s is closed", exit.Door.ShortDescription)
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
		return fmt.Errorf("unsupported exit type")
	}

	destRoom, err := GetRoom(destRoomID)
	if err != nil {
		return err
	}

	// Check if the destination room has the NoWandering flag set
	// If it does, prevent mobs from wandering into it
	if destRoom.NoWandering {
		return fmt.Errorf("room has no_wandering flag set")
	}

	// Check if the mob is trying to leave its home area
	// Only apply this restriction if the mob has a home area set
	if mob.HomeArea != "" && destRoom.Area != mob.HomeArea {
		return fmt.Errorf("mob cannot leave its home area")
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
				mob.ShortDescription, GetOppositeDirection(direction))))
		}
	}

	return nil
}

// RemoveMobFromRoom removes a mob instance from a room
func RemoveMobFromRoom(mob *MobInstance) {
	if mob == nil || mob.Room == nil {
		return
	}

	mobMutex.Lock()
	defer mobMutex.Unlock()

	roomID := mob.Room.ID

	// Find and remove the mob from the room's mob list
	for i, m := range roomMobs[roomID] {
		if m.InstanceID == mob.InstanceID {
			// Remove by swapping with last element and truncating
			roomMobs[roomID][i] = roomMobs[roomID][len(roomMobs[roomID])-1]
			roomMobs[roomID] = roomMobs[roomID][:len(roomMobs[roomID])-1]
			break
		}
	}

	// Decrease the world count for this mob type
	worldMobCounts[mob.ID]--
	if worldMobCounts[mob.ID] < 0 {
		worldMobCounts[mob.ID] = 0
	}

	// Remove from instances map
	delete(mobInstances, mob.InstanceID)

	// Log the removal
	//log.Printf("[MOB] Removed mob %s (ID: %d, Instance: %d) from room %d",
	//	mob.ShortDescription, mob.ID, mob.InstanceID, roomID)
}

// IsMobInCombat checks if any player is currently fighting this mob
func IsMobInCombat(mob *MobInstance) bool {
	playersMutex.Lock()
	defer playersMutex.Unlock()

	for _, player := range activePlayers {
		if player.IsInCombat() && player.Target == mob {
			return true
		}
	}
	return false
}

// ProcessMobWandering makes certain mobs wander randomly between rooms
func ProcessMobWandering() {
	// Global chance to process wandering at all (15% chance per pulse)
	// This means wandering will only be considered in 15% of pulses
	if rng.Intn(100) >= 15 {
		return
	}

	mobMutex.Lock()
	defer mobMutex.Unlock()

	// Process each mob instance
	for _, mob := range mobInstances {
		// Skip if this mob type shouldn't wander
		if !mob.Wandering {
			continue
		}

		// Skip if this mob is in combat with any player
		if IsMobInCombat(mob) {
			continue
		}

		// Individual mob chance to move (20% chance when wandering is processed)
		// Combined with the global 20% chance, this gives a 1% effective chance per pulse
		if rng.Intn(100) >= 20 {
			continue
		}

		// Get available exits from the current room
		if mob.Room == nil {
			continue
		}

		availableExits := make([]string, 0)
		for dir, exit := range mob.Room.Exits {
			// Skip exits with closed doors
			if exit.Door != nil && exit.Door.Closed {
				continue
			}
			availableExits = append(availableExits, dir)
		}

		// Skip if no exits
		if len(availableExits) == 0 {
			continue
		}

		// Choose a random direction
		randomDir := availableExits[rng.Intn(len(availableExits))]

		// Unlock the mutex before calling MoveMob to avoid deadlock
		// since MoveMob will acquire the lock
		mobMutex.Unlock()
		err := MoveMob(mob, randomDir)
		mobMutex.Lock() // Re-acquire the lock

		if err != nil {
			//log.Printf("Error moving mob %s: %v", mob.ShortDescription, err)
			continue
		}
	}
}

// FindMobByTarget is a helper function that abstracts the mob finding logic
// It first tries to find a mob by numeric prefix, then falls back to standard search
// This function should be used by all commands that need to target mobs
func FindMobByTarget(roomID int, targetName string) *MobInstance {
	// First try to find by numeric prefix
	mob := FindMobInRoomByNumericPrefix(roomID, targetName)

	// If not found by numeric prefix, try standard search
	if mob == nil {
		mob = FindMobInRoom(roomID, targetName)
	}

	return mob
}
