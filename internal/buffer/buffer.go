package buffer

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/MightyToolkit/mightymonitor-agent/internal/metrics"
)

const defaultMaxSize = 10

type Buffer struct {
	path    string
	maxSize int
}

func NewBuffer(path string, maxSize int) *Buffer {
	if maxSize <= 0 {
		maxSize = defaultMaxSize
	}
	return &Buffer{
		path:    path,
		maxSize: maxSize,
	}
}

func (b *Buffer) Push(payload *metrics.Payload) error {
	if payload == nil {
		return errors.New("payload is required")
	}

	items, err := b.Flush()
	if err != nil {
		return err
	}
	items = append(items, payload)
	if len(items) > b.maxSize {
		items = items[len(items)-b.maxSize:]
	}
	return b.writeAll(items)
}

func (b *Buffer) Flush() ([]*metrics.Payload, error) {
	f, err := os.Open(b.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*metrics.Payload{}, nil
		}
		return nil, err
	}
	defer f.Close()

	return readAll(f)
}

func (b *Buffer) Clear() error {
	return b.writeAll(nil)
}

func (b *Buffer) Count() (int, error) {
	items, err := b.Flush()
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func AppendPayload(path string, payload *metrics.Payload) error {
	return NewBuffer(path, defaultMaxSize).Push(payload)
}

func (b *Buffer) writeAll(items []*metrics.Payload) error {
	if err := os.MkdirAll(filepath.Dir(b.path), 0o700); err != nil {
		return err
	}

	f, err := os.OpenFile(b.path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, item := range items {
		if err := enc.Encode(item); err != nil {
			return err
		}
	}
	return f.Sync()
}

func readAll(r io.Reader) ([]*metrics.Payload, error) {
	scanner := bufio.NewScanner(r)
	items := make([]*metrics.Payload, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var payload metrics.Payload
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			continue
		}
		items = append(items, &payload)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
