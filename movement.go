package main

import (
	"fmt"     // Importing the fmt package for formatted I/O operations
	"strconv" // Importing the strconv package for converting strings to integers
	"strings" // Importing the strings package for string manipulation functions
)

// MovePlayer moves a player to a new room if possible
// Parameters:
//
//	player - a pointer to the Player struct representing the player
//	direction - a string indicating the direction to move
//
// Returns:
//
//	A pointer to the new Room the player is moving to, or an error if movement is not possible
func MovePlayer(player *Player, direction string) (*Room, error) {
	currentRoom := player.Room // Get the player's current room

	// Check if the exit in the specified direction exists
	newRoomData, exists := currentRoom.Exits[direction]
	if !exists {
		// Return the current room and an error if there is no exit in the desired direction
		return currentRoom, fmt.Errorf("you can't go that way")
	}

	// Handle different types of room movement
	switch newRoomData := newRoomData.(type) {
	case int: // Normal in-area movement, where newRoomData is an integer
		newRoomID := newRoomData           // Store the new room ID
		newRoom, err := GetRoom(newRoomID) // Retrieve the new room using the ID
		if err != nil {
			// Return current room and error if getting the new room fails
			return currentRoom, err
		}
		// Update the player's current room to the new room ID
		err = UpdatePlayerRoom(player.Name, newRoomID)
		return newRoom, err // Return the new room and any error encountered

	case string: // Cross-area movement, where newRoomData is a formatted string
		roomInfo := strings.Split(newRoomData, ":") // Split the string into area file and room ID parts
		if len(roomInfo) != 2 {
			// Return current room and an error if the room info is improperly formatted
			return currentRoom, fmt.Errorf("invalid room reference: %s", newRoomData)
		}

		areaFile, roomIDStr := roomInfo[0], roomInfo[1] // Extract area file and room ID string
		roomID, err := strconv.Atoi(roomIDStr)          // Convert room ID string to an integer
		if err != nil {
			// Return current room and an error if room ID conversion fails
			return currentRoom, fmt.Errorf("invalid room ID in reference: %s", newRoomData)
		}

		// Call function to load the new area file. Ensure the path is correct.
		loadArea("areas/" + areaFile)

		newRoom, err := GetRoom(roomID) // Retrieve the new room by its ID
		if err != nil {
			// Return current room and error if getting the new room fails
			return currentRoom, err
		}
		// Update the player's room to the new room ID
		err = UpdatePlayerRoom(player.Name, roomID)
		return newRoom, err // Return the new room and any error encountered
	}

	// Return current room and error if the exit data type is invalid
	return currentRoom, fmt.Errorf("invalid exit data type")
}
