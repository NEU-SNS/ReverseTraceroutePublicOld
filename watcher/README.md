# watcher
--
    import "github.com/NEU-SNS/ReverseTraceroute/watcher"

Package watcher watches a directory and reports back file system events

## Usage

```go
var (
	// ErrWatcherClosed is returned when the watcher is closed while waiting for an event
	ErrWatcherClosed = fmt.Errorf("The watcher was closed")
)
```

#### type Event

```go
type Event interface {
	Name() string
	Type() EventType
}
```

Event represents a file system event

#### type EventType

```go
type EventType int
```

EventType represents a file system event

```go
const (
	// Create is when a file is created
	Create EventType = iota
	// Remove is when a file is removed
	Remove
)
```

#### type Watcher

```go
type Watcher interface {
	Close() error
	GetEvent(chan struct{}) (Event, error)
}
```

Watcher watches a path

#### func  New

```go
func New(path string) (Watcher, error)
```
New creates a new watcher at the given path
