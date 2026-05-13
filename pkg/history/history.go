package history

import (
	"errors"
	"os"
	"strings"
)

type History struct {
	path     string
	Lines    []string
	modified map[int]string
	maxSize  int
	cursor   int
}

func NewHistory(path string, maxSize int) (*History, error) {
	fmtError := func(e error) error {
		if os.IsPermission(e) {
			return errors.New("permission denied: " + path)
		}
		return errors.New("invalid history file: " + e.Error())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			data = []byte{}
			if err := os.WriteFile(path, data, 0600); err != nil {
				return nil, fmtError(err)
			}
		} else {
			return nil, fmtError(err)
		}
	}
	lines := strings.Split(strings.Trim(string(data), "\n"), "\n")
	if len(lines[len(lines)-1]) > 0 {
		lines = append(lines, "")
	}
	return &History{
		path:     path,
		maxSize:  maxSize,
		Lines:    lines,
		modified: make(map[int]string),
		cursor:   len(lines) - 1,
	}, nil
}

func (h *History) Append(line string) error {
	if len(line) == 0 {
		return nil
	}

	lines := append(h.Lines[:len(h.Lines)-1], line)
	if len(lines) > h.maxSize {
		lines = lines[len(lines)-h.maxSize:]
	}
	h.Lines = append(lines, "")
	return os.WriteFile(h.path, []byte(strings.Join(h.Lines, "\n")), 0600)
}

func (h *History) Override(str string) {
	if h.cursor == len(h.Lines)-1 {
		h.Lines[h.cursor] = str
	} else if h.cursor < len(h.Lines)-1 {
		h.modified[h.cursor] = str
	}
}

func (h *History) Current() string {
	if str, prs := h.modified[h.cursor]; prs {
		return str
	}
	return h.Lines[h.cursor]
}

func (h *History) Previous() string {
	if h.cursor > 0 {
		h.cursor--
	}
	return h.Current()
}

func (h *History) Next() string {
	if h.cursor < len(h.Lines)-1 {
		h.cursor++
	}
	return h.Current()
}
