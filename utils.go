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

// stringInSlice checks if a string exists in a list
func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}
