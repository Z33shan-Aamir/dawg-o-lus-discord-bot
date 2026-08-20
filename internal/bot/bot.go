package bot

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"github.com/Z33shan-Aamir/inkwell/config"
	"github.com/Z33shan-Aamir/inkwell/internal/ai"
)

type Bot struct {
	Session         *discordgo.Session
	Config          *config.Config
	AI              *ai.Inference
	Context         map[string]*ai.ContextManager // Maps Channel IDs to their conversation histories
	contextMu       sync.RWMutex
	blockedChannels map[string]bool
	blockedMu       sync.RWMutex
	customPrompt    map[string]string
	customPromptMu  sync.RWMutex
	cooldowns       map[string]time.Time
	cooldownMu      sync.RWMutex
}

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "block-channel",
		Description: "Stop the bot from responding in this channel",
	},
	{
		Name:        "unblock-channel",
		Description: "Allow the bot to respond in this channel again",
	},
	{
		Name:        "reset-system-prompt",
		Description: "Reset the system prompt back to the default",
	},
	{
		Name:        "list-blocked-channels",
		Description: "List all currently blocked channels",
	},
	{
		Name:        "change-system-prompt",
		Description: "Send a custom prompt",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "prompt", // Must be lowercase, no spaces
				Description: "Type your prompt here",
				Required:    true, // Makes the field mandatory for the user
			},
		},
	},
}

func New(cfg *config.Config) (*Bot, error) {
	s, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, err
	}

	s.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentMessageContent

	b := &Bot{
		Session:         s,
		Config:          cfg,
		AI:              ai.New(cfg), // Instantiated once at startup
		Context:         make(map[string]*ai.ContextManager),
		blockedChannels: make(map[string]bool, 0),
		customPrompt:    make(map[string]string, 0),
		cooldowns:       make(map[string]time.Time),
	}
	// Register it as a method handler
	b.Session.AddHandler(b.OnReady)
	b.Session.AddHandler(b.GuildCreate)
	b.Session.AddHandler(b.InteractionCreate)
	b.Session.AddHandler(b.MessageCreate)

	return b, nil
}

func (b *Bot) RegisterCommands(guildID string) error {
	botID := b.Session.State.User.ID
	log.Info("registering commands", "guildID", guildID, "botID", botID)
	_, err := b.Session.ApplicationCommandBulkOverwrite(botID, guildID, commands)
	if err != nil {
		log.Error("could not register commands", "guild", guildID, "err", err)
		return err
	}
	log.Info("commands registered successfully", "guild", guildID)
	return nil
}

func (b *Bot) InteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	switch i.ApplicationCommandData().Name {
	case "block-channel":
		if !hasModPermission(i) {
			respondEphemeral(s, i, "you don't have permission for that")
			return
		}
		b.BlockChannel(i.ChannelID)
		respondEphemeral(s, i, "got it, going quiet in here")

	case "unblock-channel":
		if !hasModPermission(i) {
			respondEphemeral(s, i, "you don't have permission for that")
			return
		}
		b.UnblockChannel(i.ChannelID)
		respondEphemeral(s, i, "alright, I'm back")
	case "reset-system-prompt":
		if !hasModPermission(i) {
			respondEphemeral(s, i, "you don't have permission for that")
			return
		}
		b.customPromptMu.Lock()
		delete(b.customPrompt, i.GuildID)
		b.customPromptMu.Unlock()
		respondEphemeral(s, i, "back to my original self, unfortunately for you")

	case "list-blocked-channels":
		if !hasModPermission(i) {
			respondEphemeral(s, i, "you don't have permission for that")
			return
		}
		b.blockedMu.RLock()
		var blocked []string
		for channelID := range b.blockedChannels {
			blocked = append(blocked, fmt.Sprintf("<#%s>", channelID))
		}
		b.blockedMu.RUnlock()

		if len(blocked) == 0 {
			respondEphemeral(s, i, "no blocked channels, i'm free to roam everywhere")
			return
		}
		respondEphemeral(s, i, "blocked channels: "+strings.Join(blocked, ", "))
	case "change-system-prompt":
		if !hasModPermission(i) {
			respondEphemeral(s, i, "you don't have permission for that")
			return
		}
		options := i.ApplicationCommandData().Options
		var userInput string
		for _, opt := range options {
			if opt.Name == "prompt" {
				userInput = opt.StringValue()
			}
		}
		if strings.TrimSpace(strings.ReplaceAll(userInput, " ", "")) != "" {
			b.customPromptMu.Lock()
			b.customPrompt[i.GuildID] = userInput
			b.customPromptMu.Unlock()
			respondEphemeral(s, i, "Ok fine I now have a change of heart")
			return
		}
		respondEphemeral(s, i, "Are you an Idiot?")

	}

}

func hasModPermission(i *discordgo.InteractionCreate) bool {
	if i.Member == nil {
		return false
	}

	return i.Member.Permissions&discordgo.PermissionManageChannels != 0
}

