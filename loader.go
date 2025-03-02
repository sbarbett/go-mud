package main

import (
	"fmt"
	"io/ioutil"     // Package for reading files
	"log"           // Package for logging errors
	"path/filepath" // Package for manipulating filename paths
	"strconv"       // Package for converting strings to numeric types
	"strings"       // Package for string manipulation

	"gopkg.in/yaml.v3" // Package for parsing YAML files
)

// Area represents a collection of rooms.
type Area struct {
	Name  string        `yaml:"name"`  // Name of the area as defined in YAML
	Rooms map[int]*Room `yaml:"rooms"` // A map of room IDs to Room pointers
}

// Global storage for rooms, initialized as an empty map.
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

// Load a single area file and populate the global rooms map.
func loadArea(filePath string) error {
	// Read the YAML file.
	yamlFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		// Return an error if reading the file fails.
		return fmt.Errorf("failed to read area file %s: %v", filePath, err)
	}

	// Declare an Area variable to hold the parsed data.
	var area Area
	// Unmarshal the YAML data into the area struct.
	err = yaml.Unmarshal(yamlFile, &area)
	if err != nil {
		// Return an error if YAML parsing fails.
		return fmt.Errorf("failed to parse YAML file %s: %v", filePath, err)
	}

	// Iterate over each room in the area.
	for id, room := range area.Rooms {
		cleanedExits := make(map[string]interface{}) // Map to store validated exits of the room
		// Process each exit direction associated with the room.
		for direction, rawExit := range room.Exits {
			// Determine the type of the exit value.
			switch v := rawExit.(type) {
			case int:
				// If exit is an integer, store it directly.
				cleanedExits[direction] = v
			case string:
				// If exit is a string, attempt to convert it to an integer.
				if num, err := strconv.Atoi(v); err == nil {
					cleanedExits[direction] = num // Store the converted integer.
				} else {
					cleanedExits[direction] = v // Store the string if conversion fails.
				}
			}
		}
		// Update the room's Exits field with cleaned exit values.
		room.Exits = cleanedExits
		// Store the room in the global rooms map using its ID.
		rooms[id] = room
		// Log the successful loading of the room.
		fmt.Printf("Loaded Room [%d]: %s\n", id, room.Name)
	}
	return nil // Return nil indicating successful loading of the area.
}

// GetRoom fetches a room by its ID.
func GetRoom(id int) (*Room, error) {
	// Check if the room exists in the global rooms map.
	room, exists := rooms[id]
	if !exists {
		// Return an error if the room is not found.
		return nil, fmt.Errorf("room ID %d not found", id)
	}
	// Return the found room and a nil error.
	return room, nil
}
