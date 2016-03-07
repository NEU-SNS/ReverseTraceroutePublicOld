package watcher

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
)

// EventType represents a file system event
type EventType int

const (
	// Create is when a file is created
	Create EventType = iota
	// Remove is when a file is removed
	Remove
)

var (
	// ErrWatcherClosed is returned when the watcher is closed while waiting for an event
	ErrWatcherClosed = fmt.Errorf("The watcher was closed")
)

// Event represents a file system event
type Event interface {
	Name() string
	Type() EventType
}

type event struct {
	e     fsnotify.Event
	etype EventType
}

func (e event) Name() string {
	return e.e.Name
}

func (e event) Type() EventType {
	return e.etype
}

type watcher struct {
	w *fsnotify.Watcher
}

// Watcher watches a path
type Watcher interface {
	Close() error
	GetEvent() (Event, error)
}

// New creates a n
// ew watcher at the given path
func New(path string) (Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	err = w.Add(path)
	if err != nil {
		return nil, err
	}
	return watcher{w: w}, nil
}

// Close closes the Watcher
func (w watcher) Close() error {
	return w.w.Close()
}

// GetEvent gets the next create/remove event
func (w watcher) GetEvent() (Event, error) {
	for {
		select {
		case ev, ok := <-w.w.Events:
			if !ok {
				return nil, ErrWatcherClosed
			}
			switch {
			case ev.Op&fsnotify.Create == fsnotify.Create:
				return event{e: ev, etype: Create}, nil
			case ev.Op&fsnotify.Remove == fsnotify.Remove:
				return event{e: ev, etype: Remove}, nil
			default:
				// We don't care about other events for now
				continue
			}
		case err := <-w.w.Errors:
			if err == nil {
				return nil, ErrWatcherClosed
			}
			return nil, err
		}
	}
}
