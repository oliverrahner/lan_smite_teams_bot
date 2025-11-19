package discord

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/oliver/lan_smite_teams_bot/internal/config"
	"github.com/oliver/lan_smite_teams_bot/internal/smite"
)

// Bot represents the Discord bot instance
type Bot struct {
	session       *discordgo.Session
	gameStates    map[string]*GameState // messageID -> GameState
	mutex         sync.RWMutex
	guildCommands map[string][]*discordgo.ApplicationCommand // guildID -> commands
	ready         bool                                       // Track if bot has finished initial setup
	forceCleanup  bool                                       // Force cleanup all commands on startup
}

// GameState tracks the state of an active randomization
type GameState struct {
	MessageID    string
	GuildID      string
	ChannelID    string
	GameSetup    *smite.GameSetup
	Participants map[string]PlayerSelection // userID -> PlayerSelection
	Interaction  *discordgo.Interaction     // Store original interaction for followup messages
	mutex        sync.RWMutex
}

// PlayerSelection represents a player's role selection
type PlayerSelection struct {
	UserID     string
	Username   string
	Team       int // 1 or 2
	Role       smite.Role
	SelectedAt int64
}

// NewBot creates a new Discord bot instance
func NewBot(token string) (*Bot, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	bot := &Bot{
		session:       session,
		gameStates:    make(map[string]*GameState),
		guildCommands: make(map[string][]*discordgo.ApplicationCommand),
	}

	// Register event handlers
	session.AddHandler(bot.onInteractionCreate)
	session.AddHandler(bot.onReactionAdd)
	session.AddHandler(bot.onReactionRemove)
	session.AddHandler(bot.onGuildCreate)
	session.AddHandler(bot.onGuildDelete)
	session.AddHandler(bot.onReady)

	// Set intents
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions | discordgo.IntentsGuilds

	return bot, nil
}

// Start starts the Discord bot
func (b *Bot) Start() error {
	// Load gods data before starting the bot
	log.Println("Loading gods data...")
	if err := smite.LoadGodsFromFile(""); err != nil {
		return fmt.Errorf("failed to load gods data: %w", err)
	}
	log.Printf("Successfully loaded %d gods", smite.GetGodsCount())

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord session: %w", err)
	}

	log.Println("Discord bot started successfully")
	return nil
}

// SetForceCleanup sets whether to force cleanup all commands on startup
func (b *Bot) SetForceCleanup(force bool) {
	b.forceCleanup = force
}

// Stop stops the Discord bot
func (b *Bot) Stop() error {
	// Clean up slash commands from all guilds
	b.cleanupSlashCommands()
	return b.session.Close()
}

