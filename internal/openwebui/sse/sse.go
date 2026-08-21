// Package sse implements a defensive, line-buffered Server-Sent-Events reader
// (locked constraint D5). It only handles the documented OpenAI-style framing
// owui-term relies on: one event per "data:" line (the framing verified on the
// live 0.11.0 instance — see docs/api-notes.md), the [DONE] sentinel, and a
// tolerant ignore-everything-else posture. It is safe against fragmented
// network reads and never dispatches an incomplete line. Blank lines are
// treated as separators, so streams that also use them work unchanged.
package sse

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

// Event is one parsed SSE event: the value of its "data:" field. Unknown
// fields (event:, id:, retry:, ...) and comment lines (": ...") are ignored.
type Event struct {
	Data string
}

// IsDone reports whether this is the OpenAI [DONE] sentinel event.
func (e Event) IsDone() bool {
	return e.Data == "[DONE]"
}

// DecodeJSON unmarshals the event's data as JSON into v, returning the json
// error verbatim so the caller can decide how strictly to handle it.
func (e Event) DecodeJSON(v any) error {
	return json.Unmarshal([]byte(e.Data), v)
}

// CompletionChunk is an OpenAI-style streamed completion chunk (delta / usage).
// Only the documented fields owui-term needs are typed; extra fields are
// ignored by the JSON decoder.
type CompletionChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
		Delta        struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *Usage `json:"usage"`
}

// Usage is the token usage carried on the final completion chunk.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Reader is a line-buffered SSE event reader.
type Reader struct {
	br *bufio.Reader
}

// NewReader wraps r and returns an SSE reader. The underlying bufio.Reader
// absorbs arbitrarily fragmented reads, so a line is only dispatched once a
// newline or stream EOF is reached (it never blocks on an incomplete line).
func NewReader(r io.Reader) *Reader {
	return &Reader{br: bufio.NewReader(r)}
}

// Next returns the next event, or io.EOF when the stream is exhausted. A line
// cut short by EOF is deliberately not dispatched (never emit a partial line as
// a complete event).
func (r *Reader) Next() (Event, error) {
	for {
		line, err := r.readLine()
		if err != nil {
			// EOF / read error: no complete line remains, so stop.
			return Event{}, err
		}
		if line == "" {
			continue // blank line: separator, not an event
		}
		if strings.HasPrefix(line, ":") {
			continue // SSE comment
		}
		if field, value, ok := parseField(line); ok && field == "data" {
			return Event{Data: value}, nil
		}
		// Unknown fields are ignored (keptolerant).
	}
}

// readLine returns the next logical line with its CR/LF stripped. A partial
// final line at EOF is returned with a non-nil error so the caller can decide.
func (r *Reader) readLine() (string, error) {
	s, err := r.br.ReadString('\n')
	return strings.TrimRight(s, "\r\n"), err
}

// parseField splits "field:value" on the first colon per the SSE spec, trimming
// a single leading space from the value.
func parseField(line string) (field, value string, ok bool) {
	i := strings.IndexByte(line, ':')
	if i < 0 {
		return "", "", false
	}
	return line[:i], strings.TrimPrefix(line[i+1:], " "), true
}
