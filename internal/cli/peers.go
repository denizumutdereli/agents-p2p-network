package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var peersCmd = &cobra.Command{
	Use:   "peers",
	Short: "Manage P2P peers",
	Long:  `List, connect to, or discover peers on the network.`,
}

var peersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List connected peers",
	RunE:  runPeersList,
}

var peersDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover agents on the network",
	RunE:  runPeersDiscover,
}

func init() {
	rootCmd.AddCommand(peersCmd)
	peersCmd.AddCommand(peersListCmd)
	peersCmd.AddCommand(peersDiscoverCmd)
}

func runPeersList(cmd *cobra.Command, args []string) error {
	fmt.Println("Connected Peers:")
	fmt.Println("─────────────────────────")
	fmt.Println("  (Agent must be running. Use 'p2p-agent start' first)")
	return nil
}

func runPeersDiscover(cmd *cobra.Command, args []string) error {
	fmt.Println("Discovering agents on the network...")
	fmt.Println("  (Agent must be running. Use 'p2p-agent start' first)")
	return nil
}
