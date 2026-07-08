package main

import (
	"log/slog"
	"os"

	"github.com/bwmarrin/discordgo"
)

func main() {
	// setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// get token
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if len(token) == 0 {
		slog.Error("Failed get token from env")
		os.Exit(1)
	}

	// init bot client
	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("Failed init discord bot client", "error", err)
		os.Exit(1)
	}

	slog.Info("Successful init discord bot client")

	// open discord session
	if err := discord.Open(); err != nil {
		slog.Error("Failed open discord session", "error", err)
		os.Exit(1)
	}

	defer discord.Close()
}
