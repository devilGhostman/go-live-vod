package transcoder

import (
	"fmt"
	"github/devilGhostman/backend/internal/externalcmd"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/grafov/m3u8"
	"go.uber.org/zap"
)

type (
	VideoCodecType int
	AudioCodecType int
)

const (
	H264 VideoCodecType = iota
	H265 VideoCodecType = iota
)

const (
	AAC AudioCodecType = iota
)

const DefaultResolution = "720p"

func (vc VideoCodecType) StringFourCC() string {
	switch vc {
	case H265:
		return "hev1.1.6.L93.B0"
	case H264:
		return "avc1.4d401f,mp4a.40.2"
	}
	return "avc1.4d401f,mp4a.40.2"
}

func (ac AudioCodecType) StringFourCC() string {
	switch ac {
	case AAC:
		return "mp4a.40.5"
	}
	return "mp4a.40.5"
}

func (vc VideoCodecType) String() string {
	switch vc {
	case H265:
		return "H265"
	case H264:
		return "H264"
	}
	return "H264"
}

func (ac AudioCodecType) String() string {
	switch ac {
	case AAC:
		return "AAC"
	}
	return "AAC"
}

type Transcoder struct {
	ID             string
	Namespace      string
	FfmpegBin      string
	Source         string
	Resolutions    []string
	VideoCodec     VideoCodecType
	AudioCodec     AudioCodecType
	FrameRate      float64
	AudioEnable    bool
	MasterFileName string
	OutputDir      string
	MasterHls      string
	Mux            sync.RWMutex
}

func NewTranscoder(source, namespace, ID string, Resolutions []string, videoCodec string, audioCodec string) *Transcoder {
	ffmpegBin := os.Getenv("FFMPEG_BIN")
	tscconfig := Transcoder{
		ID:        ID,
		Namespace: namespace,
	}

	tscconfig.FfmpegBin = ffmpegBin
	tscconfig.Source = source
	tscconfig.Resolutions = Resolutions

	tscconfig.VideoCodec = H264
	tscconfig.AudioCodec = AAC

	tscconfig.MasterFileName = "playlist.m3u8"
	return &tscconfig
}

func (t *Transcoder) SetConfig(Resolutions []string, audio bool, videoCodec string, audioCodec string) {
	if len(Resolutions) >= 1 {
		t.Resolutions = Resolutions
		zap.S().Infof("setting up Resolutions %s", Resolutions)
	}

	t.AudioEnable = audio
	if audio {
		t.Resolutions = append(t.Resolutions, "audio")
	}

	if videoCodec != "" {
		zap.S().Infof("setting up videoCodec %s", videoCodec)
		switch videoCodec {
		case "h264":
			t.VideoCodec = H264
		case "h265":
			t.VideoCodec = H265
		default:
			t.VideoCodec = H264
		}
	}

	if audioCodec != "" {
		zap.S().Infof("setting up audioCodec %s", audioCodec)
		switch audioCodec {
		case "AAC":
			t.AudioCodec = AAC
		default:
			t.AudioCodec = AAC
		}
	}
}

func (t *Transcoder) Run() (string, error) {
	t.prepareOutputDir()
	cmdstring := t.generateCmdString()
	_, err := t.generateMasterHls()
	if err != nil {
		zap.S().Errorf("failed to start trasncoder, Error: %s", err)
	}

	initialMasterHls := m3u8.NewMasterPlaylist()
	baseUrl := os.Getenv("BASE_URL")

	for index, varient := range t.Resolutions {
		varientName := t.Resolutions[index]
		varientURL := fmt.Sprintf("%s/hls/%s/%s/%s.m3u8", baseUrl, t.ID, varientName, varientName)
		genrateVarient := m3u8.Variant{
			URI:           varientURL,
			VariantParams: t.getVideoMeatadata(varient),
		}
		initialMasterHls.Variants = append(initialMasterHls.Variants, &genrateVarient)

		zap.S().Infof("transcoder, generated %s.m3u8 hls file, %s", varientName)
	}

	zap.S().Infof("transcoder, generated master.m3u8 hls file, %s\nm3u8file: %s", initialMasterHls.Variants[len(initialMasterHls.Variants)-1].URI, initialMasterHls.String())

	cmdrunnerpool := externalcmd.NewPool()
	rtmpPullCmd := externalcmd.NewCmd(
		cmdrunnerpool, cmdstring, true, make(externalcmd.Environment), nil, t.ID)
	rtmpPullCmd.SetStreamID(t.ID)

	zap.S().Infof("transcoder, starting ffmpeg command: %s", cmdstring)
	return initialMasterHls.String(), nil
}

