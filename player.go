package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
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

	// Combat state
	InCombat bool
	Target   *MobInstance
	IsDead   bool // New flag to track death state

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

// RegenTick handles player regeneration on each game tick (1 minute)
func (p *Player) RegenTick() {
	// Only regenerate if player is alive
	if p.HP <= 0 {
		return
	}

	// Calculate regeneration amounts
	hpRegen := p.CON / 2
	if hpRegen < 1 {
		hpRegen = 1
	}

	mpRegen := (p.INT + p.WIS) / 4
	if mpRegen < 1 {
		mpRegen = 1
	}

	staminaRegen := 10 // 10% stamina per minute

	// Apply regeneration
	if p.HP < p.MaxHP {
		p.Heal(hpRegen)
	}

	if p.MP < p.MaxMP {
		p.RestoreMana(mpRegen)
	}

	if p.Stamina < p.MaxStamina {
		p.RestoreStamina(staminaRegen)
	}
}

// PulseUpdate handles updates that occur every second
func (p *Player) PulseUpdate() {
	// Skip update if player is dead
	if p.IsDead {
		return
	}

	// Check for low health notification
	if p.HP > 0 && p.HP < p.MaxHP/5 {
		p.Conn.Write([]byte("\r\n*Your health is critically low!*\r\n> "))
	}

	// Handle combat state - only if player is in combat
	if p.IsInCombat() {
		// Make a local copy of the target to avoid race conditions
		target := p.Target

		// Verify target is still valid
		if target == nil {
			p.ExitCombat()
			p.Conn.Write([]byte("\r\nYour target is no longer available.\r\n> "))
			return
		}

		// Verify target is still in the same room
		if target.Room == nil || target.Room.ID != p.Room.ID {
			p.ExitCombat()
			p.Conn.Write([]byte("\r\nYour target has left the room.\r\n> "))
			return
		}

		// Check if target is dead
		if target.HP <= 0 {
			p.Conn.Write([]byte(fmt.Sprintf("\r\nThe %s is dead!\r\n> ", target.ShortDescription)))
			p.ExitCombat()
			return
		}

		// Execute player's attack
		p.ExecuteAttack()

		// Check if player is still in combat after their attack
		// (they might have killed the target)
		if !p.IsInCombat() || p.Target == nil {
			return
		}

		// Add a small delay to make combat easier to follow
		time.Sleep(100 * time.Millisecond)

		// Execute mob's counter-attack if it's still alive
		if p.Target != nil && p.Target.HP > 0 {
			p.ReceiveAttack(p.Target)
		}
	}
}

// ExecuteAttack handles a player attacking a mob
func (p *Player) ExecuteAttack() {
	// Safety check - ensure player is in combat and has a valid target
	if !p.IsInCombat() || p.Target == nil || p.IsDead {
		return
	}

	// Check if the target evades the attack
	if ProcessEvasion(p.Target.Level, p.Level) {
		// Target evaded the attack
		p.Conn.Write([]byte(fmt.Sprintf("\r\nYou swing at the %s, but they dodge your attack!\r\n> ",
			p.Target.ShortDescription)))

		// Broadcast the evasion to other players in the room
		BroadcastCombatMessage(fmt.Sprintf("%s swings at the %s, but they dodge the attack!",
			p.Name, p.Target.ShortDescription), p.Room, p)
		return
	}

	// Calculate hit chance using the utility function
	finalHitChance := CalculateHitChance(p.Level, p.Target.Level)

	// Roll to hit
	hitRoll := rand.Float64()

	if hitRoll <= finalHitChance {
		// Hit! Calculate damage using the utility function
		damage := CalculateDamage(p.Level)

		// Check for critical hit
		isCritical := ProcessCriticalHit(p.Level, p.Target.Level)
		if isCritical {
			// Critical hit! Double the damage
			damage *= 2
		}

		// Apply damage to target
		p.Target.HP -= damage
		if p.Target.HP < 0 {
			p.Target.HP = 0
		}

		// Send hit message to player
		if isCritical {
			p.Conn.Write([]byte(fmt.Sprintf("\r\nYou land a CRITICAL HIT on the %s for %d damage!\r\n> ",
				p.Target.ShortDescription, damage)))

			// Broadcast the critical hit to other players in the room
			BroadcastCombatMessage(fmt.Sprintf("%s lands a CRITICAL HIT on the %s for %d damage!",
				p.Name, p.Target.ShortDescription, damage), p.Room, p)
		} else {
			p.Conn.Write([]byte(fmt.Sprintf("\r\nYou strike the %s for %d damage!\r\n> ",
				p.Target.ShortDescription, damage)))

			// Broadcast the attack to other players in the room
			BroadcastCombatMessage(fmt.Sprintf("%s strikes the %s for %d damage!",
				p.Name, p.Target.ShortDescription, damage), p.Room, p)
		}

		// Check if target is dead
		if p.Target.HP <= 0 {
			// Handle mob death
			p.HandleMobDeath(p.Target)
		}
	} else {
		// Miss
		p.Conn.Write([]byte(fmt.Sprintf("\r\nYou swing at the %s but miss!\r\n> ",
			p.Target.ShortDescription)))

		// Broadcast the miss to other players in the room
		BroadcastCombatMessage(fmt.Sprintf("%s swings at the %s but misses!",
			p.Name, p.Target.ShortDescription), p.Room, p)
	}
}