// cleanupSlashCommands removes all guild slash commands and checks for orphaned commands
func (b *Bot) cleanupSlashCommands() {
	if b.session.State.User == nil {
		return
	}

	log.Println("Starting comprehensive slash command cleanup...")

	b.mutex.RLock()
	defer b.mutex.RUnlock()

	// Clean up commands from all guilds we know about
	for guildID, commands := range b.guildCommands {
		log.Printf("Cleaning up %d commands from guild %s", len(commands), guildID)
		for _, command := range commands {
			err := b.session.ApplicationCommandDelete(b.session.State.User.ID, guildID, command.ID)
			if err != nil {
				log.Printf("Failed to delete command %s from guild %s: %v", command.Name, guildID, err)
			} else {
				log.Printf("Deleted slash command %s from guild %s", command.Name, guildID)
			}
		}
	}

	// Also check all guilds the bot is currently in for any orphaned commands
	if b.session.State.Guilds != nil {
		for _, guild := range b.session.State.Guilds {
			// Get all commands for this guild
			existingCommands, err := b.session.ApplicationCommands(b.session.State.User.ID, guild.ID)
			if err != nil {
				log.Printf("Failed to fetch commands for guild %s during cleanup: %v", guild.ID, err)
				continue
			}

			// Delete any commands that match our command names
			for _, cmd := range existingCommands {
				for _, ourCmd := range slashCommands {
					if cmd.Name == ourCmd.Name {
						err := b.session.ApplicationCommandDelete(b.session.State.User.ID, guild.ID, cmd.ID)
						if err != nil {
							log.Printf("Failed to delete orphaned command %s from guild %s: %v", cmd.Name, guild.ID, err)
						} else {
							log.Printf("Deleted orphaned slash command %s from guild %s", cmd.Name, guild.ID)
						}
						break
					}
				}
			}
		}
	}

	// Also clean up any global commands (just in case)
	globalCommands, err := b.session.ApplicationCommands(b.session.State.User.ID, "")
	if err == nil {
		for _, cmd := range globalCommands {
			for _, ourCmd := range slashCommands {
				if cmd.Name == ourCmd.Name {
					err := b.session.ApplicationCommandDelete(b.session.State.User.ID, "", cmd.ID)
					if err != nil {
						log.Printf("Failed to delete global command %s: %v", cmd.Name, err)
					} else {
						log.Printf("Deleted global slash command %s", cmd.Name)
					}
					break
				}
			}
		}
	} else {
		log.Printf("Failed to fetch global commands during cleanup: %v", err)
	}

	log.Println("Slash command cleanup completed")
} // onGuildCreate is called when the bot joins a new guild
func (b *Bot) onGuildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	log.Printf("Joined guild: %s (%s)", event.Guild.Name, event.Guild.ID)

	// Initialize default configuration for new guild
	guildConfig := config.GetGuildConfig(event.Guild.ID)
	if guildConfig.GuildID != event.Guild.ID {
		guildConfig.GuildID = event.Guild.ID
		if err := config.SetGuildConfig(event.Guild.ID, guildConfig); err != nil {
			log.Printf("Failed to save config for guild %s: %v", event.Guild.ID, err)
		}
	}

	// Only register commands if the bot has completed initial setup
	// This prevents duplicate registrations during startup
	b.mutex.RLock()
	isReady := b.ready
	b.mutex.RUnlock()

	if isReady {
		// Register slash commands for this newly joined guild
		b.registerGuildCommands(s, event.Guild.ID)
	}
}

// registerGuildCommands registers slash commands for a specific guild
func (b *Bot) registerGuildCommands(s *discordgo.Session, guildID string) {
	// Check if commands are already registered for this guild
	b.mutex.RLock()
	existingCommands, exists := b.guildCommands[guildID]
	b.mutex.RUnlock()

	if exists && len(existingCommands) > 0 {
		log.Printf("Slash commands already registered for guild %s (%d commands), skipping", guildID, len(existingCommands))
		return
	}

	log.Printf("Registering slash commands for guild %s...", guildID)

	// Use a lock to prevent concurrent registration for the same guild
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Double-check after acquiring the lock
	if existingCommands, exists := b.guildCommands[guildID]; exists && len(existingCommands) > 0 {
		log.Printf("Slash commands were registered by another goroutine for guild %s, skipping", guildID)
		return
	}

	// First, clean up any existing commands to prevent duplicates
	existingDiscordCommands, err := s.ApplicationCommands(s.State.User.ID, guildID)
	if err == nil {
		for _, cmd := range existingDiscordCommands {
			// Only delete our commands to avoid affecting other bots
			for _, ourCmd := range slashCommands {
				if cmd.Name == ourCmd.Name {
					err := s.ApplicationCommandDelete(s.State.User.ID, guildID, cmd.ID)
					if err != nil {
						log.Printf("Failed to delete existing command %s in guild %s: %v", cmd.Name, guildID, err)
					} else {
						log.Printf("Deleted existing command %s in guild %s", cmd.Name, guildID)
					}
					break
				}
			}
		}
	}

	var createdCommands []*discordgo.ApplicationCommand

	for _, cmd := range slashCommands {
		createdCmd, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
		if err != nil {
			log.Printf("Failed to create slash command %s for guild %s: %v", cmd.Name, guildID, err)
		} else {
			log.Printf("Successfully registered slash command %s for guild %s", cmd.Name, guildID)
			createdCommands = append(createdCommands, createdCmd)
		}
	}

	// Store the created commands for cleanup later
	b.guildCommands[guildID] = createdCommands
}

