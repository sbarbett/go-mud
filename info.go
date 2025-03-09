/*
 * info.go
 *
 * This file contains functions for displaying information to players.
 * It implements room descriptions, the look command, direction viewing,
 * and player scorecard functionality. The file handles formatting output
 * with appropriate colors and organizing information in a readable way
 * for players to understand their surroundings and character status.
 */

package main

import (
	"fmt"
	"sort"
	"strings"
)

// DescribeRoom prints the description of the current room
func DescribeRoom(room *Room, viewer *Player) string {
	// Get available exits and sort them
	var exits []string
	for direction := range room.Exits {
		exits = append(exits, direction)
	}
	sort.Strings(exits)

	// Get list of other players in the room (excluding the viewer)
	playersMutex.Lock()
	var otherPlayers []string
	for _, p := range activePlayers {
		if p != viewer && // Not the viewing player
			p.Room != nil && viewer.Room != nil && // Both rooms exist
			p.Room == viewer.Room { // Exact same room instance
			// Include the player's title if they have one
			if p.Title != "" {
				otherPlayers = append(otherPlayers, fmt.Sprintf("%s %s", p.Name, p.Title))
			} else {
				otherPlayers = append(otherPlayers, p.Name)
			}
		}
	}
	playersMutex.Unlock()

	// Build the room description with colors
	description := fmt.Sprintf("{C}%s{x}\n%s",
		room.Name,
		room.Description)

	// Add mobs in the room
	mobMutex.RLock()
	mobs := GetMobsInRoom(room.ID)

	if len(mobs) > 0 {
		description += "\n" // Single newline before mobs

		// Display mobs without numbering in the description
		for _, mob := range mobs {
			if mob != nil {
				// Check if this mob is in combat with any player
				combatStatus := ""
				playersMutex.Lock()
				for _, p := range activePlayers {
					if p.IsInCombat() && p.Target == mob {
						if p == viewer {
							combatStatus = " {R}[FIGHTING YOU]{x}"
						} else {
							combatStatus = fmt.Sprintf(" {R}[FIGHTING %s]{x}", p.Name)
						}
						break
					}
				}
				playersMutex.Unlock()

				description += fmt.Sprintf("%s%s\n", mob.LongDescription, combatStatus)
			}
		}
	}
	mobMutex.RUnlock()

	// Add exits after mobs
	description += fmt.Sprintf("\n{G}Available exits:{x} [%s]", strings.Join(exits, ", "))

	// Add other players if present
	if len(otherPlayers) > 0 {
		description += fmt.Sprintf("\n{Y}Also here:{x} %s", strings.Join(otherPlayers, ", "))
	}

	return description
}

// HandleLook processes the look command and its arguments
func HandleLook(player *Player, args []string) string {
	if len(args) == 0 {
		return DescribeRoom(player.Room, player)
	}

	// Check if looking at a direction
	direction := args[0]
	if fullDirection, isAlias := DirectionAliases[direction]; isAlias {
		direction = fullDirection
	}
	// If it's a direction (either an alias or full name), handle it
	if _, exists := player.Room.Exits[direction]; exists {
		return LookDirection(player.Room, direction)
	}
	// If it's a valid direction but no exit exists
	if _, isDirection := DirectionAliases[direction]; isDirection || stringInSlice(direction, []string{"north", "south", "east", "west", "up", "down"}) {
		return "Nothing special there."
	}

	// Check if looking at a mob
	lookTarget := strings.ToLower(strings.Join(args, " "))

	// Find the mob using our helper function
	mob := FindMobByTarget(player.Room.ID, lookTarget)

	if mob != nil {
		// Check if this mob is the player's combat target
		combatStatus := ""
		if player.IsInCombat() && player.Target == mob {
			combatStatus = " [FIGHTING YOU]"
		} else {
			// Check if this mob is fighting any other player
			playersMutex.Lock()
			for _, p := range activePlayers {
				if p.IsInCombat() && p.Target == mob {
					combatStatus = fmt.Sprintf(" [FIGHTING %s]", p.Name)
					break
				}
			}
			playersMutex.Unlock()
		}

		// Return the mob's description along with some basic stats and combat status
		return fmt.Sprintf("%s\n[Level %d %s] [HP: %d/%d]%s",
			mob.Description, mob.Level, mob.Toughness, mob.HP, mob.MaxHP, combatStatus)
	}

	// Check environment attributes
	for _, attr := range player.Room.Environment {
		for _, keyword := range attr.Keywords {
			if strings.ToLower(keyword) == lookTarget {
				return attr.Description
			}
		}
	}

	return "You do not see that here."
}

