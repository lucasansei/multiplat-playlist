package parser

import "testing"

func TestParseSupportedURLs(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		platform Platform
		id       string
	}{
		{
			name:     "spotify track url",
			rawURL:   "https://open.spotify.com/track/3n3Ppam7vgaVa1iaRUc9Lp",
			platform: PlatformSpotify,
			id:       "3n3Ppam7vgaVa1iaRUc9Lp",
		},
		{
			name:     "spotify uri",
			rawURL:   "spotify:track:3n3Ppam7vgaVa1iaRUc9Lp",
			platform: PlatformSpotify,
			id:       "3n3Ppam7vgaVa1iaRUc9Lp",
		},
		{
			name:     "youtube watch url",
			rawURL:   "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			platform: PlatformYouTube,
			id:       "dQw4w9WgXcQ",
		},
		{
			name:     "youtube watch url without www",
			rawURL:   "https://youtube.com/watch?v=dQw4w9WgXcQ",
			platform: PlatformYouTube,
			id:       "dQw4w9WgXcQ",
		},
		{
			name:     "youtube short url",
			rawURL:   "https://youtu.be/dQw4w9WgXcQ",
			platform: PlatformYouTube,
			id:       "dQw4w9WgXcQ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.rawURL)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got.Platform != tt.platform {
				t.Fatalf("Platform = %q, want %q", got.Platform, tt.platform)
			}
			if got.ID != tt.id {
				t.Fatalf("ID = %q, want %q", got.ID, tt.id)
			}
			if got.Original != tt.rawURL {
				t.Fatalf("Original = %q, want %q", got.Original, tt.rawURL)
			}
		})
	}
}

func TestParseInvalidURLs(t *testing.T) {
	tests := []string{
		"",
		"not a url",
		"https://example.com/watch?v=dQw4w9WgXcQ",
		"https://open.spotify.com/album/3n3Ppam7vgaVa1iaRUc9Lp",
		"spotify:album:3n3Ppam7vgaVa1iaRUc9Lp",
	}

	for _, rawURL := range tests {
		t.Run(rawURL, func(t *testing.T) {
			if got, err := Parse(rawURL); err == nil {
				t.Fatalf("Parse() = %#v, nil error; want error", got)
			}
		})
	}
}