// onGuildDelete is called when the bot leaves a guild
func (b *Bot) onGuildDelete(s *discordgo.Session, event *discordgo.GuildDelete) {
	log.Printf("Left guild: %s", event.Guild.ID)

	// Clean up commands from this guild
	b.mutex.Lock()
	if commands, exists := b.guildCommands[event.Guild.ID]; exists {
		for _, command := range commands {
			err := s.ApplicationCommandDelete(s.State.User.ID, event.Guild.ID, command.ID)
			if err != nil {
				log.Printf("Failed to delete command %s from guild %s: %v", command.Name, event.Guild.ID, err)
			} else {
				log.Printf("Deleted slash command %s from guild %s", command.Name, event.Guild.ID)
			}
		}
		delete(b.guildCommands, event.Guild.ID)
	}
	b.mutex.Unlock()

	// Clean up any game states for this guild
	b.cleanupGuildGameStates(event.Guild.ID)
}

// cleanupGuildGameStates removes all game states for a specific guild
func (b *Bot) cleanupGuildGameStates(guildID string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	for messageID, gameState := range b.gameStates {
		if gameState.GuildID == guildID {
			delete(b.gameStates, messageID)
			log.Printf("Cleaned up game state for message %s in guild %s", messageID, guildID)
		}
	}
}

// slashCommands defines the slash commands for the bot
var slashCommands = []*discordgo.ApplicationCommand{
	{
		Name:        "randomize",
		Description: "Generate random Smite 2 Conquest teams with gods for each role",
	},
	{
		Name:        "help",
		Description: "Show help information about the bot",
	},
	{
		Name:        "godinfo",
		Description: "Get detailed information about a specific god",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "god",
				Description:  "Name of the god to look up",
				Required:     true,
				Autocomplete: true,
			},
		},
	},
}

// onReady is called when the bot is ready
func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Bot is ready! Logged in as %s", s.State.User.Username)

	// Clean up any existing commands before registering new ones
	b.cleanupExistingCommands(s)

	// Register slash commands for all guilds the bot is already in
	for _, guild := range r.Guilds {
		b.registerGuildCommands(s, guild.ID)
	}

	// Mark bot as ready to prevent duplicate registrations in onGuildCreate
	b.mutex.Lock()
	b.ready = true
	b.mutex.Unlock()
}

// cleanupExistingCommands removes any existing commands on startup
func (b *Bot) cleanupExistingCommands(s *discordgo.Session) {
	if !b.forceCleanup {
		// Only do minimal cleanup unless force cleanup is enabled
		log.Println("Skipping startup cleanup (set FORCE_CLEANUP_COMMANDS=true to enable)")
		return
	}

	log.Println("Force cleaning up ALL existing commands on startup...")

	// Clean up global commands first
	globalCommands, err := s.ApplicationCommands(s.State.User.ID, "")
	if err == nil {
		log.Printf("Found %d global commands, checking for cleanup", len(globalCommands))
		for _, cmd := range globalCommands {
			for _, ourCmd := range slashCommands {
				if cmd.Name == ourCmd.Name {
					err := s.ApplicationCommandDelete(s.State.User.ID, "", cmd.ID)
					if err != nil {
						log.Printf("Failed to delete existing global command %s: %v", cmd.Name, err)
					} else {
						log.Printf("Deleted existing global command %s", cmd.Name)
					}
					break
				}
			}
		}
	} else {
		log.Printf("Failed to fetch global commands: %v", err)
	}

	// Clean up guild commands for all guilds we're in
	for _, guild := range s.State.Guilds {
		existingCommands, err := s.ApplicationCommands(s.State.User.ID, guild.ID)
		if err != nil {
			log.Printf("Failed to fetch commands for guild %s: %v", guild.ID, err)
			continue
		}

		if len(existingCommands) > 0 {
			log.Printf("Found %d commands in guild %s, checking for cleanup", len(existingCommands), guild.ID)
		}

		for _, cmd := range existingCommands {
			for _, ourCmd := range slashCommands {
				if cmd.Name == ourCmd.Name {
					err := s.ApplicationCommandDelete(s.State.User.ID, guild.ID, cmd.ID)
					if err != nil {
						log.Printf("Failed to delete existing command %s from guild %s: %v", cmd.Name, guild.ID, err)
					} else {
						log.Printf("Deleted existing command %s from guild %s", cmd.Name, guild.ID)
					}
					break
				}
			}
		}
	}

	log.Println("Startup command cleanup completed")
}

