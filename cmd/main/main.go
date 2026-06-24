package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"github.com/discord-bitch/config"
	"github.com/discord-bitch/internal/bot"
)

func main() {
	cfg := config.Load()

	Bot, err := bot.New(cfg)
	if err != nil {
		log.Fatal("bot could not start", err)
	}

	err = Bot.Start()
	if err != nil {
		log.Info("Bot has been started")
	}

	// Set status once right after starting
	err = Bot.Session.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status: string(discordgo.StatusOnline),
	})
	if err != nil {
		log.Error("Could not set status", err)
	}

	// Keep running until an OS interrupt/kill signal is received
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Clean shutdown down resources gracefully
	Bot.Close()
}
