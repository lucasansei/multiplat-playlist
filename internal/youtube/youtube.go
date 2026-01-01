package youtube

import (
	"os/exec"
	"strings"
)

func GetStreamURL(videoID string) (string, error) {
	cmd := exec.Command("yt-dlp", "-g", "-f", "bestaudio", "https://www.youtube.com/watch?v="+videoID)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