// onInteractionCreate handles slash command interactions
func (b *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Handle autocomplete interactions
	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		b.handleAutocomplete(s, i)
		return
	}

	// Only handle slash commands
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	guildConfig := config.GetGuildConfig(i.GuildID)

	// Check if interaction is in allowed channel (if configured)
	if guildConfig.AllowedChannel != "" && i.ChannelID != guildConfig.AllowedChannel {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "randomize":
		b.handleRandomizeSlashCommand(s, i, guildConfig)
	case "help":
		b.handleHelpSlashCommand(s, i, guildConfig)
	case "godinfo":
		b.handleGodInfoSlashCommand(s, i, guildConfig)
	}
}

// handleRandomizeSlashCommand creates a new randomized game via slash command
func (b *Bot) handleRandomizeSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate, guildConfig config.GuildConfig) {
	// Generate random teams
	gameSetup, err := smite.GenerateRandomTeams()
	if err != nil {
		b.respondToInteraction(s, i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Error generating random teams: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Create the embed message
	embed := b.createGameEmbed(gameSetup)

	// Respond to the interaction
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})

	if err != nil {
		log.Printf("Failed to respond to interaction: %v", err)
		return
	}

	// Get the response message
	message, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		log.Printf("Failed to get interaction response: %v", err)
		return
	}

	// Create game state
	gameState := &GameState{
		MessageID:    message.ID,
		GuildID:      i.GuildID,
		ChannelID:    i.ChannelID,
		GameSetup:    gameSetup,
		Participants: make(map[string]PlayerSelection),
		Interaction:  i.Interaction, // Store the interaction for followup messages
	}

	b.mutex.Lock()
	b.gameStates[message.ID] = gameState
	b.mutex.Unlock()

	// Add reactions for role selection
	b.addRoleReactions(s, message)

	log.Printf("Created new randomized game in guild %s, channel %s, message %s", i.GuildID, i.ChannelID, message.ID)
}

// handleHelpSlashCommand shows the help message via slash command
func (b *Bot) handleHelpSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate, guildConfig config.GuildConfig) {
	embed := &discordgo.MessageEmbed{
		Title:       "🎮 Smite 2 Random Conquest Bot",
		Description: "Organize random Conquest games with automatic team and god randomization!",
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Commands",
				Value:  "`/randomize` - Generate random teams with gods for each role\n`/godinfo` - Get detailed information about a specific god\n`/help` - Show this help message",
				Inline: false,
			},
			{
				Name:   "How to Play",
				Value:  "1. Use the `/randomize` command to generate teams\n2. React with the role emojis to claim your spot\n3. Team 1 uses: 1️⃣2️⃣3️⃣4️⃣5️⃣\n4. Team 2 uses: 🥇🥈🥉🏅🎖️\n5. Roles: Mid, Support, Jungle, Carry, Solo",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "React to the message to join a team!",
		},
	}

	b.respondToInteraction(s, i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

// handleAutocomplete handles autocomplete interactions for slash commands
func (b *Bot) handleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	if data.Name == "godinfo" {
		b.handleGodAutocomplete(s, i)
	}
}

// handleGodAutocomplete handles autocomplete for god names
func (b *Bot) handleGodAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	var currentValue string
	for _, option := range data.Options {
		if option.Name == "god" && option.Focused {
			if option.StringValue() != "" {
				currentValue = strings.ToLower(option.StringValue())
			}
			break
		}
	}

	// Get all gods and filter by current value
	gods := smite.GetAllGods()
	var choices []*discordgo.ApplicationCommandOptionChoice

	for _, god := range gods {
		if currentValue == "" || strings.Contains(strings.ToLower(god.Name), currentValue) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  god.Name,
				Value: god.Name,
			})

			// Discord limits autocomplete to 25 choices
			if len(choices) >= 25 {
				break
			}
		}
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})

	if err != nil {
		log.Printf("Failed to respond to autocomplete: %v", err)
	}
}

