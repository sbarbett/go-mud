/*
 * player.go
 *
 * This file implements the player system for the MUD.
 * It defines the Player struct and associated methods for managing player
 * characters, including combat mechanics, attribute management, experience
 * and leveling, communication, and state management. The file handles all
 * aspects of player interaction with the game world, from basic movement
 * to complex combat calculations and death/respawn mechanics.
 */

package main

import (
	"fmt"
	"log"
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
	Title string // Player's custom title
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

	// Color preferences
	ColorEnabled bool // Whether ANSI colors are enabled for this player
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

// Send sends a message to the player with color processing
func (p *Player) Send(message string) {
	// Don't send empty messages
	if message == "" {
		return
	}

	// Process color codes
	processedMessage := ProcessColors(message, p.ColorEnabled)

	// Ensure the message ends with a newline
	if !strings.HasSuffix(processedMessage, "\r\n") {
		processedMessage += "\r\n"
	}

	// Send the message to the player
	p.Conn.Write([]byte(processedMessage))
}

// SendType sends a message to the player with the default color for the specified message type
func (p *Player) SendType(message string, messageType string) {
	colorizedMessage := ColorizeByType(message, messageType)
	p.Send(colorizedMessage)
}

func BroadcastToRoom(message string, room *Room, sender *Player) {
	playersMutex.Lock()
	defer playersMutex.Unlock()

	for _, p := range activePlayers {
		if p != sender && p.Room != nil && room != nil &&
			p.Room.ID == room.ID && p.Room == room {
			p.Send(message)
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

// ExecuteAttack handles a player's attack against a mob
func (p *Player) ExecuteAttack() {
	// Check if player is in combat and has a valid target
	if !p.InCombat || p.Target == nil {
		return
	}

	// Check if target is still alive
	if p.Target.HP <= 0 {
		p.HandleMobDeath(p.Target)
		return
	}

	// Calculate hit chance
	hitChance := CalculateHitChance(p.Level, p.Target.Level)
	hitRoll := rng.Float64()

	// Check if attack misses
	if hitRoll > hitChance {
		// Attack missed
		missMessage := fmt.Sprintf("You miss %s.", p.Target.ShortDescription)
		p.SendType(missMessage, "combat")

		// Broadcast miss message to room
		roomMessage := fmt.Sprintf("%s misses %s.", p.Name, p.Target.ShortDescription)
		BroadcastCombatMessage(roomMessage, p.Room, p)
		return
	}

	// Check for evasion
	if ProcessEvasion(p.Target.Level, p.Level) {
		// Target evaded
		evadeMessage := fmt.Sprintf("%s evades your attack.", p.Target.ShortDescription)
		p.SendType(evadeMessage, "combat")

		// Broadcast evasion message to room
		roomMessage := fmt.Sprintf("%s evades %s's attack.", p.Target.ShortDescription, p.Name)
		BroadcastCombatMessage(roomMessage, p.Room, p)
		return
	}

	// Calculate damage
	damage := CalculateDamage(p.Level)

	// Check for critical hit
	isCritical := ProcessCriticalHit(p.Level, p.Target.Level)
	if isCritical {
		// Double damage for critical hits
		damage *= 2
	}

	// Apply damage to target
	p.Target.HP -= damage

	// Send attack message to player
	var attackMessage string
	if isCritical {
		attackMessage = fmt.Sprintf("You land a {R}CRITICAL{x} hit on %s for {R}%d{x} damage!", p.Target.ShortDescription, damage)
	} else {
		attackMessage = fmt.Sprintf("You hit %s for {R}%d{x} damage.", p.Target.ShortDescription, damage)
	}
	p.SendType(attackMessage, "combat")

	// Broadcast attack message to room
	var roomMessage string
	if isCritical {
		roomMessage = fmt.Sprintf("%s lands a CRITICAL hit on %s!", p.Name, p.Target.ShortDescription)
	} else {
		roomMessage = fmt.Sprintf("%s hits %s.", p.Name, p.Target.ShortDescription)
	}
	BroadcastCombatMessage(roomMessage, p.Room, p)

	// Check if target died from the attack
	if p.Target.HP <= 0 {
		p.HandleMobDeath(p.Target)
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
		evadeMessage := fmt.Sprintf("The %s swings at you, but you evade just in time!", attacker.ShortDescription)
		p.SendType(evadeMessage, "combat")

		// Broadcast the evasion to other players in the room
		roomMessage := fmt.Sprintf("The %s swings at %s, but they evade just in time!", attacker.ShortDescription, p.Name)
		BroadcastCombatMessage(roomMessage, p.Room, p)
		return
	}

	// Calculate hit chance for the mob using the utility function
	finalHitChance := CalculateHitChance(attacker.Level, p.Level)

	// Roll to hit
	hitRoll := rng.Float64()

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
		var attackMessage string
		if isCritical {
			attackMessage = fmt.Sprintf("The %s lands a {R}CRITICAL HIT{x} on you for {R}%d{x} damage!", attacker.ShortDescription, damage)
		} else {
			attackMessage = fmt.Sprintf("The %s strikes you for {R}%d{x} damage.", attacker.ShortDescription, damage)
		}
		p.SendType(attackMessage, "combat")

		// Broadcast the attack to other players in the room
		var roomMessage string
		if isCritical {
			roomMessage = fmt.Sprintf("The %s lands a CRITICAL HIT on %s for %d damage!", attacker.ShortDescription, p.Name, damage)
		} else {
			roomMessage = fmt.Sprintf("The %s strikes %s for %d damage.", attacker.ShortDescription, p.Name, damage)
		}
		BroadcastCombatMessage(roomMessage, p.Room, p)

		// Check if player died from the attack
		if p.HP <= 0 {
			p.Die(attacker)
		}
	} else {
		// Miss
		missMessage := fmt.Sprintf("The %s swings at you but misses!", attacker.ShortDescription)
		p.SendType(missMessage, "combat")

		// Broadcast the miss to other players in the room
		roomMessage := fmt.Sprintf("The %s swings at %s but misses!", attacker.ShortDescription, p.Name)
		BroadcastCombatMessage(roomMessage, p.Room, p)
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

// HandleMobDeath processes a mob's death
func (p *Player) HandleMobDeath(mob *MobInstance) {
	// Exit combat
	p.ExitCombat()

	// Calculate XP gain
	xpGain := CalculateXPGain(p.Level, mob.Level)
	p.GainXP(xpGain)

	// Send death message to player
	deathMessage := fmt.Sprintf("You have slain %s!", mob.ShortDescription)
	p.SendType(deathMessage, "combat")

	// Send XP gain message
	xpMessage := fmt.Sprintf("You gain {G}%d{x} experience points.", xpGain)
	p.Send(xpMessage)

	// Broadcast death message to room
	roomMessage := fmt.Sprintf("%s has slain %s!", p.Name, mob.ShortDescription)
	BroadcastCombatMessage(roomMessage, p.Room, p)

	// Remove the mob from the world
	RemoveMobFromRoom(mob)
}

// Die handles player death
func (p *Player) Die(killer *MobInstance) {
	// Set the player's death state
	p.IsDead = true
	p.HP = 0
	p.ExitCombat()

	// Notify the player of their death
	deathMessage := fmt.Sprintf("You have been killed by %s!", killer.ShortDescription)
	p.SendType(deathMessage, "death")

	// Broadcast the death to the room
	roomMessage := fmt.Sprintf("%s has been killed by %s!", p.Name, killer.ShortDescription)
	BroadcastToRoom(ColorizeByType(roomMessage, "death"), p.Room, p)

	// Provide instructions for respawning
	p.Send("{W}Type 'respawn' to return to life.{x}")

	// Schedule automatic respawn after a delay
	p.ScheduleRespawn()
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
			// No need to modify the players list since GetPlayersInRoom returns a new slice each time
			// and we're not storing the list of players in rooms anywhere

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
		arrivalMsg := fmt.Sprintf("%s appears in a flash of divine light.", p.Name)
		BroadcastToRoom(ColorizeByType(arrivalMsg, "system"), startRoom, p)
	}

	// Send respawn message
	p.SendType("You have been resurrected!", "system")
	p.Send("{C}Your blurred vision comes to focus and you find yourself next to the Temple Altar.{x}")

	// Update player stats in database
	UpdatePlayerHPMP(p.Name, p.HP, p.MaxHP, p.MP, p.MaxMP)
	UpdatePlayerXP(p.Name, p.XP, p.NextLevelXP)
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
	playersMutex.Lock()
	defer playersMutex.Unlock()

	colorizedMessage := ColorizeByType(message, "combat")

	for _, p := range activePlayers {
		if p != sender && p.Room != nil && room != nil &&
			p.Room.ID == room.ID && p.Room == room {
			p.Send(colorizedMessage)
		}
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
