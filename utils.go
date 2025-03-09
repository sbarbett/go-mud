/*
 * utils.go
 *
 * This file contains utility functions used throughout the MUD codebase.
 * It provides helper functions for common operations that don't fit
 * specifically into other modules. Currently, it includes a function
 * for checking if a string exists in a slice, which is used in various
 * parts of the codebase for validation and lookups.
 */

package main

import (
	"strings"
)

// stringInSlice checks if a string exists in a list
func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

// StripColorCodes removes all color codes from a string
func StripColorCodes(text string) string {
	result := text
	for code := range ColorMap {
		result = strings.ReplaceAll(result, code, "")
	}
	return result
}

// LengthWithoutColorCodes returns the length of a string without counting color codes
func LengthWithoutColorCodes(text string) int {
	return len(StripColorCodes(text))
}

// GetOppositeDirection returns the opposite of a given direction
func GetOppositeDirection(dir string) string {
	switch dir {
	case "north":
		return "south"
	case "south":
		return "north"
	case "east":
		return "west"
	case "west":
		return "east"
	case "up":
		return "down"
	case "down":
		return "up"
	default:
		return "somewhere"
	}
}
