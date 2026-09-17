// Package picker provides the interactive, fuzzy-filterable profile picker.
package picker

import (
	"fmt"
	"io"
	"os"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type item struct {
	name    string
	current bool
}

func (i item) FilterValue() string { return i.name }
func (i item) Description() string { return "" }
func (i item) Title() string {
	if i.current {
		return i.name + " (current)"
	}
	return i.name
}

type model struct {
	list     list.Model
	filter   textinput.Model
	selected string
	done     bool
}

func newModel(names []string, current string) model {
	items := make([]list.Item, len(names))
	currentIndex := 0
	for n, name := range names {
		items[n] = item{name: name, current: name == current}
		if name == current {
			currentIndex = n
		}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)
	l := list.New(items, delegate, 80, 20)
	l.SetShowTitle(false)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetStatusBarItemName("profile", "profiles")
	l.Select(currentIndex)

	filter := textinput.New()
	filter.Prompt = "Filter: "
	filter.Placeholder = "Type to filter…"
	filter.SetWidth(70)
	filter.Focus()
	return model{list: l, filter: filter}
}

func (m model) Init() tea.Cmd { return textinput.Blink }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(max(1, msg.Width), max(1, msg.Height-4))
		m.filter.SetWidth(max(1, msg.Width-10))
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.done = true
			return m, tea.Quit
		case "enter":
			if selected, ok := m.list.SelectedItem().(item); ok {
				m.selected = selected.name
				m.done = true
				return m, tea.Quit
			}
			return m, nil
		case "up", "ctrl+p":
			m.list.CursorUp()
			return m, nil
		case "down", "ctrl+n":
			m.list.CursorDown()
			return m, nil
		case "pgup":
			m.list.PrevPage()
			return m, nil
		case "pgdown":
			m.list.NextPage()
			return m, nil
		}
	}

	previous := m.filter.Value()
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	if m.filter.Value() != previous {
		// Apply the list's fuzzy filter synchronously so a quick Enter cannot
		// select a stale result while an asynchronous filter is still running.
		m.list.SetFilterText(m.filter.Value())
	}
	return m, cmd
}

func (m model) View() tea.View {
	content := ""
	if !m.done {
		title := lipgloss.NewStyle().Bold(true).Render("AWS profiles")
		help := lipgloss.NewStyle().Faint(true).Render("↑/↓ navigate • enter select • esc/ctrl+c cancel")
		content = title + "\n" + m.filter.View() + "\n" + m.list.View() + "\n" + help
	}
	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

// Run returns the selected profile, or an empty string on cancellation. UI output
// is kept separate from the export statements written by the caller.
func Run(names []string, current string, input *os.File, output io.Writer) (string, error) {
	if len(names) == 0 {
		return "", nil
	}
	result, err := tea.NewProgram(newModel(names, current), tea.WithInput(input), tea.WithOutput(output)).Run()
	if err != nil {
		return "", fmt.Errorf("running profile picker: %w", err)
	}
	return result.(model).selected, nil
}
