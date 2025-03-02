package main

import (
	"fmt"     // Importing the fmt package for formatted I/O operations
	"strconv" // Importing the strconv package for converting strings to integers
)

// DescribeRoom prints the description of the current room
// It takes a pointer to a Room object and returns a formatted string
func DescribeRoom(room *Room) string {
	// Return a formatted string combining the room's name and description
	return fmt.Sprintf("%s\n%s", room.Name, room.Description)
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
