package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"owui-term/internal/config"
)

func newTestModel() Model {
	return New(config.Config{URL: "http://localhost:3000", Token: "sk-test"}, "test")
}

func TestNewStartsLoading(t *testing.T) {
	if m := newTestModel(); m.state != stateLoading {
		t.Errorf("state = %v, want stateLoading", m.state)
	}
}

func TestReadyMsgTransitionsToReady(t *testing.T) {
	updated, _ := newTestModel().Update(readyMsg{})
	m, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", updated)
	}
	if m.state != stateReady {
		t.Errorf("state = %v, want stateReady", m.state)
	}
}

func TestQuitKeyReturnsQuitCmd(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected a non-nil command for 'q', got nil")
	}
}

func TestSetErrorNilIgnored(t *testing.T) {
	m := newTestModel().SetError(nil)
	if m.state != stateLoading {
		t.Errorf("state = %v, want stateLoading (nil error must be ignored)", m.state)
	}
}

func TestSetErrorNonNilRenders(t *testing.T) {
	m := newTestModel().SetError(fmt.Errorf("boom"))
	if m.state != stateError {
		t.Errorf("state = %v, want stateError", m.state)
	}
	if !strings.Contains(m.View(), "boom") {
		t.Errorf("error view should render the error message, got:\n%s", m.View())
	}
}

func TestViewNeverLeaksToken(t *testing.T) {
	for _, m := range []Model{newTestModel(), newTestModel().SetError(fmt.Errorf("x"))} {
		if strings.Contains(m.View(), "sk-test") {
			t.Fatal("View() leaked the bearer token (D6)")
		}
	}
}
