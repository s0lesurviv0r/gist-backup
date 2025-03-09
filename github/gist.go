package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
)

// Gist represents a GitHub gist
type Gist struct {
	ID          string              `json:"id"`
	Description string              `json:"description"`
	Public      bool                `json:"public"`
	Files       map[string]GistFile `json:"files"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	URL         string              `json:"url"`
	HTMLURL     string              `json:"html_url"`
}

// GistFile represents a file within a gist
type GistFile struct {
	Filename string `json:"filename"`
	Type     string `json:"type"`
	Language string `json:"language"`
	RawURL   string `json:"raw_url"`
	Size     int    `json:"size"`
	Content  string `json:"content"`
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client represents a GitHub API client
type Client struct {
	token      string
	httpClient HTTPClient
}

// NewClient creates a new GitHub API client
func NewClient(token string, httpClient HTTPClient) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &Client{
		token:      token,
		httpClient: httpClient,
	}
}

// GetGistsForUser fetches all gists for a given GitHub username
func (c *Client) GetGistsForUser(ctx context.Context, username string) ([]Gist, error) {
	log.Debug("Fetching gists for user: ", username)

	url := fmt.Sprintf("https://api.github.com/users/%s/gists", username)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	var gists []Gist
	if err := json.NewDecoder(resp.Body).Decode(&gists); err != nil {
		return nil, err
	}

	log.Debugf("Fetched %d gists", len(gists))

	return gists, nil
}

// DownloadGist downloads a single gist by ID to the specified directory
func (c *Client) DownloadGist(ctx context.Context, gistID, directory string) error {
	// Ensure directory exists
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.github.com/gists/%s", gistID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API error: %s", resp.Status)
	}

	var gist Gist
	if err := json.NewDecoder(resp.Body).Decode(&gist); err != nil {
		return err
	}

	// Create a subdirectory for the gist
	gistDir := filepath.Join(directory, gistID)
	if err := os.MkdirAll(gistDir, 0755); err != nil {
		return err
	}

	// Create metadata file
	metadataBytes, err := json.MarshalIndent(gist, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(gistDir, "metadata.json"), metadataBytes, 0644); err != nil {
		return err
	}

	// Download each file in the gist
	for filename, file := range gist.Files {
		filePath := filepath.Join(gistDir, filename)

		// Get file content
		fileReq, err := http.NewRequestWithContext(ctx, http.MethodGet, file.RawURL, nil)
		if err != nil {
			return err
		}

		if c.token != "" {
			fileReq.Header.Set("Authorization", "token "+c.token)
		}

		fileResp, err := c.httpClient.Do(fileReq)
		if err != nil {
			return err
		}

		if fileResp.StatusCode != http.StatusOK {
			fileResp.Body.Close()
			return fmt.Errorf("Failed to download file %s: %s", filename, fileResp.Status)
		}

		out, err := os.Create(filePath)
		if err != nil {
			fileResp.Body.Close()
			return err
		}

		_, err = io.Copy(out, fileResp.Body)
		out.Close()
		fileResp.Body.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// DownloadAllGistsForUser downloads all gists for a user to the specified directory
func (c *Client) DownloadAllGistsForUser(ctx context.Context, username, directory string) error {
	// Get all gists for the user
	gists, err := c.GetGistsForUser(ctx, username)
	if err != nil {
		return err
	}

	// Create a user directory
	userDir := filepath.Join(directory, username)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return err
	}

	// Download each gist
	for _, gist := range gists {
		log.Debugf("Downloading gist %s", gist.ID)
		if err := c.DownloadGist(ctx, gist.ID, userDir); err != nil {
			return fmt.Errorf("Failed to download gist %s: %w", gist.ID, err)
		}
	}

	return nil
}
