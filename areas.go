package main

import (
	"fmt"     // Importing the fmt package for formatted I/O operations
	"strconv" // Importing the strconv package for converting strings to integers
	"strings" // Importing the strings package for joining strings
)

// DescribeRoom prints the description of the current room
// It takes a pointer to a Room object and returns a formatted string
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
		// Debug logging to help understand what's happening
		fmt.Printf("Debug - Comparing rooms: Viewer Room ID: %d, Player %s Room ID: %d, Same pointer: %v\n",
			viewer.Room.ID, p.Name, p.Room.ID, viewer.Room == p.Room)

		// Only add players who are in exactly the same room instance
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

// CanMove checks if the player can move in a specified direction
// It takes a pointer to a Room and a string direction, returning an integer exit ID and a boolean indicating if the move is possible
func CanMove(room *Room, direction string) (int, bool) {
	// Attempt to retrieve the exit information from the room's Exits map using the given direction
	rawExit, exists := room.Exits[direction]
	// If the direction does not exist in the Exits map, return 0 and false
	if !exists {
		return 0, false
	}

	// Check if the rawExit is of type int, if so return it with true
	if exitID, ok := rawExit.(int); ok {
		return exitID, true
	} else if exitStr, ok := rawExit.(string); ok { // If the exit is of type string, convert it to an integer
		// Attempt to convert the string to an integer; if there's an error return 0 and false
		exitID, err := strconv.Atoi(exitStr)
		if err != nil {
			return 0, false
		}
		// Successfully converted string to integer exit ID
		return exitID, true
	}

	// If the exit is neither an int nor a string, return 0 and false
	return 0, false
}
