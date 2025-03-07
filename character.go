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
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("connection error during race selection: %v", err)
		}

		choice := strings.TrimSpace(input)
		if choice == "" {
			return nil, fmt.Errorf("connection closed during race selection")
		}

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
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("connection error during class selection: %v", err)
		}

		choice := strings.TrimSpace(input)
		if choice == "" {
			return nil, fmt.Errorf("connection closed during class selection")
		}

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
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("connection error during stat allocation: %v", err)
		}

		input = strings.TrimSpace(strings.ToUpper(input))
		if input == "" {
			return nil, fmt.Errorf("connection closed during stat allocation")
		}

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
	var err error
	err = CreatePlayer(name, race, class, stats)
	if err != nil {
		return nil, err
	}

	// Load the initial room
	room, err := GetRoom(3700)
	if err != nil {
		return nil, err
	}

	// Create and return the player object
	player := &Player{
		Name:         name,
		Race:         race,
		Class:        class,
		Room:         room,
		Conn:         conn,
		STR:          stats["STR"],
		DEX:          stats["DEX"],
		CON:          stats["CON"],
		INT:          stats["INT"],
		WIS:          stats["WIS"],
		PRE:          stats["PRE"],
		Level:        1,
		Stamina:      100,
		MaxStamina:   100,
		Gold:         0,    // Start with 0 gold
		ColorEnabled: true, // Default to colors enabled, will be overridden by the connection prompt
	}

	// Calculate derived stats based on class and base stats
	switch class {
	case "Warrior":
		player.MaxHP = 20 + (player.CON * 2)
		player.MaxMP = 10 + player.WIS
	case "Mage":
		player.MaxHP = 15 + player.CON
		player.MaxMP = 20 + (player.INT * 2)
	case "Rogue":
		player.MaxHP = 18 + (player.CON+player.DEX)/2
		player.MaxMP = 15 + player.INT
	case "Cleric":
		player.MaxHP = 18 + (player.CON+player.WIS)/2
		player.MaxMP = 18 + (player.WIS+player.INT)/2
	}

	// Set current HP/MP to maximum
	player.HP = player.MaxHP
	player.MP = player.MaxMP

	// Set initial XP thresholds
	player.XP = 0
	player.NextLevelXP = 1000 // Base XP needed for level 2

	// Calculate initial derived stats
	player.UpdateDerivedStats()

	return player, nil
}