func (b *Bot) BlockChannel(channelID string) {
	b.blockedMu.Lock()
	defer b.blockedMu.Unlock()
	b.blockedChannels[channelID] = true
}

func (b *Bot) UnblockChannel(channelID string) {
	b.blockedMu.Lock()
	defer b.blockedMu.Unlock()
	delete(b.blockedChannels, channelID)
}

func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) OnReady(s *discordgo.Session, r *discordgo.Ready) {
	// Set status safely inside OnReady once WebSocket is ready
	err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status: string(discordgo.StatusOnline),
	})
	if err != nil {
		log.Error("Could not set status", "error", err)
	}

	for _, guild := range r.Guilds {
		if err := b.RegisterCommands(guild.ID); err != nil {
			log.Error("failed to register commands", "guild", guild.ID, "err", err)
		}
	}
	log.Info("bot is ready", "username", r.User.Username)
}

func (b *Bot) Close() {
	log.Warn("Bot is shutting down...")
	if err := b.Session.Close(); err != nil {
		log.Error("Error closing discord session", "err", err)
	}
}

func (b *Bot) Start() error {
	if err := b.Session.Open(); err != nil {
		return err
	}
	log.Info("Bot has started")
	return nil
}

func (b *Bot) GuildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	if err := b.RegisterCommands(g.Guild.ID); err != nil {
		log.Error("failed to register commands", "guild", g.Guild.ID, "err", err)
	}
}

func (b *Bot) getSystemPrompt(guildID string) string {
	b.customPromptMu.RLock()
	defer b.customPromptMu.RUnlock()
	if prompt, ok := b.customPrompt[guildID]; ok {
		return prompt
	}
	return b.Config.SystemPrompt
}

func (b *Bot) isOnCooldown(userID string) bool {
	b.cooldownMu.RLock()
	defer b.cooldownMu.RUnlock()
	last, ok := b.cooldowns[userID]
	if !ok {
		return false
	}
	return time.Since(last) < 30*time.Second
}

func (b *Bot) setCooldown(userID string) {
	b.cooldownMu.Lock()
	defer b.cooldownMu.Unlock()
	b.cooldowns[userID] = time.Now()
}

func (b *Bot) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Don't reply to yourself
	if m.Author.ID == s.State.User.ID {
		return
	}
	if b.isOnCooldown(m.Author.ID) {
		return
	}
	if time.Since(m.Timestamp) > 5*time.Minute {
		return
	}
	b.blockedMu.RLock()
	blocked := b.blockedChannels[m.ChannelID]
	b.blockedMu.RUnlock()
	if blocked {
		return
	}
	if m.Content == "ping" {
		if os.Getenv("IS_PROD") == "false" {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Model: %s\n System Prompt: %s\n", b.AI.ModelName(), b.Config.SystemPrompt))
			return
		}
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Model: %s", b.Config.ModelName))
		return
	}

	// 1. Get or create chat history context for this specific channel
	b.contextMu.Lock()
	ctxMgr, exists := b.Context[m.ChannelID]
	if !exists {
		ctxMgr = ai.NewContextManager(50) // Retain last 20 messages
		b.Context[m.ChannelID] = ctxMgr
	}
	b.contextMu.Unlock()

	hasDirectMention := false
	// 2. Format incoming prompt payload
	formattedUserMsg := fmt.Sprintf("%s: %s", m.Author.GlobalName, m.Content)
	if m.ReferencedMessage != nil {
		formattedUserMsg = fmt.Sprintf(
			"%s (replying to %s who said \"%s\"): %s",
			m.Author.GlobalName,
			m.ReferencedMessage.Author.GlobalName,
			m.ReferencedMessage.Content,
			m.Content,
		)

		if m.ReferencedMessage.Author.ID == s.State.User.ID {
			hasDirectMention = true
		}
	}
	ctxMgr.Add("user", formattedUserMsg)

	if len(m.Mentions) >= 1 {
		if s.State.User.ID == m.Mentions[0].ID {
			hasDirectMention = true
			log.Info("Message has direct mention")
		}
	}

	// 3. Generate response utilizing stored history
	if !hasDirectMention {
		if result, err := b.AI.IsResponseRequired(formattedUserMsg, ctxMgr.Get()); err != nil || !result {
			return
		}
	}
	err := s.ChannelTyping(m.ChannelID)
	if err != nil {
		log.Error("Failed to send typing status", err)
	}
	systemPrompt := b.getSystemPrompt(m.GuildID)
	aiResponse, err := b.AI.GenerateResponse(systemPrompt, ctxMgr.Get(), formattedUserMsg)
	if err != nil {
		log.Error("Couldn't generate response: " + err.Error())
		return
	}

	// 4. Update the history manager with the exchange
	ctxMgr.Add("assistant", aiResponse)

	// 5. Send back to the channel the message originated from rather than hardcoded string
	s.ChannelMessageSendReply(m.ChannelID, aiResponse, m.Reference())
	b.setCooldown(m.Author.ID)
}
