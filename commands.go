/*
 * commands.go
 *
 * This file contains the command handling system for the MUD.
 * It defines the CommandHandler type and maps command names to their
 * respective handler functions. The file implements handlers for various
 * player commands including movement, combat, character information,
 * and system commands. The main HandleCommand function processes player
 * input and routes it to the appropriate handler.
 */

package main

import (
	"fmt"
	"log"

	//"log"
	"strconv"
	"strings"
)

// CommandHandler represents a function that handles a specific command
type CommandHandler func(player *Player, args []string) string

// commandHandlers maps command names to their handler functions
var commandHandlers = map[string]CommandHandler{
	"quit":      handleQuit,
	"look":      handleLook,
	"score":     handleScore,
	"scorecard": handleScore,
	"gainxp":    handleGainXP,
	// Combat commands
	"attack": handleAttack,
	"kill":   handleAttack,
	"flee":   handleFlee,
	"status": handleStatus,
	"combat": handleStatus,
	// Debug commands
	"debug": handleDebug,
	// Movement commands
	"north": handleMove,
	"south": handleMove,
	"east":  handleMove,
	"west":  handleMove,
	"up":    handleMove,
	"down":  handleMove,
	"n":     handleMove,
	"s":     handleMove,
	"e":     handleMove,
	"w":     handleMove,
	"u":     handleMove,
	"d":     handleMove,
	// Death commands
	"respawn": handleRespawn,
	// Color commands
	"color": handleColor,
	// Recall command
	"recall": handleRecall,
}

// HandleCommand processes a player's command and returns the appropriate response
func HandleCommand(player *Player, input string) string {
	// Handle OOC chat separately
	if input == "ooc" || strings.HasPrefix(input, "ooc ") {
		oocManager.HandleOOCCommand(player, input)
		return ""
	}

	// Store the last command for reference
	player.LastCommand = input

	// Split the input into command and arguments
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return ""
	}

	command := strings.ToLower(parts[0])
	args := parts[1:]

	// Check if player is dead
	if player.IsDead {
		// Only allow certain commands when dead
		switch command {
		case "look", "score", "quit", "respawn":
			// These commands are allowed when dead
		default:
			return "You are dead and cannot do that. Type 'respawn' to return to life."
		}
	}

	// Look up the handler for this command
	handler, exists := commandHandlers[command]
	if !exists {
		return fmt.Sprintf("Unknown command: %s", command)
	}

	// Execute the handler and return its response
	return handler(player, args)
}

// Individual command handlers

func handleQuit(player *Player, args []string) string {
	return "Goodbye!"
}

func handleScore(player *Player, args []string) string {
	return GetScorecard(player)
}

func handleGainXP(player *Player, args []string) string {
	if len(args) > 0 {
		if amount, err := strconv.Atoi(args[0]); err == nil {
			player.GainXP(amount)
			return fmt.Sprintf("Gained {G}%d{x} XP.", amount)
		}
	}
	return "Usage: gainxp <amount>"
}

func handleMove(player *Player, args []string) string {
	// Get the direction from the player's last command
	direction := strings.ToLower(player.LastCommand)

	// Handle movement
	if err := HandleMovement(player, direction); err != nil {
		return err.Error()
	}

	// Return empty string as the movement function will send the room description
	return ""
}

func handleLook(player *Player, args []string) string {
	// Get the look result from HandleLook
	lookResult := HandleLook(player, args)

	// Return the result without adding newlines
	return lookResult
}

// handleAttack processes a player's attempt to attack a mob
func handleAttack(player *Player, args []string) string {
	// Check if player is already in combat
	if player.IsInCombat() {
		return "You are already in combat!\r\n"
	}

	// Check if a target was specified
	if len(args) == 0 {
		return "Attack what?\r\n"
	}

	// Get the target name from args
	targetName := strings.ToLower(strings.Join(args, " "))

	// Find the mob in the current room
	mob := FindMobInRoom(player.Room.ID, targetName)
	if mob == nil {
		return "You don't see that here.\r\n"
	}

	// Check if the mob is already dead
	if mob.HP <= 0 {
		return fmt.Sprintf("The %s is already dead!\r\n", mob.ShortDescription)
	}

	// Set the player's combat state
	player.EnterCombat(mob)

	// Broadcast the combat initiation to other players in the room
	BroadcastCombatMessage(fmt.Sprintf("%s attacks the %s!",
		player.Name, mob.ShortDescription), player.Room, player)

	// Log the combat initiation
	// log.Printf("[COMBAT] Player %s engaged Mob ID %d (%s)",
	// 	player.Name, mob.ID, mob.ShortDescription)

	// Return success message
	return fmt.Sprintf("You attack the %s!\r\nThe %s turns to fight you!\r\n",
		mob.ShortDescription, mob.ShortDescription)
}

