// Package playerui provides a short-lived, interactive Spotify control panel.
package playerui

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/locale"
	"github.com/reathloo/shellux/internal/systeminfo"
	"github.com/reathloo/shellux/internal/theme"
)

var actions = [...]systeminfo.PlaybackAction{
	systeminfo.PlaybackPrevious,
	systeminfo.PlaybackRewind10,
	systeminfo.PlaybackToggle,
	systeminfo.PlaybackForward10,
	systeminfo.PlaybackNext,
}

type playbackMsg struct {
	playback systeminfo.Playback
	err      error
}
type actionMsg struct{ err error }
type tickMsg struct{}

type model struct {
	playback systeminfo.Playback
	read     func() (systeminfo.Playback, error)
	control  func(systeminfo.PlaybackAction) error
	colors   theme.Theme
	language locale.Language
	width    int
	height   int
	selected int
	loading  bool
	busy     bool
	status   string
}

// Run opens the player on an alternate screen, preserving the shell prompt.
func Run() error {
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		return errors.New("shellux spotify requires an interactive terminal")
	}
	settings, err := config.LoadDefault()
	if err != nil {
		return err
	}
	colors, err := theme.ByName(settings.Style)
	if err != nil {
		return err
	}
	m := &model{
		read: systeminfo.PlaybackInfo, control: systeminfo.ControlPlayback,
		colors: colors, language: locale.Resolve(settings.Language),
		width: 80, height: 24,
	}
	_, err = tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout)).Run()
	return err
}

func (m *model) Init() tea.Cmd {
	m.loading = true
	return tea.Batch(m.load(), nextTick())
}

func (m *model) load() tea.Cmd {
	read := m.read
	return func() tea.Msg {
		playback, err := read()
		return playbackMsg{playback: playback, err: err}
	}
}

func nextTick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m *model) activate(action systeminfo.PlaybackAction) tea.Cmd {
	if m.busy {
		return nil
	}
	m.busy = true
	m.status = m.tr("Working…", "Wird ausgeführt…")
	control := m.control
	return func() tea.Msg { return actionMsg{err: control(action)} }
}

func (m *model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "q", "ctrl+c":
			return m, tea.Quit
		case "left":
			m.selected = (m.selected + len(actions) - 1) % len(actions)
		case "right", "tab":
			m.selected = (m.selected + 1) % len(actions)
		case "shift+tab":
			m.selected = (m.selected + len(actions) - 1) % len(actions)
		case "enter", " ":
			return m, m.activate(actions[m.selected])
		case "p":
			return m, m.activate(systeminfo.PlaybackPrevious)
		case "b":
			return m, m.activate(systeminfo.PlaybackRewind10)
		case "f":
			return m, m.activate(systeminfo.PlaybackForward10)
		case "t":
			return m, m.activate(systeminfo.PlaybackToggle)
		case "n":
			return m, m.activate(systeminfo.PlaybackNext)
		}
	case playbackMsg:
		m.loading = false
		if msg.err == nil {
			m.playback = msg.playback
		} else {
			m.status = msg.err.Error()
		}
	case actionMsg:
		m.busy = false
		if msg.err == nil {
			m.status = m.tr("Done", "Ausgeführt")
		} else if errors.Is(msg.err, systeminfo.ErrPlaybackUnavailable) {
			m.status = m.tr("Spotify is not running", "Spotify läuft nicht")
		} else {
			m.status = msg.err.Error()
		}
		if !m.loading {
			m.loading = true
			return m, m.load()
		}
	case tickMsg:
		commands := []tea.Cmd{nextTick()}
		if !m.busy && !m.loading {
			m.loading = true
			commands = append(commands, m.load())
		}
		return m, tea.Batch(commands...)
	}
	return m, nil
}

func (m *model) tr(english, german string) string {
	if m.language == locale.German {
		return german
	}
	return english
}

func clean(value string) string {
	return strings.Map(func(character rune) rune {
		if unicode.IsControl(character) {
			return ' '
		}
		return character
	}, value)
}

func clock(value time.Duration) string {
	seconds := max(0, int(value/time.Second))
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

func (m *model) View() tea.View {
	width, height := max(1, m.width), max(1, m.height)
	accent := lipgloss.NewStyle().Foreground(m.colors.AccentColor).Bold(true)
	secondary := lipgloss.NewStyle().Foreground(m.colors.SecondaryColor)
	muted := lipgloss.NewStyle().Foreground(m.colors.MutedColor)
	focused := lipgloss.NewStyle().Foreground(lipgloss.Color("#111111")).Background(m.colors.AccentColor).Bold(true)
	var content string
	if width < 48 || height < 14 {
		content = m.tr("Make the terminal at least 48 × 14. Esc: close", "Terminal auf mindestens 48 × 14 vergrößern. Esc: schließen")
	} else {
		inner := min(66, width-8)
		title := m.tr("Shellux · Spotify player", "Shellux · Spotify-Player")
		state := m.tr("Spotify is not running", "Spotify läuft nicht")
		track, artist := "—", ""
		position := "--:-- / --:--"
		filled := 0
		if m.playback.Available {
			state = m.tr("Paused", "Pausiert")
			if m.playback.Playing {
				state = m.tr("Playing", "Wiedergabe")
			}
			if m.playback.Title != "" {
				track = clean(m.playback.Title)
				artist = clean(m.playback.Artist)
			}
			if m.playback.Duration > 0 {
				position = clock(m.playback.Position) + " / " + clock(m.playback.Duration)
				filled = min(24, max(0, int(24*m.playback.Position/m.playback.Duration)))
			}
		}
		labels := []string{
			m.tr("Previous", "Vorheriger"), "−10 s", m.tr("Play/Pause", "Start/Pause"), "+10 s", m.tr("Next", "Nächster"),
		}
		buttons := make([]string, len(labels))
		for index, label := range labels {
			button := " " + label + " "
			if index == m.selected {
				buttons[index] = focused.Render(button)
			} else {
				buttons[index] = secondary.Render("[" + button + "]")
			}
		}
		status := m.status
		if status == "" {
			status = m.tr("Choose an action", "Aktion auswählen")
		}
		lines := []string{
			accent.Render(title), "",
			secondary.Render(ansi.Truncate(track, inner, "…")),
			muted.Render(ansi.Truncate(artist, inner, "…")), "",
			muted.Render(state),
			secondary.Render(strings.Repeat("█", filled)+strings.Repeat("░", 24-filled)) + "  " + position,
			"", strings.Join(buttons, "  "), "",
			muted.Render(ansi.Truncate(clean(status), inner, "…")), "",
			muted.Render(m.tr("←/→ select · Enter run · P/B/T/F/N direct · Esc close", "←/→ wählen · Enter ausführen · P/B/T/F/N direkt · Esc schließen")),
		}
		body := strings.Join(lines, "\n")
		content = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(m.colors.MutedColor).Padding(1, 2).Render(body)
	}
	view := tea.NewView(lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content))
	view.AltScreen = true
	view.WindowTitle = "Shellux · Spotify"
	return view
}
