package whip

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pingostack/livhub/pkg/logger"
	"github.com/pingostack/livhub/pkg/plugo"
	"github.com/pion/webrtc/v3"
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

// Server implements the WHIP/WHEP signaling server
type Server struct {
	config     *Config
	engine     *gin.Engine
	peerConns  sync.Map // string -> *webrtc.PeerConnection
	httpServer *http.Server
}

var (
	// Debug mode flag, can be set via build flag: -ldflags="-X github.com/pingostack/livhub/servers/whip.debugMode=true"
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
	// global server instance for sharing across lifecycle functions
	globalServer *Server
)

func init() {
	// Register the WHIP/WHEP server plugin
	err := plugo.RegisterPlugin("whip",
		plugo.WithConfig("whip", defaultConfig, func(ctx context.Context, cfg interface{}) error {
			if config, ok := cfg.(*Config); ok {
				globalServer = NewServer(config)
				return nil
			}
			return fmt.Errorf("invalid config type: %T", cfg)
		}),
		// Init phase: setup routes
		plugo.WithInit(func(ctx context.Context) error {
			if globalServer == nil {
				return nil
			}
			globalServer.setupRoutes()
			return nil
		}),
		// Run phase: start HTTP server
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

			// Wait for context cancellation or server error
			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-errChan:
				return fmt.Errorf("http server error: %v", err)
			}
		}),
		// Exit phase: shutdown HTTP server and cleanup resources
		plugo.WithExit(func(ctx context.Context) error {
			if globalServer == nil {
				return nil
			}

			// Create a context with timeout for shutdown
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			// Shutdown HTTP server
			if globalServer.httpServer != nil {
				if err := globalServer.httpServer.Shutdown(shutdownCtx); err != nil {
					return fmt.Errorf("error shutting down server: %v", err)
				}
			}

			// Cleanup all WebRTC connections
			globalServer.peerConns.Range(func(key, value interface{}) bool {
				if pc, ok := value.(*webrtc.PeerConnection); ok {
					pc.Close()
				}
				globalServer.peerConns.Delete(key)
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
	// Set gin mode based on config
	if config.Debug {
		gin.SetMode(gin.DebugMode)
		logger.Info("Running in debug mode")
	} else {
		gin.SetMode(gin.ReleaseMode)
		logger.Info("Running in release mode")
	}

	// Create a logger with component field
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

// loggerWriter implements io.Writer interface for gin logging
type loggerWriter struct {
	logger logger.Logger
}

func (w *loggerWriter) Write(p []byte) (n int, err error) {
	w.logger.Info(string(p))
	return len(p), nil
}

// requestLogger returns a gin middleware for logging requests
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request details
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

// setupRoutes configures the HTTP routes for WHIP and WHEP
func (s *Server) setupRoutes() {
	// WHIP endpoint for publishers
	s.engine.POST(s.config.WhipEndpoint, s.handleWhipPublish)
	s.engine.DELETE(s.config.WhipEndpoint+"/*resourceId", s.handleWhipUnpublish)

	// WHEP endpoint for players
	s.engine.POST(s.config.WhepEndpoint, s.handleWhepSubscribe)
	s.engine.DELETE(s.config.WhepEndpoint+"/*resourceId", s.handleWhepUnsubscribe)
}

// handleWhipPublish handles WHIP publishing requests
func (s *Server) handleWhipPublish(c *gin.Context) {
	// TODO: Implement WHIP publishing
	c.Status(http.StatusNotImplemented)
}

// handleWhipUnpublish handles WHIP unpublishing requests
func (s *Server) handleWhipUnpublish(c *gin.Context) {
	// TODO: Implement WHIP unpublishing
	c.Status(http.StatusNotImplemented)
}

// handleWhepSubscribe handles WHEP subscription requests
func (s *Server) handleWhepSubscribe(c *gin.Context) {
	// TODO: Implement WHEP subscription
	c.Status(http.StatusNotImplemented)
}

// handleWhepUnsubscribe handles WHEP unsubscription requests
func (s *Server) handleWhepUnsubscribe(c *gin.Context) {
	// TODO: Implement WHEP unsubscription
	c.Status(http.StatusNotImplemented)
}
