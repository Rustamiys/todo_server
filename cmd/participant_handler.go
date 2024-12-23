package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func getAllParticipantsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		rows, err := db.Query("SELECT id, name FROM participants")
		if err != nil {
			http.Error(w, "Failed to fetch participants from database", http.StatusInternalServerError)
			log.Printf("Database query error: %v", err)
			return
		}
		defer rows.Close()

		var participants []Participant
		for rows.Next() {
			var participant Participant
			if err := rows.Scan(&participant.ID, &participant.Name); err != nil {
				http.Error(w, "Failed to parse participant data", http.StatusInternalServerError)
				log.Printf("Row scan error: %v", err)
				return
			}
			participants = append(participants, participant)
		}

		if err := rows.Err(); err != nil {
			http.Error(w, "Error occurred during participants iteration", http.StatusInternalServerError)
			log.Printf("Participants iteration error: %v", err)
			return
		}

		if err := json.NewEncoder(w).Encode(participants); err != nil {
			http.Error(w, "Failed to encode participants to JSON", http.StatusInternalServerError)
			log.Printf("JSON encoding error: %v", err)
		}
	}
}

func addParticipantHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var participant Participant
		if err := json.NewDecoder(r.Body).Decode(&participant); err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		log.Printf("Database insert error: %s", participant.Name)
		_, err := db.Exec("INSERT INTO participants (name) VALUES (?)", participant.Name)
		if err != nil {
			http.Error(w, "Failed to add participant to database", http.StatusInternalServerError)
			log.Printf("Database insert error: %v", err)
			return
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "Participant with name %s added successfully", participant.Name)
	}
}

func deleteParticipantHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		var participantID int
		_, err := fmt.Sscanf(id, "%d", &participantID)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		_, err = db.Exec("DELETE FROM participants WHERE id = ?", participantID)
		if err != nil {
			http.Error(w, "Failed to delete participant from database", http.StatusInternalServerError)
			log.Printf("Database delete error: %v", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Participant with ID %d deleted successfully", participantID)
	}
}
