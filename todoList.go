package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/caarlos0/env/v11"
	_ "github.com/mattn/go-sqlite3"
)

const maxBodyBytes = 1 << 20 // 1 MiB

var db *sql.DB

var (
	errTaskNotFound     = errors.New("no task found with the given ID")
	errInvalidCompleted = errors.New("invalid completed value")
)

type Task struct {
	ID          int
	Title       string
	Description string
	Completed   bool
	Date        string
}

type Config struct {
	Port   string `env:"PORT" envDefault:"8080"`
	DBName string `env:"DB_NAME" envDefault:"todoList.db"`
}

func main() {
	//CONFIG STUFF
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("%+v", err)
	}

	//DB STUFF
	var err error
	db, err = sql.Open("sqlite3", fmt.Sprintf("./%s", cfg.DBName))
	if err != nil {
		fmt.Println(err)
		return
	}

	err = initDB()
	if err != nil {
		fmt.Println(err)
		return
	}

	defer db.Close()
	fmt.Println("Connected to the SQLite database successfully.")

	//HTTP STUFF
	if cfg.Port != "0" {
		fmt.Printf("Server started on port %s\n", cfg.Port)
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), newServer()))
	} else {
		fmt.Println("Server is disabled because PORT is set to 0")
	}
}

//DB STUFF

