package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"owui-term/internal/openwebui"
	"owui-term/internal/openwebui/sse"
)

// fakeStream wraps a fixed SSE body so tests drive the probe without a server.
func fakeStream(body string) func(context.Context, openwebui.CompletionsRequest) (*sse.Reader, error) {
	return func(context.Context, openwebui.CompletionsRequest) (*sse.Reader, error) {
		return sse.NewReader(strings.NewReader(body)), nil
	}
}

func TestRunProbe(t *testing.T) {
	tests := []struct {
		name       string
		stream     func(context.Context, openwebui.CompletionsRequest) (*sse.Reader, error)
		wantFail   bool
		wantChunks int
		wantDone   bool
	}{
		{
			name:       "healthy stream: chunks then DONE",
			stream:     fakeStream("data: {\"choices\":[{\"delta\":{\"content\":\"a\"}}]}\ndata: {\"choices\":[{\"delta\":{\"content\":\"ny\"}}]}\ndata: [DONE]\n\n"),
			wantFail:   false,
			wantChunks: 2,
			wantDone:   true,
		},
		{
			name:     "zero data chunks with DONE fails",
			stream:   fakeStream("data: [DONE]\n\n"),
			wantFail: true,
		},
		{
			name:     "chunks but missing DONE fails",
			stream:   fakeStream("data: {\"choices\":[{\"delta\":{\"content\":\"a\"}}]}\ndata: {\"choices\":[{\"delta\":{\"content\":\"b\"}}]}\n\n"),
			wantFail: true,
		},
		{
			name:     "empty body fails",
			stream:   fakeStream(""),
			wantFail: true,
		},
		{
			name: "stream open error fails",
			stream: func(context.Context, openwebui.CompletionsRequest) (*sse.Reader, error) {
				return nil, errors.New("stream open failed")
			},
			wantFail: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := runProbe(context.Background(), "ci-model", tt.stream)
			if got := res.failReason != ""; got != tt.wantFail {
				t.Fatalf("runProbe fail=%v (reason %q), want fail=%v", got, res.failReason, tt.wantFail)
			}
			if !tt.wantFail {
				if res.chunks != tt.wantChunks || res.done != tt.wantDone {
					t.Fatalf("runProbe chunks=%d done=%t, want chunks=%d done=%t", res.chunks, res.done, tt.wantChunks, tt.wantDone)
				}
			}
		})
	}
}
