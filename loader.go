package main

import (
	"fmt"
	"io/ioutil"     // Package for reading files
	"log"           // Package for logging errors
	"path/filepath" // Package for manipulating filename paths

	// Package for converting strings to numeric types
	"strings" // Package for string manipulation

	"gopkg.in/yaml.v3" // Package for parsing YAML files
)

// Exit represents a direction-specific exit from a room
type Exit struct {
	ID          interface{} `yaml:"id"`          // Can be int or string (for cross-area references)
	Description string      `yaml:"description"` // Optional description of what's visible in that direction
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
}

// Area represents a collection of rooms
type Area struct {
	Name  string        `yaml:"name"`  // Name of the area as defined in YAML
	Rooms map[int]*Room `yaml:"rooms"` // A map of room IDs to Room pointers
}

// Global storage for rooms, initialized as an empty map
var rooms = make(map[int]*Room)

// LoadAreas loads all YAML files from the "areas" folder.
func LoadAreas() error {
	areaDir := "areas" // Directory containing area YAML files

	// Read the directory to get list of files.
	files, err := ioutil.ReadDir(areaDir)
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
	data, err := ioutil.ReadFile(path)
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
		room.ID = id         // Explicitly set the room ID
		room.Area = areaName // Set the area name
		rooms[id] = room     // Store in global rooms map
		fmt.Printf("Loaded Room [%d]: %s (Area: %s)\n", id, room.Name, room.Area)
	}

	return nil
}

// GetRoom fetches a room by its ID
func GetRoom(id int) (*Room, error) {
	room, exists := rooms[id]
	if !exists {
		return nil, fmt.Errorf("room ID %d not found", id)
	}

	// Debug logging
	fmt.Printf("Getting Room [%d]: %s (Area: %s)\n", id, room.Name, room.Area)
	return room, nil
}