func initDB() error {
	// Create the tasks table if it doesn't exist
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		description TEXT,
		completed BOOLEAN NOT NULL DEFAULT 0,
		date TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
	);
	`
	_, err := db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create tasks table: %v", err)
	}

	return nil
}

func addTask(title string, description string) error {
	insertQuery := `
	INSERT INTO tasks (title, description, completed)
	VALUES (?, ?, 0);
	`
	_, err := db.Exec(insertQuery, title, description)
	if err != nil {
		return fmt.Errorf("failed to add task: %v", err)
	}

	log.Printf("Task added: %s - %s", title, description)

	return nil
}

func getTasks() ([]Task, error) {
	selectQuery := `
	SELECT id, title, description, completed, date
	FROM tasks
	`
	rows, err := db.Query(selectQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve tasks: %v", err)
	}
	defer rows.Close()

	tasks := []Task{}
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Completed, &task.Date); err != nil {
			return nil, fmt.Errorf("failed to scan task: %v", err)
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred during row iteration: %v", err)
	}

	return tasks, nil
}

func getTask(id string) (*Task, error) {
	selectQuery := `
	SELECT id, title, description, completed, date
	FROM tasks
	WHERE id = ?
	`
	rows, err := db.Query(selectQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve task: %v", err)
	}
	defer rows.Close()

	var task Task
	found := false
	for rows.Next() {
		if err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Completed, &task.Date); err != nil {
			return nil, fmt.Errorf("failed to scan task: %v", err)
		}
		found = true
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred during row iteration: %v", err)
	}
	if !found {
		return nil, errTaskNotFound
	}

	return &task, nil
}

func deleteTask(id int) error {
	deleteQuery := `
	DELETE FROM tasks
	WHERE id = ?;
	`
	res, err := db.Exec(deleteQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %v", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to delete task: %v", err)
	}

	if rowsAffected == 0 {
		return errTaskNotFound
	}

	log.Printf("Task deleted: ID %d", id)
	return nil
}

func completeTask(id string, completed string) error {
	var value bool
	switch strings.ToLower(strings.TrimSpace(completed)) {
	case "":
		// No explicit value: toggle the current state.
		task, err := getTask(id)
		if err != nil {
			return err
		}
		value = !task.Completed
	case "1", "true":
		value = true
	case "0", "false":
		value = false
	default:
		return fmt.Errorf("%w: %q", errInvalidCompleted, completed)
	}

	updateQuery := `
	UPDATE tasks
	SET completed = ?
	WHERE id = ?;
	`
	res, err := db.Exec(updateQuery, value, id)
	if err != nil {
		return fmt.Errorf("failed to update task: %v", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to update task: %v", err)
	}

	if rowsAffected == 0 {
		return errTaskNotFound
	}

	log.Printf("Task updated: ID %s, Completed %t", id, value)
	return nil
}

//HTTP STUFF

const (
	mediaJSON    = "application/json"
	mediaPlain   = "text/plain"
	contentJSON  = "application/json; charset=utf-8"
	contentPlain = "text/plain; charset=utf-8"
)

func newServer() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /readyz", readyHandler)
	mux.HandleFunc("GET /tasks", getTasksHandler)
	mux.HandleFunc("POST /tasks", addTaskHandler)
	mux.HandleFunc("PUT /tasks", updateTaskHandler)
	mux.HandleFunc("DELETE /tasks", deleteTaskHandler)
	mux.HandleFunc("GET /", htmlHandler)

	return methodGuard(mux)
}

var knownMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodPost:    true,
	http.MethodPut:     true,
	http.MethodPatch:   true,
	http.MethodDelete:  true,
	http.MethodOptions: true,
	http.MethodConnect: true,
	http.MethodTrace:   true,
}

func methodGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !knownMethods[r.Method] {
			http.Error(w, "Method not implemented", http.StatusNotImplemented)
			return
		}
		if r.Method == http.MethodOptions {
			w.Header().Set("Allow", allowedMethods(r.URL.Path))
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowedMethods(path string) string {
	if path == "/tasks" {
		return "GET, HEAD, POST, PUT, DELETE, OPTIONS"
	}
	return "GET, HEAD, OPTIONS"
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", contentJSON)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func negotiate(r *http.Request) string {
	accept := r.Header.Get("Accept")
	if strings.TrimSpace(accept) == "" {
		return contentPlain
	}

	jq, js, jok := bestQuality(accept, mediaJSON)
	tq, ts, tok := bestQuality(accept, mediaPlain)

	switch {
	case jok && jq > 0 && (!tok || tq <= 0 || jq > tq || (jq == tq && js > ts)):
		return contentJSON
	case tok && tq > 0:
		return contentPlain
	default:
		return ""
	}
}

func bestQuality(accept, candidate string) (q float64, specificity int, ok bool) {
	for _, raw := range strings.Split(accept, ",") {
		media, params, err := mime.ParseMediaType(strings.TrimSpace(raw))
		if err != nil {
			continue
		}

		s := matchSpecificity(media, candidate)
		if s < 0 {
			continue
		}

		qv := 1.0
		if qs, hasQ := params["q"]; hasQ {
			parsed, err := strconv.ParseFloat(qs, 64)
			if err != nil {
				continue
			}
			qv = parsed
		}

		if !ok || s > specificity || (s == specificity && qv > q) {
			q, specificity, ok = qv, s, true
		}
	}

	return q, specificity, ok
}

func matchSpecificity(pattern, candidate string) int {
	if pattern == "*/*" {
		return 0
	}

	pType, pSub, ok := strings.Cut(pattern, "/")
	if !ok {
		return -1
	}
	cType, cSub, _ := strings.Cut(candidate, "/")

	if pType != cType {
		return -1
	}
	if pSub == "*" {
		return 1
	}
	if pSub == cSub {
		return 2
	}
	return -1
}

func htmlHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "assets/index.html")
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", contentPlain)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

func getTasksHandler(w http.ResponseWriter, r *http.Request) {
	tasks, err := getTasks()
	if err != nil {
		log.Printf("error retrieving tasks: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to retrieve tasks")
		return
	}

	w.Header().Add("Vary", "Accept")

	switch negotiate(r) {
	case contentJSON:
		writeJSON(w, http.StatusOK, tasks)
	case contentPlain:
		w.Header().Set("Content-Type", contentPlain)
		for _, task := range tasks {
			fmt.Fprintf(w, "ID: %d, Title: %s, Description: %s, Completed: %t, Date: %s\n",
				task.ID, task.Title, task.Description, task.Completed, task.Date)
		}
	default:
		http.Error(w, "No acceptable representation", http.StatusNotAcceptable)
	}
}

func addTaskHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := r.ParseMultipartForm(maxBodyBytes); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "Request body too large")
			return
		}
		writeError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")

	if strings.TrimSpace(title) == "" {
		writeError(w, http.StatusBadRequest, "Title is required")
		return
	}

	if err := addTask(title, description); err != nil {
		log.Printf("failed to add task: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to add task")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "Task added successfully"})
}

func updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	id := r.FormValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Task ID is required")
		return
	}
	if _, err := strconv.Atoi(id); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	completed := r.FormValue("completed")

	switch err := completeTask(id, completed); {
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]string{"message": "Task updated successfully"})
	case errors.Is(err, errTaskNotFound):
		writeError(w, http.StatusNotFound, "Task not found")
	case errors.Is(err, errInvalidCompleted):
		writeError(w, http.StatusBadRequest, "Invalid completed value")
	default:
		log.Printf("failed to update task: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to update task")
	}
}

func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	rawID := r.URL.Query().Get("id")
	if rawID == "" {
		writeError(w, http.StatusBadRequest, "Task ID is required")
		return
	}

	taskID, err := strconv.Atoi(rawID)
	if err != nil || taskID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	if err := deleteTask(taskID); err != nil {
		if errors.Is(err, errTaskNotFound) {
			writeError(w, http.StatusNotFound, "Task not found")
		} else {
			log.Printf("failed to delete task: %v", err)
			writeError(w, http.StatusInternalServerError, "Failed to delete task")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Task deleted successfully"})
}