// handleFlee processes a player's attempt to flee from combat
func handleFlee(player *Player, args []string) string {
	// Check if player is in combat
	if !player.IsInCombat() {
		return "You're not in combat.\r\n"
	}

	// Get the mob name for the message
	mobName := "something"
	if player.Target != nil {
		mobName = player.Target.ShortDescription
	}

	// Exit combat BEFORE broadcasting to avoid deadlocks
	player.ExitCombat()

	// Broadcast the flee to other players in the room
	BroadcastCombatMessage(fmt.Sprintf("%s flees from the %s!",
		player.Name, mobName), player.Room, player)

	// Log the flee
	//log.Printf("[COMBAT] Player %s fled from combat", player.Name)

	// Return success message
	return fmt.Sprintf("You flee from the %s!\r\n", mobName)
}

// handleStatus shows the player's current combat status
func handleStatus(player *Player, args []string) string {
	if !player.IsInCombat() {
		return "You are not in combat.\r\n"
	}

	if player.Target == nil {
		// This shouldn't happen, but just in case
		player.ExitCombat()
		return "You are not in combat.\r\n"
	}

	// Calculate hit chance using the utility function
	finalHitChance := CalculateHitChance(player.Level, player.Target.Level)

	// Calculate expected damage using the utility function
	expectedDamage := CalculateDamage(player.Level)

	return fmt.Sprintf("You are fighting %s.\r\n"+
		"Your health: %d/%d\r\n"+
		"Target health: %d/%d\r\n"+
		"Your level: %d, Target level: %d\r\n"+
		"Hit chance: %.0f%%\r\n"+
		"Expected damage per hit: %d\r\n",
		player.Target.ShortDescription,
		player.HP, player.MaxHP,
		player.Target.HP, player.Target.MaxHP,
		player.Level, player.Target.Level,
		finalHitChance*100,
		expectedDamage)
}

// handleDebug provides debug information for testing
func handleDebug(player *Player, args []string) string {
	if len(args) == 0 {
		return "Debug options: combat, room, mobs\r\n"
	}

	switch args[0] {
	case "combat":
		if !player.IsInCombat() {
			return "You are not in combat.\r\n"
		}
		return fmt.Sprintf("Combat Debug:\r\n"+
			"In Combat: %t\r\n"+
			"Target: %s (ID: %d)\r\n"+
			"Target HP: %d/%d\r\n"+
			"Target Level: %d\r\n"+
			"Your Level: %d\r\n"+
			"Hit Chance: %.2f\r\n",
			player.InCombat,
			player.Target.ShortDescription, player.Target.ID,
			player.Target.HP, player.Target.MaxHP,
			player.Target.Level,
			player.Level,
			CalculateHitChance(player.Level, player.Target.Level))

	case "room":
		return fmt.Sprintf("Room Debug:\r\n"+
			"Room ID: %d\r\n"+
			"Room Name: %s\r\n"+
			"Area: %s\r\n",
			player.Room.ID,
			player.Room.Name,
			player.Room.Area)

	case "mobs":
		mobMutex.RLock()
		mobs := GetMobsInRoom(player.Room.ID)
		mobMutex.RUnlock()

		if len(mobs) == 0 {
			return "No mobs in this room.\r\n"
		}

		result := "Mobs in room:\r\n"
		for i, mob := range mobs {
			result += fmt.Sprintf("%d. %s (ID: %d, Level: %d, HP: %d/%d)\r\n",
				i+1, mob.ShortDescription, mob.ID, mob.Level, mob.HP, mob.MaxHP)
		}
		return result

	default:
		return "Unknown debug option.\r\n"
	}
}

