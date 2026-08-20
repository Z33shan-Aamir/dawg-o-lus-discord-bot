# Discord Bot

A Discord bot written in Go that uses a local AI inference workflow and supports slash commands, channel blocking, custom prompts, and message cooldowns.

## Key Files

- `cmd/main/main.go` - application entrypoint.
- `internal/bot/bot.go` - bot logic, command handling, message processing.
- `config/config.go` - environment config loader.
- `Dockerfile` - containerized build for deployment.
- `fly.toml` - Fly.io application configuration.

## Requirements

- Go `1.26.3` or later
- `git`
- `flyctl` for Fly.io deployment
- Optionally `air` for live reload during development

## Disocrd Bot Commands

Moderators can run the following command

- `/block-channel` this commands blocks the bot from sending messages to channel where this command was used
- `/unblock-channel` command to unblocks the channel
- `/list-blocked-channels` Lists all the currently blocked channels
- `/change-system-prompt prompt:<your-prompt>` this command changes the system prompt to your prompt which is server specific.
- `/reset-system-prompt` changes the prompt to default system prompt

## Environment Variables

The app loads configuration from environment variables using `github.com/joho/godotenv`.

Required variables:

- `DISCORD_BOT_TOKEN` - Discord bot token
- `AI_API` - AI API key
- `AI_MODEL` - model identifier
- `SYSTEM_PROMPT_V1` - default system prompt
- `RESPONSE_CRITERIA_LATEST` - response criteria model identifier
- `IS_RESPONSE_REQUIRED_MODEL` - model used to determine whether a response is required

Optional environment configuration is also included in `fly.toml` under `[env]`.

## Run Locally with Go

1. Set your environment variables.

   Example using a `.env` file:

   ```bash
   cat > .env <<'EOF'
   DISCORD_BOT_TOKEN=your-token-here
   AI_API=your-api-key-here
   AI_MODEL=meta-llama/llama-3.1-8b-instruct
   SYSTEM_PROMPT_V1="<your system prompt>"
   RESPONSE_CRITERIA_LATEST=google/gemma-4-26b-a4b-it
   IS_RESPONSE_REQUIRED_MODEL=google/gemma-4-26b-a4b-it
   EOF
   ```

2. Run the app directly with Go:

   ```bash
   go run ./cmd/main
   ```

3. Or build and execute the binary:

   ```bash
   go build -o bot ./cmd/main
   ./bot
   ```

## Run Locally with `air`

`air` enables live reload during development. If you do not have it installed, install it:

```bash
go install github.com/cosmtrek/air@latest
```

Then run the app with `air` from the project root:

## Deploy to Fly.io

### 1. Install `flyctl`

Follow Fly.io install instructions at: https://fly.io/docs/getting-started/installing-flyctl/

### 2. Authenticate

```bash
fly auth login
```

### 3. Create or link your app

If you have not created the app yet, run:

```bash
fly launch --no-deply
```

This will create the app and not deply it.
You can change the settings either during the command running or in the `fly.toml` file

### 4. Configure environment variables on Fly

Set the required environment variables on Fly before deploy:

```bash
fly secrets set \
  DISCORD_BOT_TOKEN=your-token-here \
  AI_API=your-api-key-here \
```

### 5. Deploy

Use the provided `fly.toml` (if fly.io costs are less that 5$ it gets wavered) and deploy the app:

```bash
fly launch --no-deploy
fly deploy --ha=false # the flag is used to disable high availablity for low cost and because it will respond 2 times

```

### 6. Check app status

```bash
fly status
```

### 7. View logs

```bash
fly logs
```

## Docker Build (Optional)

This repo includes a `Dockerfile` for a multi-stage build.

```bash
docker build -t discord-bot .
docker run --env-file .env discord-bot
```

## Notes

- The bot uses `discordgo` for Discord interaction.
- The project expects the `cmd/main` package as the launch point.
- `fly.toml` already configures `IS_PROD=true` and AI-related env vars for production deployment.

## Troubleshooting

- If the bot exits with a missing env var error, verify each env var is defined.
- If Fly deployment fails, inspect `fly logs` and ensure the app name in `fly.toml` matches your Fly app.
- If live reload does not work, confirm `air` is installed and the `.air.toml` file is in the repository root.
