/*
 * time.go
 *
 * This file implements the time management system for the MUD.
 * It defines the TimeManager struct which handles game time events at
 * different intervals (ticks, pulses, and heartbeats). The file provides
 * functionality for registering callback functions to be executed at these
 * intervals, allowing for scheduled events like combat rounds, regeneration,
 * and world updates to occur at appropriate times.
 */

package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TimeManager handles all game time-related events
type TimeManager struct {
	// Channels for each time interval
	tickChan  chan struct{}
	pulseChan chan struct{}
	heartChan chan struct{}

	// Function registries for each time interval
	tickFuncs  []func()
	pulseFuncs []func()
	heartFuncs []func()

	// Mutex for thread safety when modifying function lists
	mu sync.RWMutex

	// Control channel to stop all goroutines
	stopChan chan struct{}

	// Track if the manager is running
	running bool
}

// NewTimeManager creates a new TimeManager instance
func NewTimeManager() *TimeManager {
	return &TimeManager{
		tickChan:   make(chan struct{}),
		pulseChan:  make(chan struct{}),
		heartChan:  make(chan struct{}),
		tickFuncs:  []func(){},
		pulseFuncs: []func(){},
		heartFuncs: []func(){},
		stopChan:   make(chan struct{}),
		running:    false,
	}
}

// Start begins the time management system
func (tm *TimeManager) Start() {
	if tm.running {
		//log.Println("TimeManager is already running")
		return
	}

	tm.running = true

	// Start the heartbeat (100ms)
	go func() {
		heartTicker := time.NewTicker(100 * time.Millisecond)
		defer heartTicker.Stop()

		for {
			select {
			case <-heartTicker.C:
				tm.heartChan <- struct{}{}
			case <-tm.stopChan:
				return
			}
		}
	}()

	// Start the pulse (1 second)
	go func() {
		pulseTicker := time.NewTicker(1 * time.Second)
		defer pulseTicker.Stop()

		for {
			select {
			case <-pulseTicker.C:
				tm.pulseChan <- struct{}{}
			case <-tm.stopChan:
				return
			}
		}
	}()

	// Start the tick (1 minute)
	go func() {
		tickTicker := time.NewTicker(60 * time.Second)
		defer tickTicker.Stop()

		for {
			select {
			case <-tickTicker.C:
				tm.tickChan <- struct{}{}
			case <-tm.stopChan:
				return
			}
		}
	}()

	// Process events from the channels
	go tm.processEvents()

	//log.Println("TimeManager started successfully")
}

// Stop halts all time-related processing
func (tm *TimeManager) Stop() {
	if !tm.running {
		return
	}

	close(tm.stopChan)
	tm.running = false
	//log.Println("TimeManager stopped")
}

// processEvents handles events from all time channels
func (tm *TimeManager) processEvents() {
	for {
		select {
		case <-tm.heartChan:
			tm.executeHeartbeatFuncs()
		case <-tm.pulseChan:
			tm.executePulseFuncs()
		case <-tm.tickChan:
			tm.executeTickFuncs()
		case <-tm.stopChan:
			return
		}
	}
}

// RegisterTickFunc adds a function to be called every tick (1 minute)
func (tm *TimeManager) RegisterTickFunc(f func()) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.tickFuncs = append(tm.tickFuncs, f)
	//log.Println("Registered new tick function")
}

// RegisterPulseFunc adds a function to be called every pulse (1 second)
func (tm *TimeManager) RegisterPulseFunc(f func()) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.pulseFuncs = append(tm.pulseFuncs, f)
	//log.Println("Registered new pulse function")
}

// RegisterHeartbeatFunc adds a function to be called every heartbeat (100ms)
func (tm *TimeManager) RegisterHeartbeatFunc(f func()) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.heartFuncs = append(tm.heartFuncs, f)
	//log.Println("Registered new heartbeat function")
}

// executeTickFuncs runs all registered tick functions
func (tm *TimeManager) executeTickFuncs() {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	//log.Println("Executing tick functions")
	for _, f := range tm.tickFuncs {
		// Execute each function in its own goroutine to prevent blocking
		go func(fn func()) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic in tick function: %v", r)
				}
			}()
			fn()
		}(f)
	}
}

