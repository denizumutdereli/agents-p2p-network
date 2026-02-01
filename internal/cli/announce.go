package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	announceType string
	announceName string
	announceURL  string
	announceDesc string
	announceTags []string
)

var announceCmd = &cobra.Command{
	Use:   "announce",
	Short: "Announce a resource to all connected agents",
	Long: `Broadcast a resource (repo, tool, skill) to all agents on the P2P network.

Example:
  p2p-agent announce --type repo --name "agents-p2p-network" \
    --url "https://github.com/denizumutdereli/agents-p2p-network" \
    --desc "P2P network for AI agents with OpenAI-compatible API" \
    --tags p2p,ai,agents,openai`,
	RunE: runAnnounce,
}

func init() {
	rootCmd.AddCommand(announceCmd)

	announceCmd.Flags().StringVar(&announceType, "type", "repo", "Resource type: repo, tool, skill, resource")
	announceCmd.Flags().StringVar(&announceName, "name", "", "Resource name (required)")
	announceCmd.Flags().StringVar(&announceURL, "url", "", "Resource URL (required)")
	announceCmd.Flags().StringVar(&announceDesc, "desc", "", "Resource description")
	announceCmd.Flags().StringSliceVar(&announceTags, "tags", []string{}, "Tags (comma-separated)")

	announceCmd.MarkFlagRequired("name")
	announceCmd.MarkFlagRequired("url")
}

func runAnnounce(cmd *cobra.Command, args []string) error {
	port := viper.GetInt("port")
	if port == 0 {
		port = 8080
	}

	apiKey := viper.GetString("api_key")
	if apiKey == "" {
		return fmt.Errorf("API key required. Set via --api-key or P2P_API_KEY env var")
	}

	payload := map[string]interface{}{
		"type":        announceType,
		"name":        announceName,
		"url":         announceURL,
		"description": announceDesc,
		"tags":        announceTags,
	}

	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("http://localhost:%d/v1/announce", port)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send announce (is agent running?): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("announce failed with status: %d", resp.StatusCode)
	}

	fmt.Printf("📢 Announced to network:\n")
	fmt.Printf("   Type: %s\n", announceType)
	fmt.Printf("   Name: %s\n", announceName)
	fmt.Printf("   URL:  %s\n", announceURL)
	if announceDesc != "" {
		fmt.Printf("   Desc: %s\n", announceDesc)
	}
	if len(announceTags) > 0 {
		fmt.Printf("   Tags: %v\n", announceTags)
	}

	return nil
}
