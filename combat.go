/*
 * combat.go
 *
 * This file contains functions related to the combat system in the MUD.
 * It implements mechanics for calculating and processing combat outcomes,
 * including evasion chances, critical hit chances, and related combat
 * calculations. The functions handle the randomized aspects of combat
 * while accounting for level differences between combatants.
 */

package main

import (
	"log"
)

// CalculateEvasionChance determines the chance to dodge an attack based on level difference
func CalculateEvasionChance(defenderLevel, attackerLevel int) float64 {
	baseEvasionChance := 0.05 // 5% base evasion chance
	levelDifference := defenderLevel - attackerLevel

	// Adjust evasion chance based on level difference
	evasionChanceAdjustment := 0.0
	if levelDifference >= 3 {
		// +10% evasion for 3+ levels higher than attacker
		evasionChanceAdjustment = 0.10
	} else if levelDifference <= -3 {
		// -5% evasion for 3+ levels lower than attacker
		evasionChanceAdjustment = -0.05
	}

	finalEvasionChance := baseEvasionChance + evasionChanceAdjustment

	// Ensure evasion chance is within bounds
	if finalEvasionChance < 0.05 {
		finalEvasionChance = 0.05 // Minimum 5% evasion chance
	} else if finalEvasionChance > 0.50 {
		finalEvasionChance = 0.50 // Maximum 50% evasion chance
	}

	return finalEvasionChance
}

// CalculateCriticalChance determines the chance to land a critical hit based on level difference
func CalculateCriticalChance(attackerLevel, defenderLevel int) float64 {
	baseCritChance := 0.05 // 5% base critical hit chance
	levelDifference := attackerLevel - defenderLevel

	// Adjust critical hit chance based on level difference
	critChanceAdjustment := 0.0
	if levelDifference >= 3 {
		// +10% crit chance for 3+ levels higher than target
		critChanceAdjustment = 0.10
	} else if levelDifference <= -3 {
		// -5% crit chance for 3+ levels lower than target
		critChanceAdjustment = -0.05
	}

	finalCritChance := baseCritChance + critChanceAdjustment

	// Ensure critical hit chance is within bounds
	if finalCritChance < 0.05 {
		finalCritChance = 0.05 // Minimum 5% critical hit chance
	} else if finalCritChance > 0.50 {
		finalCritChance = 0.50 // Maximum 50% critical hit chance
	}

	return finalCritChance
}

// ProcessEvasion checks if an attack is evaded
// Returns true if the attack is evaded, false otherwise
func ProcessEvasion(defenderLevel, attackerLevel int) bool {
	evasionChance := CalculateEvasionChance(defenderLevel, attackerLevel)
	evasionRoll := rng.Float64()

	// Log the evasion check
	if evasionRoll <= evasionChance {
		log.Printf("Combat: Evasion successful (roll: %.2f, chance: %.2f, defender level: %d, attacker level: %d)",
			evasionRoll, evasionChance, defenderLevel, attackerLevel)
		return true
	}

	return false
}

// ProcessCriticalHit checks if an attack is a critical hit
// Returns true if the attack is a critical hit, false otherwise
func ProcessCriticalHit(attackerLevel, defenderLevel int) bool {
	critChance := CalculateCriticalChance(attackerLevel, defenderLevel)
	critRoll := rng.Float64()

	// Log the critical hit check
	if critRoll <= critChance {
		log.Printf("Combat: Critical hit successful (roll: %.2f, chance: %.2f, attacker level: %d, defender level: %d)",
			critRoll, critChance, attackerLevel, defenderLevel)
		return true
	}

	return false
}
