package shai

import (
	"bytes"
	"io"
	"strings"
	"sync/atomic"
	"testing"
)

func TestCtrlCFilterDropsUntilEnabled(t *testing.T) {
	src := &chunkReader{
		data: []byte("foo\x03bar\x03baz"),
		step: 6,
	}
	filter := newCtrlCFilter(src)

	buf := make([]byte, 32)
	n, err := filter.Read(buf)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	first := string(buf[:n])
	if strings.ContainsRune(first, '\x03') {
		t.Fatalf("ctrl-c should not pass through before enable: got %q", first)
	}
	if first != "fooba" {
		t.Fatalf("unexpected data before enable: %q", first)
	}

	filter.Enable()

	n, err = filter.Read(buf)
	if err != nil {
		t.Fatalf("unexpected read error after enable: %v", err)
	}
	if got := string(buf[:n]); got != "r\x03baz" {
		t.Fatalf("expected Ctrl+C to flow after enable, got %q", got)
	}
}

type chunkReader struct {
	data []byte
	step int
}

func (c *chunkReader) Read(p []byte) (int, error) {
	if len(c.data) == 0 {
		return 0, io.EOF
	}
	n := c.step
	if n > len(c.data) {
		n = len(c.data)
	}
	if n > len(p) {
		n = len(p)
	}
	copy(p, c.data[:n])
	c.data = c.data[n:]
	return n, nil
}

func TestExecStartDetectorStripsMarker(t *testing.T) {
	var out bytes.Buffer
	var triggered atomic.Bool

	// The exact marker that will be output by the bootstrap script
	marker := "Shai sandbox started"
	detector := newExecStartDetector(&out, marker, func() {
		triggered.Store(true)
	})

	summaryLine := "Shai sandbox started using [ghcr.io/colony-2/shai-base:latest] as user [shai]. Resource sets: [foo, bar]\n"
	input := []byte("before\n" + summaryLine + "after\n")
	if _, err := detector.Write(input[:10]); err != nil {
		t.Fatalf("write chunk 1 failed: %v", err)
	}
	if _, err := detector.Write(input[10:]); err != nil {
		t.Fatalf("write chunk 2 failed: %v", err)
	}
	if err := detector.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if !triggered.Load() {
		t.Fatalf("exec start callback was not triggered")
	}
	want := "before\n" + summaryLine + "after\n"
	if got := out.String(); got != want {
		t.Fatalf("output should be preserved unchanged, got %q", got)
	}
}
