package types

type VodConfigHTTP struct {
	ID         string `json:"id" validate:"required"`
	Namespace  string `json:"namespace" validate:"required"`
	StreamName string `json:"streamName" validate:"required"`
	SourceUrl  string `json:"sourceUrl" validate:"required"`
	Config     struct {
		Codecs struct {
			Video string `json:"video"`
			Audio string `json:"audio"`
		} `json:"codecs"`
		Resolutions []string `json:"resolutions"`
	} `json:"config"`
}
