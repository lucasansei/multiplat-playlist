package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestClientGetsTrackWithPreviewURL(t *testing.T) {
	tokenRequests := 0
	trackRequests := 0

	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/token":
			tokenRequests++
			if r.Method != http.MethodPost {
				t.Fatalf("token method = %s, want POST", r.Method)
			}
			clientID, clientSecret, ok := r.BasicAuth()
			if !ok || clientID != "client-id" || clientSecret != "client-secret" {
				t.Fatalf("basic auth = %q/%q/%v, want configured credentials", clientID, clientSecret, ok)
			}
			if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/x-www-form-urlencoded") {
				t.Fatalf("Content-Type = %q, want form content type", got)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm() error = %v", err)
			}
			if got := r.Form.Get("grant_type"); got != "client_credentials" {
				t.Fatalf("grant_type = %q, want client_credentials", got)
			}
			return jsonResponse(t, http.StatusOK, tokenResponse{
				AccessToken: "access-token",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			}), nil
		case "/v1/tracks/track-id":
			trackRequests++
			if r.Method != http.MethodGet {
				t.Fatalf("track method = %s, want GET", r.Method)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
				t.Fatalf("Authorization = %q, want bearer token", got)
			}
			return jsonResponse(t, http.StatusOK, trackResponse{
				Name: "Song",
				Artists: []struct {
					Name string `json:"name"`
				}{
					{Name: "Artist One"},
					{Name: "Artist Two"},
				},
				Album: struct {
					Name string `json:"name"`
				}{Name: "Album"},
				PreviewURL: "https://preview.example/song.mp3",
			}), nil
		default:
			return jsonResponse(t, http.StatusNotFound, map[string]string{"error": "not found"}), nil
		}
	})}

	client := NewClient(
		"client-id",
		"client-secret",
		WithHTTPClient(httpClient),
		WithAuthURL("https://spotify.test/api/token"),
		WithAPIBase("https://spotify.test/v1"),
	)

	track, err := client.GetTrack(context.Background(), "track-id")
	if err != nil {
		t.Fatalf("GetTrack() error = %v", err)
	}

	if track.Name != "Song" {
		t.Fatalf("Name = %q, want Song", track.Name)
	}
	if len(track.Artists) != 2 || track.Artists[0] != "Artist One" || track.Artists[1] != "Artist Two" {
		t.Fatalf("Artists = %#v, want two artists", track.Artists)
	}
	if track.Album != "Album" {
		t.Fatalf("Album = %q, want Album", track.Album)
	}
	if track.PreviewURL != "https://preview.example/song.mp3" {
		t.Fatalf("PreviewURL = %q, want preview URL", track.PreviewURL)
	}
	if tokenRequests != 1 {
		t.Fatalf("token requests = %d, want 1", tokenRequests)
	}
	if trackRequests != 1 {
		t.Fatalf("track requests = %d, want 1", trackRequests)
	}
}

func TestClientReusesValidToken(t *testing.T) {
	tokenRequests := 0

	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/token":
			tokenRequests++
			return jsonResponse(t, http.StatusOK, tokenResponse{
				AccessToken: "access-token",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			}), nil
		case "/v1/tracks/one", "/v1/tracks/two":
			return jsonResponse(t, http.StatusOK, trackResponse{PreviewURL: "https://preview.example/song.mp3"}), nil
		default:
			return jsonResponse(t, http.StatusNotFound, map[string]string{"error": "not found"}), nil
		}
	})}

	client := NewClient(
		"client-id",
		"client-secret",
		WithHTTPClient(httpClient),
		WithAuthURL("https://spotify.test/api/token"),
		WithAPIBase("https://spotify.test/v1"),
	)

	if _, err := client.PreviewURL(context.Background(), "one"); err != nil {
		t.Fatalf("PreviewURL(one) error = %v", err)
	}
	if _, err := client.PreviewURL(context.Background(), "two"); err != nil {
		t.Fatalf("PreviewURL(two) error = %v", err)
	}
	if tokenRequests != 1 {
		t.Fatalf("token requests = %d, want 1", tokenRequests)
	}
}

func TestClientRequiresCredentials(t *testing.T) {
	client := NewClient("", "")

	_, err := client.GetTrack(context.Background(), "track-id")
	if !errors.Is(err, ErrMissingCredentials) {
		t.Fatalf("GetTrack() error = %v, want %v", err, ErrMissingCredentials)
	}
}

func TestClientPreviewURLUnavailable(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/token":
			return jsonResponse(t, http.StatusOK, tokenResponse{
				AccessToken: "access-token",
				TokenType:   "Bearer",
				ExpiresIn:   3600,
			}), nil
		case "/v1/tracks/track-id":
			return jsonResponse(t, http.StatusOK, trackResponse{}), nil
		default:
			return jsonResponse(t, http.StatusNotFound, map[string]string{"error": "not found"}), nil
		}
	})}

	client := NewClient(
		"client-id",
		"client-secret",
		WithHTTPClient(httpClient),
		WithAuthURL("https://spotify.test/api/token"),
		WithAPIBase("https://spotify.test/v1"),
	)

	_, err := client.PreviewURL(context.Background(), "track-id")
	if !errors.Is(err, ErrPreviewUnavailable) {
		t.Fatalf("PreviewURL() error = %v, want %v", err, ErrPreviewUnavailable)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(t *testing.T, statusCode int, v any) *http.Response {
	t.Helper()

	var body strings.Builder
	if err := json.NewEncoder(&body).Encode(v); err != nil {
		t.Fatalf("write JSON: %v", err)
	}
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body.String())),
	}
}