// handleGodInfoSlashCommand shows detailed god information via slash command
func (b *Bot) handleGodInfoSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate, guildConfig config.GuildConfig) {
	data := i.ApplicationCommandData()

	// Get the god name from the option
	var godName string
	for _, option := range data.Options {
		if option.Name == "god" {
			godName = option.StringValue()
			break
		}
	}

	if godName == "" {
		b.respondToInteraction(s, i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Please specify a god name.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Find the god
	god := smite.GetGodByName(godName)
	if god == nil {
		b.respondToInteraction(s, i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ God '%s' not found.", godName),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Create god info embed with a different title
	embeds := b.createStandaloneGodInfoEmbed(god)

	b.respondToInteraction(s, i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: embeds,
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

// respondToInteraction safely responds to an interaction
func (b *Bot) respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, response *discordgo.InteractionResponse) {
	if err := s.InteractionRespond(i.Interaction, response); err != nil {
		log.Printf("Failed to respond to interaction: %v", err)
	}
}

// createGameEmbed creates the embed message for a randomized game
func (b *Bot) createGameEmbed(gameSetup *smite.GameSetup) *discordgo.MessageEmbed {
	team1Emojis, team2Emojis := smite.GetRoleEmojis()

	team1Value := fmt.Sprintf("%s **Mid:** %s\n%s **Support:** %s\n%s **Jungle:** %s\n%s **Carry:** %s\n%s **Solo:** %s",
		team1Emojis[smite.RoleMid], gameSetup.Team1.Mid.Name,
		team1Emojis[smite.RoleSupport], gameSetup.Team1.Support.Name,
		team1Emojis[smite.RoleJungle], gameSetup.Team1.Jungle.Name,
		team1Emojis[smite.RoleCarry], gameSetup.Team1.Carry.Name,
		team1Emojis[smite.RoleSolo], gameSetup.Team1.Solo.Name,
	)

	team2Value := fmt.Sprintf("%s **Mid:** %s\n%s **Support:** %s\n%s **Jungle:** %s\n%s **Carry:** %s\n%s **Solo:** %s",
		team2Emojis[smite.RoleMid], gameSetup.Team2.Mid.Name,
		team2Emojis[smite.RoleSupport], gameSetup.Team2.Support.Name,
		team2Emojis[smite.RoleJungle], gameSetup.Team2.Jungle.Name,
		team2Emojis[smite.RoleCarry], gameSetup.Team2.Carry.Name,
		team2Emojis[smite.RoleSolo], gameSetup.Team2.Solo.Name,
	)

	embed := &discordgo.MessageEmbed{
		Title:       "⚔️ Random Smite 2 Conquest Teams",
		Description: "React with the emojis below to claim your role!",
		Color:       0x0066cc,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🔥 Team 1",
				Value:  team1Value,
				Inline: true,
			},
			{
				Name:   "❄️ Team 2",
				Value:  team2Value,
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Good luck and have fun!",
		},
	}

	return embed
}

// addRoleReactions adds the reaction emojis for role selection
func (b *Bot) addRoleReactions(s *discordgo.Session, message *discordgo.Message) {
	team1Emojis, team2Emojis := smite.GetRoleEmojis()
	roles := smite.GetAllRoles()

	// Add Team 1 reactions
	for _, role := range roles {
		if emoji, exists := team1Emojis[role]; exists {
			if err := s.MessageReactionAdd(message.ChannelID, message.ID, emoji); err != nil {
				log.Printf("Failed to add reaction %s: %v", emoji, err)
			}
		}
	}

	// Add Team 2 reactions
	for _, role := range roles {
		if emoji, exists := team2Emojis[role]; exists {
			if err := s.MessageReactionAdd(message.ChannelID, message.ID, emoji); err != nil {
				log.Printf("Failed to add reaction %s: %v", emoji, err)
			}
		}
	}
}

// onReactionAdd handles when users add reactions
func (b *Bot) onReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	// Ignore bot's own reactions
	if r.UserID == s.State.User.ID {
		return
	}

	b.mutex.RLock()
	gameState, exists := b.gameStates[r.MessageID]
	b.mutex.RUnlock()

	if !exists {
		return
	}

	// Get user info
	user, err := s.User(r.UserID)
	if err != nil {
		log.Printf("Failed to get user info for %s: %v", r.UserID, err)
		return
	}

	// Determine team and role from emoji
	team, role := b.getTeamAndRoleFromEmoji(r.Emoji.Name)
	if team == 0 {
		return // Unknown emoji
	}

	// Check if user already has a selection
	gameState.mutex.Lock()
	if existingSelection, hasSelection := gameState.Participants[r.UserID]; hasSelection {
		// Remove old selection
		delete(gameState.Participants, r.UserID)

		// Remove old reaction
		oldTeam1Emojis, oldTeam2Emojis := smite.GetRoleEmojis()
		var oldEmoji string
		if existingSelection.Team == 1 {
			oldEmoji = oldTeam1Emojis[existingSelection.Role]
		} else {
			oldEmoji = oldTeam2Emojis[existingSelection.Role]
		}

		gameState.mutex.Unlock()

		// Remove user's old reaction
		if err := s.MessageReactionRemove(r.ChannelID, r.MessageID, oldEmoji, r.UserID); err != nil {
			log.Printf("Failed to remove old reaction: %v", err)
		}

		gameState.mutex.Lock()
	}

	// Check if role is already taken
	roleAvailable := true
	for _, participant := range gameState.Participants {
		if participant.Team == team && participant.Role == role {
			roleAvailable = false
			break
		}
	}

	if !roleAvailable {
		gameState.mutex.Unlock()

		// Remove the reaction and send a message
		if err := s.MessageReactionRemove(r.ChannelID, r.MessageID, r.Emoji.Name, r.UserID); err != nil {
			log.Printf("Failed to remove reaction: %v", err)
		}

		// Send ephemeral message (if possible) or temporary message
		tempMsg, err := s.ChannelMessageSend(r.ChannelID, fmt.Sprintf("❌ %s, that role is already taken!", user.Username))
		if err == nil {
			// Delete the temporary message after 3 seconds
			go func() {
				time.Sleep(3 * time.Second)
				if delErr := s.ChannelMessageDelete(r.ChannelID, tempMsg.ID); delErr != nil {
					log.Printf("Failed to delete temporary message: %v", delErr)
				}
			}()
		}
		return
	}

	// Add new selection
	gameState.Participants[r.UserID] = PlayerSelection{
		UserID:     r.UserID,
		Username:   user.Username,
		Team:       team,
		Role:       role,
		SelectedAt: time.Now().Unix(),
	}
	gameState.mutex.Unlock()

	// Get the god for the selected role and team
	var selectedGod *smite.God
	gameState.mutex.RLock()
	if team == 1 {
		switch role {
		case smite.RoleMid:
			selectedGod = &gameState.GameSetup.Team1.Mid
		case smite.RoleSupport:
			selectedGod = &gameState.GameSetup.Team1.Support
		case smite.RoleJungle:
			selectedGod = &gameState.GameSetup.Team1.Jungle
		case smite.RoleCarry:
			selectedGod = &gameState.GameSetup.Team1.Carry
		case smite.RoleSolo:
			selectedGod = &gameState.GameSetup.Team1.Solo
		}
	} else {
		switch role {
		case smite.RoleMid:
			selectedGod = &gameState.GameSetup.Team2.Mid
		case smite.RoleSupport:
			selectedGod = &gameState.GameSetup.Team2.Support
		case smite.RoleJungle:
			selectedGod = &gameState.GameSetup.Team2.Jungle
		case smite.RoleCarry:
			selectedGod = &gameState.GameSetup.Team2.Carry
		case smite.RoleSolo:
			selectedGod = &gameState.GameSetup.Team2.Solo
		}
	}
	gameState.mutex.RUnlock()

	// Send ephemeral god information message
	if selectedGod != nil {
		// Create single comprehensive embed (returns slice with one element)
		embeds := b.createGodInfoEmbeds(selectedGod, role, team)
		embed := embeds[0] // Get the single embed

		// Try to send as a true ephemeral followup message using the original interaction
		gameState.mutex.RLock()
		interaction := gameState.Interaction
		gameState.mutex.RUnlock()

		if interaction != nil {
			// Send single embed
			_, err = s.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
				Embeds: []*discordgo.MessageEmbed{embed},
				Flags:  discordgo.MessageFlagsEphemeral,
			})
			if err != nil {
				log.Printf("Failed to send ephemeral followup message to user %s: %v", user.Username, err)
				// Fallback to temporary channel message if followup fails
				b.sendEmbedChannelMessage(s, r.ChannelID, embed)
			} else {
				log.Printf("Sent ephemeral god info to user %s (%s)", user.Username, r.UserID)
			}
		} else {
			log.Printf("No interaction available for ephemeral message, using fallback for user %s", user.Username)
			// Fallback to temporary channel message if no interaction available
			b.sendEmbedChannelMessage(s, r.ChannelID, embed)
		}
	}

	// Update the embed message
	b.updateGameEmbed(s, gameState)

	log.Printf("User %s (%s) selected %s on team %d in guild %s", user.Username, r.UserID, role, team, r.GuildID)
}

// onReactionRemove handles when users remove reactions
func (b *Bot) onReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	// Ignore bot's own reactions
	if r.UserID == s.State.User.ID {
		return
	}

	b.mutex.RLock()
	gameState, exists := b.gameStates[r.MessageID]
	b.mutex.RUnlock()

	if !exists {
		return
	}

	// Remove participant if they exist
	gameState.mutex.Lock()
	if _, hasSelection := gameState.Participants[r.UserID]; hasSelection {
		delete(gameState.Participants, r.UserID)
		gameState.mutex.Unlock()

		// Update the embed message
		b.updateGameEmbed(s, gameState)

		log.Printf("User %s removed their selection in guild %s", r.UserID, r.GuildID)
	} else {
		gameState.mutex.Unlock()
	}
}

