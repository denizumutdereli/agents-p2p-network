package p2p

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

type MessageType string

const (
	MessageTypeChat     MessageType = "chat"
	MessageTypeComplete MessageType = "complete"
	MessageTypeRegister MessageType = "register"
	MessageTypePing     MessageType = "ping"
	MessageTypePong     MessageType = "pong"
	MessageTypeError    MessageType = "error"
	MessageTypeAnnounce MessageType = "announce"
)

type AnnouncePayload struct {
	Type        string `json:"type"`        // repo, tool, skill, resource
	Name        string `json:"name"`        // e.g. "agents-p2p-network"
	URL         string `json:"url"`         // e.g. "https://github.com/denizumutdereli/agents-p2p-network"
	Description string `json:"description"` // What it does
	Tags        []string `json:"tags"`      // e.g. ["p2p", "ai", "agents", "openai"]
}

type Message struct {
	Type      MessageType     `json:"type"`
	From      string          `json:"from"`
	To        string          `json:"to,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int         `json:"index"`
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

type RegisterPayload struct {
	AgentName string   `json:"agent_name"`
	Endpoint  string   `json:"endpoint"`
	Models    []string `json:"models"`
}

func (h *Host) handleStream(s network.Stream) {
	defer s.Close()

	reader := bufio.NewReader(s)
	data, err := io.ReadAll(reader)
	if err != nil {
		h.logger.Error("Failed to read stream", zap.Error(err))
		return
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		h.logger.Error("Failed to unmarshal message", zap.Error(err))
		return
	}

	if h.msgHandler == nil {
		h.logger.Warn("No message handler set")
		return
	}

	response, err := h.msgHandler(h.ctx, s.Conn().RemotePeer(), &msg)
	if err != nil {
		h.logger.Error("Message handler error", zap.Error(err))
		return
	}

	if response != nil {
		respData, err := json.Marshal(response)
		if err != nil {
			h.logger.Error("Failed to marshal response", zap.Error(err))
			return
		}
		s.Write(respData)
	}
}

func (h *Host) SendMessage(ctx context.Context, peerID peer.ID, msg *Message) (*Message, error) {
	s, err := h.host.NewStream(ctx, peerID, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer s.Close()

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	if _, err := s.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write message: %w", err)
	}

	s.CloseWrite()

	reader := bufio.NewReader(s)
	respData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if len(respData) == 0 {
		return nil, nil
	}

	var response Message
	if err := json.Unmarshal(respData, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

func (h *Host) Broadcast(ctx context.Context, msg *Message) error {
	h.peersMu.RLock()
	peers := make([]peer.ID, 0, len(h.peers))
	for id, info := range h.peers {
		if info.Connected {
			peers = append(peers, id)
		}
	}
	h.peersMu.RUnlock()

	for _, peerID := range peers {
		go func(pid peer.ID) {
			if _, err := h.SendMessage(ctx, pid, msg); err != nil {
				h.logger.Debug("Failed to broadcast to peer", zap.String("peer", pid.String()), zap.Error(err))
			}
		}(peerID)
	}

	return nil
}
