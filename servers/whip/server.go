package whip

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pingostack/livhub/pkg/gortc"
	"github.com/pingostack/livhub/pkg/logger"
	"github.com/pingostack/livhub/pkg/plugo"
	"github.com/pion/webrtc/v4"
)

// Config represents the WHIP/WHEP server configuration
type Config struct {
	// HTTP server settings
	Listen string `json:"listen" yaml:"listen"`
	Port   int    `json:"port" yaml:"port"`
	Debug  bool   `json:"debug" yaml:"debug"`

	// WebRTC configuration
	ICEServers []ICEServer `json:"ice_servers" yaml:"ice_servers"`

	// WHIP/WHEP endpoints
	WhipEndpoint string `json:"whip_endpoint" yaml:"whip_endpoint"`
	WhepEndpoint string `json:"whep_endpoint" yaml:"whep_endpoint"`
}

// ICEServer represents a STUN/TURN server configuration
type ICEServer struct {
	URLs       []string `json:"urls" yaml:"urls"`
	Username   string   `json:"username" yaml:"username"`
	Credential string   `json:"credential" yaml:"credential"`
}

// Stream represents a WebRTC stream with its publisher and subscribers
type Stream struct {
	ID          string
	Publisher   *gortc.Peer
	Subscribers sync.Map // string -> *gortc.Peer
}

// Server implements the WHIP/WHEP signaling server
type Server struct {
	config     *Config
	engine     *gin.Engine
	streams    sync.Map // string -> *Stream
	httpServer *http.Server
}

var (
	debugMode = "false"

	defaultConfig = &Config{
		Listen:       "0.0.0.0",
		Port:         8080,
		Debug:        debugMode == "true",
		WhipEndpoint: "/whip",
		WhepEndpoint: "/whep",
		ICEServers: []ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
	globalServer *Server
)

func init() {
	err := plugo.RegisterPlugin("whip",
		plugo.WithConfig("whip", defaultConfig, func(ctx context.Context, cfg interface{}) error {
			if config, ok := cfg.(*Config); ok {
				globalServer = NewServer(config)
				return nil
			}
			return fmt.Errorf("invalid config type: %T", cfg)
		}),
		plugo.WithInit(func(ctx context.Context) error {
			if globalServer == nil {
				return nil
			}
			globalServer.setupRoutes()
			return nil
		}),
		plugo.WithRun(func(ctx context.Context) error {
			if globalServer == nil {
				return fmt.Errorf("server not initialized")
			}

			addr := fmt.Sprintf("%s:%d", globalServer.config.Listen, globalServer.config.Port)
			globalServer.httpServer = &http.Server{
				Addr:    addr,
				Handler: globalServer.engine,
			}

			errChan := make(chan error, 1)
			go func() {
				if err := globalServer.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					errChan <- err
				}
			}()

			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-errChan:
				return fmt.Errorf("http server error: %v", err)
			}
		}),
		plugo.WithExit(func(ctx context.Context) error {
			if globalServer == nil {
				return nil
			}

			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			if globalServer.httpServer != nil {
				if err := globalServer.httpServer.Shutdown(shutdownCtx); err != nil {
					return fmt.Errorf("error shutting down server: %v", err)
				}
			}

			// Cleanup all streams and peers
			globalServer.streams.Range(func(key, value interface{}) bool {
				if stream, ok := value.(*Stream); ok {
					if stream.Publisher != nil {
						stream.Publisher.Close()
					}
					stream.Subscribers.Range(func(_, subValue interface{}) bool {
						if sub, ok := subValue.(*gortc.Peer); ok {
							sub.Close()
						}
						return true
					})
				}
				return true
			})

			return nil
		}),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to register WHIP server plugin: %v", err))
	}
}

// NewServer creates a new WHIP/WHEP server instance
func NewServer(config *Config) *Server {
	if config.Debug {
		gin.SetMode(gin.DebugMode)
		logger.Info("Running in debug mode")
	} else {
		gin.SetMode(gin.ReleaseMode)
		logger.Info("Running in release mode")
	}

	log := logger.NewLogger().WithField("component", "gin")
	gin.DefaultWriter = &loggerWriter{logger: log}

	engine := gin.New()
	engine.Use(
		gin.Recovery(),
		requestLogger(),
	)

	return &Server{
		config: config,
		engine: engine,
	}
}

