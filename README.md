# Smite 2 Random Conquest Bot

A Discord bot that helps organize Smite 2 Conquest "all random" games by generating random teams with gods assigned to each role using modern Discord slash commands.

## Features

- 🎲 **Random Team Generation**: Creates two balanced teams with random gods for each conquest role
- ⚡ **Slash Commands**: Modern Discord slash command interface (`/randomize`, `/help`)
- � **Per-Guild Commands**: Slash commands are registered individually for each Discord server
- �🎭 **Role Selection**: Players can claim roles by reacting to messages with emojis
- 🏆 **Guild-Specific Configuration**: Supports multiple Discord servers with individual settings
- ✅ **Real-time Updates**: Live embed updates showing which roles are taken
- 🔒 **Role Protection**: Prevents multiple players from claiming the same role
- 🧹 **Auto-Cleanup**: Automatically manages command registration and cleanup

## Conquest Roles

- **Mid** (Middle lane)
- **Support** (Duo lane support)
- **Jungle** (Jungle farm and ganks)
- **Carry** (Duo lane ADC)
- **Solo** (Solo lane)

## Usage

### Commands

- `/randomize` - Generate random teams with gods for each role
- `/help` - Show help information

### Role Selection

After generating random teams, players can claim roles by reacting:

**Team 1 Emojis:**
- 1️⃣ Mid
- 2️⃣ Support  
- 3️⃣ Jungle
- 4️⃣ Carry
- 5️⃣ Solo

**Team 2 Emojis:**
- 🥇 Mid
- 🥈 Support
- 🥉 Jungle
- 🏅 Carry
- 🎖️ Solo

Players can change roles by reacting with a different emoji. The bot will automatically remove their previous selection.

## Installation & Setup

### Prerequisites

- Go 1.21 or later
- Discord Bot Token

### Building

1. Clone the repository:
```bash
git clone <repository-url>
cd lan_smite_teams_bot
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the application:
```bash
go build -o bin/smite-bot ./cmd/lan_smite_teams_bot
```

### Configuration

#### Environment Variables

- `DISCORD_BOT_TOKEN` (required) - Your Discord bot token
- `CONFIG_PATH` (optional) - Path to configuration file (default: `config.yaml`)

#### Configuration File (config.yaml)

```yaml
token: "your-discord-bot-token"  # Optional if using env var
guilds:
  "123456789012345678":  # Your Discord server ID
    guild_id: "123456789012345678"
    allowed_channel: "987654321098765432"  # Optional: restrict to specific channel
```

### Running the Bot

```bash
# Using environment variable for token
export DISCORD_BOT_TOKEN="your-bot-token-here"
./bin/smite-bot

# Or using configuration file
echo 'token: "your-bot-token-here"' > config.yaml
./bin/smite-bot
```

## Discord Bot Setup

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications)
2. Create a new application
3. Go to the "Bot" section and create a bot
4. Copy the bot token
5. Enable the following permissions:
   - Send Messages
   - Use Slash Commands
   - Read Message History
   - Add Reactions
   - Use External Emojis
6. Invite the bot to your server using the OAuth2 URL generator

## Development

### Updating God Data

The bot uses god data scraped from the Smite2 Wiki. To update this data:

```bash
# Fetch latest god data from wiki
./scripts/fetch_gods.sh
```

The god data is stored in `data/gods.json` and is automatically loaded by the bot when it starts. The scraper:
- Uses the MediaWiki API to fetch god information
- Parses role, ability, and pantheon data from wiki pages
- Maintains compatibility with the existing god data format

**Note**: The wiki scraper requires internet access and can take a few minutes to complete as it fetches data for all gods with rate limiting.

### Project Structure

```
├── cmd/
│   └── lan_smite_teams_bot/    # Main application entry point
│       └── main.go
├── internal/
│   ├── config/                 # Configuration management
│   │   └── config.go
│   ├── discord/                # Discord bot logic
│   │   └── bot.go
│   └── smite/                  # Smite game logic
│       └── gods.go
├── .github/
│   └── copilot-instructions.md # Development guidelines
├── go.mod                      # Go module definition
└── README.md
```

### Adding New Gods

Gods are automatically fetched from the Smite2 Wiki. To update the god database:

```bash
./scripts/fetch_gods.sh
```

This will:
1. Scrape all god data from https://wiki.smite2.com/w/Category:SMITE_2_gods
2. Save the data to `data/gods.json`
3. Validate the URLs for all gods

The scraper fetches:
- God names
- Roles (Mid, Support, Jungle, Carry, Solo)
- Abilities with descriptions
- Pantheon information
- Portrait images

For more details, see [cmd/fetch_gods_wiki/README.md](cmd/fetch_gods_wiki/README.md).

### Testing

Run tests with:
```bash
go test ./...
```

For verbose output:
```bash
go test -v ./...
```

## Contributing

1. Follow the coding standards outlined in `.github/copilot-instructions.md`
2. Ensure all tests pass
3. Add tests for new functionality
4. Use descriptive commit messages

## License

[Add your license information here]

## Support

For issues or feature requests, please [open an issue](link-to-your-issue-tracker) on the repository.