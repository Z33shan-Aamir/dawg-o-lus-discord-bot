package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/discord-bitch/config"
	"github.com/discord-bitch/internal/bot"
)

func main() {
	cfg := config.Load()

	Bot, err := bot.New(cfg)
	if err != nil {
		log.Fatal("bot could not be initialized", "error", err)
	}

	err = Bot.Start()
	if err != nil {
		log.Fatal("failed to start bot session", "error", err)
	}

	log.Info("Bot session started, awaiting gateway ready signal...")

	// Keep running until an OS interrupt/kill signal is received
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Clean shutdown down resources gracefully
	Bot.Close()
}