// getTeamAndRoleFromEmoji determines team and role from emoji
func (b *Bot) getTeamAndRoleFromEmoji(emoji string) (int, smite.Role) {
	team1Emojis, team2Emojis := smite.GetRoleEmojis()
	roles := smite.GetAllRoles()

	// Check team 1 emojis
	for _, role := range roles {
		if team1Emojis[role] == emoji {
			return 1, role
		}
	}

	// Check team 2 emojis
	for _, role := range roles {
		if team2Emojis[role] == emoji {
			return 2, role
		}
	}

	return 0, ""
}

// updateGameEmbed updates the embed message with current participant information
func (b *Bot) updateGameEmbed(s *discordgo.Session, gameState *GameState) {
	gameState.mutex.RLock()
	defer gameState.mutex.RUnlock()

	team1Emojis, team2Emojis := smite.GetRoleEmojis()

	// Build participant info
	team1Participants := make(map[smite.Role]string)
	team2Participants := make(map[smite.Role]string)

	for _, participant := range gameState.Participants {
		if participant.Team == 1 {
			team1Participants[participant.Role] = participant.Username
		} else {
			team2Participants[participant.Role] = participant.Username
		}
	}

	// Create updated team values
	team1Value := b.buildTeamValue(gameState.GameSetup.Team1, team1Emojis, team1Participants)
	team2Value := b.buildTeamValue(gameState.GameSetup.Team2, team2Emojis, team2Participants)

	embed := &discordgo.MessageEmbed{
		Title:       "⚔️ Random Smite 2 Conquest Teams",
		Description: "React with the emojis below to claim your role!",
		Color:       0x0066cc,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🔥 Team 1",
				Value:  team1Value,
				Inline: true,
			},
			{
				Name:   "❄️ Team 2",
				Value:  team2Value,
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Good luck and have fun!",
		},
	}

	// Check if all roles are filled
	if len(gameState.Participants) == 10 {
		embed.Description = "🎉 **All roles filled! Game is ready to start!**"
		embed.Color = 0x00ff00
	}

	_, err := s.ChannelMessageEditEmbed(gameState.ChannelID, gameState.MessageID, embed)
	if err != nil {
		log.Printf("Failed to update embed message: %v", err)
	}
}

