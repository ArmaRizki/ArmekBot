// Package scnative streams SoundCloud tracks natively via api-v2 (no yt-dlp):
// resolve → pick transcoding → signed stream URL → ffmpeg → Opus packets.
package scnative

import (
	"github.com/ArmaRizki/ArmekBot/pkg/music/opus"
	"github.com/ArmaRizki/ArmekBot/pkg/music/parsers"
)

type Streamer struct{}

func (s *Streamer) Open(track *parsers.Track, seekSec float64) (opus.Reader, func(), error) {
	return scnativeLink(track, seekSec)
}
