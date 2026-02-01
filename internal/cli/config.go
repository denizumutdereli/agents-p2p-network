package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure the P2P agent",
	Long:  `Set up API keys and other configuration options.`,
}

var configSetKeyCmd = &cobra.Command{
	Use:   "set-key",
	Short: "Set the OpenAI API key",
	Long:  `Securely store your OpenAI API key for agent authentication.`,
	RunE:  runSetKey,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runShowConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetKeyCmd)
	configCmd.AddCommand(configShowCmd)
}

func runSetKey(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter your OpenAI API key: ")
	key, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read API key: %w", err)
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	if !strings.HasPrefix(key, "sk-") {
		fmt.Println("⚠️  Warning: API key doesn't start with 'sk-'. Make sure it's a valid OpenAI key.")
	}

	viper.Set("api_key", key)

	configPath := getConfigPath()
	if err := viper.WriteConfigAs(configPath); err != nil {
		if err := viper.SafeWriteConfigAs(configPath); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	fmt.Printf("✅ API key saved to %s\n", configPath)
	return nil
}

func runShowConfig(cmd *cobra.Command, args []string) error {
	fmt.Println("Current Configuration:")
	fmt.Println("─────────────────────────")

	apiKey := viper.GetString("api_key")
	if apiKey != "" {
		masked := apiKey[:7] + "..." + apiKey[len(apiKey)-4:]
		fmt.Printf("  API Key:    %s\n", masked)
	} else {
		fmt.Println("  API Key:    (not set)")
	}

	fmt.Printf("  HTTP Port:  %d\n", viper.GetInt("port"))
	fmt.Printf("  P2P Port:   %d\n", viper.GetInt("p2p_port"))
	fmt.Printf("  Agent Name: %s\n", viper.GetString("name"))
	fmt.Printf("  Bootstrap:  %s\n", viper.GetString("bootstrap"))

	return nil
}