// handleRespawn processes a player's attempt to respawn after death
func handleRespawn(player *Player, args []string) string {
	if !player.IsDead {
		return "You are not dead!"
	}

	// Reset player state
	player.IsDead = false
	player.HP = player.MaxHP / 2 // Respawn with half health
	player.MP = player.MaxMP / 2 // Respawn with half mana

	// Get the respawn room (Temple of Midgaard)
	respawnRoomID := 3001 // Temple of Midgaard
	startRoom, err := GetRoom(respawnRoomID)
	if err != nil {
		log.Printf("Error getting respawn room: %v", err)
		return "{R}Error during respawn. Please contact an administrator.{x}"
	}

	// Move player to respawn room
	oldRoom := player.Room
	player.Room = startRoom

	// Update player's room in database
	if err := UpdatePlayerRoom(player.Name, respawnRoomID); err != nil {
		log.Printf("Error updating player room in database: %v", err)
	}

	// Broadcast departure and arrival messages
	if oldRoom != startRoom {
		BroadcastToRoom(fmt.Sprintf("%s's body fades away.", player.Name), oldRoom, player)
	}
	BroadcastToRoom(ColorizeByType(fmt.Sprintf("%s appears in a flash of divine light.", player.Name), "system"), startRoom, player)

	return "{G}You feel your spirit being pulled back to the world of the living...{x}"
}

// handleColor toggles ANSI color on or off for the player
func handleColor(player *Player, args []string) string {
	if len(args) == 0 {
		// Display current color setting
		if player.ColorEnabled {
			return "Colors are currently {G}ON{x}. Use 'color off' to disable."
		} else {
			return "Colors are currently OFF. Use 'color on' to enable."
		}
	}

	switch strings.ToLower(args[0]) {
	case "on":
		player.ColorEnabled = true
		// Update the player's preference in the database
		err := UpdatePlayerColorPreference(player.Name, true)
		if err != nil {
			return "Error saving color preference. Colors enabled for this session only."
		}
		return "{G}Colors enabled.{x} You will now see colored text."
	case "off":
		player.ColorEnabled = false
		// Update the player's preference in the database
		err := UpdatePlayerColorPreference(player.Name, false)
		if err != nil {
			return "Error saving color preference. Colors disabled for this session only."
		}
		return "Colors disabled. You will no longer see colored text."
	default:
		return "Usage: color [on|off]"
	}
}

// handleRecall processes a player's attempt to recall to Room 3001 (Temple Square)
func handleRecall(player *Player, args []string) string {
	// Check if player is in combat
	if player.IsInCombat() {
		return "You cannot recall while fighting!"
	}

	// Get the destination room (Room 3001)
	destRoom, err := GetRoom(3001)
	if err != nil {
		log.Printf("[ERROR] Recall destination Room 3001 not found: %v", err)
		return "The recall magic fizzles. The destination seems to be missing."
	}

	// Store the old room for notifications
	oldRoom := player.Room

	// Update player's room in the database
	err = UpdatePlayerRoom(player.Name, 3001)
	if err != nil {
		log.Printf("[ERROR] Failed to update player room during recall: %v", err)
		return "The recall magic fizzles. Something went wrong."
	}

	// Update player's room in memory
	player.Room = destRoom

	// Log the recall event
	log.Printf("[RECALL] Player %s recalled to Room 3001.", player.Name)

	// Notify players in the old room about departure
	playersMutex.Lock()
	for _, p := range activePlayers {
		if p != player && p.Room == oldRoom {
			p.Send(fmt.Sprintf("%s disappears in a flash of light.", player.Name))
		}
	}
	playersMutex.Unlock()

	// Notify players in the new room about arrival
	playersMutex.Lock()
	for _, p := range activePlayers {
		if p != player && p.Room == destRoom {
			p.Send(fmt.Sprintf("%s appears in a flash of light.", player.Name))
		}
	}
	playersMutex.Unlock()

	// Send success message and room description to the player
	player.Send("A bright flash surrounds you, and you find yourself back at the Temple Square.")
	player.Send(DescribeRoom(destRoom, player))

	return ""
}
