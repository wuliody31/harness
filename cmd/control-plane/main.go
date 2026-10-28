package main

import (
	"company.example/harness/internal/harness"
	"context"
	"encoding/json"
	"errors"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type server struct {
	repo *harness.Repository
	tc   client.Client
}

func write(w http.ResponseWriter, c int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	_ = json.NewEncoder(w).Encode(v)
}
func (s *server) actors(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		write(w, 405, nil)
		return
	}
	var a harness.Actor
	if e := json.NewDecoder(r.Body).Decode(&a); e != nil {
		write(w, 400, map[string]string{"error": e.Error()})
		return
	}
	if a.ID == "" {
		a.ID = harness.NewID()
	}
	if a.TenantID == "" {
		a.TenantID = "default"
	}
	if e := s.repo.CreateActor(r.Context(), a); e != nil {
		write(w, 500, map[string]string{"error": e.Error()})
		return
	}
	write(w, 201, a)
}
func (s *server) tasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		write(w, 405, nil)
		return
	}
	var x struct {
		TenantID     string `json:"tenant_id"`
		Title        string `json:"title"`
		ParentTaskID string `json:"parent_task_id"`
	}
	if e := json.NewDecoder(r.Body).Decode(&x); e != nil || strings.TrimSpace(x.Title) == "" {
		write(w, 400, map[string]string{"error": "title is required"})
		return
	}
	if x.TenantID == "" {
		x.TenantID = "default"
	}
	t := harness.Task{ID: harness.NewID(), TenantID: x.TenantID, ParentTaskID: x.ParentTaskID, Title: x.Title, Status: "OPEN", CreatedAt: time.Now().UTC()}
	if e := s.repo.CreateTask(r.Context(), t); e != nil {
		write(w, 500, map[string]string{"error": e.Error()})
		return
	}
	write(w, 201, t)
}
func (s *server) messages(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		write(w, 405, nil)
		return
	}
	var m harness.Message
	if e := json.NewDecoder(r.Body).Decode(&m); e != nil || m.TaskID == "" || m.Body == "" {
		write(w, 400, map[string]string{"error": "task_id and body are required"})
		return
	}
	if m.ID == "" {
		m.ID = harness.NewID()
	}
	if m.TenantID == "" {
		m.TenantID = "default"
	}
	if m.Type == "" {
		m.Type = "COMMENT"
	}
	m.CreatedAt = time.Now().UTC()
	if e := s.repo.CreateMessage(r.Context(), m); e != nil {
		write(w, 500, map[string]string{"error": e.Error()})
		return
	}
	write(w, 201, m)
}
func (s *server) task(w http.ResponseWriter, r *http.Request) {
	p := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(p) < 3 {
		write(w, 404, nil)
		return
	}
	id := p[2]
	if len(p) == 4 && p[3] == "messages" {
		v, e := s.repo.ListMessages(r.Context(), id)
		if e != nil {
			write(w, 500, map[string]string{"error": e.Error()})
			return
		}
		write(w, 200, v)
		return
	}
	if len(p) == 4 && p[3] == "supervise" {
		if r.Method != "POST" {
			write(w, 405, nil)
			return
		}
		run, e := s.tc.ExecuteWorkflow(r.Context(), client.StartWorkflowOptions{ID: "supervisor-" + id, TaskQueue: "harness-supervisor"}, "SupervisorWorkflow", harness.SupervisorInput{TaskID: id})
		if e != nil {
			var already *serviceerror.WorkflowExecutionAlreadyStarted
			if errors.As(e, &already) {
				write(w, 202, map[string]string{"workflow_id": "supervisor-" + id, "status": "already_running"})
				return
			}
			write(w, 500, map[string]string{"error": e.Error()})
			return
		}
		write(w, 202, map[string]string{"workflow_id": run.GetID(), "run_id": run.GetRunID()})
		return
	}
	if r.Method != "GET" {
		write(w, 405, nil)
		return
	}
	v, e := s.repo.GetTask(r.Context(), id)
	if errors.Is(e, context.Canceled) {
		return
	}
	if e != nil {
		write(w, 404, map[string]string{"error": "task not found"})
		return
	}
	write(w, 200, v)
}
func main() {
	ctx := context.Background()
	db := os.Getenv("DATABASE_URL")
	if db == "" {
		db = "postgres://harness:harness@localhost:5432/harness?sslmode=disable"
	}
	repo, e := harness.OpenRepository(ctx, db)
	if e != nil {
		log.Fatal(e)
	}
	defer repo.Close()
	addr := os.Getenv("TEMPORAL_ADDRESS")
	if addr == "" {
		addr = "localhost:7233"
	}
	tc, e := client.Dial(client.Options{HostPort: addr})
	if e != nil {
		log.Fatal(e)
	}
	defer tc.Close()
	s := &server{repo: repo, tc: tc}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { write(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("/v1/actors", s.actors)
	mux.HandleFunc("/v1/tasks", s.tasks)
	mux.HandleFunc("/v1/tasks/", s.task)
	mux.HandleFunc("/v1/messages", s.messages)
	log.Fatal(http.ListenAndServe(":8080", mux))
}
