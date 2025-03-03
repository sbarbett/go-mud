package main

import (
	"fmt"
	"sort"
	"strings"
)

// DescribeRoom prints the description of the current room
func DescribeRoom(room *Room, viewer *Player) string {
	// Get available exits and sort them
	var exits []string
	for direction := range room.Exits {
		exits = append(exits, direction)
	}
	sort.Strings(exits)

	// Get list of other players in the room (excluding the viewer)
	playersMutex.Lock()
	var otherPlayers []string
	for _, p := range activePlayers {
		if p != viewer && // Not the viewing player
			p.Room != nil && viewer.Room != nil && // Both rooms exist
			p.Room == viewer.Room { // Exact same room instance
			otherPlayers = append(otherPlayers, p.Name)
		}
	}
	playersMutex.Unlock()

	// Build the room description
	description := fmt.Sprintf("%s\n%s\nAvailable exits: [%s]",
		room.Name,
		room.Description,
		strings.Join(exits, ", "))

	// Add other players if present
	if len(otherPlayers) > 0 {
		description += fmt.Sprintf("\nAlso here: %s", strings.Join(otherPlayers, ", "))
	}

	return description
}

// HandleLook processes the look command and its arguments
func HandleLook(player *Player, args []string) string {
	if len(args) == 0 {
		return DescribeRoom(player.Room, player)
	}

	// Check if looking at a direction
	direction := args[0]
	if fullDirection, isAlias := DirectionAliases[direction]; isAlias {
		direction = fullDirection
	}
	// If it's a direction (either an alias or full name), handle it
	if _, exists := player.Room.Exits[direction]; exists {
		return LookDirection(player.Room, direction)
	}
	// If it's a valid direction but no exit exists
	if _, isDirection := DirectionAliases[direction]; isDirection || stringInSlice(direction, []string{"north", "south", "east", "west", "up", "down"}) {
		return "Nothing special there."
	}

	// Check environment attributes
	lookTarget := strings.ToLower(strings.Join(args, " "))
	for _, attr := range player.Room.Environment {
		for _, keyword := range attr.Keywords {
			if strings.ToLower(keyword) == lookTarget {
				return attr.Description
			}
		}
	}

	return "You do not see that here."
}

// LookDirection returns the description of what's visible in a given direction
func LookDirection(room *Room, direction string) string {
	exit, exists := room.Exits[direction]
	if !exists {
		return "Nothing special there."
	}

	if exit != nil && exit.Description != "" {
		return exit.Description
	}

	return fmt.Sprintf("You see a passage leading %s.", direction)
}
