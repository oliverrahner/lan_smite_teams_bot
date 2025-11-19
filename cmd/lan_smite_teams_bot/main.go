package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/oliver/lan_smite_teams_bot/internal/config"
	"github.com/oliver/lan_smite_teams_bot/internal/discord"
)

func main() {
	log.Println("Starting Smite 2 Random Conquest Bot...")

	// Load configuration
	configPath := getConfigPath()
	if err := config.LoadConfig(configPath); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Check if force cleanup is requested
	forceCleanup := os.Getenv("FORCE_CLEANUP_COMMANDS") == "true"
	if forceCleanup {
		log.Println("Force cleanup mode enabled - will clean all commands on startup")
	}

	// Create Discord bot
	bot, err := discord.NewBot(config.GetToken())
	if err != nil {
		log.Fatalf("Failed to create Discord bot: %v", err)
	}

	// Set cleanup mode if requested
	if forceCleanup {
		bot.SetForceCleanup(true)
	}

	// Start the bot
	if err := bot.Start(); err != nil {
		log.Fatalf("Failed to start Discord bot: %v", err)
	}

	log.Println("Bot is now running. Press CTRL+C to exit.")

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down bot...")

	// Gracefully close the Discord session
	if err := bot.Stop(); err != nil {
		log.Printf("Error stopping bot: %v", err)
	}

	log.Println("Bot stopped successfully.")
}

// getConfigPath returns the configuration file path
func getConfigPath() string {
	if configPath := os.Getenv("CONFIG_PATH"); configPath != "" {
		return configPath
	}
	return "config.yaml"
}
