package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

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
	// For now, return a placeholder message
	return fmt.Sprintf("Lyrics placeholder for %s - %s from the album %s", artist, track, album), nil
}

func main() {
	// Run cmus-remote -Q to get current song information
	cmd := exec.Command("cmus-remote", "-Q")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error executing cmus-remote: %v\n", err)
		return
	}

	// Check if cmus is playing something
	outputStr := string(output)
	if !regexp.MustCompile(`status (playing|paused)`).MatchString(outputStr) {
		fmt.Println("No song is currently playing in cmus")
		return
	}

	// Parse the output to get song info
	artist, album, title := parseCmusOutput(outputStr)
	
	if artist == "" || title == "" {
		fmt.Println("Could not find artist or title information for the current song")
		return
	}

	// Fetch lyrics
	lyrics, err := fetchLyrics(artist, album, title)
	if err != nil {
		fmt.Printf("Error fetching lyrics: %v\n", err)
		return
	}

	// Display song info and lyrics
	fmt.Printf("\nNow playing: %s - %s\n", artist, title)
	if album != "" {
		fmt.Printf("Album: %s\n", album)
	}
	fmt.Println("\nLyrics:\n")
	fmt.Println(lyrics)
}