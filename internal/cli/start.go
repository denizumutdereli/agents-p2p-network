package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/denizumutdereli/agents-p2p-network/internal/agent"
	"github.com/denizumutdereli/agents-p2p-network/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	p2pPort       int
	bootstrapPeer string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the P2P agent node",
	Long:  `Start the P2P agent node with OpenAI-compatible API endpoints.`,
	RunE:  runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().IntVar(&p2pPort, "p2p-port", 9000, "P2P network port")
	startCmd.Flags().StringVar(&bootstrapPeer, "bootstrap", "", "Bootstrap peer multiaddr")

	viper.BindPFlag("p2p_port", startCmd.Flags().Lookup("p2p-port"))
	viper.BindPFlag("bootstrap", startCmd.Flags().Lookup("bootstrap"))
}

func runStart(cmd *cobra.Command, args []string) error {
	cfg := &config.Config{
		APIKey:        viper.GetString("api_key"),
		HTTPPort:      viper.GetInt("port"),
		P2PPort:       viper.GetInt("p2p_port"),
		AgentName:     viper.GetString("name"),
		BootstrapPeer: viper.GetString("bootstrap"),
	}

	// Validate configuration
	if errs := cfg.Validate(); errs.HasErrors() {
		fmt.Println("❌ Configuration errors:")
		for _, e := range errs {
			fmt.Printf("   • %s: %s\n", e.Field, e.Message)
		}
		return fmt.Errorf("invalid configuration")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ag, err := agent.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	if err := ag.Start(ctx); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	fmt.Printf("🚀 Agent '%s' started\n", cfg.AgentName)
	fmt.Printf("   HTTP API: http://localhost:%d\n", cfg.HTTPPort)
	fmt.Printf("   P2P Port: %d\n", cfg.P2PPort)
	fmt.Printf("   Peer ID:  %s\n", ag.PeerID())

	<-sigCh
	fmt.Println("\n⏹️  Shutting down...")
	ag.Stop()

	return nil
}
