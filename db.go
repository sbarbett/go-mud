package main

import (
	"database/sql" // Import the database/sql package to enable SQL database operations
	"log"          // Import log package for logging error messages

	_ "modernc.org/sqlite" // Import the SQLite driver for database connections
)

// Global variable to hold the database connection
var db *sql.DB

// InitDB initializes the database connection and creates the players table if it doesn't exist
func InitDB() {
	var err error
	// Open a connection to the SQLite database located at ./mud.db
	db, err = sql.Open("sqlite", "./mud.db")
	if err != nil {
		// Log a fatal error if the database connection fails
		log.Fatal("Failed to connect to database:", err)
	}

	// Execute a SQL command to create the players table if it does not currently exist
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS players (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		race TEXT NOT NULL,
		class TEXT NOT NULL,
		room_id INTEGER NOT NULL DEFAULT 1
	);
	`)
	if err != nil {
		// Log a fatal error if creating the players table fails
		log.Fatal("Failed to create players table:", err)
	}
}

// CreatePlayer adds a new player to the players table in the database
func CreatePlayer(name, race, class string) error {
	// Insert the new player's details into the players table
	_, err := db.Exec("INSERT INTO players (name, race, class, room_id) VALUES (?, ?, ?, ?)", name, race, class, 1)
	return err // Return any error encountered during the process
}

// PlayerExists checks if a player with the given name exists in the database
func PlayerExists(name string) bool {
	var exists bool
	// Query the database to check for the existence of the player by name
	err := db.QueryRow("SELECT EXISTS (SELECT 1 FROM players WHERE name = ?)", name).Scan(&exists)
	// Return true if no error occurred and the player exists, otherwise return false
	return err == nil && exists
}

// LoadPlayer retrieves a player's race, class, and room ID from the database using their name
func LoadPlayer(name string) (string, string, int, error) {
	var race, class string
	var roomID int
	// Execute a query to get the race, class, and room_id for the specified player name
	err := db.QueryRow("SELECT race, class, room_id FROM players WHERE name = ?", name).Scan(&race, &class, &roomID)
	if err != nil {
		// Return an error if the query fails, along with empty strings and 0 for the roomID
		return "", "", 0, err
	}
	// Return the retrieved details: race, class, and roomID
	return race, class, roomID, nil
}

// UpdatePlayerRoom updates the room ID for a player, moving them to a new room
func UpdatePlayerRoom(playerName string, roomID int) error {
	// Execute an update query to change the player's room_id in the players table
	_, err := db.Exec("UPDATE players SET room_id = ? WHERE name = ?", roomID, playerName)
	return err // Return any error encountered during the process
}
