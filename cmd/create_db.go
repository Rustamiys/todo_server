package main

import (
	"database/sql"
	"log"
)

func initDB(db *sql.DB) {
	queryUsers := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		login TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		surename TEXT NOT NULL,
		password TEXT NOT NULL
	);`

	queryTasks := `
	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		priority TEXT NOT NULL,
		task TEXT NOT NULL,
		start_date DATETIME NOT NULL,
		end_date DATETIME NOT NULL,
		status TEXT NOT NULL
	);`

	queryParticipants := `
	CREATE TABLE IF NOT EXISTS participants (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);`

	queryTaskParticipants := `
	CREATE TABLE IF NOT EXISTS task_participants (
		task_id INTEGER NOT NULL,
		participant_id INTEGER NOT NULL,
		PRIMARY KEY (task_id, participant_id),
		FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
		FOREIGN KEY (participant_id) REFERENCES participants(id) ON DELETE CASCADE
	);`

	if _, err := db.Exec(queryUsers); err != nil {
		log.Fatalf("Failed to create 'users' table: %v", err)
	}

	if _, err := db.Exec(queryTasks); err != nil {
		log.Fatalf("Failed to create 'tasks' table: %v", err)
	}

	if _, err := db.Exec(queryParticipants); err != nil {
		log.Fatalf("Failed to create 'participants' table: %v", err)
	}

	if _, err := db.Exec(queryTaskParticipants); err != nil {
		log.Fatalf("Failed to create 'task_participants' table: %v", err)
	}
}
