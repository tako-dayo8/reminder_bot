package database

import (
	"database/sql"
	"time"

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
	done		INTEGER NOT NULL DEFAULT FALSE,
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

type NewSchedule struct {
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

func OpenSQL(path string) (*sql.DB, error) {
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"

	// func name is "open" but not connection to dsn(file path?) check only
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// defer db.Close()
	db.SetMaxOpenConns(1) // Avoidance conflict

	// verify a connection for the db
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// initialize sqlite
func InitSQL(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}

func CreateSchedule(db *sql.DB, schedule NewSchedule) (int64, error) {
	now := time.Now().Unix()

	const SQL = "INSERT INTO schedules (title, description, channel_id, user_id ,remind_at, interval, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	res, err := db.Exec(SQL, schedule.title, schedule.description, schedule.channelID, schedule.userID, schedule.remindAt, schedule.interval, now, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
