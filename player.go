package main

import (
	"net"
	"sync"
)

// Player represents an active player session with attributes like name, race, class, current room, and connection.
type Player struct {
	Name  string   // Player's name
	Race  string   // Player's race
	Class string   // Player's class
	Room  *Room    // Current room the player is in
	Conn  net.Conn // Network connection for the player
}

// Global player-related variables
var (
	activePlayers = make(map[string]*Player) // Maps player names to Player objects
	playersMutex  sync.Mutex                 // Mutex to safely manage the activePlayers map
)

// BroadcastToRoom sends a message to all players in a specific room except the sender
func BroadcastToRoom(message string, room *Room, sender *Player) {
	playersMutex.Lock()
	defer playersMutex.Unlock()

	for _, p := range activePlayers {
		if p != sender && p.Room != nil && room != nil &&
			p.Room.ID == room.ID && p.Room == room {
			p.Conn.Write([]byte(message + "\r\n"))
		}
	}
}

// GetPlayersInRoom returns a slice of player names in the specified room
func GetPlayersInRoom(room *Room) []string {
	playersMutex.Lock()
	defer playersMutex.Unlock()

	var players []string
	for _, p := range activePlayers {
		if p.Room != nil && room != nil &&
			p.Room.ID == room.ID && p.Room == room &&
			len(p.Name) > 0 {
			players = append(players, p.Name)
		}
	}
	return players
}

// RemovePlayer removes a player from the active players list
func RemovePlayer(player *Player) {
	playersMutex.Lock()
	defer playersMutex.Unlock()
	delete(activePlayers, player.Name)
}

// AddPlayer adds a player to the active players list
func AddPlayer(player *Player) {
	playersMutex.Lock()
	defer playersMutex.Unlock()
	activePlayers[player.Name] = player
}
