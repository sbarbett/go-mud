/*
 * stats.go
 *
 * This file defines the character statistics system for the MUD.
 * It contains the base statistics for different races, functions for
 * retrieving and validating character stats, and constants for character
 * creation. The file provides the foundation for the attribute system
 * that influences character capabilities and performance in the game.
 */

package main

// BaseStats defines the starting stats for each race
var BaseStats = map[string]map[string]int{
	"Human": {
		"STR": 10, "DEX": 10, "CON": 10,
		"INT": 10, "WIS": 10, "PRE": 10,
	},
	"Elf": {
		"STR": 7, "DEX": 12, "CON": 8,
		"INT": 13, "WIS": 11, "PRE": 10,
	},
	"Dwarf": {
		"STR": 12, "DEX": 8, "CON": 13,
		"INT": 7, "WIS": 11, "PRE": 8,
	},
	"Orc": {
		"STR": 14, "DEX": 9, "CON": 13,
		"INT": 6, "WIS": 7, "PRE": 8,
	},
}

const BONUS_POINTS = 8 // Number of bonus points to allocate

// GetBaseStats returns the base stats for a given race
func GetBaseStats(race string) map[string]int {
	if stats, exists := BaseStats[race]; exists {
		// Return a copy of the stats to prevent modifying the base values
		statsCopy := make(map[string]int)
		for k, v := range stats {
			statsCopy[k] = v
		}
		return statsCopy
	}
	// Return default human stats if race not found
	return BaseStats["Human"]
}

// ValidateStat checks if a stat value is within acceptable range
func ValidateStat(value int) bool {
	return value >= 3 && value <= 18
}
