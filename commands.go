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
	"save":      handleSave,
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
	// Title command
	"title": handleTitle,
	// Who command
	"who": handleWho,
	// Help command
	"help": handleHelp,
	// Door commands
	"open":  handleOpen,
	"close": handleClose,
	// Teleport command
	"goto": handleGoto,
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
	// Save player's progress before quitting
	if err := UpdatePlayerXP(player.Name, player.XP, player.NextLevelXP); err != nil {
		log.Printf("Error saving player XP on quit: %v", err)
	}

	if err := UpdatePlayerHPMP(player.Name, player.HP, player.MaxHP, player.MP, player.MaxMP); err != nil {
		log.Printf("Error saving player HP/MP on quit: %v", err)
	}

	if err := UpdatePlayerStats(player.Name, player.HP, player.MaxHP, player.MP, player.MaxMP, player.Stamina, player.MaxStamina); err != nil {
		log.Printf("Error saving player stats on quit: %v", err)
	}

	return "Your progress has been saved. Goodbye!"
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

func handleSave(player *Player, args []string) string {
	// Save player's current state to the database
	if err := UpdatePlayerXP(player.Name, player.XP, player.NextLevelXP); err != nil {
		log.Printf("Error saving player XP: %v", err)
		return "Error saving your progress."
	}

	if err := UpdatePlayerHPMP(player.Name, player.HP, player.MaxHP, player.MP, player.MaxMP); err != nil {
		log.Printf("Error saving player HP/MP: %v", err)
		return "Error saving your progress."
	}

	if err := UpdatePlayerStats(player.Name, player.HP, player.MaxHP, player.MP, player.MaxMP, player.Stamina, player.MaxStamina); err != nil {
		log.Printf("Error saving player stats: %v", err)
		return "Error saving your progress."
	}

	return "Your progress has been saved."
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

	// Find the mob using our helper function
	mob := FindMobByTarget(player.Room.ID, targetName)

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

	// Get the respawn room
	respawnRoomID := RespawnRoomID // Use the constant from player.go
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

// handleRecall processes a player's attempt to recall to the respawn point (RespawnRoomID)
func handleRecall(player *Player, args []string) string {
	// Check if player is in combat
	if player.IsInCombat() {
		return "You cannot recall while fighting!"
	}

	// Get the destination room
	destRoom, err := GetRoom(RespawnRoomID)
	if err != nil {
		log.Printf("[ERROR] Recall destination Room %d not found: %v", RespawnRoomID, err)
		return "The recall magic fizzles. The destination seems to be missing."
	}

	// Store the old room for notifications
	oldRoom := player.Room

	// Update player's room in the database
	err = UpdatePlayerRoom(player.Name, RespawnRoomID)
	if err != nil {
		log.Printf("[ERROR] Failed to update player room during recall: %v", err)
		return "The recall magic fizzles. Something went wrong."
	}

	// Update player's room in memory
	player.Room = destRoom

	// Log the recall event
	log.Printf("[RECALL] Player %s recalled to Room %d.", player.Name, RespawnRoomID)

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

// handleTitle processes the title command
func handleTitle(player *Player, args []string) string {
	// If no arguments, remove the title
	if len(args) == 0 {
		player.Title = ""
		// Update the title in the database
		if err := UpdatePlayerTitle(player.Name, ""); err != nil {
			log.Printf("Error updating player title in database: %v", err)
		}
		return "Your title has been removed."
	}

	// Combine all arguments into a single title
	title := strings.Join(args, " ")

	// Trim any trailing spaces
	title = strings.TrimSpace(title)

	// Check if the title is too long (40 characters max, excluding color codes)
	if LengthWithoutColorCodes(title) > 40 {
		return "Titles must be no longer than 40 characters."
	}

	// Ensure the title ends with a color reset code
	if !strings.HasSuffix(title, "{x}") {
		// Check if any color code was used
		hasColorCode := false
		for code := range ColorMap {
			if strings.Contains(title, code) {
				hasColorCode = true
				break
			}
		}

		// Add reset code at the end if any color code was used
		if hasColorCode {
			title += "{x}"
		}
	}

	// Set the player's title
	player.Title = title

	// Update the title in the database
	if err := UpdatePlayerTitle(player.Name, title); err != nil {
		log.Printf("Error updating player title in database: %v", err)
	}

	return fmt.Sprintf("Your title is now: %s", title)
}

// handleWho displays a list of all players currently online
func handleWho(player *Player, args []string) string {
	playersMutex.Lock()
	defer playersMutex.Unlock()

	// Check if there are any players online
	if len(activePlayers) == 0 {
		return "There are no players currently online."
	}

	// Build the header for the who list
	output := "{Y}Players currently online:{x}\r\n"
	output += "{C}----------------------------------------{x}\r\n"

	// Format each player's information
	for _, p := range activePlayers {
		// Format the player's race, class, and level within brackets
		bracketInfo := fmt.Sprintf("[{G}%-6s{x} {B}%-8s{x} {M}%-3d{x}]",
			p.Race, p.Class, p.Level)

		// Add the player's name and title (if they have one)
		if p.Title != "" {
			output += fmt.Sprintf("%s {W}%s{x} %s\r\n", bracketInfo, p.Name, p.Title)
		} else {
			output += fmt.Sprintf("%s {W}%s{x}\r\n", bracketInfo, p.Name)
		}
	}

	// Add a footer with the total count
	output += "{C}----------------------------------------{x}\r\n"
	output += fmt.Sprintf("{Y}Total players online: {W}%d{x}\r\n", len(activePlayers))

	return output
}

// handleOpen processes the open command
func handleOpen(player *Player, args []string) string {
	if len(args) == 0 {
		return "Open what?"
	}

	// Get the target to open (direction or keyword)
	target := strings.ToLower(args[0])

	// Check if the target is a direction
	if fullDirection, isAlias := DirectionAliases[target]; isAlias {
		target = fullDirection
	}

	// Check if the target is a valid direction
	exit, exists := player.Room.Exits[target]
	if exists {
		// Check if there's a door in that direction
		if exit.Door == nil {
			return fmt.Sprintf("There is no door to the %s.", target)
		}

		// Check if the door is already open
		if !exit.Door.Closed {
			return fmt.Sprintf("The %s %s is already open.", target, exit.Door.ShortDescription)
		}

		// Check if the door is locked
		if exit.Door.Locked {
			return fmt.Sprintf("The %s %s is locked.", target, exit.Door.ShortDescription)
		}

		// Open the door
		exit.Door.Closed = false

		// Synchronize with the connected room's door
		SynchronizeDoor(player.Room.ID, target, false)

		// Notify the player
		message := fmt.Sprintf("You open the %s.", exit.Door.ShortDescription)

		// Notify other players in the room
		BroadcastToRoom(fmt.Sprintf("%s opens the %s.", player.Name, exit.Door.ShortDescription), player.Room, player)

		return message
	}

	// If not a direction, check if it's a door keyword
	for direction, exit := range player.Room.Exits {
		if exit.Door != nil {
			for _, keyword := range exit.Door.Keywords {
				if strings.ToLower(keyword) == target {
					// Check if the door is already open
					if !exit.Door.Closed {
						return fmt.Sprintf("The %s is already open.", exit.Door.ShortDescription)
					}

					// Check if the door is locked
					if exit.Door.Locked {
						return fmt.Sprintf("The %s is locked.", exit.Door.ShortDescription)
					}

					// Open the door
					exit.Door.Closed = false

					// Synchronize with the connected room's door
					SynchronizeDoor(player.Room.ID, direction, false)

					// Notify the player
					message := fmt.Sprintf("You open the %s to the %s.", exit.Door.ShortDescription, direction)

					// Notify other players in the room
					BroadcastToRoom(fmt.Sprintf("%s opens the %s to the %s.", player.Name, exit.Door.ShortDescription, direction), player.Room, player)

					return message
				}
			}
		}
	}

	return "You don't see that here."
}

// handleClose processes the close command
func handleClose(player *Player, args []string) string {
	if len(args) == 0 {
		return "Close what?"
	}

	// Get the target to close (direction or keyword)
	target := strings.ToLower(args[0])

	// Check if the target is a direction
	if fullDirection, isAlias := DirectionAliases[target]; isAlias {
		target = fullDirection
	}

	// Check if the target is a valid direction
	exit, exists := player.Room.Exits[target]
	if exists {
		// Check if there's a door in that direction
		if exit.Door == nil {
			return fmt.Sprintf("There is no door to the %s.", target)
		}

		// Check if the door is already closed
		if exit.Door.Closed {
			return fmt.Sprintf("The %s %s is already closed.", target, exit.Door.ShortDescription)
		}

		// Close the door
		exit.Door.Closed = true

		// Synchronize with the connected room's door
		SynchronizeDoor(player.Room.ID, target, true)

		// Notify the player
		message := fmt.Sprintf("You close the %s.", exit.Door.ShortDescription)

		// Notify other players in the room
		BroadcastToRoom(fmt.Sprintf("%s closes the %s.", player.Name, exit.Door.ShortDescription), player.Room, player)

		return message
	}

	// If not a direction, check if it's a door keyword
	for direction, exit := range player.Room.Exits {
		if exit.Door != nil {
			for _, keyword := range exit.Door.Keywords {
				if strings.ToLower(keyword) == target {
					// Check if the door is already closed
					if exit.Door.Closed {
						return fmt.Sprintf("The %s is already closed.", exit.Door.ShortDescription)
					}

					// Close the door
					exit.Door.Closed = true

					// Synchronize with the connected room's door
					SynchronizeDoor(player.Room.ID, direction, true)

					// Notify the player
					message := fmt.Sprintf("You close the %s to the %s.", exit.Door.ShortDescription, direction)

					// Notify other players in the room
					BroadcastToRoom(fmt.Sprintf("%s closes the %s to the %s.", player.Name, exit.Door.ShortDescription, direction), player.Room, player)

					return message
				}
			}
		}
	}

	return "You don't see that here."
}

// handleGoto teleports a player to a specified room ID
func handleGoto(player *Player, args []string) string {
	// Check if a room ID was provided
	if len(args) < 1 {
		return "Goto where? Please specify a room ID."
	}

	// Parse the room ID
	roomID, err := strconv.Atoi(args[0])
	if err != nil {
		return "Invalid room ID. Please specify a numeric room ID."
	}

	// Check if the room exists
	newRoom, err := GetRoom(roomID)
	if err != nil {
		return fmt.Sprintf("Room %d does not exist.", roomID)
	}

	// Update the player's location in the database
	err = UpdatePlayerRoom(player.Name, roomID)
	if err != nil {
		log.Printf("Error updating player room: %v", err)
		return "An error occurred while teleporting."
	}

	// Update the player's room in memory
	player.Room = newRoom

	// Log the teleportation for debugging
	log.Printf("Player %s teleported to room %d (%s)", player.Name, roomID, newRoom.Name)

	// Return success message
	return fmt.Sprintf("You teleport to Room %d (%s).", roomID, newRoom.Name)
}
