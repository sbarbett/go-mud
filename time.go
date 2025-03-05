package main

import (
	"fmt"
	"log"
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
		log.Println("TimeManager is already running")
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

	log.Println("TimeManager started successfully")
}

// Stop halts all time-related processing
func (tm *TimeManager) Stop() {
	if !tm.running {
		return
	}

	close(tm.stopChan)
	tm.running = false
	log.Println("TimeManager stopped")
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
	log.Println("Registered new tick function")
}

// RegisterPulseFunc adds a function to be called every pulse (1 second)
func (tm *TimeManager) RegisterPulseFunc(f func()) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.pulseFuncs = append(tm.pulseFuncs, f)
	log.Println("Registered new pulse function")
}

// RegisterHeartbeatFunc adds a function to be called every heartbeat (100ms)
func (tm *TimeManager) RegisterHeartbeatFunc(f func()) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.heartFuncs = append(tm.heartFuncs, f)
	log.Println("Registered new heartbeat function")
}

// executeTickFuncs runs all registered tick functions
func (tm *TimeManager) executeTickFuncs() {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	log.Println("Executing tick functions")
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
	funcCount := len(tm.pulseFuncs)
	log.Printf("[DEBUG] Executing %d pulse functions", funcCount)
	defer tm.mu.RUnlock()

	for i, f := range tm.pulseFuncs {
		go func(fn func(), idx int) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic in pulse function %d: %v", idx, r)
				}
			}()
			log.Printf("[DEBUG] Starting pulse function %d", idx)
			fn()
			log.Printf("[DEBUG] Completed pulse function %d", idx)
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
