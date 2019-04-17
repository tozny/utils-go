package utils

import (
	"log"
	"sync"
)

// Initializer provides an interface an object to provide methods for startup and shutdown.
type Initializer interface {
	Initialize()
	Close()
}

// CloseFunc is a function that gracefully shuts down a connection as a side effect.
type CloseFunc func()

// ConnectionManager manages initialization and shutdown of log lived connections.
// Each connection object must match the Initializer interface. Initialization happens
// in parallel. The waitgroup can be used to wait util all connections are initialized.
// Close is a method allowing the proper shutdown of all connections.
type ConnectionManager struct {
	closerChan chan CloseFunc
	Close      CloseFunc
	WG         sync.WaitGroup
}

// NewConnectionManager initializes a new ConnectionManager object that can be used
// to manage the life of long lived remote connections such as to a database.
func NewConnectionManager(logger *log.Logger) ConnectionManager {
	closerChan := make(chan CloseFunc)
	shutdown := make(chan struct{})
	var stopwg sync.WaitGroup
	go func() {
		closers := []CloseFunc{}
	loop:
		for {
			select {
			case <-shutdown:
				logger.Println("Shutting Down")
				break loop
			case c := <-closerChan:
				stopwg.Add(1)
				closers = append(closers, c)
			}
		}

		for _, c := range closers {
			go func() {
				c()
				stopwg.Done()
			}()
			stopwg.Wait()
		}
	}()
	return ConnectionManager{
		closerChan: closerChan,
		Close: func() {
			shutdown <- struct{}{}
		},
	}
}

// DoInit initializes an object matching the Initializer interface, setting its close
// operation to run when the Connection Manager's Close method is called.
func (cm *ConnectionManager) DoInit(initializer Initializer) {
	cm.WG.Add(1)
	go func() {
		initializer.Initialize()
		cm.closerChan <- initializer.Close
		cm.WG.Done()
	}()
}
