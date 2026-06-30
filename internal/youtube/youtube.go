package youtube

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func GetStreamURL(ctx context.Context, videoID string) (string, error) {
	cmd := exec.CommandContext(ctx, "yt-dlp", "-g", "-f", "bestaudio", "https://www.youtube.com/watch?v="+videoID)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return "", fmt.Errorf("yt-dlp: %s: %w", stderr, err)
			}
		}
		return "", fmt.Errorf("yt-dlp: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