// LookDirection returns the description of what's visible in a given direction
func LookDirection(room *Room, direction string) string {
	exit, exists := room.Exits[direction]
	if !exists {
		return "Nothing special there."
	}

	if exit != nil && exit.Description != "" {
		return exit.Description
	}

	return fmt.Sprintf("You see a passage leading %s.", direction)
}

// GetScorecard returns a formatted string containing the player's complete stats
func GetScorecard(player *Player) string {
	// Update derived stats before displaying
	player.UpdateDerivedStats()

	// Format status effects (placeholder for now)
	status := "[No active effects]"

	// Build the scorecard using strings.Builder for efficiency
	var sb strings.Builder

	// Use consistent width and simple borders
	sb.WriteString("-------------------------------------------------\n")
	sb.WriteString("                CHARACTER SCORECARD               \n")
	sb.WriteString("-------------------------------------------------\n")
	sb.WriteString(fmt.Sprintf(" Name:         %-12s  Level:     %-6d\n", player.Name, player.Level))
	sb.WriteString(fmt.Sprintf(" Race:         %-12s  Class:     %-6s\n", player.Race, player.Class))

	// Display title or [not set] if empty
	titleToShow := player.Title
	if titleToShow == "" {
		titleToShow = "[not set]"
	}
	sb.WriteString(fmt.Sprintf(" Title:        %s\n", titleToShow))

	sb.WriteString(fmt.Sprintf(" XP:           %-12s  Gold:      %-6d\n", fmt.Sprintf("%d / %d", player.XP, player.NextLevelXP), player.Gold))
	sb.WriteString(fmt.Sprintf(" Status:       %-32s\n", status))
	sb.WriteString("-------------------------------------------------\n")
	sb.WriteString("                   ATTRIBUTES                     \n")
	sb.WriteString("-------------------------------------------------\n")
	sb.WriteString(fmt.Sprintf(" Strength:     %-8d  Dexterity:    %-8d\n", player.STR, player.DEX))
	sb.WriteString(fmt.Sprintf(" Constitution: %-8d  Intelligence: %-8d\n", player.CON, player.INT))
	sb.WriteString(fmt.Sprintf(" Wisdom:       %-8d  Presence:     %-8d\n", player.WIS, player.PRE))
	sb.WriteString("-------------------------------------------------\n")
	sb.WriteString("                  COMBAT STATS                    \n")
	sb.WriteString("-------------------------------------------------\n")
	sb.WriteString(fmt.Sprintf(" HP:           %-12s  MP:          %-12s\n", fmt.Sprintf("%d / %d", player.HP, player.MaxHP), fmt.Sprintf("%d / %d", player.MP, player.MaxMP)))
	sb.WriteString(fmt.Sprintf(" Stamina:      %-12s  Hit%%:        %-12s\n", fmt.Sprintf("%d%%", player.Stamina), fmt.Sprintf("%.1f%%", player.HitChance)))
	sb.WriteString(fmt.Sprintf(" Evasion:      %-12s  Crit%%:       %-12s\n", fmt.Sprintf("%.1f%%", player.EvasionChance), fmt.Sprintf("%.1f%%", player.CritChance)))
	sb.WriteString(fmt.Sprintf(" Crit DMG:     %-12s  Attack SPD:  %-12s\n", fmt.Sprintf("%.1f%%", player.CritDamage), fmt.Sprintf("%.1f%%", player.AttackSpeed)))
	sb.WriteString(fmt.Sprintf(" Cast SPD:     %-12s\n", fmt.Sprintf("%.1f%%", player.CastSpeed)))
	sb.WriteString("-------------------------------------------------\n")

	return sb.String()
}
