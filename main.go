package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS schedules (
	id			INTEGER PRIMARY KEY,
	title		TEXT	NOT NULL,
	description TEXT,
	channel_id  TEXT 	NOT NULL,	-- target channel
	user_id		TEXT	NOT NULL,	-- target user for example roll, everyone and user
	remind_at	INTEGER NOT NULL,	-- Unix Seconds
	interval	INTEGER,			-- NULL = not interval
	done		INTEGER NOT NULL DEFAULT 0,
	created_at	INTEGER NOT NULL,
	updated_at	INTEGER NOT NULL
) STRICT;
`

type Schedule struct {
	id          int64
	title       string
	description *string
	channelID   string
	userID      string
	remindAt    int64
	interval    *int64
	done        bool
	createdAt   int64
	updateAt    int64
}

type CreateSchedule struct {
	title       string
	description *string
	channelID   string
	userID      string
	remindAt    int64
	interval    *int64
}

type UpdateSchedule struct {
	title       *string
	description *string
	channelID   *string
	userID      *string
	remindAt    *int64
	interval    *int64
	done        *bool
}

func openSQL(path string) (*sql.DB, error) {
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"

	// func name is "open" but not connection to dsn(file path?) check only
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1) // Avoidance conflict

	// verify a connection for the db
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// initialize sqlite
func initSQL(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}

func createSchedule(db *sql.DB, schedule CreateSchedule) (int64, error) {
	now := time.Now().Unix()

	const SQL = "INSERT INTO schedules (title, description, channel_id, user_id ,remind_at, interval, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	res, err := db.Exec(SQL, schedule.title, schedule.description, schedule.channelID, schedule.userID, schedule.remindAt, schedule.interval, now, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// definition add commands
var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "ping",
		Description: "greet reminder bot",
	},
}

// mapping command name -> command handler
var commandHandlers = map[string]func(session *discordgo.Session, interaction *discordgo.InteractionCreate){
	"ping": pingHandler,
}

// ping command handler
func pingHandler(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
	slog.Info("Run ping command", "GuildID", interaction.GuildID)

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
	db, err := openSQL(filepath.Join(databasePath, "db.sqlite"))
	if err != nil {
		slog.Error("Failed open sql", "error", err)
		os.Exit(1)
	}
	if err := initSQL(db); err != nil {
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

	slog.Info("Succuss startup. please Ctl+C to End")
	// wait
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
