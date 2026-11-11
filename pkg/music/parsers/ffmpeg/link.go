package ffmpeg

import (
	"github.com/ArmaRizki/ArmekBot/pkg/music/opus"
)

func ffmpegLink(url string) (opus.Reader, func(), error) {
	return OpusReader(NewPCMCommand(url, 0, true, "ffmpeg-link"), "ffmpeg")
}
