package config

import (
	"fmt"
	"net"
	"strings"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

func (c *Config) Validate() ValidationErrors {
	var errors ValidationErrors

	// API Key validation
	if err := validateAPIKey(c.APIKey); err != nil {
		errors = append(errors, *err)
	}

	// Agent name validation
	if err := validateAgentName(c.AgentName); err != nil {
		errors = append(errors, *err)
	}

	// Port validation
	if err := validatePort(c.HTTPPort, "http_port"); err != nil {
		errors = append(errors, *err)
	}
	if err := validatePort(c.P2PPort, "p2p_port"); err != nil {
		errors = append(errors, *err)
	}

	// Port conflict check
	if c.HTTPPort == c.P2PPort {
		errors = append(errors, ValidationError{
			Field:   "ports",
			Message: "HTTP port and P2P port cannot be the same",
		})
	}

	// Check if ports are available
	if err := checkPortAvailable(c.HTTPPort, "http_port"); err != nil {
		errors = append(errors, *err)
	}
	if err := checkPortAvailable(c.P2PPort, "p2p_port"); err != nil {
		errors = append(errors, *err)
	}

	return errors
}

func validateAPIKey(key string) *ValidationError {
	if key == "" {
		return &ValidationError{
			Field:   "api_key",
			Message: "API key is required. Use --api-key flag or set P2P_API_KEY env var",
		}
	}

	// OpenAI API keys start with "sk-"
	if !strings.HasPrefix(key, "sk-") {
		return &ValidationError{
			Field:   "api_key",
			Message: "Invalid API key format. OpenAI API keys start with 'sk-'",
		}
	}

	// Minimum length check (OpenAI keys are typically 51+ chars)
	if len(key) < 40 {
		return &ValidationError{
			Field:   "api_key",
			Message: "API key appears to be too short. Please check your key",
		}
	}

	return nil
}

func validateAgentName(name string) *ValidationError {
	if name == "" {
		return &ValidationError{
			Field:   "agent_name",
			Message: "Agent name is required. Use --name flag to set it",
		}
	}

	// Check for valid characters (alphanumeric, dash, underscore)
	for _, c := range name {
		if !isValidNameChar(c) {
			return &ValidationError{
				Field:   "agent_name",
				Message: "Agent name can only contain letters, numbers, dashes, and underscores",
			}
		}
	}

	// Length check
	if len(name) < 2 {
		return &ValidationError{
			Field:   "agent_name",
			Message: "Agent name must be at least 2 characters",
		}
	}

	if len(name) > 32 {
		return &ValidationError{
			Field:   "agent_name",
			Message: "Agent name cannot exceed 32 characters",
		}
	}

	return nil
}

func isValidNameChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}

func validatePort(port int, field string) *ValidationError {
	if port < 1 || port > 65535 {
		return &ValidationError{
			Field:   field,
			Message: "Port must be between 1 and 65535",
		}
	}

	if port < 1024 {
		return &ValidationError{
			Field:   field,
			Message: "Port below 1024 requires elevated privileges. Use a port >= 1024",
		}
	}

	return nil
}

func checkPortAvailable(port int, field string) *ValidationError {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("Port %d is already in use", port),
		}
	}
	listener.Close()
	return nil
}
