package main

import (
	"fmt"
	"log"
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
	STR         int
	DEX         int
	CON         int
	INT         int
	WIS         int
	PRE         int
	Level       int
	XP          int
	NextLevelXP int
	MaxHP       int
	HP          int
	MaxMP       int
	MP          int

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

// Add function to calculate XP needed for next level
func calculateNextLevelXP(level int) int {
	return (level * 1000) + ((level - 1) * 500)
}

// Add function to handle XP gain and level ups
func (p *Player) GainXP(amount int) {
	p.XP += amount
	p.Conn.Write([]byte(fmt.Sprintf("You gain %d experience points.\r\n", amount)))

	for p.XP >= p.NextLevelXP {
		overflowXP := p.XP - p.NextLevelXP
		p.Level++
		p.XP = overflowXP
		p.NextLevelXP = calculateNextLevelXP(p.Level)

		// Calculate HP and MP gains
		hpGain := (p.CON * 5) + 10
		mpGain := ((p.INT + p.WIS) * 3) + 8

		// Update max values
		p.MaxHP += hpGain
		p.MaxMP += mpGain

		// Fully restore HP and MP
		p.HP = p.MaxHP
		p.MP = p.MaxMP

		// Announce level up and stat increases
		levelUpMsg := fmt.Sprintf("\r\nCONGRATULATIONS! You have reached level %d!\r\n", p.Level)
		levelUpMsg += fmt.Sprintf("Your Max HP increased by %d! Your Max MP increased by %d!\r\n", hpGain, mpGain)
		p.Conn.Write([]byte(levelUpMsg))

		// Update the database
		if err := UpdatePlayerLevel(p.Name, p.Level, p.XP, p.NextLevelXP); err != nil {
			log.Printf("Error updating player level: %v", err)
		}
		if err := UpdatePlayerHPMP(p.Name, p.HP, p.MaxHP, p.MP, p.MaxMP); err != nil {
			log.Printf("Error updating player HP/MP: %v", err)
		}
	}
}

// Add healing and mana restoration methods
func (p *Player) Heal(amount int) {
	p.HP += amount
	if p.HP > p.MaxHP {
		p.HP = p.MaxHP
	}
	if p.HP < 0 {
		p.HP = 0
	}
	p.Conn.Write([]byte(fmt.Sprintf("You are healed for %d points.\r\n", amount)))
	UpdatePlayerHPMP(p.Name, p.HP, p.MaxHP, p.MP, p.MaxMP)
}

func (p *Player) RestoreMana(amount int) {
	p.MP += amount
	if p.MP > p.MaxMP {
		p.MP = p.MaxMP
	}
	if p.MP < 0 {
		p.MP = 0
	}
	p.Conn.Write([]byte(fmt.Sprintf("You recover %d mana points.\r\n", amount)))
	UpdatePlayerHPMP(p.Name, p.HP, p.MaxHP, p.MP, p.MaxMP)
}
