package sync

import (
	"testing"
)

func TestAtomicFlagDefault(t *testing.T) {
	f := NewAtomicFlag(true)
	if !f.Get() {
		t.Error("Expected true")
	}
	f2 := NewAtomicFlag(false)
	if f2.Get() {
		t.Error("Expected false")
	}
}

func TestAtomicFlagSetGet(t *testing.T) {
	f := NewAtomicFlag(true)
	if !f.Get() {
		t.Error("Expected true")
	}
	ret := f.Set(false)
	if ret {
		t.Error("Set should return false")
	}
	if f.Get() {
		t.Error("Expected false after Set(false)")
	}
}

func TestEventBusBasic(t *testing.T) {
	const (
		EvtA EventKind = iota
		EvtB
		EvtC
		EvtD
		EvtE
	)

	bus := NewEventBus()
	ready := make(chan bool, 1)
	sum := 0
	iterations := 0

	go func() {
		bus.Set(EvtA, 10)
		bus.Set(EvtB, 20)
		ready <- true
		<-ready
		bus.Set(EvtC, 30)
		bus.Set(EvtD, 40)
		ready <- true
	}()

	<-ready

	bus.Wait(func(events *EventMap) {
		iterations++
		for _, v := range *events {
			sum += v.(int)
		}
		events.Clear()
	})

	ready <- true
	<-ready

	bus.Wait(func(events *EventMap) {
		iterations++
		for _, v := range *events {
			sum += v.(int)
		}
		events.Clear()
	})

	if iterations != 2 {
		t.Errorf("Expected 2 iterations, got %d", iterations)
	}
	if sum != 100 {
		t.Errorf("Expected sum 100, got %d", sum)
	}
}

func TestEventBusPeek(t *testing.T) {
	const EvtTest EventKind = 1
	bus := NewEventBus()

	if bus.Peek(EvtTest) {
		t.Error("Should not peek empty bus")
	}

	bus.Set(EvtTest, nil)
	if !bus.Peek(EvtTest) {
		t.Error("Should peek set event")
	}
}

func TestEventBusWatchUnwatch(t *testing.T) {
	const EvtTest EventKind = 1
	bus := NewEventBus()

	bus.Unwatch(EvtTest)
	bus.Watch(EvtTest)
	// Just verify no panic
}

func TestEventBusWaitFor(t *testing.T) {
	const EvtTarget EventKind = 42
	bus := NewEventBus()
	done := make(chan bool)

	go func() {
		bus.WaitFor(EvtTarget)
		done <- true
	}()

	bus.Set(EvtTarget, "data")
	<-done
}
