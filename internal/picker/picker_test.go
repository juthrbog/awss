package picker

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func press(m model, key tea.KeyPressMsg) (model, tea.Cmd) {
	next, cmd := m.Update(key)
	return next.(model), cmd
}

func typeText(m model, text string) model {
	for _, r := range text {
		m, _ = press(m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func TestCurrentProfile(t *testing.T) {
	m := newModel([]string{"default", "production", "staging"}, "production")
	if got := m.list.SelectedItem().(item).name; got != "production" {
		t.Fatalf("initial selection = %q", got)
	}
	if !strings.Contains(m.View().Content, "production (current)") {
		t.Fatal("current profile not indicated")
	}
	m, cmd := press(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.selected != "production" || !m.done || cmd == nil {
		t.Fatalf("selection failed: selected=%q done=%v", m.selected, m.done)
	}
	if m.View().Content != "" {
		t.Fatal("finished picker should render nothing")
	}
}

func TestFuzzyFilterAndSelect(t *testing.T) {
	m := newModel([]string{"default", "production", "staging"}, "default")
	m = typeText(m, "pdn") // Non-contiguous match, not a substring.
	if got := len(m.list.VisibleItems()); got != 1 {
		t.Fatalf("visible items = %d, want 1", got)
	}
	m, _ = press(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.selected != "production" {
		t.Fatalf("selected = %q", m.selected)
	}
}

func TestNavigateFilteredResults(t *testing.T) {
	m := newModel([]string{"dev-admin", "dev-readonly", "production"}, "")
	m = typeText(m, "dev")
	m, _ = press(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if got := m.list.SelectedItem().(item).name; got != "dev-readonly" {
		t.Fatalf("down selected %q", got)
	}
	m, _ = press(m, tea.KeyPressMsg{Code: tea.KeyUp})
	m, _ = press(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.selected != "dev-admin" {
		t.Fatalf("selected = %q", m.selected)
	}
}

func TestNoMatchesAndBackspace(t *testing.T) {
	m := typeText(newModel([]string{"default"}, ""), "z")
	m, cmd := press(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.done || m.selected != "" || cmd != nil {
		t.Fatal("enter with no matches must not select or quit")
	}
	m, _ = press(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	if len(m.list.VisibleItems()) != 1 {
		t.Fatal("clearing filter did not restore profiles")
	}
	m, _ = press(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.selected != "default" {
		t.Fatalf("selected = %q", m.selected)
	}
}

func TestCancel(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{{Code: tea.KeyEscape}, {Code: 'c', Mod: tea.ModCtrl}} {
		for _, filter := range []string{"", "prod", "no-match"} {
			t.Run(key.String()+"/"+filter, func(t *testing.T) {
				m := typeText(newModel([]string{"production"}, "production"), filter)
				m, cmd := press(m, key)
				if m.selected != "" || !m.done || cmd == nil {
					t.Fatal("cancel should quit without selecting")
				}
			})
		}
	}
}

func TestQFiltersRatherThanQuits(t *testing.T) {
	m := typeText(newModel([]string{"qa", "production"}, ""), "q")
	if m.done || m.list.SelectedItem().(item).name != "qa" {
		t.Fatal("q must filter, not quit")
	}
}

func TestWindowSizes(t *testing.T) {
	names := make([]string, 60)
	for i := range names {
		names[i] = fmt.Sprintf("profile-%02d-%s", i, strings.Repeat("x", 100))
	}
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 120, Height: 40}, {Width: 60, Height: 15}} {
		t.Run(fmt.Sprintf("%dx%d", size.Width, size.Height), func(t *testing.T) {
			m := newModel(names, "")
			next, _ := m.Update(size)
			view := next.(model).View()
			if !view.AltScreen {
				t.Fatal("picker should restore the terminal using the alternate screen")
			}
			if w, h := lipgloss.Size(view.Content); w > size.Width || h > size.Height {
				t.Fatalf("view is %dx%d, exceeds %dx%d", w, h, size.Width, size.Height)
			}
		})
	}
}

func TestRunEmpty(t *testing.T) {
	selected, err := Run(nil, "", nil, nil)
	if selected != "" || err != nil {
		t.Fatalf("Run(empty) = %q, %v", selected, err)
	}
}
