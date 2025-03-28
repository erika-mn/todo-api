package utils

import (
    "database/sql"
    "log"

    _ "modernc.org/sqlite" 
)

var db *sql.DB

func InitDB() {
    var err error
    db, err = sql.Open("sqlite", "./tasks.db")
    if err != nil {
        log.Fatal(err)
    }

    query := `
	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT, -- Ensure AUTOINCREMENT is used
		title TEXT NOT NULL,
		description TEXT,
		position INTEGER NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_position ON tasks(position);
	`
    _, err = db.Exec(query)
    if err != nil {
        log.Fatal(err)
    }
}

func CloseDB() {
    db.Close()
}

func GetDB() *sql.DB {
    return db
}