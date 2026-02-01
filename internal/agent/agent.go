package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/denizumutdereli/agents-p2p-network/internal/api"
	"github.com/denizumutdereli/agents-p2p-network/internal/config"
	"github.com/denizumutdereli/agents-p2p-network/internal/p2p"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

type Agent struct {
	config     *config.Config
	p2pHost    *p2p.Host
	apiServer  *api.Server
	logger     *zap.Logger
	httpClient *http.Client

	agentRegistry map[string]*AgentRecord
}

type AgentRecord struct {
	PeerID   peer.ID
	Name     string
	Endpoint string
	Models   []string
}

func New(cfg *config.Config) (*Agent, error) {
	logger, _ := zap.NewProduction()

	a := &Agent{
		config:        cfg,
		logger:        logger,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		agentRegistry: make(map[string]*AgentRecord),
	}

	return a, nil
}

func (a *Agent) Start(ctx context.Context) error {
	var err error
	a.p2pHost, err = p2p.NewHost(ctx, a.config.P2PPort, a.logger)
	if err != nil {
		return fmt.Errorf("failed to create P2P host: %w", err)
	}

	a.p2pHost.SetLocalName(a.config.AgentName)
	a.p2pHost.SetMessageHandler(a.handleP2PMessage)

	if err := a.p2pHost.StartMDNS(); err != nil {
		a.logger.Warn("Failed to start mDNS discovery", zap.Error(err))
	}

	a.p2pHost.StartDHTDiscovery()

	if a.config.BootstrapPeer != "" {
		if err := a.p2pHost.ConnectBootstrap(a.config.BootstrapPeer); err != nil {
			a.logger.Warn("Failed to connect to bootstrap peer", zap.Error(err))
		}
	}

	a.apiServer = api.NewServer(a.config.HTTPPort, a.config.APIKey, a, a.logger)
	if err := a.apiServer.Start(); err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}

	a.broadcastRegistration(ctx)

	return nil
}

func (a *Agent) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if a.apiServer != nil {
		a.apiServer.Stop(ctx)
	}
	if a.p2pHost != nil {
		a.p2pHost.Close()
	}
}

func (a *Agent) PeerID() string {
	return a.p2pHost.ID().String()
}

func (a *Agent) handleP2PMessage(ctx context.Context, from peer.ID, msg *p2p.Message) (*p2p.Message, error) {
	switch msg.Type {
	case p2p.MessageTypeRegister:
		return a.handleRegister(from, msg)
	case p2p.MessageTypeChat:
		return a.handleChatRequest(ctx, from, msg)
	case p2p.MessageTypePing:
		return a.handlePing(from, msg)
	case p2p.MessageTypeAnnounce:
		return a.handleAnnounce(from, msg)
	default:
		a.logger.Warn("Unknown message type", zap.String("type", string(msg.Type)))
		return nil, nil
	}
}

func (a *Agent) handleRegister(from peer.ID, msg *p2p.Message) (*p2p.Message, error) {
	var payload p2p.RegisterPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, err
	}

	// Check for duplicate agent name
	if err := a.p2pHost.RegisterAgentName(payload.AgentName, from); err != nil {
		a.logger.Warn("Duplicate agent name rejected", 
			zap.String("name", payload.AgentName), 
			zap.String("peer_id", from.String()),
			zap.Error(err))
		
		errPayload, _ := json.Marshal(map[string]string{"error": err.Error()})
		return &p2p.Message{
			Type:    p2p.MessageTypeError,
			From:    a.p2pHost.ID().String(),
			Payload: errPayload,
		}, nil
	}

	a.agentRegistry[from.String()] = &AgentRecord{
		PeerID:   from,
		Name:     payload.AgentName,
		Endpoint: payload.Endpoint,
		Models:   payload.Models,
	}

	a.logger.Info("Agent registered", zap.String("name", payload.AgentName), zap.String("peer_id", from.String()))

	return &p2p.Message{
		Type: p2p.MessageTypePong,
		From: a.p2pHost.ID().String(),
	}, nil
}

func (a *Agent) handleChatRequest(ctx context.Context, from peer.ID, msg *p2p.Message) (*p2p.Message, error) {
	var chatReq api.ChatCompletionRequest
	if err := json.Unmarshal(msg.Payload, &chatReq); err != nil {
		return nil, err
	}

	resp, err := a.forwardToOpenAI(ctx, &chatReq)
	if err != nil {
		return nil, err
	}

	respPayload, _ := json.Marshal(resp)
	return &p2p.Message{
		Type:      p2p.MessageTypeComplete,
		From:      a.p2pHost.ID().String(),
		RequestID: msg.RequestID,
		Payload:   respPayload,
	}, nil
}

