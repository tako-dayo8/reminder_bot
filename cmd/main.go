package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"reminder_bot/cmd/database"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

var minValue = 0.0

func commandLog(commandName string, interaction *discordgo.InteractionCreate) {
	var user *discordgo.User
	if interaction.Member != nil {
		user = interaction.Member.User
	} else {
		user = interaction.User
	}

	t, err := discordgo.SnowflakeTimestamp(interaction.ID)
	if err != nil {
		slog.Warn("Failed get interaction timestamp")
	}

	var interactionType = map[discordgo.InteractionType]string{
		1: "InteractionPing",
		2: "InteractionApplicationCommand",
		3: "InteractionMessageComponent",
		4: "InteractionApplicationCommandAutocomplete",
		5: "InteractionModalSubmit",
	}

	slog.Info(fmt.Sprintf("Run %s command", commandName),
		"interactionID", interaction.ID,
		"type", interactionType[interaction.Type],
		"timeStamp", t.Local().String(),
		"commandName", interaction.ApplicationCommandData().Name,
		"commandOptions", interaction.ApplicationCommandData().Options,
		"guildID", interaction.GuildID,
		"channelID", interaction.ChannelID,
		"userID", user.ID,
		"userName", user.Username,
		"appPermissions", interaction.AppPermissions,
	)
}

// definition add commands
var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "ping",
		Description: "greet reminder bot",
	},
	{
		Name:        "remind",
		Description: "create new remind or update remind",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "add",
				Description: "add remind",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "title",
						Description: "remind title",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "hour",
						Description: "remind at hour (0~24)",
						Required:    true,
						MinValue:    &minValue,
						MaxValue:    24,
					},
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "minutes",
						Description: "remind at minutes (0~60)",
						Required:    true,
						MinValue:    &minValue,
						MaxValue:    60,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "description",
						Description: "remind description",
						Required:    false,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "repeat",
						Description: "repeat",
						Required:    false,
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							{
								Name: "None", Value: "none",
							},
							{
								Name: "EveryDay", Value: "daily",
							},
							{
								Name: "EveryWeek", Value: "weekly",
							},
						},
					},
				},
			},
		},
	},
}

// mapping command name -> command handler
var commandHandlers = map[string]func(session *discordgo.Session, interaction *discordgo.InteractionCreate){
	"ping":   pingHandler,
	"remind": remindHandler,
}

// ping command handler
func pingHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	commandLog("ping", interaction)

	// response interaction
	err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "hello!!!", // message context
		},
	})
	if err != nil {
		slog.Error("Failed interaction response", "error", err, "GuildID", interaction.GuildID)
	}
}


// embed color
const (
	colorRemind = 0xEF9F27 
	colorError  = 0xED4245
)

// remind command handler
// TODO: create remindHandler
func remindHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	commandLog("remind", interaction)

	data := interaction.ApplicationCommandData()
	sub := data.Options[0]

	switch sub.Name {
	case "add":
		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(sub.Options))

		for _, opt := range sub.Options {
			optionMap[opt.Name] = opt
		}

		// get options
		// title
		title := optionMap["title"].StringValue()
		// hour
		hour := optionMap["hour"].IntValue()
		// minutes
		minutes := optionMap["minutes"].IntValue()

		// description
		// description := "none"
		// if opt, ok := optionMap["description"]; ok {
		// 	description = opt.StringValue()
		// }

		// repeat
		// repeat := "none"
		// if opt, ok := optionMap["repeat"]; ok {
		// 	repeat = opt.StringValue()
		// }

		// Debug
		slog.Info("check int64 to int", "hour", int(hour), "minutes",int(minutes))

		now := time.Now()
		unix := time.Date(now.Year(), now.Month(), now.Day(), 15, 0, 0, 0, time.Local)

		// Debug
		slog.Info("check unix func", "unix.Unix()", unix.Unix())

		//TODO: schedule reminder

		// TODO:bug - mismatch between reminder time and displayed time
		embed := &discordgo.MessageEmbed{
			Author: &discordgo.MessageEmbedAuthor{
				Name: "🔔 Set a reminder", 
			},
			Title: title, 
			Color: colorRemind,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "remind at",
					Value:  fmt.Sprintf("<t:%d:f>", unix.Unix()), 
					Inline: true,
				},
				{
					Name:   "until",
					Value:  fmt.Sprintf("<t:%d:R>", unix.Unix()), 
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				// TODO: get a remind id
				Text: fmt.Sprintf("ID: %s ・ /remind delete to delete", "a"),
			},
		}
	
		// response interaction
		err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				// Content: "🔔 Set a reminder", // message context
				Embeds: []*discordgo.MessageEmbed{embed},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})

		if err != nil {
			slog.Error("Failed interaction response", "error", err, "GuildID", interaction.GuildID)
		}

		return
	}

	err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "no selected sub command", // message context
		},
	})

	if err != nil {
		slog.Error("Failed interaction response", "error", err, "GuildID", interaction.GuildID)
	}
}

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
	// get guildID
	guildID := os.Getenv("GUILD_ID")
	if len(guildID) == 0 {
		slog.Warn("Failed get guildID from env")
	}
	// get database path (default: ./database)
	var databasePath string
	databasePath = os.Getenv("DB_PATH")
	if len(databasePath) == 0 {
		slog.Warn("Failed get databasePath from env")
		databasePath = "database"
	}

	err := os.MkdirAll(databasePath, 0755)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed make %s directory", databasePath), "error", err)
		os.Exit(1)
	}

	// init sqlite3
	db, err := database.OpenSQL(filepath.Join(databasePath, "db.sqlite"))
	if err != nil {
		slog.Error("Failed open sql", "error", err)
		os.Exit(1)
	}
	if err := database.InitSQL(db); err != nil {
		slog.Error("Failed init sql", "error", err)
		os.Exit(1)
	}
	slog.Info("Successful init sqlite")

	// init bot client
	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		slog.Error("Failed init discord bot client", "error", err)
		os.Exit(1)
	}
	slog.Info("Successful init discord bot client")

	// add handler before connect gateway
	discord.AddHandler(func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
		// command only filter from interaction type
		if interaction.Type != discordgo.InteractionApplicationCommand {
			return
		}
		if handler, ok := commandHandlers[interaction.ApplicationCommandData().Name]; ok {
			handler(session, interaction)
		}
	})

	// connect discord gateway
	if err := discord.Open(); err != nil {
		slog.Error("Failed open discord session", "error", err)
		os.Exit(1)
	}
	defer discord.Close()

	// create command after connect gateway
	for _, cmd := range commands {
		if _, err := discord.ApplicationCommandCreate(discord.State.User.ID, guildID, cmd); err != nil {
			slog.Error("Failed command create", "error", err, "cmd", cmd.Name)
		}
	}

	slog.Info("Succuss startup. please Ctl+C to End", "id", discord.State.User.ID, "username", discord.State.User.Username)
	// wait
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
