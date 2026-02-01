package p2p

import (
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
)

const (
	ProtocolID       = "/p2p-agent/1.0.0"
	AgentServiceName = "p2p-agent-network"
)

type Host struct {
	host       host.Host
	dht        *dht.IpfsDHT
	logger     *zap.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	msgHandler MessageHandler
	localName  string

	peersMu    sync.RWMutex
	peers      map[peer.ID]*PeerInfo
	agentNames map[string]peer.ID // Track agent names to detect duplicates
}

type PeerInfo struct {
	ID        peer.ID
	Name      string
	Addrs     []multiaddr.Multiaddr
	Connected bool
}

type MessageHandler func(ctx context.Context, from peer.ID, msg *Message) (*Message, error)

func NewHost(ctx context.Context, port int, logger *zap.Logger) (*Host, error) {
	ctx, cancel := context.WithCancel(ctx)

	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)

	h, err := libp2p.New(
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
		libp2p.NATPortMap(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	kadDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeAutoServer))
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	if err := kadDHT.Bootstrap(ctx); err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	p2pHost := &Host{
		host:       h,
		dht:        kadDHT,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
		peers:      make(map[peer.ID]*PeerInfo),
		agentNames: make(map[string]peer.ID),
	}

	h.SetStreamHandler(protocol.ID(ProtocolID), p2pHost.handleStream)

	h.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, c network.Conn) {
			p2pHost.onPeerConnected(c.RemotePeer())
		},
		DisconnectedF: func(n network.Network, c network.Conn) {
			p2pHost.onPeerDisconnected(c.RemotePeer())
		},
	})

	return p2pHost, nil
}

func (h *Host) ID() peer.ID {
	return h.host.ID()
}

func (h *Host) Addrs() []multiaddr.Multiaddr {
	return h.host.Addrs()
}

func (h *Host) MultiAddrs() []string {
	addrs := h.host.Addrs()
	result := make([]string, len(addrs))
	for i, addr := range addrs {
		result[i] = fmt.Sprintf("%s/p2p/%s", addr, h.host.ID())
	}
	return result
}

func (h *Host) SetMessageHandler(handler MessageHandler) {
	h.msgHandler = handler
}

func (h *Host) SetLocalName(name string) {
	h.localName = name
}

func (h *Host) RegisterAgentName(name string, peerID peer.ID) error {
	h.peersMu.Lock()
	defer h.peersMu.Unlock()

	if existingPeer, exists := h.agentNames[name]; exists {
		if existingPeer != peerID {
			return fmt.Errorf("agent name '%s' is already taken by peer %s", name, existingPeer.String()[:12])
		}
	}

	h.agentNames[name] = peerID
	return nil
}

func (h *Host) IsNameTaken(name string) (bool, peer.ID) {
	h.peersMu.RLock()
	defer h.peersMu.RUnlock()

	if peerID, exists := h.agentNames[name]; exists {
		return true, peerID
	}
	return false, ""
}

func (h *Host) StartMDNS() error {
	mdnsService := mdns.NewMdnsService(h.host, AgentServiceName, &mdnsNotifee{host: h})
	return mdnsService.Start()
}

func (h *Host) StartDHTDiscovery() {
	routingDiscovery := drouting.NewRoutingDiscovery(h.dht)

	go func() {
		for {
			select {
			case <-h.ctx.Done():
				return
			default:
				peerChan, err := routingDiscovery.FindPeers(h.ctx, AgentServiceName)
				if err != nil {
					h.logger.Debug("DHT discovery error", zap.Error(err))
					continue
				}

				for p := range peerChan {
					if p.ID == h.host.ID() || len(p.Addrs) == 0 {
						continue
					}
					h.Connect(h.ctx, p)
				}
			}
		}
	}()
}

func (h *Host) Connect(ctx context.Context, pi peer.AddrInfo) error {
	if h.host.Network().Connectedness(pi.ID) == network.Connected {
		return nil
	}

	if err := h.host.Connect(ctx, pi); err != nil {
		return fmt.Errorf("failed to connect to peer %s: %w", pi.ID, err)
	}

	h.logger.Info("Connected to peer", zap.String("peer_id", pi.ID.String()))
	return nil
}

func (h *Host) ConnectBootstrap(addr string) error {
	if addr == "" {
		return nil
	}

	ma, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return fmt.Errorf("invalid bootstrap address: %w", err)
	}

	pi, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		return fmt.Errorf("failed to parse bootstrap peer info: %w", err)
	}

	return h.Connect(h.ctx, *pi)
}

func (h *Host) GetPeers() []*PeerInfo {
	h.peersMu.RLock()
	defer h.peersMu.RUnlock()

	peers := make([]*PeerInfo, 0, len(h.peers))
	for _, p := range h.peers {
		peers = append(peers, p)
	}
	return peers
}

func (h *Host) Close() error {
	h.cancel()
	if h.dht != nil {
		h.dht.Close()
	}
	return h.host.Close()
}

func (h *Host) onPeerConnected(peerID peer.ID) {
	h.peersMu.Lock()
	defer h.peersMu.Unlock()

	if _, exists := h.peers[peerID]; !exists {
		h.peers[peerID] = &PeerInfo{
			ID:        peerID,
			Connected: true,
		}
	} else {
		h.peers[peerID].Connected = true
	}

	h.logger.Info("Peer connected", zap.String("peer_id", peerID.String()))
}

func (h *Host) onPeerDisconnected(peerID peer.ID) {
	h.peersMu.Lock()
	defer h.peersMu.Unlock()

	if p, exists := h.peers[peerID]; exists {
		p.Connected = false
	}

	h.logger.Info("Peer disconnected", zap.String("peer_id", peerID.String()))
}

type mdnsNotifee struct {
	host *Host
}

func (n *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID == n.host.host.ID() {
		return
	}
	n.host.logger.Debug("Found peer via mDNS", zap.String("peer_id", pi.ID.String()))
	n.host.Connect(n.host.ctx, pi)
}
