package types

// RtmpConfigHTTP is a struct to handle rtmp config http request
type RtmpConfigHTTP struct {
	ID         string `json:"id" validate:"required"`
	Namespace  string `json:"namespace" validate:"required"`
	StreamName string `json:"streamName" validate:"required"`
	RtmpUrl    string `json:"rtmpUrl"`
	Config     struct {
		Codecs struct {
			Video string `json:"video"`
			Audio string `json:"audio"`
		}
		Resolutions []string `json:"resolutions"`
	} `json:"config"`
}
