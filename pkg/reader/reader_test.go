package reader

import (
	"testing"
	"time"

	fzfsync "github.com/fzf/finder/pkg/sync"
)

func TestReadFromCommand(t *testing.T) {
	strs := []string{}
	eb := fzfsync.NewEventBus()
	reader := NewReader(
		func(s []byte) bool { strs = append(strs, string(s)); return true },
		eb, false, true)

	reader.StartEventPoller()

	if eb.Peek(EvtReadNew) {
		t.Error("EvtReadNew should not be set yet")
	}

	counter := 0
	ready := func() {
		counter++
	}
	reader.Fin(reader.ReadFromCommand(`echo abc&&echo def`, nil, ready))
	if len(strs) != 2 || strs[0] != "abc" || strs[1] != "def" || counter != 1 {
		t.Errorf("%s", strs)
	}

	eb.WaitFor(EvtReadFin)

	eb.Wait(func(events *fzfsync.EventMap) {
		events.Clear()
	})

	if eb.Peek(EvtReadNew) {
		t.Error("EvtReadNew should not be set")
	}

	time.Sleep(ReaderPollIntervalMax)

	reader.StartEventPoller()

	reader.Fin(reader.ReadFromCommand(`no-such-command`, nil, ready))
	strs = []string{}
	if len(strs) > 0 || counter != 2 {
		t.Errorf("%s", strs)
	}

	if eb.Peek(EvtReadNew) {
		t.Error("Command failed. EvtReadNew should not be set")
	}
	if !eb.Peek(EvtReadFin) {
		t.Error("EvtReadFin should be set")
	}
}

func TestReadChannel(t *testing.T) {
	strs := []string{}
	eb := fzfsync.NewEventBus()
	reader := NewReader(
		func(s []byte) bool { strs = append(strs, string(s)); return true },
		eb, false, false)

	ch := make(chan string, 3)
	ch <- "hello"
	ch <- "world"
	ch <- "test"
	close(ch)

	reader.ReadChannel(ch)
	if len(strs) != 3 || strs[0] != "hello" || strs[1] != "world" || strs[2] != "test" {
		t.Errorf("Expected [hello, world, test], got %v", strs)
	}
}

func TestFeedWithDelimiter(t *testing.T) {
	strs := []string{}
	eb := fzfsync.NewEventBus()
	reader := NewReader(
		func(s []byte) bool { strs = append(strs, string(s)); return true },
		eb, false, false)

	data := "line1\nline2\nline3"
	reader.Feed(stringReader(data))
	if len(strs) != 3 || strs[0] != "line1" || strs[1] != "line2" || strs[2] != "line3" {
		t.Errorf("Expected [line1, line2, line3], got %v", strs)
	}
}

func TestFeedNilDelimiter(t *testing.T) {
	strs := []string{}
	eb := fzfsync.NewEventBus()
	reader := NewReader(
		func(s []byte) bool { strs = append(strs, string(s)); return true },
		eb, true, false)

	data := "a\x00b\x00c"
	reader.Feed(stringReader(data))
	if len(strs) != 3 || strs[0] != "a" || strs[1] != "b" || strs[2] != "c" {
		t.Errorf("Expected [a, b, c], got %v", strs)
	}
}

type strReader struct {
	data []byte
	pos  int
}

func stringReader(s string) *strReader {
	return &strReader{data: []byte(s)}
}

func (r *strReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	if r.pos >= len(r.data) {
		return n, nil
	}
	return n, nil
}
