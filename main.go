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

// handleConnection manages player login and the overall lifecycle of the player's session
func handleConnection(conn net.Conn) {
	defer conn.Close()              // Ensure the connection is closed when the function exits
	reader := bufio.NewReader(conn) // Create a buffered reader for reading from the connection

	// Prompt the player to enter their character name
	conn.Write([]byte("Enter your character name: "))
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

		// Notify the player of the successful character creation
		conn.Write([]byte(fmt.Sprintf("Character created! Welcome, %s the %s %s!\r\n", player.Name, player.Race, player.Class)))

		// After successful player creation or loading, use AddPlayer
		AddPlayer(player)

		// Broadcast player join
		oocManager.BroadcastMessage(fmt.Sprintf("[OOC] %s has connected.", player.Name), player)

		// Send initial room description to the player
		player.Conn.Write([]byte(fmt.Sprintf("%s\r\n", DescribeRoom(player.Room, player))))

		// Calculate derived stats for loaded player
		player.UpdateDerivedStats()

		playGame(player, reader) // Start the game for the newly created player

		// When player disconnects, use RemovePlayer
		RemovePlayer(player)
		oocManager.BroadcastMessage(fmt.Sprintf("[OOC] %s has disconnected.", player.Name), player)
		return
	}
	// Player already exists; load their existing information from the database
	race, class, roomID, str, dex, con, int_, wis, pre, level, xp, nextLevelXP, hp, maxHP, mp, maxMP, stamina, maxStamina, gold, err := LoadPlayer(name)
	if err != nil {
		conn.Write([]byte("Error loading character.\r\n")) // Handle loading errors
		return
	}

	// Fetch the room associated with the loaded player
	room, err := GetRoom(roomID)
	if err != nil {
		conn.Write([]byte("Error loading game world.\r\n")) // Handle room loading errors
		return
	}

	// Initialize the player object with loaded data
	player := &Player{
		Name:        name,
		Race:        race,
		Class:       class,
		Room:        room,
		Conn:        conn,
		STR:         str,
		DEX:         dex,
		CON:         con,
		INT:         int_,
		WIS:         wis,
		PRE:         pre,
		Level:       level,
		XP:          xp,
		NextLevelXP: nextLevelXP,
		HP:          hp,
		MaxHP:       maxHP,
		MP:          mp,
		MaxMP:       maxMP,
		Stamina:     stamina,
		MaxStamina:  maxStamina,
		Gold:        gold,
	}

	// Calculate derived stats for loaded player
	player.UpdateDerivedStats()

	// Welcome the player back
	conn.Write([]byte(fmt.Sprintf("Welcome back, %s!\r\n", name)))

	// After successful player creation or loading, use AddPlayer
	AddPlayer(player)

	// Broadcast player join
	oocManager.BroadcastMessage(fmt.Sprintf("[OOC] %s has connected.", player.Name), player)

	// Send initial room description to the player
	player.Conn.Write([]byte(fmt.Sprintf("%s\r\n", DescribeRoom(player.Room, player))))

	playGame(player, reader) // Start the game for the existing player

	// When player disconnects, use RemovePlayer
	RemovePlayer(player)
	oocManager.BroadcastMessage(fmt.Sprintf("[OOC] %s has disconnected.", player.Name), player)
}

// playGame handles gameplay commands and maintains the player's session state
func playGame(player *Player, reader *bufio.Reader) {
	for {
		player.Conn.Write([]byte("> "))
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading from connection: %v", err)
			return
		}

		input = strings.TrimSpace(strings.ToLower(input))
		if input == "" {
			continue
		}

		// Store the original command for movement handling
		player.LastCommand = input

		// Process the command
		response := HandleCommand(player, input)

		// Send the response to the player
		if response != "" {
			player.Conn.Write([]byte(response))
		}

		// Check if the player is quitting
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

	// Initialize the random number generator with a time-based seed
	rand.Seed(time.Now().UnixNano())

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

	// Register player pulse updates - ensure this is properly registered
	timeManager.RegisterPulseFunc(func() {
		// Log that the pulse is running for debugging
		log.Printf("[DEBUG] Processing pulse update for %d active players", len(activePlayers))

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

	// Start the time manager
	timeManager.Start()

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
