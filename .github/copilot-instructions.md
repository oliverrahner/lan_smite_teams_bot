# Copilot Instructions for lan_discord_bot

## Language & Framework
- **Primary Language**: Golang
- **Discord Library**: `github.com/bwmarrin/discordgo` (or similar)
- **Build Tool**: Go modules

## Project Structure
- Organize code into clear packages: `discord`, `ai`, `mapping`, `screenshot`, `config`, `test`, etc.
- Use Go modules for dependency management
- Place main entry point in `cmd/lan_discord_bot/main.go`
- Keep internal packages in `internal/` to prevent external imports

## Code Style & Best Practices
- Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` for consistent formatting
- Prefer explicit error handling; never ignore errors
- Use descriptive names for variables, functions, and types
- Export only what needs to be public; keep internal implementation details private
- Write idiomatic Go code following standard library conventions

## Concurrency & Robustness
- Use goroutines and channels for asynchronous tasks (e.g., screenshot monitoring, Discord events)
- Protect shared resources with mutexes or other synchronization primitives
- Ensure robust error handling and logging for all external interactions (Discord API, filesystem, AI models)
- Use context for cancellation and timeout management
- Handle graceful shutdown of all goroutines

## Configuration Management
- **Guild-Specific Configuration**: All bot configuration must be guild-specific (Discord's internal name for "server" or "instance")
- Store configuration per guild, allowing the bot to serve multiple Discord servers simultaneously
- Use guild ID as the primary key for configuration lookups
- Store configuration in a file (e.g., `config.yaml` or `config.json`) or database with guild-scoped settings
- Validate configuration at startup and handle missing/invalid values gracefully
- Support environment variables for sensitive data (tokens, API keys)
- Document all configuration options
- Ensure voice channels, text channels, and all other settings are scoped to specific guilds

## Discord Integration
- Use a well-maintained Go Discord library (e.g., `github.com/bwmarrin/discordgo`)
- Implement command structure for bot operations (`sort`, `unite`, etc.) using slash commands. Don't use simple prefix messages!
- Make the slash commands per-guild, not global ones
- Clean up command registrations on shutdown
- Use batch requests to create and remove slash commands
- **Guild Context**: Always process commands and events in the context of the specific guild
- Retrieve guild-specific configuration for each command and event
- Ensure proper permissions and error handling for channel operations
- Log all Discord API interactions for debugging with guild ID context
- Handle rate limiting appropriately
- Support the bot being present in multiple guilds simultaneously


## Testing Strategy
- Develop a test harness for screenshot interpretation
- Use Go's `testing` package for unit and integration tests
- Write table-driven tests where appropriate
- Document how to add new tests and expected outputs
- Aim for high test coverage on critical paths
- Include test fixtures for screenshots and expected results

## Logging & Auditing
- Log all user movements, mapping changes, and errors
- Use structured logging (e.g., `logrus`, `zap`, or standard `log/slog` package)
- Include contextual information (user IDs, timestamps, operation types)
- Separate log levels appropriately (debug, info, warn, error)
- Consider log rotation for production deployments

## Error Handling
- Always check and handle errors explicitly
- Wrap errors with context using `fmt.Errorf` with `%w` verb
- Return errors up the stack; don't panic in library code
- Use sentinel errors or custom error types for expected error conditions
- Log errors with sufficient context for debugging

## Documentation
- Document all exported functions, types, and packages with godoc comments
- Maintain a clear README with setup and usage instructions
- Update documentation for new features and changes
- Include examples for complex functionality
- Document API endpoints and command syntax

## Security Considerations
- Never commit Discord tokens or API keys to version control
- Use environment variables or secure configuration for secrets
- Validate all user input from Discord commands
- Implement rate limiting to prevent abuse
- Follow principle of least privilege for Discord bot permissions

## Performance
- Avoid blocking operations in Discord event handlers
- Use buffered channels appropriately to prevent goroutine leaks
- Profile and optimize hot paths
- Consider caching for frequently accessed data
- Monitor memory usage, especially for image processing

