package main

import (
	"fmt"
	"strings"
)

// DescribeRoom prints the description of the current room
func DescribeRoom(room *Room, viewer *Player) string {
	// Get available exits
	var exits []string
	for direction := range room.Exits {
		exits = append(exits, direction)
	}

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

	direction := args[0]
	// Check if it's a direction alias
	if fullDirection, isAlias := DirectionAliases[direction]; isAlias {
		direction = fullDirection
	}
	return LookDirection(player.Room, direction)
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
