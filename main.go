/*
 * main.go
 *
 * This file contains the main entry point and core server functionality for the MUD.
 * It handles server initialization, connection management, player authentication,
 * and the main game loop. The file implements functions for handling new connections,
 * processing player input, managing the game state, and gracefully shutting down
 * the server when needed.
 */

package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Global variables
var oocManager *OOCManager
var timeManager *TimeManager

// Global random number generator
var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// handleConnection manages player login and the overall lifecycle of the player's session
func handleConnection(conn net.Conn) {
	defer conn.Close()              // Ensure the connection is closed when the function exits
	reader := bufio.NewReader(conn) // Create a buffered reader for reading from the connection

	// First, ask about ANSI color before showing any colored content
	conn.Write([]byte("Would you like to enable ANSI colors? (yes/no): "))

	colorResponse, _ := reader.ReadString('\n')
	colorResponse = strings.TrimSpace(strings.ToLower(colorResponse))
	colorEnabled := colorResponse != "no" // Enable colors unless explicitly declined

	// Now display the splash screen with or without colors
	if colorEnabled {
		conn.Write([]byte("\x1b[1;36m" +
			"  ▄████  ▒█████      ███▄ ▄███▓  █    ██ ▓█████▄ \r\n" +
			"  ██▒ ▀█▒▒██▒  ██▒   ▓██▒▀█▀ ██▒ ██  ▓██▒▒██▀ ██▌\r\n" +
			" ▒██░▄▄▄░▒██░  ██▒   ▓██    ▓██░▓██  ▒██░░██   █▌\r\n" +
			" ░▓█  ██▓▒██   ██░   ▒██    ▒██ ▓▓█  ░██░░▓█▄   ▌\r\n" +
			" ░▒▓███▀▒░ ████▓▒░   ▒██▒   ░██▒▒▒█████▓ ░▒████▓ \r\n" +
			"  ░▒   ▒ ░ ▒░▒░▒░    ░ ▒░   ░  ░░▒▓▒ ▒ ▒  ▒▒▓  ▒ \r\n" +
			"   ░   ░   ░ ▒ ▒░    ░  ░      ░░░▒░ ░ ░  ░ ▒  ▒ \r\n" +
			" ░ ░   ░ ░ ░ ░ ▒     ░      ░    ░░░ ░ ░  ░ ░  ░ \r\n" +
			"       ░     ░ ░            ░      ░        ░    \r\n" +
			"                                           ░      \x1b[0m\r\n\r\n" +
			"\x1b[1;32m  Welcome to Go-MUD!\x1b[0m\r\n" +
			"\x1b[0;36m  A text-based multiplayer adventure\x1b[0m\r\n\r\n" +
			"\x1b[0;33m  Created with ❤️ by shanevapid\x1b[0m\r\n\r\n"))
	} else {
		conn.Write([]byte("\r\n" +
			"  GO-MUD\r\n\r\n" +
			"  Welcome to Go-MUD!\r\n" +
			"  A text-based multiplayer adventure\r\n\r\n" +
			"  Created with <3 by shanevapid\r\n\r\n"))
	}

	// Prompt the player to enter their character name
	if colorEnabled {
		conn.Write([]byte("\x1b[1;37mWhat's your name, traveler? \x1b[0m"))
	} else {
		conn.Write([]byte("What's your name, traveler? "))
	}

	name, _ := reader.ReadString('\n') // Read name input from the player
	name = strings.TrimSpace(name)     // Remove any surrounding whitespace

	// Check if the player already exists in the system
	if !PlayerExists(name) {
		// If the player does not exist, prompt to create a new character
		conn.Write([]byte("Character not found. Would you like to create a new character? (yes/no) "))
		response, _ := reader.ReadString('\n')                  // Read the player's response
		response = strings.TrimSpace(strings.ToLower(response)) // Normalize the response to lowercase

		if response != "yes" { // If the response is not "yes"
			conn.Write([]byte("Goodbye!\r\n")) // Bid goodbye and exit
			return
		}

		// Create a new character for the player
		player, err := CreateNewCharacter(conn, reader, name)
		if err != nil {
			conn.Write([]byte("Error creating character. Please try again.\r\n")) // Handle creation errors
			return
		}

		// Set the color preference from the initial prompt
		player.ColorEnabled = colorEnabled

		// Update the player's color preference in the database
		err = UpdatePlayerColorPreference(name, colorEnabled)
		if err != nil {
			// Just log the error, don't fail character creation
			log.Printf("Error saving color preference: %v\n", err)
		}

		// Notify the player of the successful character creation
		player.Send(fmt.Sprintf("Character created! Welcome, %s the %s %s!", player.Name, player.Race, player.Class))

		// After successful player creation or loading, use AddPlayer
		AddPlayer(player)

		// Broadcast player join
		oocManager.BroadcastMessage(fmt.Sprintf("[OOC] %s has connected.", player.Name), player)

		// Send initial room description to the player
		player.Send(DescribeRoom(player.Room, player))

		// Calculate derived stats for loaded player
		player.UpdateDerivedStats()

		playGame(player, reader) // Start the game for the newly created player

		// When player disconnects, use RemovePlayer
		RemovePlayer(player)
		oocManager.BroadcastMessage(fmt.Sprintf("[OOC] %s has disconnected.", player.Name), player)
		return
	}
	// Player already exists; load their existing information from the database
	race, class, title, roomID, str, dex, con, int_, wis, pre, level, xp, nextLevelXP, hp, maxHP, mp, maxMP, stamina, maxStamina, gold, dbColorEnabled, err := LoadPlayer(name)
	if err != nil {
		log.Printf("Error loading player %s: %v", name, err)
		conn.Write([]byte("Error loading character.\r\n")) // Handle loading errors
		return
	}

	// Fetch the room associated with the loaded player
	room, err := GetRoom(roomID)
	if err != nil {
		log.Printf("Error getting room %d for player %s: %v", roomID, name, err)
		conn.Write([]byte("Error loading game world.\r\n")) // Handle room loading errors
		return
	}

	// Create a new player with the loaded information
	player := &Player{
		Name:         name,
		Race:         race,
		Class:        class,
		Title:        title,
		STR:          str,
		DEX:          dex,
		CON:          con,
		INT:          int_,
		WIS:          wis,
		PRE:          pre,
		Level:        level,
		XP:           xp,
		NextLevelXP:  nextLevelXP,
		HP:           hp,
		MaxHP:        maxHP,
		MP:           mp,
		MaxMP:        maxMP,
		Stamina:      stamina,
		MaxStamina:   maxStamina,
		Gold:         gold,
		Room:         room,
		Conn:         conn,
		ColorEnabled: dbColorEnabled,
	}

	// Update the player's color preference in the database if it's different from the stored value
	if colorEnabled != dbColorEnabled {
		err = UpdatePlayerColorPreference(name, colorEnabled)
		if err != nil {
			log.Printf("Error updating color preference: %v\n", err)
		}
	}

	// Welcome the player back
	player.Send(fmt.Sprintf("Welcome back, %s!", player.Name))

	// After successful player creation or loading, use AddPlayer
	AddPlayer(player)

	// Broadcast player join
	oocManager.BroadcastMessage(fmt.Sprintf("[OOC] %s has connected.", player.Name), player)

	// Send initial room description to the player
	player.Send(DescribeRoom(player.Room, player))

	// Calculate derived stats for loaded player
	player.UpdateDerivedStats()

	playGame(player, reader) // Start the game for the loaded player

	// When player disconnects, use RemovePlayer
	RemovePlayer(player)
	oocManager.BroadcastMessage(fmt.Sprintf("[OOC] %s has disconnected.", player.Name), player)
}

