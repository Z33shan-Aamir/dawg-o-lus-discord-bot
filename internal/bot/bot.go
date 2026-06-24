package bot

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"github.com/discord-bitch/config"
	"github.com/discord-bitch/internal/ai"
)

type Bot struct {
	Session *discordgo.Session
	Config  *config.Config
	AI      *ai.Inference
	Context map[string]*ai.ContextManager // Maps Channel IDs to their conversation histories
}

func New(cfg *config.Config) (*Bot, error) {
	s, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, err
	}

	s.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentMessageContent

	b := &Bot{
		Session: s,
		Config:  cfg,
		AI:      ai.New(cfg), // Instantiated once at startup
		Context: make(map[string]*ai.ContextManager),
	}

	// Register it as a method handler
	b.Session.AddHandler(b.MessageCreate)
	return b, nil
}

func (b *Bot) Start() error {
	if err := b.Session.Open(); err != nil {
		return err
	}
	log.Info("Bot has started")
	return nil
}

func (b *Bot) Close() {
	log.Warn("Bot has stopped")
	b.Session.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status: string(discordgo.StatusOffline),
	})
	b.Session.Close()
}

func (b *Bot) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Don't reply to yourself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Model: %s\n System Prompt: %s\n", b.AI.ModelName(), b.Config.SystemPrompt))
		return
	}

	// Process message if within 5 minutes
	if time.Since(m.Timestamp) <= 5*time.Minute {
		// 1. Get or create chat history context for this specific channel
		ctxMgr, exists := b.Context[m.ChannelID]
		if !exists {
			ctxMgr = ai.NewContextManager(20) // Retain last 20 messages
			b.Context[m.ChannelID] = ctxMgr
		}

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
		}
		// 3. Generate response utilizing stored history

		if result, err := b.AI.IsResponseRequired(formattedUserMsg, ctxMgr.Get()); err != nil || result == false {
			return
		}

		err := s.ChannelTyping(m.ChannelID)

		if err != nil {
			log.Error("Failed to send typing status", err)
		}

		aiResponse, err := b.AI.GenerateResponse(ctxMgr.Get(), formattedUserMsg)
		if err != nil {
			log.Error("Couldn't generate response: " + err.Error())
			return
		}

		ctxMgr.Add("user", formattedUserMsg)
		// 4. Update the history manager with the exchange
		ctxMgr.Add("assistant", aiResponse)

		// 5. Send back to the channel the message originated from rather than hardcoded string
		s.ChannelMessageSendReply(m.ChannelID, aiResponse, m.Reference())
	}
}
