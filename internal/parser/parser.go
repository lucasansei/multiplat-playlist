package parser

import (
	"errors"
	"regexp"
)

type Platform string

const (
	PlatformSpotify Platform = "spotify"
	PlatformYouTube Platform = "youtube"
	PlatformUnknown Platform = "unknown"
)

type ParsedURL struct {
	Platform Platform
	ID       string
	Original string
}

var (
	spotifyRegex = regexp.MustCompile(`(?:https?://open\.spotify\.com/track/|spotify:track:)([a-zA-Z0-9]+)`)
	youtubeRegex = regexp.MustCompile(`(?:https?://(?:www\.)?youtube\.com/watch\?v=|https?://youtu\.be/)([a-zA-Z0-9_-]+)`)
)

func Parse(url string) (*ParsedURL, error) {
	if matches := spotifyRegex.FindStringSubmatch(url); len(matches) > 1 {
		return &ParsedURL{
			Platform: PlatformSpotify,
			ID:       matches[1],
			Original: url,
		}, nil
	}

	if matches := youtubeRegex.FindStringSubmatch(url); len(matches) > 1 {
		return &ParsedURL{
			Platform: PlatformYouTube,
			ID:       matches[1],
			Original: url,
		}, nil
	}

	return nil, errors.New("unsupported URL format")
}
