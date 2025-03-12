package main

import (
	"flag"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the application state
type model struct {
	viewport      viewport.Model
	statusBar     string
	artist        string
	album         string
	title         string
	lyrics        string
	ready         bool
	lastChecked   time.Time
	showHelpFooter bool
}

// Init initializes the Bubble Tea program
func (m model) Init() tea.Cmd {
	return checkCmusCmd()
}

// Update handles events and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			m.viewport.LineDown(1)
		case "k", "up":
			m.viewport.LineUp(1)
		case "g":
			m.viewport.GotoTop()
		case "G":
			m.viewport.GotoBottom()
		case "ctrl+d":
			// m.viewport.LineDown(10)
			m.viewport.HalfViewDown()
		case "ctrl+u":
			// m.viewport.LineUp(10)
			m.viewport.HalfViewUp()
		case "r": // Manually refresh
			cmds = append(cmds, checkCmusCmd())
		}

	case tea.WindowSizeMsg:
		headerHeight := 1 // Status bar
		footerHeight := 1 // Help text
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
			m.viewport.SetContent(m.lyrics)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - headerHeight - footerHeight
		}

	case songUpdateMsg:
		// Only update if song changed
		if m.artist != msg.artist || m.title != msg.title || m.lyrics == "" {
			m.artist = msg.artist
			m.album = msg.album
			m.title = msg.title
			m.lyrics = msg.lyrics
			m.updateStatusBar()
			m.viewport.SetContent(m.lyrics)

			// Scroll back to top when song changes
			m.viewport.GotoTop()
		}

		// Schedule next check
		cmds = append(cmds, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return checkCmusTick{}
		}))

	case checkCmusTick:
		cmds = append(cmds, checkCmusCmd())
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the application UI
func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	statusBarStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#0088CC")).
		Bold(true).
		Width(m.viewport.Width).
		Padding(0, 1)
		
	// Render the status bar
	statusBar := statusBarStyle.Render(m.statusBar)
	
	// Calculate scroll percentage
	scrollPercent := 0
	if m.viewport.ScrollPercent() >= 0 {
		scrollPercent = int(m.viewport.ScrollPercent() * 100)
	}
	
	var footer string
	if m.showHelpFooter {
		// Help text with keybindings
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))
	
		helpText := "j/k: scroll • g/G: top/bottom • C-d/C-u: page down/up • r: refresh • q: quit"
		
		// Show both help text and percentage
		percentStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Bold(true)
		
		// Join help text with percentage
		footer = lipgloss.JoinHorizontal(
			lipgloss.Left,
			helpStyle.Render(helpText),
			lipgloss.NewStyle().Width(m.viewport.Width - lipgloss.Width(helpText) - 4).Render(""),
			percentStyle.Render(fmt.Sprintf("%3d%%", scrollPercent)),
		)
	} else {
		// Only show percentage when help is hidden
		percentStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Bold(true).
			Width(m.viewport.Width).
			Align(lipgloss.Right)
		
		footer = percentStyle.Render(fmt.Sprintf("%3d%%", scrollPercent))
	}

	return fmt.Sprintf("%s\n%s\n%s", statusBar, m.viewport.View(), footer)
}

func (m *model) updateStatusBar() {
	if m.album != "" {
		m.statusBar = fmt.Sprintf("%s - %s [%s]", m.artist, m.title, m.album)
	} else {
		m.statusBar = fmt.Sprintf("%s - %s", m.artist, m.title)
	}
}

// Message types for tea.Cmd
type checkCmusTick time.Time

type songUpdateMsg struct {
	artist string
	album  string
	title  string
	lyrics string
}

// Extract information from cmus-remote -Q output
func parseCmusOutput(output string) (artist, album, title string) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "tag artist ") {
			artist = strings.TrimPrefix(line, "tag artist ")
		} else if strings.HasPrefix(line, "tag album ") {
			album = strings.TrimPrefix(line, "tag album ")
		} else if strings.HasPrefix(line, "tag title ") {
			title = strings.TrimPrefix(line, "tag title ")
		}
	}
	return
}

