package connectionmanager

import (
	"sync"

	"github.com/tozny/utils-go/logging"
)

// Initializer is the interface that initializes a connection of some kind.
type Initializer interface {
	Initialize()
}

// Closer is the interface that gracefully closes a connection of some kind.
type Closer interface {
	Close()
}

// InitializerCloser is the interface that both initializes and gracefully closes a
// connection of some kind.
type InitializerCloser interface {
	Initializer
	Closer
}

// CloseFunc is a function that gracefully shuts down a connection as a side effect.
type CloseFunc func()

// Manager allows multiple items needing initialization or shutdown to be
// managed as a group.
//
// Initialization and Close of items are managed independently of each other. Once
// created the manager can accept any number of items supporting initialization,
// close, or both. The ManageInitialization, ManageClose, and ManageLifecycle methods can
// be called as many times as needed in any order to add managed items. They are variadic
// functions, so multiple items can be added in a single call.
//
// Initialization items will immediately start initialization in a separate go routine
// once the item is added to the lifecycle.Manager. An internally managed sync.WaitGroup
// is made available. Calling WG.Wait() on the lifecycle.Manager will block the current
// go routine until all initialization functions are complete.
//
// Closers are queued up internally running only when the lifecycle.Manager's Close method
// is called. The Manager runs each Close method in a separate go routine and blocks
// until all are complete.
type Manager struct {
	closerChan chan CloseFunc
	Close      CloseFunc
	WG         sync.WaitGroup
}

// NewManager initializes a new lifecycle.Manager object that can be used
// to manage the lifecycle of items such as connections to a database.
func NewManager(logger logging.Logger) Manager {
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
			go func(c func()) {
				c()
				stopwg.Done()
			}(c)
		}
	}()
	return Manager{
		closerChan: closerChan,
		Close: func() {
			shutdown <- struct{}{}
			stopwg.Wait()
		},
	}
}

// ManageInitialization allows the lifecycle manager to accept any number of items
// matching the Initializer interface and initializes each in parallel. The wait
// group is managed to allow callers to block until all managed initialization
// methods are complete.
func (m *Manager) ManageInitialization(initializers ...Initializer) {
	for _, initializer := range initializers {
		m.WG.Add(1)
		go func(i Initializer) {
			i.Initialize()
			m.WG.Done()
		}(initializer)
	}
}

// ManageClose allows the lifecycle manager to accept any number of items matching
// the Closer interface. It queues them up internally. When Close is called on
// the lifecycle manager, all queued Close methods are executed in parallel.
// The close method block until all managed Closers are complete.
func (m *Manager) ManageClose(closers ...Closer) {
	for _, closer := range closers {
		m.closerChan <- closer.Close
	}
}

// ManageLifecycle accepts any number of items matching the InitializerCloser
// interface and manages both an item's initialization and close.
//
// The close method of the managed item is queued first to ensure it is present
// before running the item's initialization which happens immediately when calling
// the ManageInitialization method. Without this order, close may not get managed
// if something interupts before initialization is complete.
func (m *Manager) ManageLifecycle(initializerClosers ...InitializerCloser) {
	for _, ic := range initializerClosers {
		m.ManageClose(ic)
		m.ManageInitialization(ic)
	}
}
