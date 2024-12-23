package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

const (
	dbPath      = "users.db"
	jwtSecret   = "mysecretkey"
	port        = "8080"
	baseURLAuth = "/api/users"
	baseURLTask = "/api/task"
)

type User struct {
	ID       int    `json:"id"`
	Login    string `json:"login"`
	Name     string `json:"name"`
	Surename string `json:"surename"`
	Password string `json:"password,omitempty"`
}

type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

type Task struct {
	ID           int           `json:"id"`
	Priority     string        `json:"priority"`
	Task         string        `json:"task"`
	StartDate    string        `json:"startDate"`
	EndDate      string        `json:"endDate"`
	Status       string        `json:"status"`
	Participants []Participant `json:"participants"`
}

type Participant struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to SQLite: %v", err)
	}
	defer db.Close()

	initDB(db)

	authMux := http.NewServeMux()

	taskMux := mux.NewRouter()
	taskMux.Use(corsMiddleware)

	taskMux.HandleFunc(baseURLAuth+"/register", registerHandler(db))
	taskMux.HandleFunc(baseURLAuth+"/login", loginHandler(db))
	taskMux.HandleFunc(baseURLAuth+"/auth", authHandler())

	taskMux.HandleFunc(baseURLTask, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getAllTasksHandler(db)(w, r)
		case http.MethodPost:
			addTaskHandler(db)(w, r)
		case http.MethodPut:
			updateTaskHandler(db)(w, r)
		case http.MethodDelete:
			deleteTaskHandler(db)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	taskMux.HandleFunc("/api/task/{id}", deleteTaskHandler(db)).Methods("DELETE")
	taskMux.HandleFunc("/api/task/{id}", getTaskHandler(db)).Methods("GET")
	taskMux.HandleFunc("/api/task/{id}", updateTaskHandler(db)).Methods("PUT")

	taskMux.HandleFunc("/api/participants", getAllParticipantsHandler(db)).Methods("GET")
	taskMux.HandleFunc("/api/participants", addParticipantHandler(db)).Methods("POST")
	taskMux.HandleFunc("/api/participants/{id}", deleteParticipantHandler(db)).Methods("DELETE")

	if err := http.ListenAndServe(":8080", corsMiddleware(taskMux)); err != nil {
		fmt.Println("Ошибка при запуске сервера:", err)
	}

	http.Handle(baseURLAuth, corsMiddleware(authMux))
	http.Handle(baseURLTask, corsMiddleware(taskMux))

	fmt.Println("Server running at http://localhost:8080")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