func (t *Transcoder) generateCmdString() string {
	// Base FFmpeg command with latency and reconnect optimizations
	prefix := fmt.Sprintf(
		"%s -re -i \"%s\" "+
			"-reconnect 1 -reconnect_streamed 1 -reconnect_delay_max 10 -reconnect_at_eof 1 "+
			"-c:v libx264 -c:a aac -ac 1 -strict -2 -crf 23 -preset ultrafast -tune zerolatency ",
		t.FfmpegBin, t.Source,
	)

	var commands []string

	// Process variants
	for _, variant := range t.Resolutions {
		segmentFilename := fmt.Sprintf("%d_", time.Now().Unix()) + "%04d.ts"

		if variant != "audio" {
			// Video variant with dynamic bitrate and segment management
			resolution := t.getVideoMeatadata(variant).Resolution
			bitrate := t.getVariantBitrate(variant)
			maxRate := bitrate * 2
			bufSize := bitrate * 4

			cmd := fmt.Sprintf(
				"-s %s -b:v %dk -maxrate %dk -bufsize %dk "+
					"-hls_time 6 -hls_list_size 6 "+
					"-hls_flags delete_segments+omit_endlist "+
					"-hls_delete_threshold 10 -hls_segment_type fmp4 "+
					"-hls_segment_filename %s/%s/%s "+
					"-f hls %s/%s/%s.m3u8 ",
				resolution, bitrate, maxRate, bufSize,
				t.OutputDir, variant, segmentFilename,
				t.OutputDir, variant, variant,
			)
			commands = append(commands, cmd)
		} else {
			// Audio-only variant
			cmd := fmt.Sprintf(
				"-map 0:a -b:a 128k "+
					"-hls_time 4 -hls_list_size 4 "+
					"-hls_flags delete_segments+independent_segments "+
					"-hls_delete_threshold 10 "+
					"-hls_segment_filename %s/audio/%s "+
					"-f hls %s/audio/audio.m3u8 ",
				t.OutputDir, segmentFilename, t.OutputDir,
			)
			commands = append(commands, cmd)
		}
	}

	generatethumbnailcmd := fmt.Sprintf(" -ss 1.4 -frames:v 1 %s/thumbnail.jpg", t.OutputDir)
	commands = append(commands, generatethumbnailcmd)

	// Combine all parts into a single command string
	finalCmd := fmt.Sprintf("%s %s", prefix, strings.Join(commands, " "))
	zap.S().Infof("Generated FFmpeg command: %s", finalCmd)

	return finalCmd
}

// Helper to dynamically get variant bitrate (example implementation)
func (t *Transcoder) getVariantBitrate(variant string) int {
	switch variant {
	case "240p":
		return 200 // 200 kbps
	case "360p":
		return 596 // 600 kbps
	case "480p":
		return 800 // 600 kbps
	case "720p":
		return 1200 // 1200 kbps
	case "1080p":
		return 2500 // 2500 kbps
	case "1440p":
		return 5000 // 5000 kbps
	case "2160p":
		return 7000 // 7000 kbps
	default:
		return 2500 // 2500 kbps
	}
}

