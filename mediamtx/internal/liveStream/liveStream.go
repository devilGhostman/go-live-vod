package livestream

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/bluenviron/mediamtx/internal/utils"
)

type LiveStream struct {
	Id         string
	Namespace  string
	streamName string

	config struct {
		codecs struct {
			video string
			audio string
		}
		resolutions []string
	}
}

var GlobalLiveStreamInstance map[string]*LiveStream = make(map[string]*LiveStream)

func NewLiveStream(streamName string) *LiveStream {
	id := strings.Split(streamName, "/")[1]
	namespace := strings.Split(streamName, "/")[0]
	LiveStreamInstance := &LiveStream{
		Id:         id,
		Namespace:  namespace,
		streamName: streamName,
	}

	LiveStreamInstance.config.codecs.video = "h264"
	LiveStreamInstance.config.codecs.audio = "aac"

	LiveStreamInstance.config.resolutions = []string{"144p", "240p", "360p", "480p", "720p", "1080p"}

	AddLiveStreamInstance(LiveStreamInstance)
	return LiveStreamInstance
}

func AddLiveStreamInstance(instance *LiveStream) {
	log.Default().Printf("addLiveStreamInstance %s\n", instance.streamName)
	GlobalLiveStreamInstance[instance.streamName] = instance
}

func GetLiveStreamInstance(streamName string) *LiveStream {
	log.Default().Printf("getLiveStreamInstance %s\n", streamName)
	return GlobalLiveStreamInstance[streamName]
}

func (ls *LiveStream) StartPublishToTranscoder() error {
	transcodeUrl := os.Getenv("API_BASE_URL")
	rtmlBaseUrl := os.Getenv("RTMP_SERVER_URL")
	payload := struct {
		Id         string `json:"id"`
		Namespace  string `json:"namespace"`
		StreamName string `json:"streamName"`
		RtmpUrl    string `json:"rtmpUrl"`
		Config     struct {
			Codecs struct {
				Video string `json:"video"`
				Audio string `json:"audio"`
			} `json:"codecs"`
			Resolutions []string `json:"resolutions"`
		}
	}{
		Id:         ls.Id,
		Namespace:  ls.Namespace,
		StreamName: ls.streamName,
	}

	payload.Config.Codecs.Video = ls.config.codecs.video
	payload.Config.Codecs.Audio = ls.config.codecs.audio
	payload.Config.Resolutions = ls.config.resolutions
	payload.RtmpUrl = rtmlBaseUrl + "/" + ls.streamName

	log.Printf("payload: %v", payload)
	payloadbytes, _ := json.Marshal(payload)

	header := make(map[string]string)
	header["Content-Type"] = "application/json"

	transcodeResp, err := utils.RetryApi(http.MethodPost, transcodeUrl, bytes.NewBuffer(payloadbytes), header)
	if err != nil {
		log.Printf("error while publishing to hlsproxy: %v", err)
		return err
	}

	if transcodeResp == nil {
		return errors.New("transcodeResp is nil")
	}
	log.Printf("response Status: %s", transcodeResp)
	return nil
}

func (ls *LiveStream) StopPublishToTranscoder() error {
	baseUrl := os.Getenv("API_BASE_URL")
	transcodeUrl := fmt.Sprintf("%s/?id=%s&namespace=%s", baseUrl, ls.Id, ls.Namespace)

	header := make(map[string]string)
	header["Content-Type"] = "application/json"

	transcodeResp, err := utils.RetryApi(http.MethodDelete, transcodeUrl, nil, header)
	if err != nil {
		log.Printf("error while publishing to hlsproxy: %v", err)
		return err
	}

	if transcodeResp == nil {
		return errors.New("transcodeResp is nil")
	}
	log.Printf("response Status: %s", transcodeResp)
	return nil
}
