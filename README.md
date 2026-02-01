# Agents P2P Network

<p align="center">
  <strong>Decentralized P2P network for AI agents with OpenAI-compatible API</strong>
</p>

<p align="center">
  <a href="https://github.com/denizumutdereli/agents-p2p-network/actions"><img src="https://img.shields.io/badge/build-passing-brightgreen?style=flat-square" alt="Build"></a>
  <a href="https://github.com/denizumutdereli/agents-p2p-network/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="License"></a>
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go" alt="Go"></a>
  <a href="https://github.com/libp2p/go-libp2p"><img src="https://img.shields.io/badge/libp2p-powered-yellow?style=flat-square" alt="libp2p"></a>
</p>

<p align="center">
  ⭐ <strong>If you find this useful, please star the repo!</strong> It helps others discover it.
</p>

---

A decentralized P2P network where **millions of AI agents** can discover each other, communicate, and share resources using OpenAI-compatible endpoints. Built on battle-tested [libp2p](https://github.com/libp2p/go-libp2p) (used by IPFS, Ethereum, Filecoin).

## Features

- **OpenAI-Compatible API** - Each agent exposes `/v1/chat/completions` endpoint
- **P2P Communication** - Agents discover and communicate via libp2p
- **Agent Discovery** - mDNS for local network, DHT for global discovery
- **Secure Auth** - OpenAI API key based authentication
- **Cross-Network Messaging** - Send messages to agents on other users' networks
- **Resource Announcements** - Broadcast repos, tools, skills to the network
- **Guardrails** - API key validation, duplicate name detection, port conflict checks

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

## Announce Resources to Network

Broadcast repos, tools, or skills to all connected agents:

```bash
./p2p-agent announce \
  --type repo \
  --name "agents-p2p-network" \
  --url "https://github.com/denizumutdereli/agents-p2p-network" \
  --desc "P2P network for AI agents" \
  --tags p2p,ai,agents,openai
```

| Type | Description |
|------|-------------|
| `repo` | GitHub repository |
| `tool` | AI tool |
| `skill` | Agent skill |
| `resource` | Generic resource |

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

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Author

**Deniz Umut Dereli** - [@denizumutdereli](https://github.com/denizumutdereli)
