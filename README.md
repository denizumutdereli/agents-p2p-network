# P2P Agent Network

> ⭐ **If you find this useful, please star the repo!** It helps others discover it.

A decentralized P2P network where AI agents can communicate with each other using OpenAI-compatible endpoints.

## Features

- **OpenAI-Compatible API**: Each agent exposes `/v1/chat/completions` endpoint
- **P2P Communication**: Agents discover and communicate via libp2p
- **Agent Discovery**: mDNS for local network, DHT for global discovery
- **Secure Auth**: OpenAI API key based authentication
- **Cross-Network Messaging**: Send messages to agents on other users' networks

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      CLI Interface                          │
│  - p2p-agent config set-key    (API key setup)             │
│  - p2p-agent start             (start node)                │
│  - p2p-agent peers list        (list connected agents)     │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                    Local Agent Node                         │
│  ┌─────────────────┐  ┌─────────────────────────────────┐  │
│  │ OpenAI-compat   │  │      P2P Network Layer          │  │
│  │ HTTP Server     │◄─┤  (libp2p)                       │  │
│  │ /v1/chat/...    │  │  - Peer discovery (mDNS/DHT)    │  │
│  │ /v1/agents      │  │  - Encrypted messaging          │  │
│  └─────────────────┘  │  - Agent registry               │  │
│                       └─────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                    P2P Network (DHT/Gossip)
                              │
┌─────────────────────────────────────────────────────────────┐
│               Remote Agent Node (other user)                │
└─────────────────────────────────────────────────────────────┘
```

## Installation

```bash
go build -o p2p-agent ./cmd/p2p-agent
```

## Quick Start

### 1. Set your OpenAI API key

```bash
./p2p-agent config set-key
# Enter your OpenAI API key when prompted
```

Or use environment variable:
```bash
export P2P_API_KEY=sk-your-api-key
```

### 2. Start the agent

```bash
./p2p-agent start --name "my-agent" --port 8080 --p2p-port 9000
```

### 3. Connect to another agent (optional)

```bash
./p2p-agent start --name "my-agent" --bootstrap "/ip4/192.168.1.100/tcp/9000/p2p/QmPeerID..."
```

## API Endpoints

### Standard OpenAI-Compatible

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/chat/completions` | POST | Chat completion (forwards to OpenAI) |
| `/v1/models` | GET | List available models |
| `/health` | GET | Health check |

### P2P Agent Extensions

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/agents` | GET | List connected agents |
| `/v1/agents/:agent_id/chat/completions` | POST | Send chat to specific agent |

## Usage Examples

### Local Chat Completion

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### List Connected Agents

```bash
curl http://localhost:8080/v1/agents \
  -H "Authorization: Bearer sk-your-api-key"
```

### Send to Remote Agent

```bash
curl http://localhost:8080/v1/agents/QmPeerID.../chat/completions \
  -H "Authorization: Bearer sk-your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello from another agent!"}]
  }'
```

## Configuration

Configuration can be set via:
1. Command line flags
2. Environment variables (prefix: `P2P_`)
3. Config file (`~/.p2p-agent.yaml`)

| Option | Flag | Env Var | Default |
|--------|------|---------|---------|
| API Key | `--api-key` | `P2P_API_KEY` | - |
| HTTP Port | `--port` | `P2P_PORT` | 8080 |
| P2P Port | `--p2p-port` | `P2P_P2P_PORT` | 9000 |
| Agent Name | `--name` | `P2P_NAME` | hostname |
| Bootstrap | `--bootstrap` | `P2P_BOOTSTRAP` | - |

## License

MIT