// buildTeamValue builds the value string for a team field
func (b *Bot) buildTeamValue(team smite.Team, emojis map[smite.Role]string, participants map[smite.Role]string) string {
	roles := []struct {
		role smite.Role
		god  string
	}{
		{smite.RoleMid, team.Mid.Name},
		{smite.RoleSupport, team.Support.Name},
		{smite.RoleJungle, team.Jungle.Name},
		{smite.RoleCarry, team.Carry.Name},
		{smite.RoleSolo, team.Solo.Name},
	}

	var lines []string
	for _, roleInfo := range roles {
		emoji := emojis[roleInfo.role]
		participant := participants[roleInfo.role]

		if participant != "" {
			lines = append(lines, fmt.Sprintf("%s **%s:** %s ✅ *%s*", emoji, roleInfo.role, roleInfo.god, participant))
		} else {
			lines = append(lines, fmt.Sprintf("%s **%s:** %s", emoji, roleInfo.role, roleInfo.god))
		}
	}

	return strings.Join(lines, "\n")
}

// sendTemporaryChannelMessage sends a message that stays in channel (fallback for when ephemeral isn't available)
func (b *Bot) sendTemporaryChannelMessage(s *discordgo.Session, channelID, username string, role smite.Role, team int, godInfo string) {
	messageContent := fmt.Sprintf("✅ **%s**, you claimed **%s** on Team %d!\n\n%s",
		username, role, team, godInfo)

	_, err := s.ChannelMessageSend(channelID, messageContent)
	if err != nil {
		log.Printf("Failed to send god info message: %v", err)
	} else {
		log.Printf("Sent god info message for user %s (fallback method)", username)
	}
}

