package main

import (
	"fmt"
	"log"
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
}

// HandleCommand processes a player's command and returns the appropriate response
func HandleCommand(player *Player, input string) string {
	// Handle OOC chat separately
	if input == "ooc" || strings.HasPrefix(input, "ooc ") {
		oocManager.HandleOOCCommand(player, input)
		return ""
	}

	// Split the input into command and arguments
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return ""
	}

	command := strings.ToLower(parts[0])
	args := parts[1:]

	// Look up the handler for this command
	handler, exists := commandHandlers[command]
	if !exists {
		return "Unknown command.\r\n"
	}

	// Execute the handler
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

	// Set the player's combat state
	player.EnterCombat(mob)

	// Log the combat initiation
	log.Printf("[COMBAT] Player %s engaged Mob ID %d (%s)",
		player.Name, mob.ID, mob.ShortDescription)

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

	// Exit combat
	player.ExitCombat()

	// Log the flee
	log.Printf("[COMBAT] Player %s fled from combat", player.Name)

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

	return fmt.Sprintf("You are fighting %s.\r\nYour health: %d/%d\r\nTarget health: %d/%d\r\n",
		player.Target.ShortDescription,
		player.HP, player.MaxHP,
		player.Target.HP, player.Target.MaxHP)
}
