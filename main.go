package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

// Global variables
var oocManager *OOCManager

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

		playGame(player, reader) // Start the game for the newly created player

		// When player disconnects, use RemovePlayer
		RemovePlayer(player)
		oocManager.BroadcastMessage(fmt.Sprintf("[OOC] %s has disconnected.", player.Name), player)
		return
	}

	// Player already exists; load their existing information from the database
	race, class, roomID, err := LoadPlayer(name)
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
	player := &Player{Name: name, Race: race, Class: class, Room: room, Conn: conn}

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
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "ooc" || strings.HasPrefix(input, "ooc ") {
			oocManager.HandleOOCCommand(player, input)
			continue
		}

		switch input {
		case "quit":
			player.Conn.Write([]byte("Goodbye!\r\n"))
			return

		case "look":
			player.Conn.Write([]byte(fmt.Sprintf("%s\n", DescribeRoom(player.Room, player))))

		// Handle movement commands
		case "north", "south", "east", "west", "up", "down",
			"n", "s", "e", "w", "u", "d":
			if err := HandleMovement(player, input); err != nil {
				player.Conn.Write([]byte(err.Error() + "\n"))
			}

		default:
			player.Conn.Write([]byte("Unknown command.\r\n"))
		}
	}
}

// main initializes the MUD server and starts listening for connections
func main() {
	// Initialize the database
	InitDB()

	// Initialize OOC manager with the player mutex and active players map
	oocManager = NewOOCManager(&playersMutex, activePlayers)

	// Load all areas from YAML
	fmt.Println("Loading areas...")
	if err := LoadAreas(); err != nil {
		log.Fatalf("Error loading areas: %v", err)
	}

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