// fetchLyrics attempts to fetch lyrics from azlyrics
func fetchLyrics(artist, album, track string) (string, error) {
	// This function will be implemented by the user
	// For now, return a placeholder message with some multiline content to demonstrate scrolling
	placeholder := fmt.Sprintf("Lyrics for %s - %s", artist, track)
	if album != "" {
		placeholder += fmt.Sprintf(" from the album %s", album)
	}
	placeholder += "\n\n"

	// Add some dummy content to demonstrate scrolling with a more realistic lyrics format
	verses := []string{
		"This is the first verse of the song\nIt has multiple lines\nTo demonstrate how lyrics look\nIn this TUI application",
		"This is the chorus of the song\nIt repeats several times\nAnd often has a catchy melody\nThat listeners will remember",
		"This is the second verse of the song\nWith different lyrics than the first\nBut still following the same pattern\nAnd building on the song's theme",
		"[Chorus repeats here]",
		"This is the bridge section\nOften with a different feel\nBefore returning to the chorus\nOne last dramatic time",
		"[Final chorus]\nSometimes with slight variations\nOr additional emphasis\nTo bring the song to closure",
		"This is the second verse of the song\nWith different lyrics than the first\nBut still following the same pattern\nAnd building on the song's theme",
		"[Chorus repeats here]",
		"This is the bridge section\nOften with a different feel\nBefore returning to the chorus\nOne last dramatic time",
		"[Final chorus]\nSometimes with slight variations\nOr additional emphasis\nTo bring the song to closure",
		"This is the second verse of the song\nWith different lyrics than the first\nBut still following the same pattern\nAnd building on the song's theme",
		"[Chorus repeats here]",
		"This is the bridge section\nOften with a different feel\nBefore returning to the chorus\nOne last dramatic time",
		"[Final chorus]\nSometimes with slight variations\nOr additional emphasis\nTo bring the song to closure",
		"This is the second verse of the song\nWith different lyrics than the first\nBut still following the same pattern\nAnd building on the song's theme",
		"[Chorus repeats here]",
		"This is the bridge section\nOften with a different feel\nBefore returning to the chorus\nOne last dramatic time",
		"[Final chorus]\nSometimes with slight variations\nOr additional emphasis\nTo bring the song to closure",
		"This is the second verse of the song\nWith different lyrics than the first\nBut still following the same pattern\nAnd building on the song's theme",
		"[Chorus repeats here]",
		"This is the bridge section\nOften with a different feel\nBefore returning to the chorus\nOne last dramatic time",
		"[Final chorus]\nSometimes with slight variations\nOr additional emphasis\nTo bring the song to closure",
		"This is the second verse of the song\nWith different lyrics than the first\nBut still following the same pattern\nAnd building on the song's theme",
		"[Chorus repeats here]",
		"This is the bridge section\nOften with a different feel\nBefore returning to the chorus\nOne last dramatic time",
		"[Final chorus]\nSometimes with slight variations\nOr additional emphasis\nTo bring the song to closure",
		"This is the second verse of the song\nWith different lyrics than the first\nBut still following the same pattern\nAnd building on the song's theme",
		"[Chorus repeats here]",
		"This is the bridge section\nOften with a different feel\nBefore returning to the chorus\nOne last dramatic time",
		"[Final chorus]\nSometimes with slight variations\nOr additional emphasis\nTo bring the song to closure",
		"This is the second verse of the song\nWith different lyrics than the first\nBut still following the same pattern\nAnd building on the song's theme",
		"[Chorus repeats here]",
		"This is the bridge section\nOften with a different feel\nBefore returning to the chorus\nOne last dramatic time",
		"[Final chorus]\nSometimes with slight variations\nOr additional emphasis\nTo bring the song to closure",
		"This is the second verse of the song\nWith different lyrics than the first\nBut still following the same pattern\nAnd building on the song's theme",
		"[Chorus repeats here]",
		"This is the bridge section\nOften with a different feel\nBefore returning to the chorus\nOne last dramatic time",
		"[Final chorus]\nSometimes with slight variations\nOr additional emphasis\nTo bring the song to closure",
	}

	for _, verse := range verses {
		placeholder += verse + "\n\n"
	}

	return placeholder, nil
}

// checkCmusCmd checks cmus status and updates the song if changed
func checkCmusCmd() tea.Cmd {
	return func() tea.Msg {
		// Run cmus-remote -Q to get current song information
		cmd := exec.Command("cmus-remote", "-Q")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return songUpdateMsg{
				artist: "",
				album:  "",
				title:  "Error: cmus not running or not available",
				lyrics: fmt.Sprintf("Error executing cmus-remote: %v\n", err),
			}
		}

		// Check if cmus is playing something
		outputStr := string(output)
		if !regexp.MustCompile(`status (playing|paused)`).MatchString(outputStr) {
			return songUpdateMsg{
				artist: "",
				album:  "",
				title:  "No song playing",
				lyrics: "No song is currently playing in cmus",
			}
		}

		// Parse the output to get song info
		artist, album, title := parseCmusOutput(outputStr)

		if artist == "" || title == "" {
			return songUpdateMsg{
				artist: "",
				album:  "",
				title:  "Unknown song",
				lyrics: "Could not find artist or title information for the current song",
			}
		}

		// Fetch lyrics
		lyrics, err := fetchLyrics(artist, album, title)
		if err != nil {
			return songUpdateMsg{
				artist: artist,
				album:  album,
				title:  title,
				lyrics: fmt.Sprintf("Error fetching lyrics: %v\n", err),
			}
		}

		return songUpdateMsg{
			artist: artist,
			album:  album,
			title:  title,
			lyrics: lyrics,
		}
	}
}

func main() {
	// Define command line flags
	showHelpFooter := flag.Bool("show-help-footer", false, "Show keybinding help text in the footer")
	
	// Parse flags
	flag.Parse()
	
	initialModel := model{
		statusBar:     "Loading...",
		lyrics:        "Fetching current song information...",
		showHelpFooter: *showHelpFooter,
	}

	p := tea.NewProgram(initialModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
	}
}
