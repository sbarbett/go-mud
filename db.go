/*
 * db.go
 *
 * This file handles database operations for the MUD.
 * It provides functions for initializing the SQLite database connection,
 * creating and managing database tables, and performing CRUD operations
 * on player data. The file includes functions for creating new players,
 * loading player information, updating player attributes, and checking
 * if players exist in the database.
 */

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
		title TEXT,
		room_id INTEGER NOT NULL DEFAULT 3700,
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

	// Check if the title column exists, and add it if it doesn't
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('players') WHERE name='title'`).Scan(&count)
	if err != nil {
		log.Fatal("Failed to check if title column exists:", err)
	}
	if count == 0 {
		_, err = db.Exec(`ALTER TABLE players ADD COLUMN title TEXT;`)
		if err != nil {
			log.Fatal("Failed to add title column:", err)
		}
	}

	// Helper function to check if a column exists and add it if it doesn't
	addColumnIfNotExists := func(columnName, columnDef string) {
		var columnExists bool
		err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('players') WHERE name=?", columnName).Scan(&columnExists)
		if err != nil {
			log.Fatal("Failed to check if column exists:", columnName, err)
		}
		if !columnExists {
			_, err := db.Exec("ALTER TABLE players ADD COLUMN " + columnName + " " + columnDef)
			if err != nil {
				log.Fatal("Failed to add column:", columnName, err)
			}
			log.Printf("Added column: %s", columnName)
		}
	}

	// Add all required columns
	addColumnIfNotExists("level", "INTEGER NOT NULL DEFAULT 1")
	addColumnIfNotExists("xp", "INTEGER")
	addColumnIfNotExists("next_level_xp", "INTEGER")
	addColumnIfNotExists("hp", "INTEGER")
	addColumnIfNotExists("max_hp", "INTEGER")
	addColumnIfNotExists("mp", "INTEGER")
	addColumnIfNotExists("max_mp", "INTEGER")
	addColumnIfNotExists("stamina", "INTEGER")
	addColumnIfNotExists("max_stamina", "INTEGER")
	addColumnIfNotExists("gold", "INTEGER")
	addColumnIfNotExists("color_enabled", "INTEGER NOT NULL DEFAULT 1") // 1 = true, 0 = false
}

// CreatePlayer adds a new player to the database with their stats
func CreatePlayer(name, race, class string, stats map[string]int) error {
	_, err := db.Exec(`
		INSERT INTO players (
			name, race, class, title, str, dex, con, int, wis, pre,
			level, xp, next_level_xp, hp, max_hp, mp, max_mp,
			stamina, max_stamina, color_enabled
		) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 0, 1000, 100, 100, 100, 100, 100, 100, 1)`,
		name, race, class, "the Newbie",
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
func LoadPlayer(name string) (race string, class string, title string, roomID int, str int, dex int, con int, int_ int, wis int, pre int, level int, xp int, nextLevelXP int, hp int, maxHP int, mp int, maxMP int, stamina int, maxStamina int, gold int, colorEnabled bool, err error) {
	// Set default values
	gold = 0
	colorEnabled = true // Default to true if not found in DB

	log.Printf("Loading player data for: %s", name)

	// Check if the title column exists
	var titleColumnExists bool
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('players') WHERE name='title'`).Scan(&titleColumnExists)
	if err != nil {
		log.Printf("Error checking if title column exists: %v", err)
		// Continue anyway, we'll handle missing columns
	}

	// Query the database for the player's information
	var colorEnabledInt int
	var goldNull sql.NullInt64   // Use NullInt64 to handle NULL values
	var titleNull sql.NullString // Use NullString to handle NULL values

	// Build the query based on whether the title column exists
	var query string
	if titleColumnExists {
		query = `
			SELECT race, class, title, room_id, str, dex, con, int, wis, pre, 
			level, xp, next_level_xp, hp, max_hp, mp, max_mp, stamina, max_stamina, gold, 
			COALESCE(color_enabled, 1) 
			FROM players WHERE name = ?`
		err = db.QueryRow(query, name).Scan(
			&race, &class, &titleNull, &roomID, &str, &dex, &con, &int_, &wis, &pre,
			&level, &xp, &nextLevelXP, &hp, &maxHP, &mp, &maxMP, &stamina, &maxStamina, &goldNull,
			&colorEnabledInt)
	} else {
		query = `
			SELECT race, class, room_id, str, dex, con, int, wis, pre, 
			level, xp, next_level_xp, hp, max_hp, mp, max_mp, stamina, max_stamina, gold, 
			COALESCE(color_enabled, 1) 
			FROM players WHERE name = ?`
		err = db.QueryRow(query, name).Scan(
			&race, &class, &roomID, &str, &dex, &con, &int_, &wis, &pre,
			&level, &xp, &nextLevelXP, &hp, &maxHP, &mp, &maxMP, &stamina, &maxStamina, &goldNull,
			&colorEnabledInt)
	}

	if err != nil {
		log.Printf("Error loading player %s: %v", name, err)
		return
	}

	// Convert NullInt64 to int
	if goldNull.Valid {
		gold = int(goldNull.Int64)
	}

	// Convert NullString to string
	if titleNull.Valid {
		title = titleNull.String
	} else {
		// No title found, leave it empty
		title = ""
	}

	log.Printf("Successfully loaded player %s: race=%s, class=%s, room=%d", name, race, class, roomID)
	colorEnabled = colorEnabledInt == 1
	return
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

// Add new function to update player stats including stamina
func UpdatePlayerStats(name string, hp, maxHP, mp, maxMP, stamina, maxStamina int) error {
	_, err := db.Exec(`
		UPDATE players 
		SET hp = ?, max_hp = ?, mp = ?, max_mp = ?, stamina = ?, max_stamina = ?
		WHERE name = ?`,
		hp, maxHP, mp, maxMP, stamina, maxStamina, name)
	return err
}

// UpdatePlayerAttributes updates the core attributes of a player in the database
func UpdatePlayerAttributes(name string, str, dex, con, int_, wis, pre int) error {
	_, err := db.Exec(`
		UPDATE players 
		SET str = ?, dex = ?, con = ?, int = ?, wis = ?, pre = ?
		WHERE name = ?`,
		str, dex, con, int_, wis, pre, name)
	return err
}

// UpdatePlayerColorPreference updates a player's color preference in the database
func UpdatePlayerColorPreference(name string, colorEnabled bool) error {
	_, err := db.Exec("UPDATE players SET color_enabled = ? WHERE name = ?", colorEnabled, name)
	return err
}

// UpdatePlayerTitle updates the player's title in the database
func UpdatePlayerTitle(name string, title string) error {
	_, err := db.Exec("UPDATE players SET title = ? WHERE name = ?", title, name)
	return err
}