// sendEmbedChannelMessage sends an embed message to the channel (fallback for when ephemeral isn't available)
func (b *Bot) sendEmbedChannelMessage(s *discordgo.Session, channelID string, embed *discordgo.MessageEmbed) {
	_, err := s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		log.Printf("Failed to send embed god info message: %v", err)
	} else {
		log.Printf("Sent embed god info message (fallback method)")
	}
}

// createGodInfoEmbeds creates a single comprehensive embed for god information
func (b *Bot) createGodInfoEmbeds(god *smite.God, role smite.Role, team int) []*discordgo.MessageEmbed {
	title := fmt.Sprintf("✅ You claimed %s (%s) on Team %d!", god.Name, role, team)
	return b.createGodEmbed(god, title)
}

// createStandaloneGodInfoEmbed creates god information embed for the godinfo slash command
func (b *Bot) createStandaloneGodInfoEmbed(god *smite.God) []*discordgo.MessageEmbed {
	title := fmt.Sprintf("📖 God Information: %s", god.Name)
	return b.createGodEmbed(god, title)
}

// createGodEmbed creates a god information embed with the specified title
func (b *Bot) createGodEmbed(god *smite.God, title string) []*discordgo.MessageEmbed {
	// Convert roles to strings for joining
	roleStrings := make([]string, len(god.Roles))
	for i, r := range god.Roles {
		roleStrings[i] = string(r)
	}

	// Create main god info embed with detailed ability information
	var abilityTexts []string
	for _, ability := range god.Abilities {
		// Strip HTML tags and format ability
		cleanDesc := smite.StripHTMLTags(ability.Description)
		abilityText := fmt.Sprintf("**%s (%s)**\n%s", ability.Name, ability.Slot, cleanDesc)
		abilityTexts = append(abilityTexts, abilityText)
	}

	// Generate URLs for god information
	smiteURL := smite.GetGodSmite2ComURL(god.Name)
	wikiURL := smite.GetGodWikiURL(god.Name)
	smiteLiveURL := smite.GetGodSmite2LiveURL(god.Name)

	mainEmbed := &discordgo.MessageEmbed{
		Title: title,
		URL:   smiteLiveURL,
		Description: fmt.Sprintf("🔸 **%s** (%s)\n🔸 **Damage Type:** %s\n🔸 **Attack Type:** %s\n🔸 **Roles:** %s\n\n**Abilities:**\n%s\n\n**Links:**\n🔗 [smite2.com](%s)\n📖 [Smite 2 Official Wiki](%s)\n📖 [Smite 2 Live (Builds and METAs)](%s)",
			god.Name, god.Pantheon,
			god.DamageType,
			god.AttackType,
			strings.Join(roleStrings, ", "),
			strings.Join(abilityTexts, "\n\n"),
			smiteURL,
			wikiURL,
			smiteLiveURL,
		),
		Color: 0x0066cc,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: god.PortraitURL,
		},
	}

	// Return just the single main embed
	return []*discordgo.MessageEmbed{mainEmbed}
}
