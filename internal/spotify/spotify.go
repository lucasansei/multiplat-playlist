package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authURL = "https://accounts.spotify.com/api/token"
	apiBase = "https://api.spotify.com/v1"
	timeout = 10 * time.Second
)

var (
	ErrMissingCredentials = errors.New("spotify credentials are not configured")
	ErrPreviewUnavailable = errors.New("spotify preview URL is not available")
)

type Client struct {
	clientID     string
	clientSecret string
	accessToken  string
	tokenExpiry  time.Time
	httpClient   *http.Client
	authURL      string
	apiBase      string
	now          func() time.Time
}

type Option func(*Client)

type Track struct {
	Name       string
	Artists    []string
	Album      string
	PreviewURL string
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type trackResponse struct {
	Name    string `json:"name"`
	Artists []struct {
		Name string `json:"name"`
	} `json:"artists"`
	Album struct {
		Name string `json:"name"`
	} `json:"album"`
	PreviewURL string `json:"preview_url"`
}

func NewClient(clientID, clientSecret string, opts ...Option) *Client {
	c := &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		authURL:      authURL,
		apiBase:      apiBase,
		now:          time.Now,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

func WithAuthURL(rawURL string) Option {
	return func(c *Client) {
		c.authURL = strings.TrimRight(rawURL, "/")
	}
}

func WithAPIBase(rawURL string) Option {
	return func(c *Client) {
		c.apiBase = strings.TrimRight(rawURL, "/")
	}
}

func (c *Client) PreviewURL(ctx context.Context, trackID string) (string, error) {
	track, err := c.GetTrack(ctx, trackID)
	if err != nil {
		return "", err
	}
	if track.PreviewURL == "" {
		return "", ErrPreviewUnavailable
	}
	return track.PreviewURL, nil
}

func (c *Client) GetTrack(ctx context.Context, trackID string) (*Track, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}

	reqURL := c.apiBase + "/tracks/" + url.PathEscape(trackID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create spotify track request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get spotify track: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("spotify track request failed: status %d", resp.StatusCode)
	}

	var trackResp trackResponse
	if err := json.NewDecoder(resp.Body).Decode(&trackResp); err != nil {
		return nil, fmt.Errorf("decode spotify track response: %w", err)
	}

	artists := make([]string, 0, len(trackResp.Artists))
	for _, artist := range trackResp.Artists {
		artists = append(artists, artist.Name)
	}

	return &Track{
		Name:       trackResp.Name,
		Artists:    artists,
		Album:      trackResp.Album.Name,
		PreviewURL: trackResp.PreviewURL,
	}, nil
}

func (c *Client) ensureToken(ctx context.Context) error {
	if c.clientID == "" || c.clientSecret == "" {
		return ErrMissingCredentials
	}
	if c.accessToken != "" && c.now().Before(c.tokenExpiry) {
		return nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.authURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create spotify token request: %w", err)
	}
	req.SetBasicAuth(c.clientID, c.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request spotify token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("spotify token request failed: status %d", resp.StatusCode)
	}

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return fmt.Errorf("decode spotify token response: %w", err)
	}
	if token.AccessToken == "" {
		return errors.New("spotify token response missing access token")
	}
	if token.TokenType != "" && !strings.EqualFold(token.TokenType, "bearer") {
		return fmt.Errorf("spotify token response has unsupported token type %q", token.TokenType)
	}

	c.accessToken = token.AccessToken
	c.tokenExpiry = c.now().Add(time.Duration(token.ExpiresIn) * time.Second)
	if token.ExpiresIn > 60 {
		c.tokenExpiry = c.tokenExpiry.Add(-60 * time.Second)
	}
	return nil
}
