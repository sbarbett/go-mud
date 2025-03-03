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
		room_id INTEGER NOT NULL DEFAULT 1,
		str INTEGER NOT NULL DEFAULT 10,
		dex INTEGER NOT NULL DEFAULT 10,
		con INTEGER NOT NULL DEFAULT 10,
		int INTEGER NOT NULL DEFAULT 10,
		wis INTEGER NOT NULL DEFAULT 10,
		pre INTEGER NOT NULL DEFAULT 10
	);
	`)
	if err != nil {
		// Log a fatal error if creating the players table fails
		log.Fatal("Failed to create players table:", err)
	}

	// Add new columns to the players table
	_, err = db.Exec(`
	ALTER TABLE players 
	ADD COLUMN level INTEGER NOT NULL DEFAULT 1;
	`)
	if err != nil {
		// Ignore error if column already exists
		log.Printf("Note: level column may already exist: %v", err)
	}

	_, err = db.Exec(`
	ALTER TABLE players 
	ADD COLUMN xp INTEGER NOT NULL DEFAULT 0;
	`)
	if err != nil {
		log.Printf("Note: xp column may already exist: %v", err)
	}

	_, err = db.Exec(`
	ALTER TABLE players 
	ADD COLUMN next_level_xp INTEGER NOT NULL DEFAULT 1000;
	`)
	if err != nil {
		log.Printf("Note: next_level_xp column may already exist: %v", err)
	}

	// Add HP and MP columns
	_, err = db.Exec(`
	ALTER TABLE players 
	ADD COLUMN hp INTEGER NOT NULL DEFAULT 100;
	`)
	if err != nil {
		log.Printf("Note: hp column may already exist: %v", err)
	}

	_, err = db.Exec(`
	ALTER TABLE players 
	ADD COLUMN max_hp INTEGER NOT NULL DEFAULT 100;
	`)
	if err != nil {
		log.Printf("Note: max_hp column may already exist: %v", err)
	}

	_, err = db.Exec(`
	ALTER TABLE players 
	ADD COLUMN mp INTEGER NOT NULL DEFAULT 100;
	`)
	if err != nil {
		log.Printf("Note: mp column may already exist: %v", err)
	}

	_, err = db.Exec(`
	ALTER TABLE players 
	ADD COLUMN max_mp INTEGER NOT NULL DEFAULT 100;
	`)
	if err != nil {
		log.Printf("Note: max_mp column may already exist: %v", err)
	}
}

// CreatePlayer adds a new player to the database with their stats
func CreatePlayer(name, race, class string, stats map[string]int) error {
	_, err := db.Exec(`
		INSERT INTO players (
			name, race, class, str, dex, con, int, wis, pre,
			level, xp, next_level_xp, hp, max_hp, mp, max_mp
		) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 0, 1000, 100, 100, 100, 100)`,
		name, race, class,
		stats["STR"], stats["DEX"], stats["CON"],
		stats["INT"], stats["WIS"], stats["PRE"])
	return err
}

// PlayerExists checks if a player with the given name exists in the database
func PlayerExists(name string) bool {
	var exists bool
	// Query the database to check for the existence of the player by name
	err := db.QueryRow("SELECT EXISTS (SELECT 1 FROM players WHERE name = ?)", name).Scan(&exists)
	// Return true if no error occurred and the player exists, otherwise return false
	return err == nil && exists
}

// LoadPlayer retrieves a player's information from the database
func LoadPlayer(name string) (string, string, int, int, int, int, int, int, int, int, int, int, int, int, int, int, error) {
	var race, class string
	var roomID, str, dex, con, int_, wis, pre, level, xp, nextLevelXP, hp, maxHP, mp, maxMP int
	err := db.QueryRow(`
		SELECT race, class, room_id, str, dex, con, int, wis, pre, 
			   level, xp, next_level_xp, hp, max_hp, mp, max_mp 
		FROM players WHERE name = ?`, name).Scan(
		&race, &class, &roomID, &str, &dex, &con, &int_, &wis, &pre,
		&level, &xp, &nextLevelXP, &hp, &maxHP, &mp, &maxMP)
	if err != nil {
		return "", "", 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, err
	}
	return race, class, roomID, str, dex, con, int_, wis, pre, level, xp, nextLevelXP, hp, maxHP, mp, maxMP, nil
}

// UpdatePlayerRoom updates the room ID for a player, moving them to a new room
func UpdatePlayerRoom(playerName string, roomID int) error {
	// Execute an update query to change the player's room_id in the players table
	_, err := db.Exec("UPDATE players SET room_id = ? WHERE name = ?", roomID, playerName)
	return err // Return any error encountered during the process
}

// Add function to update player level info
func UpdatePlayerLevel(name string, level, xp, nextLevelXP int) error {
	_, err := db.Exec(`
		UPDATE players 
		SET level = ?, xp = ?, next_level_xp = ? 
		WHERE name = ?`,
		level, xp, nextLevelXP, name)
	return err
}

// Add function to update player HP and MP
func UpdatePlayerHPMP(name string, hp, maxHP, mp, maxMP int) error {
	_, err := db.Exec(`
		UPDATE players 
		SET hp = ?, max_hp = ?, mp = ?, max_mp = ? 
		WHERE name = ?`,
		hp, maxHP, mp, maxMP, name)
	return err
}
