package sync

import "sync"

type EventKind int

type EventMap map[EventKind]any

func (em *EventMap) Clear() {
	for k := range *em {
		delete(*em, k)
	}
}

type EventBus struct {
	pending EventMap
	cond    *sync.Cond
	muted   map[EventKind]bool
}

func NewEventBus() *EventBus {
	return &EventBus{
		pending: make(EventMap),
		cond:    sync.NewCond(&sync.Mutex{}),
		muted:   make(map[EventKind]bool),
	}
}

func (eb *EventBus) Set(kind EventKind, data any) {
	eb.cond.L.Lock()
	eb.pending[kind] = data
	if !eb.muted[kind] {
		eb.cond.Broadcast()
	}
	eb.cond.L.Unlock()
}

func (eb *EventBus) Wait(fn func(*EventMap)) {
	eb.cond.L.Lock()
	if len(eb.pending) == 0 {
		eb.cond.Wait()
	}
	fn(&eb.pending)
	eb.cond.L.Unlock()
}

func (eb *EventBus) Peek(kind EventKind) bool {
	eb.cond.L.Lock()
	_, found := eb.pending[kind]
	eb.cond.L.Unlock()
	return found
}

func (eb *EventBus) Watch(kinds ...EventKind) {
	eb.cond.L.Lock()
	for _, k := range kinds {
		delete(eb.muted, k)
	}
	eb.cond.L.Unlock()
}

func (eb *EventBus) Unwatch(kinds ...EventKind) {
	eb.cond.L.Lock()
	for _, k := range kinds {
		eb.muted[k] = true
	}
	eb.cond.L.Unlock()
}

func (eb *EventBus) WaitFor(kind EventKind) {
	waiting := true
	for waiting {
		eb.Wait(func(events *EventMap) {
			for k := range *events {
				if k == kind {
					waiting = false
					return
				}
			}
		})
	}
}
