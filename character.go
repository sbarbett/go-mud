package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

// Character creation and customization functions
func CreateNewCharacter(conn net.Conn, reader *bufio.Reader, name string) (*Player, error) {
	// Present race options
	races := []string{"Human", "Elf", "Dwarf", "Orc"}
	conn.Write([]byte("\nChoose your race:\n"))
	for i, race := range races {
		conn.Write([]byte(fmt.Sprintf("%d. %s\n", i+1, race)))
	}

	// Get race selection
	var race string
	for {
		conn.Write([]byte("Enter your choice (1-4): "))
		input, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(input)
		if num := choice[0] - '0'; num >= 1 && num <= 4 {
			race = races[num-1]
			break
		}
		conn.Write([]byte("Invalid choice. Please try again.\n"))
	}

	// Present class options
	classes := []string{"Warrior", "Mage", "Rogue", "Cleric"}
	conn.Write([]byte("\nChoose your class:\n"))
	for i, class := range classes {
		conn.Write([]byte(fmt.Sprintf("%d. %s\n", i+1, class)))
	}

	// Get class selection
	var class string
	for {
		conn.Write([]byte("Enter your choice (1-4): "))
		input, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(input)
		if num := choice[0] - '0'; num >= 1 && num <= 4 {
			class = classes[num-1]
			break
		}
		conn.Write([]byte("Invalid choice. Please try again.\n"))
	}

	// Get base stats for the selected race
	stats := GetBaseStats(race)

	// Handle bonus point allocation
	remainingPoints := BONUS_POINTS
	conn.Write([]byte(fmt.Sprintf("\nYou have %d bonus points to allocate to your stats.\n", remainingPoints)))
	conn.Write([]byte("Current stats based on your race:\n"))

	statNames := []string{"STR", "DEX", "CON", "INT", "WIS", "PRE"}
	for _, stat := range statNames {
		conn.Write([]byte(fmt.Sprintf("%s: %d\n", stat, stats[stat])))
	}

	// Allocate bonus points
	for remainingPoints > 0 {
		conn.Write([]byte(fmt.Sprintf("\nRemaining points: %d\n", remainingPoints)))
		conn.Write([]byte("Enter stat to increase (STR/DEX/CON/INT/WIS/PRE) or 'done' to finish: "))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToUpper(input))

		if input == "DONE" {
			break
		}

		if _, exists := stats[input]; !exists {
			conn.Write([]byte("Invalid stat. Please try again.\n"))
			continue
		}

		if !ValidateStat(stats[input] + 1) {
			conn.Write([]byte("Cannot increase stat above 18.\n"))
			continue
		}

		stats[input]++
		remainingPoints--
	}

	// Create the character in the database
	err := CreatePlayer(name, race, class, stats)
	if err != nil {
		return nil, err
	}

	// Load the initial room
	room, err := GetRoom(3700)
	if err != nil {
		return nil, err
	}

	// Create and return the player object
	return &Player{
		Name:  name,
		Race:  race,
		Class: class,
		Room:  room,
		Conn:  conn,
		STR:   stats["STR"],
		DEX:   stats["DEX"],
		CON:   stats["CON"],
		INT:   stats["INT"],
		WIS:   stats["WIS"],
		PRE:   stats["PRE"],
	}, nil
}