// ReceiveAttack handles an attack from a mob against the player
func (p *Player) ReceiveAttack(attacker *MobInstance) {
	// Safety check - ensure player is in combat and has a valid target
	if !p.IsInCombat() || p.Target == nil || p.Target != attacker || p.IsDead {
		return
	}

	// Check if the player evades the attack
	if ProcessEvasion(p.Level, attacker.Level) {
		// Player evaded the attack
		p.Conn.Write([]byte(fmt.Sprintf("\r\nThe %s swings at you, but you evade just in time!\r\n> ",
			attacker.ShortDescription)))

		// Broadcast the evasion to other players in the room
		BroadcastCombatMessage(fmt.Sprintf("The %s swings at %s, but they evade just in time!",
			attacker.ShortDescription, p.Name), p.Room, p)
		return
	}

	// Calculate hit chance for the mob using the utility function
	finalHitChance := CalculateHitChance(attacker.Level, p.Level)

	// Roll to hit
	hitRoll := rand.Float64()

	if hitRoll <= finalHitChance {
		// Hit! Calculate damage using the utility function
		damage := CalculateDamage(attacker.Level)

		// Check for critical hit
		isCritical := ProcessCriticalHit(attacker.Level, p.Level)
		if isCritical {
			// Critical hit! Double the damage
			damage *= 2
		}

		// Apply damage to player
		p.HP -= damage
		if p.HP < 0 {
			p.HP = 0
		}

		// Send hit message to player
		if isCritical {
			p.Conn.Write([]byte(fmt.Sprintf("\r\nThe %s lands a CRITICAL HIT on you for %d damage!\r\n> ",
				attacker.ShortDescription, damage)))

			// Broadcast the critical hit to other players in the room
			BroadcastCombatMessage(fmt.Sprintf("The %s lands a CRITICAL HIT on %s for %d damage!",
				attacker.ShortDescription, p.Name, damage), p.Room, p)
		} else {
			p.Conn.Write([]byte(fmt.Sprintf("\r\nThe %s strikes you for %d damage!\r\n> ",
				attacker.ShortDescription, damage)))

			// Broadcast the attack to other players in the room
			BroadcastCombatMessage(fmt.Sprintf("The %s strikes %s for %d damage!",
				attacker.ShortDescription, p.Name, damage), p.Room, p)
		}

		// Check if player is dead
		if p.HP <= 0 {
			// Handle player death
			p.Die(attacker)
		}
	} else {
		// Miss
		p.Conn.Write([]byte(fmt.Sprintf("\r\nThe %s swings at you but misses!\r\n> ",
			attacker.ShortDescription)))

		// Broadcast the miss to other players in the room
		BroadcastCombatMessage(fmt.Sprintf("The %s swings at %s but misses!",
			attacker.ShortDescription, p.Name), p.Room, p)
	}
}

// EnterCombat puts the player in combat with the specified mob
func (p *Player) EnterCombat(target *MobInstance) {
	p.InCombat = true
	p.Target = target
}