func (t *Transcoder) prepareOutputDir() {
	wd, _ := os.Getwd()
	t.OutputDir = path.Join(
		wd,
		"output",
		t.Namespace,
		t.ID,
	)
	zap.S().Infof("setting up output dir: %s", t.OutputDir)

	for _, varient := range t.Resolutions {
		varientPath := path.Join(t.OutputDir, varient)
		err := os.MkdirAll(varientPath, os.ModePerm)
		if err != nil {
			zap.S().Warnf("transcoder: failed to mkdir, Error: %s", err)
		}
	}
}

func (t *Transcoder) generateMasterHls() (m3u8.MasterPlaylist, error) {
	masterHls := m3u8.NewMasterPlaylist()

	for _, varient := range t.Resolutions {
		genrateVarient := m3u8.Variant{
			URI:           fmt.Sprintf("%s/%s.m3u8", varient, varient),
			VariantParams: t.getVideoMeatadata(varient),
		}
		masterHls.Variants = append(masterHls.Variants, &genrateVarient)
	}

	zap.S().Infof("transcoder, generated master.m3u8 hls file, %s", masterHls.String())
	masterfilepath := path.Join(t.OutputDir, t.MasterFileName)
	t.MasterHls = masterfilepath
	err := t.writeFile(masterfilepath, []byte(masterHls.String()))
	return *masterHls, err
}

func (t *Transcoder) writeFile(filepath string, data []byte) error {
	err := os.WriteFile(filepath, data, 0o755)
	return err
}

func (t *Transcoder) getVideoMeatadata(varient string) m3u8.VariantParams {
	ONEKBPS := uint32(1000)

	videoResolutions := map[string]m3u8.VariantParams{
		"240p": {
			ProgramId:  1,
			Resolution: "426x240",
			Name:       "240p",
			Codecs:     t.VideoCodec.StringFourCC(),
			Bandwidth:  400 * ONEKBPS,
			FrameRate:  t.FrameRate,
		},
		"360p": {
			ProgramId:  2,
			Resolution: "640x360",
			Name:       "360p",
			Codecs:     t.VideoCodec.StringFourCC(),
			Bandwidth:  1200 * ONEKBPS,
			FrameRate:  t.FrameRate,
		},
		"480p": {
			ProgramId:  3,
			Resolution: "854x480",
			Name:       "480p",
			Codecs:     t.VideoCodec.StringFourCC(),
			Bandwidth:  2000 * ONEKBPS,
			FrameRate:  t.FrameRate,
		},
		"720p": {
			ProgramId:  4,
			Resolution: "1280x720",
			Name:       "720p",
			Codecs:     t.VideoCodec.StringFourCC(),
			Bandwidth:  3500 * ONEKBPS,
			FrameRate:  t.FrameRate,
		},
		"1080p": {
			ProgramId:  5,
			Resolution: "1920x1080",
			Name:       "1080p",
			Codecs:     t.VideoCodec.StringFourCC(),
			Bandwidth:  5000 * ONEKBPS,
			FrameRate:  t.FrameRate,
		},
		"1440p": {
			ProgramId:  6,
			Resolution: "2560x1440",
			Name:       "2K",
			Codecs:     t.VideoCodec.StringFourCC(),
			Bandwidth:  7000 * ONEKBPS,
			FrameRate:  t.FrameRate,
		},
		"2160p": {
			ProgramId:  7,
			Resolution: "3840x2160",
			Name:       "4K",
			Codecs:     t.VideoCodec.StringFourCC(),
			Bandwidth:  10000 * ONEKBPS,
			FrameRate:  t.FrameRate,
		},
		"audio": {
			ProgramId: 8,
			Name:      "audio",
			Codecs:    t.AudioCodec.StringFourCC(),
			Bandwidth: 192 * ONEKBPS,
		},
	}

	metadata, found := videoResolutions[varient]
	if !found {
		return videoResolutions[DefaultResolution]
	}
	return metadata
}
