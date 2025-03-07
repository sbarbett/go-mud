package main

import (
	"fmt"
	"strings"
	"sync"
)

// OOCManager handles out-of-character communication functionality
type OOCManager struct {
	playersMutex *sync.Mutex
	players      map[string]*Player
}

// NewOOCManager creates a new OOCManager instance
func NewOOCManager(playersMutex *sync.Mutex, players map[string]*Player) *OOCManager {
	return &OOCManager{
		playersMutex: playersMutex,
		players:      players,
	}
}

// HandleOOCCommand processes out-of-character messages
func (m *OOCManager) HandleOOCCommand(player *Player, input string) {
	// If the input is exactly "ooc", show the help message
	if input == "ooc" {
		player.Send("OOC (Out of Character) lets you chat with other players.\r\nUsage: ooc <message>")
		return
	}

	// Otherwise, strip the "ooc " prefix and broadcast the message
	message := strings.TrimPrefix(input, "ooc ")
	m.BroadcastMessage(fmt.Sprintf("[OOC] %s: %s", player.Name, message), nil)
}

// BroadcastMessage sends a message to all connected players, excluding the specified player (if any)
func (m *OOCManager) BroadcastMessage(message string, exclude *Player) {
	m.playersMutex.Lock()
	defer m.playersMutex.Unlock()

	for _, p := range m.players {
		if p != exclude {
			p.Send(message)
		}
	}
}