// setupRoutes configures the HTTP routes for WHIP and WHEP
func (s *Server) setupRoutes() {
	s.engine.POST(s.config.WhipEndpoint, s.handleWhipPublish)
	s.engine.DELETE(s.config.WhipEndpoint+"/*resourceId", s.handleWhipUnpublish)
	s.engine.POST(s.config.WhepEndpoint, s.handleWhepSubscribe)
	s.engine.DELETE(s.config.WhepEndpoint+"/*resourceId", s.handleWhepUnsubscribe)
}

// handleWhipPublish handles WHIP publishing requests
func (s *Server) handleWhipPublish(c *gin.Context) {
	log := logger.WithField("handler", "whip_publish")

	var offer webrtc.SessionDescription
	if err := c.ShouldBindJSON(&offer); err != nil {
		log.WithError(err).Error("Failed to parse SDP offer")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid SDP offer"})
		return
	}

	// Create peer config
	config := gortc.PeerConfig{
		ICEServers: make([]webrtc.ICEServer, len(s.config.ICEServers)),
	}
	for i, server := range s.config.ICEServers {
		config.ICEServers[i] = webrtc.ICEServer{
			URLs:       server.URLs,
			Username:   server.Username,
			Credential: server.Credential,
		}
	}

	// Generate IDs
	streamID := uuid.New().String()
	resourceID := uuid.New().String()

	// Create publisher peer
	publisher, err := gortc.NewPeer(resourceID, gortc.PeerTypePublisher, config)
	if err != nil {
		log.WithError(err).Error("Failed to create peer")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create peer"})
		return
	}

	// Create stream
	stream := &Stream{
		ID:        streamID,
		Publisher: publisher,
	}

	// Handle new tracks
	go func() {
		for trackInfo := range publisher.OnTrack() {
			info := trackInfo
			log.WithField("track_id", info.ID).Info("New track received")

			// Forward track to all subscribers
			stream.Subscribers.Range(func(_, value interface{}) bool {
				subscriber := value.(*gortc.Peer)
				if _, err := subscriber.AddTrack(info); err != nil {
					log.WithError(err).Error("Failed to add track to subscriber")
				}
				return true
			})
		}
	}()

	// Handle close
	go func() {
		<-publisher.OnClose()
		s.cleanupPublisher(streamID, resourceID)
	}()

	// Set remote description
	if err = publisher.SetRemoteDescription(offer.SDP); err != nil {
		log.WithError(err).Error("Failed to set remote description")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set remote description"})
		return
	}

	// Store the stream
	s.streams.Store(streamID, stream)

	// Set response headers
	c.Header("Location", fmt.Sprintf("%s/%s", s.config.WhipEndpoint, resourceID))
	c.Header("Link", "</whep>;rel=channel")

	// Return the answer
	c.String(http.StatusCreated, publisher.GetLocalDescription())
}

// handleWhipUnpublish handles WHIP unpublishing requests
func (s *Server) handleWhipUnpublish(c *gin.Context) {
	resourceID := strings.TrimPrefix(c.Param("resourceId"), "/")
	if resourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing resource ID"})
		return
	}

	if err := s.cleanupPublisher("", resourceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cleanup publisher"})
		return
	}

	c.Status(http.StatusOK)
}