// ExitCombat takes the player out of combat
func (p *Player) ExitCombat() {
	p.InCombat = false
	p.Target = nil
}

// IsInCombat checks if the player is in combat
func (p *Player) IsInCombat() bool {
	return p.InCombat && p.Target != nil
}

// HandleMobDeath handles the death of a mob killed by a player
func (p *Player) HandleMobDeath(mob *MobInstance) {
	// Calculate XP gain
	xpGain := CalculateXPGain(p.Level, mob.Level)

	// Award XP to the player
	p.GainXP(xpGain)

	// Award gold to the player
	goldGain := mob.Level * 5 // Simple formula: 5 gold per mob level
	p.Gold += goldGain

	// Send death message to player
	p.Conn.Write([]byte(fmt.Sprintf("\r\nYou have slain the %s!\r\n", mob.ShortDescription)))
	p.Conn.Write([]byte(fmt.Sprintf("You gain %d experience and %d gold.\r\n> ", xpGain, goldGain)))

	// Broadcast the death to other players in the room
	BroadcastCombatMessage(fmt.Sprintf("%s has slain the %s!",
		p.Name, mob.ShortDescription), p.Room, p)

	// Remove the mob from the room (this would be handled by a mob management function)
	// For now, we'll just log it
	log.Printf("Mob %s killed by player %s", mob.ShortDescription, p.Name)

	// Exit combat
	p.ExitCombat()

	// Schedule mob respawn would be handled by a mob management function
	// For now, we'll just log it
	log.Printf("Scheduling respawn for mob %s", mob.ShortDescription)
}

// Die handles the death of a player
func (p *Player) Die(killer *MobInstance) {
	// Set player as dead
	p.IsDead = true

	// Exit combat
	p.ExitCombat()

	// Send death message to player
	p.Conn.Write([]byte(fmt.Sprintf("\r\n\x1b[31mYou have been killed by the %s!\x1b[0m\r\n",
		killer.ShortDescription)))

	// Broadcast the death to other players in the room
	BroadcastCombatMessage(fmt.Sprintf("\x1b[31m%s has been killed by the %s!\x1b[0m",
		p.Name, killer.ShortDescription), p.Room, p)

	// Calculate XP loss (10% of current XP)
	xpLoss := p.XP / 10
	if xpLoss < 1 {
		xpLoss = 1 // Minimum XP loss of 1
	}

	// Apply XP loss
	p.XP -= xpLoss
	if p.XP < 0 {
		p.XP = 0
	}

	// Inform player of XP loss
	p.Conn.Write([]byte(fmt.Sprintf("You lose %d experience points.\r\n", xpLoss)))

	// Schedule player respawn
	go p.ScheduleRespawn()
}

// ScheduleRespawn schedules a player to respawn after a delay
func (p *Player) ScheduleRespawn() {
	// Wait for respawn time (5 seconds)
	time.Sleep(5 * time.Second)

	// Respawn the player
	p.IsDead = false
	p.HP = p.MaxHP / 2 // Respawn with half health
	p.MP = p.MaxMP / 2 // Respawn with half mana

	// Move player to Temple Altar (room 3054)
	respawnRoomID := 3054
	startRoom, err := GetRoom(respawnRoomID)
	if err != nil {
		log.Printf("Error getting respawn room: %v", err)
		// If respawn room doesn't exist, use current room
		startRoom = p.Room
	}

	if startRoom != nil {
		// Store old room for broadcasting departure
		oldRoom := p.Room

		// Remove from current room
		if p.Room != nil {
			for i, name := range GetPlayersInRoom(p.Room) {
				if name == p.Name {
					// This is a bit inefficient but works for now
					players := GetPlayersInRoom(p.Room)
					players = append(players[:i], players[i+1:]...)
					break
				}
			}

			// Broadcast departure from old room if it's different from respawn room
			if oldRoom != startRoom {
				BroadcastToRoom(fmt.Sprintf("%s's body fades away.", p.Name), oldRoom, p)
			}
		}

		// Add to respawn room
		p.Room = startRoom

		// Update player's room in database
		if err := UpdatePlayerRoom(p.Name, respawnRoomID); err != nil {
			log.Printf("Error updating player room in database: %v", err)
		}

		// Broadcast arrival to respawn room
		BroadcastToRoom(fmt.Sprintf("%s appears in a flash of divine light.", p.Name), startRoom, p)
	}

	// Send respawn message
	p.Conn.Write([]byte("\r\n\x1b[32mYou have been resurrected!\x1b[0m\r\n"))
	p.Conn.Write([]byte("Your blurred vision comes to focus and you find yourself next to the Temple Altar.\r\n> "))

	// Update player stats in database
	UpdatePlayerHPMP(p.Name, p.HP, p.MaxHP, p.MP, p.MaxMP)
	// We would need to implement UpdatePlayerXP or use an existing function
	log.Printf("Updating XP for player %s: %d", p.Name, p.XP)
}

