package main

import (
	"fmt"
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
			return "You are dead and cannot do that. Type 'respawn' to return to life.\r\n"
		}
	}

	// Look up the handler for this command
	handler, exists := commandHandlers[command]
	if !exists {
		return fmt.Sprintf("Unknown command: %s\r\n", command)
	}

	// Execute the handler and return its response
	return handler(player, args)
}

// Individual command handlers

func handleQuit(player *Player, args []string) string {
	return "Goodbye!\r\n"
}

func handleScore(player *Player, args []string) string {
	return GetScorecard(player)
}

func handleGainXP(player *Player, args []string) string {
	if len(args) > 0 {
		if amount, err := strconv.Atoi(args[0]); err == nil {
			player.GainXP(amount)
			return fmt.Sprintf("Gained %d XP.\r\n", amount)
		}
	}
	return "Usage: gainxp <amount>\r\n"
}

func handleMove(player *Player, args []string) string {
	direction := strings.ToLower(player.LastCommand)
	if err := HandleMovement(player, direction); err != nil {
		return err.Error() + "\r\n"
	}
	return ""
}

func handleLook(player *Player, args []string) string {
	return HandleLook(player, args) + "\r\n"
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

// handleRespawn allows a player to manually respawn if they're dead
func handleRespawn(player *Player, args []string) string {
	if !player.IsDead {
		return "You are already alive!\r\n"
	}

	// Force immediate respawn
	go player.ScheduleRespawn()

	return "You feel your spirit being pulled back to the world of the living...\r\n"
}
