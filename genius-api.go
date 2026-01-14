package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
)

type SearchResponse struct {
	Response struct {
		Hits []struct {
			Type   string `json:"type"`
			Result struct {
				ID          int64  `json:"id"`
				Title       string `json:"title"`
				ArtistNames string `json:"artist_names"`
			} `json:"result"`
		} `json:"hits"`
	} `json:"response"`
}

type GetSongResponse struct {
	Response struct {
		Song struct {
			Path string `json:"path"`
		} `json:"song"`
	} `json:"response"`
}

type GeniusAPIClient struct {
	accessToken string
}

func NewGeniusAPIClient(accessToken string) *GeniusAPIClient {
	c := &GeniusAPIClient{
		accessToken: accessToken,
	}
	return c
}

func (c *GeniusAPIClient) search(ctx context.Context, query string) (SearchResponse, error) {
	baseURL := "https://api.genius.com/search"

	// Create URL with properly encoded query parameter
	params := url.Values{}
	params.Add("q", query)
	requestURL := baseURL + "?" + params.Encode()

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return SearchResponse{}, errors.Wrap(err, "create request")
	}

	// Set authorization header
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return SearchResponse{}, errors.Wrap(err, "send request")
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return SearchResponse{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Decode response
	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return SearchResponse{}, errors.Wrap(err, "decode response")
	}

	return searchResp, nil
}

func (c *GeniusAPIClient) getSong(ctx context.Context, id int64) (GetSongResponse, error) {
	requestURL := fmt.Sprintf("https://api.genius.com/songs/%d", id)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return GetSongResponse{}, errors.Wrap(err, "create request")
	}

	// Set authorization header
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return GetSongResponse{}, errors.Wrap(err, "send request")
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return GetSongResponse{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Decode response
	var songResp GetSongResponse
	if err := json.NewDecoder(resp.Body).Decode(&songResp); err != nil {
		return GetSongResponse{}, errors.Wrap(err, "decode response")
	}

	return songResp, nil
}

func (c *GeniusAPIClient) getLyrics(ctx context.Context, path string) (string, error) {
	// Construct the full URL
	fullURL := fmt.Sprintf("https://genius.com%s", path)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return "", errors.Wrap(err, "create request")
	}

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "send request")
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "parse HTML")
	}

	// Find the lyrics container by data attribute and class prefix
	var lyricsText strings.Builder
	doc.Find("[data-lyrics-container=\"true\"]").Each(func(i int, s *goquery.Selection) {
		// Remove elements that should be excluded from selection
		s.Find("[data-exclude-from-selection=\"true\"]").Remove()

		// Get the HTML content and append to our builder
		html, err := s.Html()
		if err != nil {
			return
		}
		lyricsText.WriteString(html)
	})

	if lyricsText.Len() == 0 {
		return "", errors.New("no lyrics found on page")
	}

	// Replace HTML line breaks with actual newlines
	lyrics := strings.ReplaceAll(lyricsText.String(), "<br/>", "\n")
	lyrics = strings.ReplaceAll(lyrics, "<br>", "\n")

	// Create a new document to parse the lyrics HTML and extract just the text
	lyricDoc, err := goquery.NewDocumentFromReader(strings.NewReader("<div>" + lyrics + "</div>"))
	if err != nil {
		return "", errors.Wrap(err, "parse lyrics HTML")
	}

	// Extract text content
	cleanLyrics := strings.TrimSpace(lyricDoc.Text())

	return cleanLyrics, nil
}

func (c *GeniusAPIClient) GetLyrics(ctx context.Context, artist string, title string) (string, error) {
	searchResp, err := c.search(ctx, fmt.Sprintf("%s %s", artist, title))
	if err != nil {
		return "", errors.Wrap(err, "search genius api")
	}

	if len(searchResp.Response.Hits) == 0 {
		return "", errors.New("no results")
	}

	songResp, err := c.getSong(ctx, searchResp.Response.Hits[0].Result.ID)
	if err != nil {
		return "", errors.Wrap(err, "get song from genius api")
	}

	lyrics, err := c.getLyrics(ctx, songResp.Response.Song.Path)
	if err != nil {
		return "", errors.Wrap(err, "scrape lyrics from genius webpage")
	}

	return lyrics, nil
}
