package main

import (
	"fmt"
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
