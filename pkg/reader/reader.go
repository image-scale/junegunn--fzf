package reader

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	fzfsync "github.com/fzf/finder/pkg/sync"
)

const (
	ReaderBufferSize      = 64 * 1024
	ReaderSlabSize        = 128 * 1024
	ReaderPollIntervalMin = 10 * time.Millisecond
	ReaderPollIntervalStep = 5 * time.Millisecond
	ReaderPollIntervalMax = 50 * time.Millisecond
)

const (
	EvtReadNew fzfsync.EventKind = 100
	EvtReadFin fzfsync.EventKind = 101
	EvtReady   fzfsync.EventKind = 102
)

type Reader struct {
	pusher   func([]byte) bool
	eventBox *fzfsync.EventBus
	delimNil bool
	event    int32
	finChan  chan bool
	mutex    sync.Mutex
	killed   bool
	termFunc func()
	command  *string
	wait     bool
}

func NewReader(pusher func([]byte) bool, eventBox *fzfsync.EventBus, delimNil bool, wait bool) *Reader {
	return &Reader{
		pusher:   pusher,
		eventBox: eventBox,
		delimNil: delimNil,
		event:    int32(EvtReady),
		finChan:  make(chan bool, 1),
		termFunc: func() { os.Stdin.Close() },
		wait:     wait,
	}
}

func (r *Reader) StartEventPoller() {
	go func() {
		ptr := &r.event
		pollInterval := ReaderPollIntervalMin
		for {
			if atomic.CompareAndSwapInt32(ptr, int32(EvtReadNew), int32(EvtReady)) {
				r.eventBox.Set(EvtReadNew, (*string)(nil))
				pollInterval = ReaderPollIntervalMin
			} else if atomic.LoadInt32(ptr) == int32(EvtReadFin) {
				if r.wait {
					r.finChan <- true
				}
				return
			} else {
				pollInterval += ReaderPollIntervalStep
				if pollInterval > ReaderPollIntervalMax {
					pollInterval = ReaderPollIntervalMax
				}
			}
			time.Sleep(pollInterval)
		}
	}()
}

func (r *Reader) Fin(success bool) {
	atomic.StoreInt32(&r.event, int32(EvtReadFin))
	if r.wait {
		<-r.finChan
	}

	r.mutex.Lock()
	ret := r.command
	if success || r.killed {
		ret = nil
	}
	r.mutex.Unlock()

	r.eventBox.Set(EvtReadFin, ret)
}

func (r *Reader) Terminate() {
	r.mutex.Lock()
	r.killed = true
	if r.termFunc != nil {
		r.termFunc()
		r.termFunc = nil
	}
	r.mutex.Unlock()
}

func (r *Reader) ReadChannel(inputChan chan string) bool {
	for {
		item, more := <-inputChan
		if !more {
			break
		}
		if r.pusher([]byte(item)) {
			atomic.StoreInt32(&r.event, int32(EvtReadNew))
		}
	}
	return true
}

func (r *Reader) Feed(src io.Reader) {
	delim := byte('\n')
	if r.delimNil {
		delim = '\000'
	}

	slab := make([]byte, ReaderSlabSize)
	leftover := []byte{}
	var err error
	for {
		n := 0
		scope := slab[:min(len(slab), ReaderBufferSize)]
		for range 100 {
			n, err = src.Read(scope)
			if n > 0 || err != nil {
				break
			}
		}

		if n == 0 {
			break
		}

		buf := slab[:n]
		slab = slab[n:]

		for len(buf) > 0 {
			if i := bytes.IndexByte(buf, delim); i >= 0 {
				slice := buf[:i+1]
				buf = buf[i+1:]
				slice = slice[:len(slice)-1]
				if len(leftover) > 0 {
					slice = append(leftover, slice...)
					leftover = []byte{}
				}
				if (err == nil || len(slice) > 0) && r.pusher(slice) {
					atomic.StoreInt32(&r.event, int32(EvtReadNew))
				}
			} else {
				leftover = append(leftover, buf...)
				break
			}
		}

		if err == io.EOF {
			leftover = append(leftover, buf...)
			break
		}

		if len(slab) == 0 {
			slab = make([]byte, ReaderSlabSize)
		}
	}
	if len(leftover) > 0 && r.pusher(leftover) {
		atomic.StoreInt32(&r.event, int32(EvtReadNew))
	}
}

func (r *Reader) ReadFromStdin() bool {
	r.Feed(os.Stdin)
	return true
}

func (r *Reader) ReadFromCommand(command string, environ []string, signalReady func()) bool {
	r.mutex.Lock()

	r.killed = false
	r.termFunc = nil
	r.command = &command

	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	cmd := exec.Command(shell, "-c", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if environ != nil {
		cmd.Env = environ
	}
	execOut, err := cmd.StdoutPipe()
	if err != nil || cmd.Start() != nil {
		signalReady()
		r.mutex.Unlock()
		return false
	}

	r.termFunc = func() {
		execOut.Close()
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	signalReady()
	r.mutex.Unlock()

	r.Feed(execOut)
	return cmd.Wait() == nil
}
