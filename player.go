package main

import (
	"net"
	"sync"
)

// Player represents an active player session
type Player struct {
	// Character data
	Name  string
	Race  string
	Class string
	// Core Stats
	STR int
	DEX int
	CON int
	INT int
	WIS int
	PRE int

	// Session-specific data
	Room *Room    // Current room the player is in
	Conn net.Conn // Network connection for the player
}

// Global session management
var (
	activePlayers = make(map[string]*Player)
	playersMutex  sync.Mutex
)

// Session management functions
func AddPlayer(player *Player) {
	playersMutex.Lock()
	defer playersMutex.Unlock()
	activePlayers[player.Name] = player
}

func RemovePlayer(player *Player) {
	playersMutex.Lock()
	defer playersMutex.Unlock()
	delete(activePlayers, player.Name)
}

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
