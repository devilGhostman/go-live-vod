package transcoder

import (
	"fmt"
	"github/devilGhostman/backend/internal/externalcmd"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/grafov/m3u8"
	"go.uber.org/zap"
)

type VodTranscoder struct {
	ID             string
	Namespace      string
	FFmpegBin      string
	Source         string
	Resolutions    []string
	VideoCodec     VideoCodecType
	AudioCodec     AudioCodecType
	FrameRate      float64
	MasterFileName string
	OutputDir      string
	MasterHls      string
	Mux            sync.RWMutex
}

func NewVodTranscoder(source, namespace, ID string, Resolutions []string, videoCodec string, audioCodec string) *VodTranscoder {
	ffmpegBin := os.Getenv("FFMPEG_BIN")
	tscconfig := VodTranscoder{
		ID:        ID,
		Namespace: namespace,
	}

	tscconfig.FFmpegBin = ffmpegBin
	tscconfig.Source = source
	tscconfig.Resolutions = Resolutions

	tscconfig.VideoCodec = H264
	tscconfig.AudioCodec = AAC
	tscconfig.FrameRate = 30

	tscconfig.MasterFileName = "playlist.m3u8"
	tscconfig.OutputDir = "output"
	return &tscconfig
}

func (vt *VodTranscoder) Run() error {
	vt.prepareOutputDir()
	cmdString := vt.generateCmdString()
	_, err := vt.generateMasterHls()
	if err != nil {
		zap.S().Errorf("failed to start trasncoder, Error: %s", err)
	}

	cmdrunnerpool := externalcmd.NewPool()
	vodpullcmd := externalcmd.NewCmd(cmdrunnerpool, cmdString, false, make(externalcmd.Environment), nil, vt.ID)
	vodpullcmd.SetStreamID(vt.ID)

	zap.S().Infof("transcoder, starting ffmpeg command: %s", cmdString)

	cmdrunnerpool.Close()
	err = vt.onCompleteTranscode(vt.ID)
	if err != nil {
		zap.S().Errorf("failed to start trasncoder, Error: %s", err)
		// return err
	}

	return nil

}

func (vt *VodTranscoder) prepareOutputDir() {
	wd, _ := os.Getwd()
	vt.OutputDir = path.Join(
		wd,
		"output",
		vt.Namespace,
		vt.ID,
	)

	zap.S().Infof("setting up output dir: %s", vt.OutputDir)
	for _, varient := range vt.Resolutions {
		varientPath := path.Join(vt.OutputDir, varient)
		err := os.MkdirAll(varientPath, os.ModePerm)
		if err != nil {
			zap.S().Warnf("transcoder: failed to mkdir, Error: %s", err)
		}
	}
}

func (vt *VodTranscoder) generateCmdString() string {
	prefix := fmt.Sprintf(
		"%s -re -i \"%s\" ",
		vt.FFmpegBin, vt.Source,
	)

	var commands []string

	for _, variant := range vt.Resolutions {
		segmentFilename := fmt.Sprintf("%d_", time.Now().Unix()) + "%04d.ts"
		resolution := vt.getVideoScaledata(variant)
		bitrate := vt.getVariantBitrate(variant)
		framerate := "30"

		cmd := fmt.Sprintf(
			" -r %s -vf %s -c:v libx264 -b:v %dk -c:a aac -b:a 128k -ac 2 -preset fast "+
				"-f hls -hls_time 10 -hls_playlist_type vod -hls_segment_filename %s/%s/%s -start_number 0 %s/%s/%s.m3u8",
			framerate, resolution, bitrate, vt.OutputDir, variant, segmentFilename, vt.OutputDir, variant, variant)

		commands = append(commands, cmd)

	}

	generatethumbnailcmd := fmt.Sprintf(" -ss 1.4 -frames:v 1 %s/thumbnail.jpg", vt.OutputDir)
	commands = append(commands, generatethumbnailcmd)

	finalCmd := fmt.Sprintf("%s %s", prefix, strings.Join(commands, ""))

	return finalCmd

}

