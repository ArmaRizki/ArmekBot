package kkdai

import (
	"errors"
	"fmt"

	"github.com/ArmaRizki/ArmekBot/pkg/music/opus"
	"github.com/ArmaRizki/ArmekBot/pkg/music/parsers"
	ffmpegparser "github.com/ArmaRizki/ArmekBot/pkg/music/parsers/ffmpeg"

	"github.com/kkdai/youtube/v2"
)

func kkdaiLink(track *parsers.Track, seekSec float64) (opus.Reader, func(), error) {
	videoID, err := extractYouTubeID(track.URL)
	if err != nil {
		return nil, nil, err
	}

	client := &youtube.Client{}
	video, err := client.GetVideo(videoID)
	if err != nil {
		return nil, nil, fmt.Errorf("kkdai: youtube client: %w", err)
	}
	if video == nil {
		return nil, nil, errors.New("kkdai: youtube client: no video")
	}

	track.Duration = video.Duration
	track.Title = video.Title

	formats := video.Formats.WithAudioChannels()
	if len(formats) == 0 {
		return nil, nil, errors.New("kkdai: no audio formats found")
	}

	link, err := client.GetStreamURL(video, &formats[0])
	if err != nil {
		return nil, nil, fmt.Errorf("kkdai: get stream url: %w", err)
	}

	return ffmpegparser.OpusReader(ffmpegparser.NewPCMCommand(link, seekSec, true, "kkdai-link"), "kkdai")
}
