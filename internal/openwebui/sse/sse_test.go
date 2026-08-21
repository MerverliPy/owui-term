package sse

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

// fixture returns the real captured SSE stream (see docs/api-notes.md).
func fixture(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../testdata/stream-0.11.0.sse")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return b
}

// collect reads every event (and the final error) from r.
func collect(t *testing.T, r *Reader) []Event {
	t.Helper()
	var evs []Event
	for {
		ev, err := r.Next()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				t.Fatalf("unexpected read error: %v", err)
			}
			return evs
		}
		evs = append(evs, ev)
	}
}

// chunkReader caps each Read at size bytes to simulate fragmented network reads.
type chunkReader struct {
	r    io.Reader
	size int
}

func (cr *chunkReader) Read(p []byte) (int, error) {
	if len(p) > cr.size {
		p = p[:cr.size]
	}
	return cr.r.Read(p)
}

func TestCapturedFixture(t *testing.T) {
	evs := collect(t, NewReader(strings.NewReader(string(fixture(t)))))
	if len(evs) != 4 {
		t.Fatalf("got %d events, want 4 (3 chunks + [DONE])", len(evs))
	}

	var chunks []CompletionChunk
	done := 0
	for _, ev := range evs {
		if ev.IsDone() {
			done++
			continue
		}
		var c CompletionChunk
		if err := ev.DecodeJSON(&c); err != nil {
			t.Fatalf("decode chunk: %v", err)
		}
		chunks = append(chunks, c)
	}

	if done != 1 {
		t.Errorf("found %d [DONE] events, want 1", done)
	}
	if len(chunks) != 3 {
		t.Fatalf("got %d chunks, want 3", len(chunks))
	}

	// First chunk: partial content "P", no finish_reason.
	if chunks[0].Model != "qwen2.5:1.5b" {
		t.Errorf("chunk0.Model = %q", chunks[0].Model)
	}
	if chunks[0].Choices[0].Delta.Content != "P" || chunks[0].Choices[0].FinishReason != "" {
		t.Errorf("chunk0 delta = %+v", chunks[0].Choices[0])
	}
	// Second chunk: continuation "ONG".
	if chunks[1].Choices[0].Delta.Content != "ONG" {
		t.Errorf("chunk1 content = %q", chunks[1].Choices[0].Delta.Content)
	}
	// Third chunk: finish_reason=stop + usage.
	if chunks[2].Choices[0].FinishReason != "stop" {
		t.Errorf("chunk2 finish_reason = %q", chunks[2].Choices[0].FinishReason)
	}
	if chunks[2].Usage == nil || chunks[2].Usage.TotalTokens != 38 || chunks[2].Usage.PromptTokens != 35 {
		t.Errorf("chunk2 usage = %+v", chunks[2].Usage)
	}
}

// TestFragmentedRead feeds the fixture one byte / a few bytes at a time to
// prove the reader is tolerant of arbitrary network fragmentation.
func TestFragmentedRead(t *testing.T) {
	raw := fixture(t)
	for _, size := range []int{1, 2, 3, 7, 16} {
		r := NewReader(&chunkReader{r: strings.NewReader(string(raw)), size: size})
		if evs := collect(t, r); len(evs) != 4 {
			t.Errorf("size=%d: got %d events, want 4", size, len(evs))
		}
	}
}

func TestDone(t *testing.T) {
	evs := collect(t, NewReader(strings.NewReader("data: [DONE]\n\n")))
	if len(evs) != 1 || !evs[0].IsDone() {
		t.Fatalf("events = %+v, want single [DONE]", evs)
	}
}

func TestErrorEventDelivered(t *testing.T) {
	body := "data: {\"error\":{\"message\":\"boom\",\"code\":\"invalid_request\"}}\n\n"
	evs := collect(t, NewReader(strings.NewReader(body)))
	if len(evs) != 1 {
		t.Fatalf("got %d events, want 1", len(evs))
	}
	var errObj struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := evs[0].DecodeJSON(&errObj); err != nil {
		t.Fatalf("decode error event: %v", err)
	}
	if errObj.Error.Message != "boom" {
		t.Errorf("error message = %q, want %q", errObj.Error.Message, "boom")
	}
}

func TestMalformedJSONIsDelivered(t *testing.T) {
	// Parser must not choke on malformed data; it just delivers the raw line.
	body := "data: {not json}\n\ndata: [DONE]\n\n"
	evs := collect(t, NewReader(strings.NewReader(body)))
	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2", len(evs))
	}
	if evs[0].Data != "{not json}" {
		t.Errorf("data = %q", evs[0].Data)
	}
}

func TestCommentsAndUnknownFieldsIgnored(t *testing.T) {
	body := ": keep-alive\n" + // comment
		"event: message\n" + // unknown field (ignored)
		"data: {\"x\":1}\n" + // real data
		"id: 5\n" + // unknown field (ignored)
		"\n" + // end event
		"data: [DONE]\n\n"
	evs := collect(t, NewReader(strings.NewReader(body)))
	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2", len(evs))
	}
	if evs[0].Data != `{"x":1}` {
		t.Errorf("data = %q", evs[0].Data)
	}
}

func TestNoTrailingBlankLine(t *testing.T) {
	// A single event with no trailing blank line must still be emitted.
	evs := collect(t, NewReader(strings.NewReader("data: {\"a\":1}\n")))
	if len(evs) != 1 || evs[0].Data != `{"a":1}` {
		t.Fatalf("events = %+v", evs)
	}
}

func TestIncompleteFinalLineDropped(t *testing.T) {
	// A data line cut short by EOF must NOT be emitted as a complete event.
	evs := collect(t, NewReader(strings.NewReader("data: [DONE]\n\ndata: {\"cut")))
	if len(evs) != 1 || !evs[0].IsDone() {
		t.Fatalf("events = %+v, want only the [DONE] event", evs)
	}
}

func TestMultipleDataLinesEmitSeparateEvents(t *testing.T) {
	// Open-WebUI frames one event per data: line (see the captured fixture);
	// blank lines are optional separators, so two data lines -> two events.
	body := "data: line1\ndata: line2\n\n"
	evs := collect(t, NewReader(strings.NewReader(body)))
	if len(evs) != 2 || evs[0].Data != "line1" || evs[1].Data != "line2" {
		t.Fatalf("events = %+v", evs)
	}
}