func (vt *VodTranscoder) getVideoScaledata(variant string) string {
	switch variant {
	case "240p":
		return "scale=w=426:h=240"
	case "360p":
		return "scale=w=640:h=360"
	case "480p":
		return "scale=w=854:h=480"
	case "720p":
		return "scale=w=1280:h=720"
	case "1080p":
		return "scale=w=1920:h=1080"
	case "1440p":
		return "scale=w=640:h=360"
	case "2160p":
		return "scale=w=1440:h=2160"
	default:
		return "scale=w=1280:h=720"
	}
}

func (vt *VodTranscoder) getVariantBitrate(variant string) int {
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

func (vt *VodTranscoder) generateMasterHls() (m3u8.MasterPlaylist, error) {
	masterHls := m3u8.NewMasterPlaylist()

	for _, varient := range vt.Resolutions {
		genrateVarient := m3u8.Variant{
			URI:           fmt.Sprintf("%s/%s.m3u8", varient, varient),
			VariantParams: vt.getVideoMeatadata(varient),
		}
		masterHls.Variants = append(masterHls.Variants, &genrateVarient)
	}
	masterfilepath := path.Join(vt.OutputDir, vt.MasterFileName)
	vt.MasterHls = masterfilepath
	err := vt.writeFile(masterfilepath, []byte(masterHls.String()))
	return *masterHls, err
}

func (vt *VodTranscoder) writeFile(filepath string, data []byte) error {
	err := os.WriteFile(filepath, data, 0o755)
	return err
}

func (vt *VodTranscoder) getVideoMeatadata(varient string) m3u8.VariantParams {
	ONEKBPS := uint32(1000)

	videoResolutions := map[string]m3u8.VariantParams{
		"240p": {
			ProgramId:  1,
			Resolution: "426x240",
			Name:       "240p",
			Bandwidth:  400 * ONEKBPS,
			FrameRate:  vt.FrameRate,
		},
		"360p": {
			ProgramId:  2,
			Resolution: "640x360",
			Name:       "360p",
			Bandwidth:  1200 * ONEKBPS,
			FrameRate:  vt.FrameRate,
		},
		"480p": {
			ProgramId:  3,
			Resolution: "854x480",
			Name:       "480p",
			Bandwidth:  2000 * ONEKBPS,
			FrameRate:  vt.FrameRate,
		},
		"720p": {
			ProgramId:  4,
			Resolution: "1280x720",
			Name:       "720p",
			Bandwidth:  3500 * ONEKBPS,
			FrameRate:  vt.FrameRate,
		},
		"1080p": {
			ProgramId:  5,
			Resolution: "1920x1080",
			Name:       "1080p",
			Bandwidth:  5000 * ONEKBPS,
			FrameRate:  vt.FrameRate,
		},
		"1440p": {
			ProgramId:  6,
			Resolution: "2560x1440",
			Name:       "2K",
			Bandwidth:  7000 * ONEKBPS,
			FrameRate:  vt.FrameRate,
		},
		"2160p": {
			ProgramId:  7,
			Resolution: "3840x2160",
			Name:       "4K",
			Bandwidth:  10000 * ONEKBPS,
			FrameRate:  vt.FrameRate,
		},
	}

	metadata, found := videoResolutions[varient]
	if !found {
		return videoResolutions[DefaultResolution]
	}
	return metadata
}

func (vt *VodTranscoder) onCompleteTranscode(id string) error {
	activeCmd := externalcmd.GloblaActiveCmds[id]
	zap.S().Infof("total number of runnings hls streaming Count:%s \n current activeCmd: %s", len(externalcmd.GloblaActiveCmds), activeCmd)

	defer activeCmd.Close()

	zap.S().Infof("closing gracefully...\ncmdstring: %s \nprocessid: %s", activeCmd.GetCmdString(), activeCmd.GetProcess().Pid)
	syscall.Kill(-activeCmd.GetProcess().Pid, syscall.SIGINT)
	err := activeCmd.GetProcess().Kill()

	if err != nil {
		zap.S().Errorf("failed to terminate hls stream ID:%s", activeCmd.StreamID)
	}
	zap.S().Infof("closed processid: %s", activeCmd.GetProcess().Pid)
	delete(externalcmd.GloblaActiveCmds, id)

	return nil
}
