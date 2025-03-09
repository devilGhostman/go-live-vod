package server

import (
	outputcontroller "github/devilGhostman/backend/internal/outputController"
	"github/devilGhostman/backend/internal/services/livestream"
	videoOnDemand "github/devilGhostman/backend/internal/services/vod"

	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	Address string
	gin     *gin.Engine
	root    *gin.RouterGroup
}

func NewServer(address string) *Server {
	zap.S().Infof("starting httproxy server at %s", address)
	server := &Server{
		Address: address,
		gin:     gin.Default(),
	}
	return server
}

func (s *Server) SetupRouter() {

	// cors
	s.gin.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"}, // Allow specific origin
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           24 * 3600, // 24 hours
	}))

	s.root = s.gin.Group("/")
	s.gin.MaxMultipartMemory = 100 << 20

	// health check
	s.root.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "success",
		})
	})

	// liveStream routers
	liveStream := s.root.Group("/livestream")
	liveStream.POST("/auth", livestream.Auth)
	liveStream.POST("/", livestream.StartHlsTranscode)
	liveStream.DELETE("/", livestream.StopHlsTranscode)
	liveStream.GET("/", livestream.GetAllLiveStreams)

	// vod routers
	vod := s.root.Group("/vod")
	vod.POST("/", videoOnDemand.StartVodTranscode)
	vod.GET("/", videoOnDemand.GetAllVods)

	// for serving hls file
	hls := s.root.Group("/hls")
	hls.Use(outputcontroller.HlsOutputControllerMiddleware())
	hls.StaticFS("/", http.Dir("output"))

}

func (s *Server) Run() {
	zap.S().Infof("started listening on %s", s.Address)
	s.gin.Run(s.Address)
}
