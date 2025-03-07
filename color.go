package main

import (
	"strings"
)

/*
ANSI Color System for Go-MUD

This system implements ROM-style color codes for text output in the MUD.
Players can toggle colors on/off using the 'color' command.

Color Codes:
  {R} - Red
  {G} - Green
  {Y} - Yellow
  {B} - Blue
  {M} - Magenta
  {C} - Cyan
  {W} - White
  {D} - Dark Gray
  {x} - Reset (default color)

Usage Examples:
  - "{R}The cityguard attacks you!{x}" -> Red text followed by reset
  - "{G}You gain 100 experience points.{x}" -> Green text followed by reset
  - "{C}Market Square{x}" -> Cyan text followed by reset

Default Color Scheme:
  - Room Descriptions: {C} Cyan
  - Combat Messages: {R} Red
  - Dialogue/Text: {Y} Yellow
  - System Messages: {W} White
  - Player Deaths: {M} Magenta
  - Items: {G} Green
  - Skills: {B} Blue
  - Notifications: {D} Dark Gray

To use colors in your code:
  1. For direct player output: player.Send("{R}Colored text{x}")
  2. For typed messages: player.SendType("Message text", "combat")
  3. For room broadcasts: BroadcastToRoom(ColorizeByType("Message", "room"), room, player)
*/

// ANSI color codes
const (
	// Reset
	Reset = "\033[0m"

	// Regular Colors
	Red      = "\033[31m"
	Green    = "\033[32m"
	Yellow   = "\033[33m"
	Blue     = "\033[34m"
	Magenta  = "\033[35m"
	Cyan     = "\033[36m"
	White    = "\033[37m"
	DarkGray = "\033[90m"

	// Bold/Bright Colors
	BoldRed     = "\033[1;31m"
	BoldGreen   = "\033[1;32m"
	BoldYellow  = "\033[1;33m"
	BoldBlue    = "\033[1;34m"
	BoldMagenta = "\033[1;35m"
	BoldCyan    = "\033[1;36m"
	BoldWhite   = "\033[1;37m"
)

// ColorMap maps ROM-style color codes to ANSI escape sequences
var ColorMap = map[string]string{
	"{R}": Red,
	"{G}": Green,
	"{Y}": Yellow,
	"{B}": Blue,
	"{M}": Magenta,
	"{C}": Cyan,
	"{W}": White,
	"{D}": DarkGray,
	"{x}": Reset,
}

// Default color scheme for different types of messages
var DefaultColorScheme = map[string]string{
	"room":         "{C}", // Cyan for room descriptions
	"combat":       "{R}", // Red for combat messages
	"dialogue":     "{Y}", // Yellow for dialogue/text
	"system":       "{W}", // White for system messages
	"death":        "{M}", // Magenta for player deaths
	"item":         "{G}", // Green for items
	"skill":        "{B}", // Blue for skills
	"notification": "{D}", // Dark gray for notifications
}

// ProcessColors replaces ROM-style color codes with ANSI escape sequences
// If colorEnabled is false, it strips color codes instead
func ProcessColors(text string, colorEnabled bool) string {
	if !colorEnabled {
		// Strip color codes if colors are disabled
		for code := range ColorMap {
			text = strings.ReplaceAll(text, code, "")
		}
		return text
	}

	// Replace color codes with ANSI escape sequences
	for code, ansi := range ColorMap {
		text = strings.ReplaceAll(text, code, ansi)
	}

	// Check if the text contains any color codes but doesn't end with a reset
	if !strings.HasSuffix(text, Reset) {
		// Check if any color code was used
		for _, ansi := range ColorMap {
			if strings.Contains(text, ansi) {
				// Add reset code at the end
				text += Reset
				break
			}
		}
	}

	return text
}

// ColorizeByType applies the default color for a specific message type
func ColorizeByType(text string, messageType string) string {
	colorCode, exists := DefaultColorScheme[messageType]
	if !exists {
		return text // Return unmodified if message type doesn't exist
	}

	// Add color code at the beginning and reset at the end
	return colorCode + text + "{x}"
}
