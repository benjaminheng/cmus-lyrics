package main

import (
	"context"
	"flag"
	"fmt"
	"log"
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
	viewport        viewport.Model
	showHelpFooter  bool
	geniusAPIClient *GeniusAPIClient

	statusBar   string
	artist      string
	album       string
	title       string
	lyrics      string
	ready       bool
	lastChecked time.Time

	// Track if we've already fetched lyrics for the current song
	currentSongID string
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

			// Reflow lyrics if window size changes
			m.updateLyrics(m.lyrics)
		}

	case songInfoMsg:
		// Only update if song changed
		if m.artist != msg.artist || m.title != msg.title {
			m.artist = msg.artist
			m.album = msg.album
			m.title = msg.title
			m.updateStatusBar()

			m.viewport.SetContent(m.centerText("Loading..."))

			// Scroll back to top when song changes
			m.viewport.GotoTop()
		}

		// Schedule next check
		cmds = append(cmds, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return checkCmusTick{}
		}))

		// Schedule lyrics to be fetched asynchronously
		cmds = append(cmds, fetchLyricsCmd(m.geniusAPIClient, m.artist, m.album, m.title))

	case songLyricsMsg:
		if msg.err != nil {
			m.viewport.SetContent(msg.err.Error())
		} else {
			m.lyrics = msg.lyrics
			m.updateLyrics(m.lyrics)
		}

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
			lipgloss.NewStyle().Width(m.viewport.Width-lipgloss.Width(helpText)-4).Render(""),
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
		m.statusBar = fmt.Sprintf("%s - %s - %s", m.artist, m.album, m.title)
	} else {
		m.statusBar = fmt.Sprintf("%s - %s", m.artist, m.title)
	}
}

func (m *model) updateLyrics(lyrics string) {
	centeredLyrics := m.centerText(lyrics)
	m.viewport.SetContent(centeredLyrics)
}

func (m *model) centerText(text string) string {
	// Center each line of the lyrics
	centeredLyrics := ""
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		// Use lipgloss to center each line within the viewport width
		centeredLine := lipgloss.NewStyle().
			Width(m.viewport.Width).
			Align(lipgloss.Center).
			Render(line)
		centeredLyrics += centeredLine + "\n"
	}
	// Remove trailing newline
	if len(centeredLyrics) > 0 {
		centeredLyrics = centeredLyrics[:len(centeredLyrics)-1]
	}
	return centeredLyrics
}

// Message types for tea.Cmd
type checkCmusTick time.Time

// songInfoMsg contains just the song metadata, without lyrics
type songInfoMsg struct {
	artist string
	album  string
	title  string
	err    error
}

// songLyricsMsg contains the song metadata and fetched lyrics
type songLyricsMsg struct {
	artist string
	album  string
	title  string
	lyrics string
	err    error
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

// generateSongID creates a unique identifier for a song
func generateSongID(artist, title string) string {
	return fmt.Sprintf("%s-%s", strings.ToLower(artist), strings.ToLower(title))
}

// fetchLyricsCmd is a command to fetch lyrics asynchronously
func fetchLyricsCmd(client *GeniusAPIClient, artist, album, title string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		lyrics, err := client.GetLyrics(ctx, artist, title)
		if err != nil {
			return songLyricsMsg{
				artist: artist,
				album:  album,
				title:  title,
				lyrics: fmt.Sprintf("Error fetching lyrics: %v\n", err),
				err:    err,
			}
		}

		return songLyricsMsg{
			artist: artist,
			album:  album,
			title:  title,
			lyrics: lyrics,
			err:    nil,
		}
	}
}

// checkCmusCmd checks cmus status and updates the song info if changed
func checkCmusCmd() tea.Cmd {
	return func() tea.Msg {
		// Run cmus-remote -Q to get current song information
		cmd := exec.Command("cmus-remote", "-Q")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return songInfoMsg{
				artist: "",
				album:  "",
				title:  "Error: cmus not running or not available",
				err:    err,
			}
		}

		// Check if cmus is playing something
		outputStr := string(output)
		if !regexp.MustCompile(`status (playing|paused)`).MatchString(outputStr) {
			return songInfoMsg{
				artist: "",
				album:  "",
				title:  "No song playing",
				err:    nil,
			}
		}

		// Parse the output to get song info
		artist, album, title := parseCmusOutput(outputStr)

		if artist == "" || title == "" {
			return songInfoMsg{
				artist: "",
				album:  "",
				title:  "Unknown song",
				err:    fmt.Errorf("missing artist or title information"),
			}
		}

		// Return the song info without fetching lyrics yet
		return songInfoMsg{
			artist: artist,
			album:  album,
			title:  title,
			err:    nil,
		}
	}
}

func main() {
	// Define command line flags
	showHelpFooter := flag.Bool("show-help-footer", false, "Show keybinding help text in the footer")
	singleQuery := flag.String("query", "", "Do a one-off query for lyrics and print to stdout. For best results, query \"<artist> <track>\".")

	// Parse flags
	flag.Parse()

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	geniusAPIClient := NewGeniusAPIClient(config.GeniusAccessToken)

	if *singleQuery != "" {
		lyrics, err := geniusAPIClient.GetLyrics(context.Background(), *singleQuery, "")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(lyrics)
	} else {
		initialModel := model{
			statusBar:       "Loading...",
			lyrics:          "Loading...",
			showHelpFooter:  *showHelpFooter,
			geniusAPIClient: geniusAPIClient,
		}

		p := tea.NewProgram(initialModel, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			log.Fatal(err)
		}
	}
}
