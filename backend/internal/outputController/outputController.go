package outputcontroller

import (
	"os"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)


// HlsOutputController is a controller for HLS output
type HlsOutputController struct {
	ctx        *gin.Context
	input      string // input is the path to the input file
	output     string // output is the path to the output file
	namespace  string // namespace is the namespace of the stream
	streamid   string // streamid is the stream id
	resolution string // resolution is the resolution of the stream
}

// Middleware wraps the gin.HandlerFunc with the output controller
func HlsOutputControllerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		hls := NewHlsOutputController(c, c.Request.URL.Path)
		if hls.shouldIgnore() {
			zap.S().Info("ignoring hls file ", zap.String("path", c.Request.URL.Path))
			c.Next()
			return
		}
		err := hls.Send()
		if err != nil {
			c.Next()
			return
		}
	}
}

func NewHlsOutputController(ctx *gin.Context, input string) *HlsOutputController {
	hoc := &HlsOutputController{
		ctx:    ctx,
		input:  input,
		output: "output",
	}
	hoc.loadMetadata() // load metadata
	return hoc
}

func (hoc *HlsOutputController) Send() error {

	// fix path
	suffixpath := strings.Split(hoc.input, "/hls/")
	wd, _ := os.Getwd()
	hoc.input = path.Join(wd, hoc.output, suffixpath[1])

	zap.S().Info("serving hls file ", zap.String("path", hoc.input))

	// check if segment file exists
	if _, err := os.Stat(hoc.input); os.IsNotExist(err) {
		zap.S().Warn("segment file does not exist yet ", hoc.input)
		return nil
	}

	// setting up the headers
	hoc.ctx.Header("X-Served-By", "hls")
	hoc.ctx.Header("X-Server-Time", time.Now().Format(time.RFC3339))

	zap.S().Info("served hls file through HlsOutputController ", zap.String("path", hoc.input))
	hoc.ctx.File(hoc.input)
	return nil
}

func (hoc *HlsOutputController) shouldIgnore() bool {
	// if master playlist, ignore
	if strings.Contains(hoc.input, "playlist.m3u8") {
		return true
	}

	// if ts file, ignore
	if strings.Contains(hoc.input, ".ts") {
		return true
	}

	return false
}

func (hoc *HlsOutputController) loadMetadata() {
	path := hoc.ctx.Request.URL.Path

	if strings.Split(path, "/")[1] != "hls" {
		zap.S().Warn("invalid path ", zap.String("path", path))
		return
	}

	hoc.namespace = strings.Split(path, "/")[2]
	hoc.streamid = strings.Split(path, "/")[3]
	hoc.resolution = strings.Split(path, "/")[4]

	zap.S().Info("loaded metadata ", zap.String("namespace", hoc.namespace), zap.String("streamid", hoc.streamid), zap.String("resolution", hoc.resolution))
}
