package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"reminder_bot/cmd/database"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

var db *sql.DB
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
						Description: "remind hours from now",
						Required:    true,
						MinValue:    &minValue,
						MaxValue:    23,
					},
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "minutes",
						Description: "remind minutes from now",
						Required:    true,
						MinValue:    &minValue,
						MaxValue:    59,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "description",
						Description: "remind description",
						Required:    false,
					},
					// {
					// 	Type:        discordgo.ApplicationCommandOptionString,
					// 	Name:        "interval",
					// 	Description: "interval",
					// 	Required:    false,
					// 	Choices: []*discordgo.ApplicationCommandOptionChoice{
					// 		{
					// 			Name: "None", Value: "none",
					// 		},
					// 		{
					// 			Name: "EveryDay", Value: "daily",
					// 		},
					// 		{
					// 			Name: "EveryWeek", Value: "weekly",
					// 		},
					// 	},
					// },
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
	colorFire   = 0x5865F2
)

// remind command handler
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
		// hours
		hours := optionMap["hour"].IntValue()
		// minutes
		minutes := optionMap["minutes"].IntValue()

		// description
		description := ""
		if opt, ok := optionMap["description"]; ok {
			description = opt.StringValue()
		}

		// interval
		// interval := "none"
		// if opt, ok := optionMap["interval"]; ok {
		// 	interval = opt.StringValue()
		// }

		now := time.Now()
		// add hours and minutes
		remindTime := now.Add(time.Duration(hours) * time.Hour).Add(time.Duration(minutes) * time.Minute)


		var userID string

		if interaction.User != nil {
			userID = interaction.User.ID
		} else if interaction.Member != nil && interaction.Member.User != nil  {
			userID = interaction.Member.User.ID
		} else {
			slog.Error("Failed get to UserID and Member.UserID")
		}

		slog.Debug("get userID", "userID", userID)

		newSchedule := database.NewSchedule{
			Title: title,
			Description: &description,
			ChannelID: interaction.ChannelID,
			UserID: userID,
			RemindAt: remindTime.Unix(),
		}

		slog.Debug("newSchedule", "value", newSchedule)

		//TODO: schedule reminder
		insertID, err := database.CreateSchedule(db, newSchedule)
		if err != nil {
			slog.Error("Failed create schedule for database" , "error", err)
			err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Sorry!! Failed create new schedule", 
				Flags: discordgo.MessageFlagsEphemeral,
			},
			})

			if err != nil {
				slog.Error("Failed interaction response", "error", err, "GuildID", interaction.GuildID)
			}
			return
		}
		slog.Debug("Succuss save new schedule for database", "insertID", insertID)

		// create embed
		embed := &discordgo.MessageEmbed{
			Author: &discordgo.MessageEmbedAuthor{
				Name: "🔔 Set a reminder", 
			},
			Title: title, 
			Description: description,
			Color: colorRemind,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "remind at",
					Value:  fmt.Sprintf("<t:%d:f>", remindTime.Unix()), 
					Inline: true,
				},
				{
					Name:   "until",
					Value:  fmt.Sprintf("<t:%d:R>", remindTime.Unix()), 
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("ID: %d ・ /remind delete to delete", insertID),
			},
		}
	
		// response interaction
		if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				// Content: "🔔 Set a reminder", // message context
				Embeds: []*discordgo.MessageEmbed{embed},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		}); err != nil {
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

func runScheduler(ctx context.Context, s *discordgo.Session) {
	slog.Debug("check runScheduler start")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <- ctx.Done():
			slog.Info("scheduler stopped.")
			return
		case <- ticker.C:
			// slog.Debug("check select case.")
			if err := tick(ctx, s); err != nil {
				slog.Error("tick failed", "err", err)
			}
		}
	}
}

type due struct {
	id int64
	channelID string
	userID string
	title string
	description sql.NullString
}

func tick(ctx context.Context, s *discordgo.Session) error {
	now := time.Now().Unix()

	const SQL = `
	SELECT id, channel_id, user_id, title, description
	FROM schedules
	WHERE done = 0 AND remind_at <= ?
	ORDER BY remind_at
	LIMIT 50
	`

	rows, err := db.QueryContext(ctx, SQL, now)
	if err != nil {
		return fmt.Errorf("query due schedules: %w", err)
	}

	var list []due
	for rows.Next() {
		var d due
		if err := rows.Scan(&d.id, &d.channelID, &d.userID ,&d.title, &d.description ); err != nil {
			rows.Close()
			return fmt.Errorf("scan: %w", err)
		}
		slog.Debug("check", "d", d)
		list = append(list, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}

	var wg sync.WaitGroup
	for _, d := range list {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := notify(s, d); err != nil {
				slog.Error("notify failed", "id", d.id, "err", err)
				return 
			}
			if err := database.MarkDone(db,d.id); err != nil {
				slog.Error("mark done failed", "id", d.id, "err", err)
			}
		}()
	}
	wg.Wait() 

	return nil
}


func notify(s *discordgo.Session, d due)  error {
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name: "🔔 Reminder",
		},
		Title:     d.title,
		Color:     colorFire,
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("ID: %d", d.id),
		},
	}

	_, err := s.ChannelMessageSendComplex(d.channelID, &discordgo.MessageSend{
		Content: fmt.Sprintf("<@%s>", d.userID),
		Embeds:  []*discordgo.MessageEmbed{embed},
	})
	return err
}


func main() {
	var logLevel slog.Leveler
	debugFlag := os.Getenv("DEBUG_MODE")
	
	if strings.ToUpper(debugFlag) == "TRUE" || strings.ToUpper(debugFlag) == "YES" {
		logLevel = slog.LevelDebug
	} else {
		logLevel = slog.LevelInfo
	}

	// setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
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
	db, err = database.OpenSQL(filepath.Join(databasePath, "db.sqlite"))
	if err != nil {
		slog.Error("Failed open sql", "error", err)
		os.Exit(1)
	}
	if err := database.InitSQL(db); err != nil {
		slog.Error("Failed init sql", "error", err)
		os.Exit(1)
	}

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

	ctx, cancel := context.WithCancel(context.Background())

	schedulerDone := make(chan struct{})
	go func() {
		defer close(schedulerDone)
		runScheduler(ctx, discord)
	}()


	slog.Info("Succuss startup. please Ctl+C to End", "id", discord.State.User.ID, "username", discord.State.User.Username)
	// wait
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	cancel()
	<- schedulerDone
	db.Close()
}
