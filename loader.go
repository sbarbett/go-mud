/*
 * loader.go
 *
 * This file handles loading game world data from YAML files.
 * It defines the data structures for rooms, exits, areas, and environment
 * attributes, and provides functions for loading area files from the
 * filesystem. The file implements the core world-building functionality
 * by parsing area definitions and making them available to the game engine.
 */

package main

import (
	"fmt"
	"log"           // Package for logging errors
	"os"            // Package for OS functionality, including file operations
	"path/filepath" // Package for manipulating filename paths
	"strconv"       // Package for string conversion
	"strings"       // Package for string manipulation

	"gopkg.in/yaml.v3" // Package for parsing YAML files
)

// Exit represents a direction-specific exit from a room
type Exit struct {
	ID          interface{} `yaml:"id"`             // Can be int or string (for cross-area references)
	Description string      `yaml:"description"`    // Optional description of what's visible in that direction
	Door        *Door       `yaml:"door,omitempty"` // Optional door information
}

// Door represents a door that can be opened, closed, and locked
type Door struct {
	ShortDescription string   `yaml:"short_description"` // Short description of the door
	Keywords         []string `yaml:"keywords"`          // Keywords that can be used to refer to the door
	Locked           bool     `yaml:"locked"`            // Whether the door is locked
	Closed           bool     `yaml:"closed,omitempty"`  // Whether the door is closed (defaults to true if door exists)
}

// EnvironmentAttribute represents a lookable object or detail in a room
type EnvironmentAttribute struct {
	Keywords    []string `yaml:"keywords"`
	Description string   `yaml:"description"`
}

// Room represents a location in the game
type Room struct {
	ID          int                    `yaml:"-"`
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Area        string                 `yaml:"-"`
	Exits       map[string]*Exit       `yaml:"exits"`
	Environment []EnvironmentAttribute `yaml:"environment,omitempty"`
	NoWandering bool                   `yaml:"no_wandering,omitempty"` // If true, mobs cannot wander into this room
}

// Area represents a collection of rooms
type Area struct {
	Name      string        `yaml:"name"`
	Rooms     map[int]*Room `yaml:"rooms"`
	Mobiles   map[int]*Mob  `yaml:"mobiles"`
	MobResets []MobReset    `yaml:"mob_resets"`
}

// Global storage for rooms, initialized as an empty map
var rooms = make(map[int]*Room)

// LoadAreas loads all YAML files from the "areas" folder.
func LoadAreas() error {
	areaDir := "areas" // Directory containing area YAML files

	// Read the directory to get list of files.
	files, err := os.ReadDir(areaDir)
	if err != nil {
		// Return an error if the directory cannot be read.
		return fmt.Errorf("failed to read areas directory: %v", err)
	}

	// Iterate over each file in the directory.
	for _, file := range files {
		// Check if the file has a .yml extension.
		if strings.HasSuffix(file.Name(), ".yml") {
			// Generate the full path to the area file.
			areaPath := filepath.Join(areaDir, file.Name())
			// Load the area from the file and log any errors.
			if err := loadArea(areaPath); err != nil {
				log.Printf("Error loading area: %v", err)
			}
		}
	}
	return nil // Return nil indicating success in loading areas.
}

// LoadArea loads a single area file
func loadArea(path string) error {
	areaName := filepath.Base(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var area Area
	err = yaml.Unmarshal(data, &area)
	if err != nil {
		return err
	}

	// Set the area name and ID for each room
	for id, room := range area.Rooms {
		room.ID = id
		room.Area = areaName

		// Set default closed state for doors
		for _, exit := range room.Exits {
			if exit.Door != nil && !exit.Door.Closed {
				exit.Door.Closed = true // Default to closed if door exists
			}
		}

		rooms[id] = room
		//fmt.Printf("Loaded Room [%d]: %s (Area: %s)\n", id, room.Name, room.Area)
	}

	// After all rooms are loaded, ensure door consistency between connected rooms
	for id, room := range area.Rooms {
		for direction, exit := range room.Exits {
			if exit.Door != nil {
				// Get the destination room
				var destRoomID int
				switch exitID := exit.ID.(type) {
				case int:
					destRoomID = exitID
				case string:
					// Handle cross-area references
					roomInfo := strings.Split(exitID, ":")
					if len(roomInfo) != 2 {
						continue
					}
					var err error
					destRoomID, err = strconv.Atoi(roomInfo[1])
					if err != nil {
						continue
					}
				default:
					continue
				}

				// Get the destination room
				destRoom, exists := rooms[destRoomID]
				if !exists {
					continue // Destination room not loaded yet
				}

				// Find the opposite direction
				oppositeDirection := GetOppositeDirection(direction)

				// Check if the destination room has a corresponding exit
				destExit, exists := destRoom.Exits[oppositeDirection]
				if !exists {
					// Create a corresponding exit with a door
					log.Printf("[WARNING] Room %d has a door to %d, but %d has no exit back. Adding reciprocal exit.",
						id, destRoomID, destRoomID)
					destRoom.Exits[oppositeDirection] = &Exit{
						ID:          id,
						Description: fmt.Sprintf("You see %s.", room.Name),
						Door: &Door{
							ShortDescription: exit.Door.ShortDescription,
							Keywords:         exit.Door.Keywords,
							Locked:           exit.Door.Locked,
							Closed:           exit.Door.Closed,
						},
					}
				} else if destExit.Door == nil {
					// Add a door to the destination exit
					log.Printf("[WARNING] Room %d has a door to %d, but %d has no door back. Adding reciprocal door.",
						id, destRoomID, destRoomID)
					destExit.Door = &Door{
						ShortDescription: exit.Door.ShortDescription,
						Keywords:         exit.Door.Keywords,
						Locked:           exit.Door.Locked,
						Closed:           exit.Door.Closed,
					}
				} else {
					// Ensure door states are synchronized
					destExit.Door.Closed = exit.Door.Closed
					destExit.Door.Locked = exit.Door.Locked
				}
			}
		}
	}

	// Load mobs from the mobiles section
	for id, mob := range area.Mobiles {
		//fmt.Printf("Loading mob [%d]: %s\nLong Description: %s\n", id, mob.ShortDescription, mob.LongDescription)
		mob.ID = id
		RegisterMob(mob)
	}

	// Store mob resets
	mobResets = append(mobResets, area.MobResets...)

	return nil
}

// GetRoom fetches a room by its ID
func GetRoom(id int) (*Room, error) {
	room, exists := rooms[id]
	if !exists {
		return nil, fmt.Errorf("room ID %d not found", id)
	}

	// Debug logging
	//fmt.Printf("Getting Room [%d]: %s (Area: %s)\n", id, room.Name, room.Area)
	return room, nil
}
