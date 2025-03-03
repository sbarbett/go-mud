package main

import (
	"bufio"
	"net"
	"strings"
)

// CreateNewCharacter function prompts the player to select a race and a class for their new character.
// It takes a network connection, a buffered reader for input, and the character name as arguments.
// It returns a pointer to a Player struct and an error if any occurs during character creation.
func CreateNewCharacter(conn net.Conn, reader *bufio.Reader, name string) (*Player, error) {
	// Define available races
	races := []string{"Human", "Elf", "Dwarf", "Orc"}
	// Send message to the client to choose a race
	conn.Write([]byte("Choose your race: [Human, Elf, Dwarf, Orc]\r\n"))
	var race string
	// Loop until a valid race is chosen
	for {
		// Prompt for input
		conn.Write([]byte("> "))
		// Read the input from the player
		raceInput, _ := reader.ReadString('\n')
		// Trim whitespace and capitalize the first letter of the input
		race = strings.TrimSpace(strings.Title(raceInput))

		// Check if the chosen race is valid
		if stringInSlice(race, races) {
			break // Exit loop if valid race
		}
		// Inform the user if the input was invalid
		conn.Write([]byte("Invalid race. Choose from: Human, Elf, Dwarf, Orc.\r\n"))
	}

	// Define available classes
	classes := []string{"Warrior", "Mage", "Thief", "Cleric"}
	// Send message to the client to choose a class
	conn.Write([]byte("Choose your class: [Warrior, Mage, Thief, Cleric]\r\n"))
	var class string
	// Loop until a valid class is selected
	for {
		// Prompt for input
		conn.Write([]byte("> "))
		// Read the input from the player
		classInput, _ := reader.ReadString('\n')
		// Trim whitespace and capitalize the first letter of the input
		class = strings.TrimSpace(strings.Title(classInput))

		// Check if the chosen class is valid
		if stringInSlice(class, classes) {
			break // Exit loop if valid class
		}
		// Inform the user if the input was invalid
		conn.Write([]byte("Invalid class. Choose from: Warrior, Mage, Thief, Cleric.\r\n"))
	}

	// Attempt to save the new character to the database
	err := CreatePlayer(name, race, class)
	if err != nil {
		// Return nil and the error if the player could not be created
		return nil, err
	}

	// Load the initial room for the player, starting in room 3700 (Mud School)
	room, err := GetRoom(3700)
	if err != nil {
		// Return nil and the error if the room could not be retrieved
		return nil, err
	}

	// Return a new Player struct with the player's information
	return &Player{Name: name, Race: race, Class: class, Room: room, Conn: conn}, nil
}
