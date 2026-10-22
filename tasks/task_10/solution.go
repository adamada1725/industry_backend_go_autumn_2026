package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List(done bool) []Task
	SetDone(id string, done bool) (Task, error)
}

type Clock interface {
	Now() time.Time
}

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("invalid title")
)

var globalTaskID atomic.Uint64

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	tasks map[string]Task
}

func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	return &inMemoryTaskRepo{
		clock: clock,
		tasks: make(map[string]Task),
	}
}

func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrInvalidTitle
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	task := Task{
		ID:        strconv.FormatUint(globalTaskID.Add(1), 10),
		Title:     title,
		Done:      false,
		UpdatedAt: r.clock.Now(),
	}

	r.tasks[task.ID] = task
	return task, nil
}

func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, ok := r.tasks[id]
	return task, ok
}

func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]Task, 0)

	for _, task := range r.tasks {
		if task.Done == done {
			tasks = append(tasks, task)
		}
	}

	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].UpdatedAt.Equal(tasks[j].UpdatedAt) {
			return tasks[i].ID < tasks[j].ID
		}

		return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
	})

	return tasks
}

func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}

	if task.Done != done {
		task.Done = done
		task.UpdatedAt = r.clock.Now()
		r.tasks[id] = task
	}

	return task, nil
}

// Читает ровно один JSON-объект.
// Проверка структуры и обязательных полей выполняется вызывающим кодом.
func decodeObject(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("extra JSON value")
		}
		return err
	}

	return nil
}

func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func NewHTTPHandler(repo TaskRepo) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			values, exists := r.URL.Query()["done"]
			done := false

			if exists {
				if len(values) != 1 ||
					(values[0] != "true" && values[0] != "false") {
					http.Error(w, "invalid done", http.StatusBadRequest)
					return
				}

				done = values[0] == "true"
			}

			jsonResponse(w, http.StatusOK, repo.List(done))

		case http.MethodPost:
			var req struct {
				Title *string `json:"title"`
			}

			if err := decodeObject(r, &req); err != nil || req.Title == nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}

			task, err := repo.Create(*req.Title)
			if err != nil {
				if errors.Is(err, ErrInvalidTitle) {
					http.Error(w, err.Error(), http.StatusBadRequest)
				} else {
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
				return
			}

			jsonResponse(w, http.StatusCreated, task)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/tasks/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/tasks/")

		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			task, ok := repo.Get(id)
			if !ok {
				http.NotFound(w, r)
				return
			}

			jsonResponse(w, http.StatusOK, task)

		case http.MethodPatch:
			var req struct {
				Done *bool `json:"done"`
			}

			if err := decodeObject(r, &req); err != nil || req.Done == nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}

			task, err := repo.SetDone(id, *req.Done)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					http.NotFound(w, r)
				} else {
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
				return
			}

			jsonResponse(w, http.StatusOK, task)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux
}