func (a *Agent) handlePing(from peer.ID, msg *p2p.Message) (*p2p.Message, error) {
	return &p2p.Message{
		Type: p2p.MessageTypePong,
		From: a.p2pHost.ID().String(),
	}, nil
}

func (a *Agent) handleAnnounce(from peer.ID, msg *p2p.Message) (*p2p.Message, error) {
	var payload p2p.AnnouncePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, err
	}

	a.logger.Info("📢 Received announcement",
		zap.String("from", from.String()[:12]),
		zap.String("type", payload.Type),
		zap.String("name", payload.Name),
		zap.String("url", payload.URL),
		zap.Strings("tags", payload.Tags))

	return &p2p.Message{
		Type: p2p.MessageTypePong,
		From: a.p2pHost.ID().String(),
	}, nil
}

func (a *Agent) broadcastRegistration(ctx context.Context) {
	payload := p2p.RegisterPayload{
		AgentName: a.config.AgentName,
		Endpoint:  fmt.Sprintf("http://localhost:%d", a.config.HTTPPort),
		Models:    []string{"gpt-4", "gpt-3.5-turbo"},
	}

	payloadBytes, _ := json.Marshal(payload)
	msg := &p2p.Message{
		Type:    p2p.MessageTypeRegister,
		From:    a.p2pHost.ID().String(),
		Payload: payloadBytes,
	}

	a.p2pHost.Broadcast(ctx, msg)
}

func (a *Agent) forwardToOpenAI(ctx context.Context, req *api.ChatCompletionRequest) (*api.ChatCompletionResponse, error) {
	body, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.config.APIKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var chatResp api.ChatCompletionResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	return &chatResp, nil
}

func (a *Agent) HandleChatCompletion(ctx context.Context, req *api.ChatCompletionRequest) (*api.ChatCompletionResponse, error) {
	return a.forwardToOpenAI(ctx, req)
}

func (a *Agent) HandleListModels(ctx context.Context) (*api.ModelsResponse, error) {
	return &api.ModelsResponse{
		Object: "list",
		Data: []api.Model{
			{ID: "gpt-4", Object: "model", Created: time.Now().Unix(), OwnedBy: "openai"},
			{ID: "gpt-3.5-turbo", Object: "model", Created: time.Now().Unix(), OwnedBy: "openai"},
		},
	}, nil
}

func (a *Agent) HandleListAgents(ctx context.Context) (*api.AgentsResponse, error) {
	peers := a.p2pHost.GetPeers()
	agents := make([]api.AgentInfo, 0)

	for _, p := range peers {
		record, exists := a.agentRegistry[p.ID.String()]
		agentInfo := api.AgentInfo{
			ID:        p.ID.String(),
			PeerID:    p.ID.String(),
			Connected: p.Connected,
		}

		if exists {
			agentInfo.Name = record.Name
			agentInfo.Endpoint = record.Endpoint
			agentInfo.Models = record.Models
		}

		agents = append(agents, agentInfo)
	}

	return &api.AgentsResponse{
		Object: "list",
		Data:   agents,
	}, nil
}

func (a *Agent) HandleSendToAgent(ctx context.Context, agentID string, req *api.ChatCompletionRequest) (*api.ChatCompletionResponse, error) {
	peerID, err := peer.Decode(agentID)
	if err != nil {
		return nil, fmt.Errorf("invalid agent ID: %w", err)
	}

	payload, _ := json.Marshal(req)
	msg := &p2p.Message{
		Type:      p2p.MessageTypeChat,
		From:      a.p2pHost.ID().String(),
		To:        agentID,
		RequestID: uuid.New().String(),
		Payload:   payload,
	}

	resp, err := a.p2pHost.SendMessage(ctx, peerID, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send to agent: %w", err)
	}

	if resp == nil {
		return nil, fmt.Errorf("no response from agent")
	}

	var chatResp api.ChatCompletionResponse
	if err := json.Unmarshal(resp.Payload, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse agent response: %w", err)
	}

	return &chatResp, nil
}

func (a *Agent) HandleAnnounce(ctx context.Context, req *api.AnnounceRequest) error {
	payload := p2p.AnnouncePayload{
		Type:        req.Type,
		Name:        req.Name,
		URL:         req.URL,
		Description: req.Description,
		Tags:        req.Tags,
	}

	payloadBytes, _ := json.Marshal(payload)
	msg := &p2p.Message{
		Type:    p2p.MessageTypeAnnounce,
		From:    a.p2pHost.ID().String(),
		Payload: payloadBytes,
	}

	a.logger.Info("Broadcasting announcement",
		zap.String("type", req.Type),
		zap.String("name", req.Name),
		zap.String("url", req.URL))

	return a.p2pHost.Broadcast(ctx, msg)
}
