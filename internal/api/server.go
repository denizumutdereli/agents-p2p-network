package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	router     *gin.Engine
	httpServer *http.Server
	logger     *zap.Logger
	apiKey     string
	handler    RequestHandler
}

type RequestHandler interface {
	HandleChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error)
	HandleListModels(ctx context.Context) (*ModelsResponse, error)
	HandleListAgents(ctx context.Context) (*AgentsResponse, error)
	HandleSendToAgent(ctx context.Context, agentID string, req *ChatCompletionRequest) (*ChatCompletionResponse, error)
}

func NewServer(port int, apiKey string, handler RequestHandler, logger *zap.Logger) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	s := &Server{
		router:  router,
		logger:  logger,
		apiKey:  apiKey,
		handler: handler,
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: router,
		},
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.GET("/health", s.healthCheck)

	v1 := s.router.Group("/v1")
	v1.Use(s.authMiddleware())
	{
		v1.GET("/models", s.listModels)
		v1.POST("/chat/completions", s.chatCompletions)

		v1.GET("/agents", s.listAgents)
		v1.POST("/agents/:agent_id/chat/completions", s.agentChatCompletions)
	}
}

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Missing Authorization header",
					"type":    "invalid_request_error",
				},
			})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if token != s.apiKey {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Invalid API key",
					"type":    "invalid_request_error",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Unix(),
	})
}

func (s *Server) listModels(c *gin.Context) {
	resp, err := s.handler.HandleListModels(c.Request.Context())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Server) chatCompletions(c *gin.Context) {
	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	resp, err := s.handler.HandleChatCompletion(c.Request.Context(), &req)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Server) listAgents(c *gin.Context) {
	resp, err := s.handler.HandleListAgents(c.Request.Context())
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Server) agentChatCompletions(c *gin.Context) {
	agentID := c.Param("agent_id")

	var req ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	resp, err := s.handler.HandleSendToAgent(c.Request.Context(), agentID, &req)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Server) errorResponse(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"message": message,
			"type":    "api_error",
		},
	})
}

func (s *Server) Start() error {
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", zap.Error(err))
		}
	}()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
