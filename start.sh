#!/bin/bash

# Smite 2 Discord Bot Startup Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored message
print_message() {
    echo -e "${2:-$NC}$1${NC}"
}

# Check if required environment variables are set
check_env() {
    if [ -z "$DISCORD_BOT_TOKEN" ] && [ ! -f "config.yaml" ]; then
        print_message "❌ Error: DISCORD_BOT_TOKEN environment variable not set and config.yaml not found" $RED
        print_message "Please set DISCORD_BOT_TOKEN or create a config.yaml file" $YELLOW
        exit 1
    fi
}

# Check if binary exists
check_binary() {
    if [ ! -f "bin/smite-bot" ]; then
        print_message "🔨 Binary not found. Building..." $YELLOW
        make build
    fi
}

# Display startup banner
show_banner() {
    print_message "
    ⚔️  Smite 2 Random Conquest Discord Bot  ⚔️
    
    Starting bot with configuration:
    - Config file: ${CONFIG_PATH:-config.yaml}
    - Token source: ${DISCORD_BOT_TOKEN:+Environment Variable}"${DISCORD_BOT_TOKEN:-Configuration File} $BLUE
}

# Main execution
main() {
    print_message "🚀 Starting Smite 2 Discord Bot..." $GREEN
    
    show_banner
    check_env
    check_binary
    
    print_message "✅ All checks passed. Starting bot..." $GREEN
    print_message "🛑 Press Ctrl+C to stop the bot" $YELLOW
    
    # Start the bot
    ./bin/smite-bot
}

# Handle cleanup on exit
cleanup() {
    print_message "
🛑 Shutting down bot..." $YELLOW
    print_message "👋 Goodbye!" $GREEN
}

# Set trap for cleanup
trap cleanup EXIT INT TERM

# Run main function
main "$@"