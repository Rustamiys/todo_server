package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func getAllTasksHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		taskRows, err := db.Query("SELECT id, priority, task, start_date, end_date, status FROM tasks")
		if err != nil {
			http.Error(w, "Failed to fetch tasks from database", http.StatusInternalServerError)
			log.Printf("Database query error: %v", err)
			return
		}
		defer taskRows.Close()

		var tasks []Task
		for taskRows.Next() {
			var task Task
			if err := taskRows.Scan(&task.ID, &task.Priority, &task.Task, &task.StartDate, &task.EndDate, &task.Status); err != nil {
				http.Error(w, "Failed to parse task data", http.StatusInternalServerError)
				log.Printf("Row scan error: %v", err)
				return
			}

			participantRows, err := db.Query(`
				SELECT p.id, p.name 
				FROM participants p
				JOIN task_participants tp ON p.id = tp.participant_id
				WHERE tp.task_id = ?`, task.ID)
			if err != nil {
				http.Error(w, "Failed to fetch participants from database", http.StatusInternalServerError)
				log.Printf("Participants query error for task %d: %v", task.ID, err)
				return
			}

			var participants []Participant
			for participantRows.Next() {
				var participant Participant
				if err := participantRows.Scan(&participant.ID, &participant.Name); err != nil {
					http.Error(w, "Failed to parse participant data", http.StatusInternalServerError)
					log.Printf("Row scan error for participants: %v", err)
					return
				}
				participants = append(participants, participant)
			}
			participantRows.Close()

			task.Participants = participants
			tasks = append(tasks, task)
		}

		if err := taskRows.Err(); err != nil {
			http.Error(w, "Error occurred during tasks iteration", http.StatusInternalServerError)
			log.Printf("Tasks iteration error: %v", err)
			return
		}

		if err := json.NewEncoder(w).Encode(tasks); err != nil {
			http.Error(w, "Failed to encode tasks to JSON", http.StatusInternalServerError)
			log.Printf("JSON encoding error: %v", err)
		}
	}
}

func addTaskHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var newTask Task
		if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
			http.Error(w, "Invalid task data", http.StatusBadRequest)
			log.Printf("JSON decoding error: %v", err)
			return
		}

		query := `INSERT INTO tasks (priority, task, start_date, end_date, status) VALUES (?, ?, ?, ?, ?)`
		result, err := db.Exec(query, newTask.Priority, newTask.Task, newTask.StartDate, newTask.EndDate, newTask.Status)
		if err != nil {
			http.Error(w, "Failed to insert task into database", http.StatusInternalServerError)
			log.Printf("Database insert error: %v", err)
			return
		}

		taskID, err := result.LastInsertId()
		if err != nil {
			http.Error(w, "Failed to retrieve last insert ID", http.StatusInternalServerError)
			log.Printf("LastInsertId error: %v", err)
			return
		}

		newTask.ID = int(taskID)

		for _, participant := range newTask.Participants {
			_, err := db.Exec(`INSERT INTO task_participants (task_id, participant_id) VALUES (?, ?)`, newTask.ID, participant.ID)
			if err != nil {
				http.Error(w, "Failed to add participants to task", http.StatusInternalServerError)
				log.Printf("Insert participant error: %v", err)
				return
			}
		}

		if err := json.NewEncoder(w).Encode(newTask); err != nil {
			http.Error(w, "Failed to encode task to JSON", http.StatusInternalServerError)
			log.Printf("JSON encoding error: %v", err)
		}
	}
}

func getTaskHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		id := vars["id"]
		if id == "" {
			http.Error(w, "Task ID is required", http.StatusBadRequest)
			return
		}

		var task Task
		query := `
			SELECT t.id, t.priority, t.task, t.start_date, t.end_date, t.status
			FROM tasks t
			WHERE t.id = ?`
		err := db.QueryRow(query, id).Scan(&task.ID, &task.Priority, &task.Task, &task.StartDate, &task.EndDate, &task.Status)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Task not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("Error retrieving task: %v", err), http.StatusInternalServerError)
			}
			return
		}

		participantQuery := `
			SELECT p.id, p.name
			FROM participants p
			JOIN task_participants tp ON p.id = tp.participant_id
			WHERE tp.task_id = ?`
		rows, err := db.Query(participantQuery, id)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error retrieving participants: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var participant Participant
			if err := rows.Scan(&participant.ID, &participant.Name); err != nil {
				http.Error(w, fmt.Sprintf("Error scanning participant: %v", err), http.StatusInternalServerError)
				return
			}
			task.Participants = append(task.Participants, participant)
		}

		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf("Error reading participants: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(task)
	}
}

func deleteTaskHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		id := vars["id"]
		if id == "" {
			http.Error(w, "Task ID is required", http.StatusBadRequest)
			return
		}

		_, err := db.Exec("DELETE FROM task_participants WHERE task_id = ?", id)
		if err != nil {
			http.Error(w, "Failed to remove participants for task", http.StatusInternalServerError)
			log.Printf("Delete participants error: %v", err)
			return
		}

		query := "DELETE FROM tasks WHERE id = ?"
		result, err := db.Exec(query, id)
		if err != nil {
			http.Error(w, "Failed to delete task from database", http.StatusInternalServerError)
			log.Printf("Database delete error: %v", err)
			return
		}

		affectedRows, err := result.RowsAffected()
		if err != nil {
			http.Error(w, "Failed to retrieve affected rows", http.StatusInternalServerError)
			log.Printf("RowsAffected error: %v", err)
			return
		}

		if affectedRows == 0 {
			http.Error(w, "No task found with the given ID", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func updateTaskHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		id := vars["id"]
		if id == "" {
			http.Error(w, "Task ID is required", http.StatusBadRequest)
			return
		}

		var updatedTask Task
		if err := json.NewDecoder(r.Body).Decode(&updatedTask); err != nil {
			http.Error(w, "Invalid task data", http.StatusBadRequest)
			log.Printf("JSON decoding error: %v", err)
			return
		}

		query := `UPDATE tasks SET priority = ?, task = ?, start_date = ?, end_date = ?, status = ? WHERE id = ?`
		result, err := db.Exec(query, updatedTask.Priority, updatedTask.Task, updatedTask.StartDate, updatedTask.EndDate, updatedTask.Status, id)
		if err != nil {
			http.Error(w, "Failed to update task in database", http.StatusInternalServerError)
			log.Printf("Database update error: %v", err)
			return
		}

		affectedRows, err := result.RowsAffected()
		if err != nil {
			http.Error(w, "Failed to retrieve affected rows", http.StatusInternalServerError)
			log.Printf("RowsAffected error: %v", err)
			return
		}

		if affectedRows == 0 {
			http.Error(w, "No task found with the given ID", http.StatusNotFound)
			return
		}

		_, err = db.Exec(`DELETE FROM task_participants WHERE task_id = ?`, id)
		if err != nil {
			http.Error(w, "Failed to remove existing participants", http.StatusInternalServerError)
			log.Printf("Delete participants error: %v", err)
			return
		}

		for _, participant := range updatedTask.Participants {
			_, err := db.Exec(`INSERT INTO task_participants (task_id, participant_id) VALUES (?, ?)`, id, participant.ID)
			if err != nil {
				http.Error(w, "Failed to add participants to task", http.StatusInternalServerError)
				log.Printf("Insert participant error: %v", err)
				return
			}
		}

		if err := json.NewEncoder(w).Encode(updatedTask); err != nil {
			http.Error(w, "Failed to encode task to JSON", http.StatusInternalServerError)
			log.Printf("JSON encoding error: %v", err)
		}
	}
}
