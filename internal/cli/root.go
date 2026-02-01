package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	apiKey     string
	listenPort int
	agentName  string
)

var rootCmd = &cobra.Command{
	Use:   "p2p-agent",
	Short: "P2P Agent Network with OpenAI-compatible API",
	Long: `A decentralized P2P network where AI agents can communicate
with each other using OpenAI-compatible endpoints.

Each agent exposes an OpenAI-compatible API and can discover
and communicate with other agents on the network.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.p2p-agent.yaml)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "OpenAI API key for authentication")
	rootCmd.PersistentFlags().IntVar(&listenPort, "port", 8080, "HTTP API port")
	rootCmd.PersistentFlags().StringVar(&agentName, "name", "", "Agent name for discovery")

	viper.BindPFlag("api_key", rootCmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("name", rootCmd.PersistentFlags().Lookup("name"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".p2p-agent")
	}

	viper.SetEnvPrefix("P2P")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".p2p-agent.yaml")
}
