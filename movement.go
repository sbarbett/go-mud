/*
 * movement.go
 *
 * This file implements the movement system for players in the MUD.
 * It provides functions for handling player movement between rooms,
 * processing direction commands, and managing the transitions between
 * different areas of the game world. The file includes logic for
 * validating movement requests and updating player locations in both
 * memory and the database.
 */

package main

import (
	"fmt"     // Importing the fmt package for formatted I/O operations
	"log"     // Importing the log package for logging
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
	currentRoom := player.Room
	// fmt.Printf("Debug - Current Room: ID=%d, Name=%s, Area=%s\n",
	// 	currentRoom.ID, currentRoom.Name, currentRoom.Area)

	// Check if the exit in the specified direction exists
	exit, exists := currentRoom.Exits[direction]
	if !exists {
		return currentRoom, fmt.Errorf("you can't go that way")
	}

	// Check if there's a closed door blocking the way
	if exit.Door != nil && exit.Door.Closed {
		return currentRoom, fmt.Errorf("the %s is closed", exit.Door.ShortDescription)
	}

	// Debug logging
	// fmt.Printf("Debug - MovePlayer: Moving from Room %d to %v\n",
	// 	currentRoom.ID, exit)

	// Handle different types of room movement based on exit ID type
	switch exitID := exit.ID.(type) {
	case int:
		newRoom, err := GetRoom(exitID)
		if err != nil {
			return currentRoom, err
		}
		err = UpdatePlayerRoom(player.Name, exitID)
		if err != nil {
			return currentRoom, err
		}
		// fmt.Printf("Debug - Moved to Room: ID=%d, Name=%s, Area=%s\n",
		// 	newRoom.ID, newRoom.Name, newRoom.Area)
		return newRoom, nil

	case string:
		// Handle cross-area movement
		roomInfo := strings.Split(exitID, ":")
		if len(roomInfo) != 2 {
			return currentRoom, fmt.Errorf("invalid room reference")
		}

		roomID, err := strconv.Atoi(roomInfo[1])
		if err != nil {
			return currentRoom, fmt.Errorf("invalid room ID")
		}

		newRoom, err := GetRoom(roomID)
		if err != nil {
			return currentRoom, err
		}

		err = UpdatePlayerRoom(player.Name, roomID)
		if err != nil {
			return currentRoom, err
		}
		// fmt.Printf("Debug - Moved to Room (cross-area): ID=%d, Name=%s, Area=%s\n",
		// 	newRoom.ID, newRoom.Name, newRoom.Area)
		return newRoom, nil
	}

	return currentRoom, fmt.Errorf("invalid exit type")
}

// DirectionAliases maps shorthand commands to full direction names
var DirectionAliases = map[string]string{
	"n": "north",
	"s": "south",
	"e": "east",
	"w": "west",
	"u": "up",
	"d": "down",
}

// HandleMovement processes movement commands and executes the movement
func HandleMovement(player *Player, command string) error {
	// Check if the player is in combat
	if player.IsInCombat() {
		return fmt.Errorf("you can't move while in combat")
	}

	// Check if the command is a shorthand direction and convert it
	if fullDirection, isAlias := DirectionAliases[command]; isAlias {
		command = fullDirection
	}

	// Store the old room for notifications
	oldRoom := player.Room

	// Attempt to move the player
	newRoom, err := MovePlayer(player, command)
	if err != nil {
		return err
	}

	// Notify players in the old room about departure
	playersMutex.Lock()
	for _, p := range activePlayers {
		if p != player && p.Room == oldRoom {
			p.Send(fmt.Sprintf("%s leaves %s.", player.Name, command))
		}
	}
	playersMutex.Unlock()

	// Update player's room
	player.Room = newRoom

	// Send movement message and room description to moving player
	player.Send(fmt.Sprintf("You move %s.", command))
	player.Send(DescribeRoom(newRoom, player))

	// Notify players in the new room about arrival
	playersMutex.Lock()
	for _, p := range activePlayers {
		if p != player && p.Room == newRoom {
			p.Send(fmt.Sprintf("%s arrives.", player.Name))
		}
	}
	playersMutex.Unlock()

	return nil
}

// SynchronizeDoor ensures that a door's state is synchronized between connected rooms
// Parameters:
//
//	sourceRoomID - the ID of the room where the door state was changed
//	direction - the direction of the exit with the door
//	closed - the new state of the door (true = closed, false = open)
func SynchronizeDoor(sourceRoomID int, direction string, closed bool) {
	// Get the source room
	sourceRoom, err := GetRoom(sourceRoomID)
	if err != nil {
		log.Printf("[ERROR] Failed to get source room %d: %v", sourceRoomID, err)
		return
	}

	// Get the exit in the specified direction
	exit, exists := sourceRoom.Exits[direction]
	if !exists || exit.Door == nil {
		return // No exit or no door in that direction
	}

	// Get the destination room
	var destRoomID int
	switch exitID := exit.ID.(type) {
	case int:
		destRoomID = exitID
	case string:
		// Handle cross-area references
		roomInfo := strings.Split(exitID, ":")
		if len(roomInfo) != 2 {
			return
		}
		var err error
		destRoomID, err = strconv.Atoi(roomInfo[1])
		if err != nil {
			return
		}
	default:
		return
	}

	destRoom, err := GetRoom(destRoomID)
	if err != nil {
		log.Printf("[ERROR] Failed to get destination room %d: %v", destRoomID, err)
		return
	}

	// Find the opposite direction
	oppositeDirection := GetOppositeDirection(direction)

	// Update the door state in the destination room
	destExit, exists := destRoom.Exits[oppositeDirection]
	if exists && destExit.Door != nil {
		destExit.Door.Closed = closed

		// Notify players in the destination room about the door state change
		playersMutex.Lock()
		for _, p := range activePlayers {
			if p.Room != nil && p.Room.ID == destRoomID {
				if closed {
					p.Send(fmt.Sprintf("The %s closes.", destExit.Door.ShortDescription))
				} else {
					p.Send(fmt.Sprintf("The %s opens.", destExit.Door.ShortDescription))
				}
			}
		}
		playersMutex.Unlock()
	}
}

// getOppositeDirection returns the opposite of a given direction
// ... existing code ...