// CalculateHitChance determines the chance to hit based on level difference
func CalculateHitChance(attackerLevel, defenderLevel int) float64 {
	baseHitChance := 0.80 // 80% base hit chance
	levelDifference := attackerLevel - defenderLevel

	// Adjust hit chance based on level difference
	hitChanceAdjustment := 0.0
	if levelDifference >= 2 {
		hitChanceAdjustment = 0.10 // +10% for 2+ levels higher
	} else if levelDifference == 1 {
		hitChanceAdjustment = 0.05 // +5% for 1 level higher
	} else if levelDifference == -1 {
		hitChanceAdjustment = -0.05 // -5% for 1 level lower
	} else if levelDifference <= -2 {
		hitChanceAdjustment = -0.10 // -10% for 2+ levels lower
	}

	finalHitChance := baseHitChance + hitChanceAdjustment

	// Ensure hit chance is within bounds
	if finalHitChance < 0.05 {
		finalHitChance = 0.05 // Minimum 5% hit chance
	} else if finalHitChance > 1.0 {
		finalHitChance = 1.0 // Maximum 100% hit chance
	}

	return finalHitChance
}

// CalculateDamage determines the damage dealt based on attacker level
func CalculateDamage(attackerLevel int) int {
	baseMultiplier := 2
	return attackerLevel * baseMultiplier
}

// BroadcastCombatMessage sends a combat message to all players in the room except the sender
func BroadcastCombatMessage(message string, room *Room, sender *Player) {
	// Make a copy of the players to avoid holding the lock while writing to connections
	var playersToNotify []*Player

	playersMutex.Lock()
	for _, p := range activePlayers {
		if p != sender && p.Room != nil && room != nil &&
			p.Room.ID == room.ID && p.Room == room {
			playersToNotify = append(playersToNotify, p)
		}
	}
	playersMutex.Unlock()

	// Now send the message to each player without holding the lock
	for _, p := range playersToNotify {
		p.Conn.Write([]byte(message + "\r\n"))
	}
}

// Add a constant for the respawn room ID
const (
	RespawnRoomID = 3001 // Temple of Midgaard (or whatever room you want as respawn point)
)

// Add function to calculate XP based on level difference
func CalculateXPGain(playerLevel, mobLevel int) int {
	// Base XP calculation
	baseXP := 100 * mobLevel

	// Calculate level difference
	levelDiff := mobLevel - playerLevel

	// Apply level modifier based on the difference
	var levelModifier float64
	switch {
	case levelDiff >= 5:
		levelModifier = 2.0 // x2 XP multiplier
	case levelDiff >= 3:
		levelModifier = 1.5 // x1.5 XP multiplier
	case levelDiff >= 2:
		levelModifier = 1.25 // x1.25 XP multiplier
	case levelDiff >= 0:
		levelModifier = 1.0 // x1.0 XP multiplier
	case levelDiff == -1:
		levelModifier = 0.75 // x0.75 XP multiplier
	case levelDiff == -2:
		levelModifier = 0.5 // x0.5 XP multiplier
	case levelDiff == -3:
		levelModifier = 0.25 // x0.25 XP multiplier
	default:
		levelModifier = 0.0 // No XP for mobs 4+ levels below player
	}

	// Calculate final XP
	return int(float64(baseXP) * levelModifier)
}
