package main

import (
	"fmt"
	"log"
	"net"
	"strings"
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
	Stamina     int
	MaxStamina  int
	Gold        int

	// Derived Combat Stats
	HitChance     float64
	EvasionChance float64
	CritChance    float64
	CritDamage    float64
	AttackSpeed   float64
	CastSpeed     float64

	// Session-specific data
	Room        *Room    // Current room the player is in
	Conn        net.Conn // Network connection for the player
	LastCommand string   // Store the last command for reference
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

		// Update derived stats after level up
		p.UpdateDerivedStats()

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

// Add new function for stamina restoration
func (p *Player) RestoreStamina(amount int) {
	p.Stamina += amount
	if p.Stamina > p.MaxStamina {
		p.Stamina = p.MaxStamina
	}
	if p.Stamina < 0 {
		p.Stamina = 0
	}
	p.Conn.Write([]byte(fmt.Sprintf("You recover %d%% stamina.\r\n", amount)))
	UpdatePlayerStats(p.Name, p.HP, p.MaxHP, p.MP, p.MaxMP, p.Stamina, p.MaxStamina)
}

// Add function to update derived stats
func (p *Player) UpdateDerivedStats() {
	// Calculate derived stats using the formulas
	p.HitChance = 50 + (float64(p.DEX) * 1.5) + (float64(p.WIS) * 0.5) + (float64(p.Level) * 0.5)
	p.EvasionChance = (float64(p.DEX) * 1.8) + (float64(p.WIS) * 0.2) - (float64(p.Level) * 0.3)
	p.CritChance = (float64(p.DEX) * 0.5) + (float64(p.PRE) * 0.7) + (float64(p.Level) * 0.2)
	p.CritDamage = 150 + (float64(p.STR) * 1.2) + (float64(p.WIS) * 0.5)
	p.AttackSpeed = 100 + (float64(p.DEX) * 1.5) + (float64(p.STR) * 0.5)
	p.CastSpeed = 100 + (float64(p.INT) * 1.3) + (float64(p.WIS) * 0.8)
}

// Update GetStatsDisplay to include combat stats
func (p *Player) GetStatsDisplay() string {
	// Update derived stats before displaying
	p.UpdateDerivedStats()

	return fmt.Sprintf(
		"=== Basic Stats ===\n"+
			"HP: %d/%d\n"+
			"MP: %d/%d\n"+
			"Stamina: %d%%\n"+
			"Level: %d\n"+
			"XP: %d/%d\n\n"+
			"=== Combat Stats ===\n"+
			"Hit %%: %.1f%%\n"+
			"Evasion %%: %.1f%%\n"+
			"Crit %%: %.1f%%\n"+
			"Crit DMG %%: %.1f%%\n"+
			"Attack Speed %%: %.1f%%\n"+
			"Cast Speed %%: %.1f%%\n",
		p.HP, p.MaxHP,
		p.MP, p.MaxMP,
		p.Stamina,
		p.Level, p.XP, p.NextLevelXP,
		p.HitChance,
		p.EvasionChance,
		p.CritChance,
		p.CritDamage,
		p.AttackSpeed,
		p.CastSpeed)
}

// ModifyAttribute safely changes a core attribute value and updates derived stats
func (p *Player) ModifyAttribute(attribute string, amount int) error {
	switch strings.ToUpper(attribute) {
	case "STR":
		p.STR += amount
	case "DEX":
		p.DEX += amount
	case "CON":
		p.CON += amount
	case "INT":
		p.INT += amount
	case "WIS":
		p.WIS += amount
	case "PRE":
		p.PRE += amount
	default:
		return fmt.Errorf("invalid attribute: %s", attribute)
	}

	// Recalculate derived stats whenever core attributes change
	p.UpdateDerivedStats()

	// Update the database with new attribute values
	// Note: We'll need to create this function in db.go
	return UpdatePlayerAttributes(p.Name, p.STR, p.DEX, p.CON, p.INT, p.WIS, p.PRE)
}
