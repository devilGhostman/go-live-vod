package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// VideoUrls represents the structure for different video quality URLs and the poster
type VideoUrls struct {
	Master string `bson:"master,omitempty" json:"master,omitempty"`
	U144p  string `bson:"144p,omitempty" json:"144p,omitempty"`
	U240p  string `bson:"240p,omitempty" json:"240p,omitempty"`
	U360p  string `bson:"360p,omitempty" json:"360p,omitempty"`
	U480p  string `bson:"480p,omitempty" json:"480p,omitempty"`
	U720p  string `bson:"720p,omitempty" json:"720p,omitempty"`
	U1080p string `bson:"1080p,omitempty" json:"1080p,omitempty"`
	Poster string `bson:"poster,omitempty" json:"poster,omitempty"`
	Audio  string `bson:"audio,omitempty" json:"audio,omitempty"`
}

type Config struct {
	Codecs struct {
		Video string `bson:"video,omitempty" json:"video,omitempty"`
		Audio string `bson:"audio,omitempty" json:"audio,omitempty"`
	} `bson:"codecs,omitempty" json:"codecs,omitempty"`
	Resolutions []string `bson:"resolutions,omitempty" json:"resolutions,omitempty"`
}

// Transcode represents the schema for a transcoding job
type Transcode struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Title     string             `bson:"title" json:"title"`
	VideoUrls VideoUrls          `bson:"videoUrls" json:"videoUrls"`
	Config    Config             `bson:"config,omitempty" json:"config,omitempty"`
	Status    string             `bson:"status" json:"status"`
	CreatedAt string             `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
	UpdatedAt string             `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

// NewTranscode creates a new Transcode struct with default values
func NewTranscode(title string) *Transcode {
	return &Transcode{
		Title: title,
	}
}