// executePulseFuncs runs all registered pulse functions
func (tm *TimeManager) executePulseFuncs() {
	tm.mu.RLock()
	//funcCount := len(tm.pulseFuncs)
	//log.Printf("[DEBUG] Executing %d pulse functions", funcCount)
	defer tm.mu.RUnlock()

	for i, f := range tm.pulseFuncs {
		go func(fn func(), idx int) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic in pulse function %d: %v", idx, r)
				}
			}()
			//log.Printf("[DEBUG] Starting pulse function %d", idx)
			fn()
			//log.Printf("[DEBUG] Completed pulse function %d", idx)
		}(f, i)
	}
}

// executeHeartbeatFuncs runs all registered heartbeat functions
func (tm *TimeManager) executeHeartbeatFuncs() {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	for _, f := range tm.heartFuncs {
		go func(fn func()) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic in heartbeat function: %v", r)
				}
			}()
			fn()
		}(f)
	}
}

// Debug functions to help monitor the time system

// DebugTick prints a message when a tick occurs
func DebugTick() {
	fmt.Println("TICK: Game time advanced by 1 minute")
}

// DebugPulse prints a message when a pulse occurs
func DebugPulse() {
	fmt.Println("PULSE: 1 second has passed")
}

// DebugHeartbeat prints a message when a heartbeat occurs
func DebugHeartbeat() {
	fmt.Println("HEARTBEAT: 100ms has passed")
}

// ResetDoors closes all doors in the game world
func ResetDoors() {
	//log.Println("[TIME] Resetting doors to closed state (15-minute interval)")

	// Track which doors have already been processed to avoid duplicates
	processedDoors := make(map[string]bool)

	// Iterate through all rooms
	for roomID, room := range rooms {
		// Check each exit for doors
		for direction, exit := range room.Exits {
			if exit.Door != nil && !exit.Door.Closed {
				// Create a unique key for this door connection
				var destRoomID int
				switch exitID := exit.ID.(type) {
				case int:
					destRoomID = exitID
				case string:
					// Handle cross-area references
					roomInfo := strings.Split(exitID, ":")
					if len(roomInfo) != 2 {
						continue
					}
					var err error
					destRoomID, err = strconv.Atoi(roomInfo[1])
					if err != nil {
						continue
					}
				default:
					continue
				}

				// Create a unique key for this door (smaller room ID first)
				doorKey := ""
				if roomID < destRoomID {
					doorKey = fmt.Sprintf("%d:%d", roomID, destRoomID)
				} else {
					doorKey = fmt.Sprintf("%d:%d", destRoomID, roomID)
				}

				// Skip if we've already processed this door
				if processedDoors[doorKey] {
					continue
				}

				// Mark this door as processed
				processedDoors[doorKey] = true

				// Close the door in both rooms
				exit.Door.Closed = true

				// Use SynchronizeDoor to update the connected room
				SynchronizeDoor(roomID, direction, true)

				// Notify players in this room
				playersMutex.Lock()
				for _, p := range activePlayers {
					if p.Room != nil && p.Room.ID == roomID {
						p.Send(fmt.Sprintf("The %s closes.", exit.Door.ShortDescription))
					}
				}
				playersMutex.Unlock()
			}
		}
	}
}

// ResetMobs respawns mobs according to the reset configuration
func ResetMobs() {
	//log.Println("[TIME] Resetting mobs in the world (15-minute interval)")
	ProcessMobResets()
}

// AutoSaveAllPlayers saves the progress of all active players
func AutoSaveAllPlayers() {
	playersMutex.Lock()
	defer playersMutex.Unlock()

	//log.Printf("Auto-saving progress for %d active players", len(activePlayers))

	for _, player := range activePlayers {
		player.AutoSave()
	}
}

// ScheduleResets registers all the reset functions with the time manager
func ScheduleResets(tm *TimeManager) {
	// Counter for 15-minute resets
	resetCounter := 0

	// Counter for 5-minute auto-saves
	saveCounter := 0

	// Register a tick function to handle resets every 15 minutes
	tm.RegisterTickFunc(func() {
		resetCounter++
		saveCounter++

		// Process resets every 15 minutes (15 ticks)
		if resetCounter >= 15 {
			resetCounter = 0

			// Reset doors
			ResetDoors()

			// Reset mobs
			ResetMobs()
		}

		// Auto-save player progress every 5 minutes (5 ticks)
		if saveCounter >= 5 {
			saveCounter = 0

			// Save all players' progress
			AutoSaveAllPlayers()
		}
	})
}