// playGame handles the main game loop for a player
func playGame(player *Player, reader *bufio.Reader) {
	// Display initial prompt
	displayPrompt(player)

	for {
		// Read input from the player
		input, err := reader.ReadString('\n')
		if err != nil {
			// Handle connection errors
			log.Printf("Error reading from connection: %v", err)
			return
		}

		// Process the input
		input = strings.TrimSpace(input)
		if input == "" {
			// Display prompt again if empty input
			displayPrompt(player)
			continue
		}

		// Store the last command for reference (needed for movement)
		player.LastCommand = input

		// Handle the command and get the response
		response := HandleCommand(player, input)

		// Send the response back to the player
		if response != "" {
			player.Send(response)
		}

		// Always display the prompt after a command
		displayPrompt(player)

		// Check if the player wants to quit
		if input == "quit" {
			return
		}
	}
}

// setupSignalHandler sets up a signal handler for graceful shutdown
func setupSignalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("Shutting down server...")

		// Stop the time manager
		if timeManager != nil {
			timeManager.Stop()
		}

		// Close database connection
		if db != nil {
			db.Close()
		}

		fmt.Println("Server shutdown complete")
		os.Exit(0)
	}()
}

// main initializes the MUD server and starts listening for connections
func main() {
	// Setup signal handler for graceful shutdown
	setupSignalHandler()

	// No need to seed the global rand anymore as we're using our own rng instance
	// rand.Seed(time.Now().UnixNano())

	// Initialize the database
	InitDB()

	// Initialize OOC manager with the player mutex and active players map
	oocManager = NewOOCManager(&playersMutex, activePlayers)

	// Initialize and start the time manager
	timeManager = NewTimeManager()

	// Register debug functions if in debug mode
	// Uncomment these for debugging
	// timeManager.RegisterTickFunc(DebugTick)
	// timeManager.RegisterPulseFunc(DebugPulse)
	// timeManager.RegisterHeartbeatFunc(DebugHeartbeat)

	// Register player regeneration on tick
	timeManager.RegisterTickFunc(func() {
		playersMutex.Lock()
		defer playersMutex.Unlock()

		for _, player := range activePlayers {
			player.RegenTick()
		}
	})

	// Register periodic mob resets (every 5 minutes)
	tickCounter := 0
	timeManager.RegisterTickFunc(func() {
		tickCounter++

		// Process mob resets every 5 minutes
		if tickCounter >= 5 {
			tickCounter = 0
			ProcessMobResets()
		}
	})

	// Register player pulse updates - ensure this is properly registered
	timeManager.RegisterPulseFunc(func() {
		// Log that the pulse is running for debugging
		//log.Printf("[DEBUG] Processing pulse update for %d active players", len(activePlayers))

		// Make a copy of the players to avoid holding the lock while processing
		var playersToUpdate []*Player

		playersMutex.Lock()
		for _, player := range activePlayers {
			playersToUpdate = append(playersToUpdate, player)
		}
		playersMutex.Unlock()

		// Now process each player without holding the global lock
		for _, player := range playersToUpdate {
			// Process each player in its own goroutine to avoid blocking
			go func(p *Player) {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[ERROR] Panic in player pulse update for %s: %v", p.Name, r)
					}
				}()
				p.PulseUpdate()
			}(player)
		}
	})

	// Register mob wandering behavior
	timeManager.RegisterPulseFunc(ProcessMobWandering)

	// Start the time manager
	timeManager.Start()

	// Initialize the help system
	fmt.Println("Initializing help system...")
	InitHelpSystem()

	// Load all areas from YAML
	fmt.Println("Loading areas...")
	if err := LoadAreas(); err != nil {
		log.Fatalf("Error loading areas: %v", err)
	}

	// Process mob resets after loading areas
	ProcessMobResets()

	// Start the MUD server
	listener, err := net.Listen("tcp", "0.0.0.0:4000")
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer listener.Close()

	fmt.Println("MUD server listening on port 4000...")

	// Accept and handle incoming connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go handleConnection(conn)
	}
}

// displayPrompt shows the player's current stats (HP, MP, Stamina) as a prompt
func displayPrompt(player *Player) {
	// Format: [HP: 100/100 | MP: 100/100 | ST: 100/100]>
	prompt := fmt.Sprintf("[HP: %d/%d | MP: %d/%d | ST: %d/%d]> ",
		player.HP, player.MaxHP,
		player.MP, player.MaxMP,
		player.Stamina, player.MaxStamina)

	// Apply color to the prompt based on health percentage
	healthPercent := float64(player.HP) / float64(player.MaxHP)

	if player.ColorEnabled {
		var colorCode string
		if healthPercent < 0.3 {
			// Red for low health
			colorCode = "{R}"
		} else if healthPercent < 0.6 {
			// Yellow for medium health
			colorCode = "{Y}"
		} else {
			// Green for good health
			colorCode = "{G}"
		}

		// Send the colored prompt directly to avoid double newlines
		coloredPrompt := ProcessColors(colorCode+prompt+"{x}", player.ColorEnabled)
		player.Conn.Write([]byte(coloredPrompt))
	} else {
		// Send the plain prompt directly to avoid double newlines
		player.Conn.Write([]byte(prompt))
	}
}