// handleWhepSubscribe handles WHEP subscription requests
func (s *Server) handleWhepSubscribe(c *gin.Context) {
	log := logger.WithField("handler", "whep_subscribe")

	var offer webrtc.SessionDescription
	if err := c.ShouldBindJSON(&offer); err != nil {
		log.WithError(err).Error("Failed to parse SDP offer")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid SDP offer"})
		return
	}

	streamID := extractStreamIDFromSDP(offer.SDP)
	if streamID == "" {
		log.Error("No stream ID found in SDP")
		c.JSON(http.StatusBadRequest, gin.H{"error": "No stream ID specified"})
		return
	}

	streamValue, ok := s.streams.Load(streamID)
	if !ok {
		log.WithField("stream_id", streamID).Error("Stream not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Stream not found"})
		return
	}
	stream := streamValue.(*Stream)

	// Create peer config
	config := gortc.PeerConfig{
		ICEServers: make([]webrtc.ICEServer, len(s.config.ICEServers)),
	}
	for i, server := range s.config.ICEServers {
		config.ICEServers[i] = webrtc.ICEServer{
			URLs:       server.URLs,
			Username:   server.Username,
			Credential: server.Credential,
		}
	}

	// Generate resource ID
	resourceID := uuid.New().String()

	// Create subscriber peer
	subscriber, err := gortc.NewPeer(resourceID, gortc.PeerTypeSubscriber, config)
	if err != nil {
		log.WithError(err).Error("Failed to create peer")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create peer"})
		return
	}

	// Handle close
	go func() {
		<-subscriber.OnClose()
		s.cleanupSubscriber(streamID, resourceID)
	}()

	// Set remote description
	if err = subscriber.SetRemoteDescription(offer.SDP); err != nil {
		log.WithError(err).Error("Failed to set remote description")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set remote description"})
		return
	}

	// Store the subscriber
	stream.Subscribers.Store(resourceID, subscriber)

	// Set response headers
	c.Header("Location", fmt.Sprintf("%s/%s", s.config.WhepEndpoint, resourceID))

	// Return the answer
	c.String(http.StatusCreated, subscriber.GetLocalDescription())
}

// handleWhepUnsubscribe handles WHEP unsubscription requests
func (s *Server) handleWhepUnsubscribe(c *gin.Context) {
	resourceID := strings.TrimPrefix(c.Param("resourceId"), "/")
	if resourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing resource ID"})
		return
	}

	if err := s.cleanupSubscriber("", resourceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cleanup subscriber"})
		return
	}

	c.Status(http.StatusOK)
}

// Helper functions

func (s *Server) cleanupPublisher(streamID, resourceID string) error {
	// If stream ID is not provided, find it
	if streamID == "" {
		s.streams.Range(func(key, value interface{}) bool {
			stream := value.(*Stream)
			if stream.Publisher != nil && stream.Publisher.ID() == resourceID {
				streamID = key.(string)
				return false
			}
			return true
		})
	}

	// Remove the stream if found
	if streamID != "" {
		if streamValue, ok := s.streams.LoadAndDelete(streamID); ok {
			stream := streamValue.(*Stream)
			// Close publisher
			if stream.Publisher != nil {
				stream.Publisher.Close()
			}
			// Close all subscribers
			stream.Subscribers.Range(func(_, value interface{}) bool {
				if subscriber, ok := value.(*gortc.Peer); ok {
					subscriber.Close()
				}
				return true
			})
		}
	}

	return nil
}

func (s *Server) cleanupSubscriber(streamID, resourceID string) error {
	// If stream ID is not provided, find it
	if streamID == "" {
		s.streams.Range(func(key, value interface{}) bool {
			stream := value.(*Stream)
			if sub, ok := stream.Subscribers.LoadAndDelete(resourceID); ok {
				if subscriber, ok := sub.(*gortc.Peer); ok {
					subscriber.Close()
				}
				streamID = key.(string)
				return false
			}
			return true
		})
	} else if streamValue, ok := s.streams.Load(streamID); ok {
		stream := streamValue.(*Stream)
		if sub, ok := stream.Subscribers.LoadAndDelete(resourceID); ok {
			if subscriber, ok := sub.(*gortc.Peer); ok {
				subscriber.Close()
			}
		}
	}

	return nil
}

type loggerWriter struct {
	logger logger.Logger
}

func (w *loggerWriter) Write(p []byte) (n int, err error) {
	w.logger.Info(string(p))
	return len(p), nil
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		if raw != "" {
			path = path + "?" + raw
		}

		log := logger.WithFields(map[string]interface{}{
			"status":     c.Writer.Status(),
			"method":     c.Request.Method,
			"path":       path,
			"latency":    time.Since(start).String(),
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		})

		if len(c.Errors) > 0 {
			log.Error(c.Errors.String())
		} else {
			log.Info("request completed")
		}
	}
}

func extractStreamIDFromSDP(sdp string) string {
	lines := strings.Split(sdp, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "a=group:BUNDLE ") {
			parts := strings.Split(line, " ")
			if len(parts) > 1 {
				return parts[1]
			}
		}
	}
	return ""
}
