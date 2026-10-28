package main

import (
	"company.example/harness/internal/harness"
	"context"
	"errors"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"log"
	"os"
	"time"
)

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
	acts := &harness.SupervisorActivities{Repo: repo}
	w := worker.New(tc, "harness-supervisor", worker.Options{})
	w.RegisterWorkflow(harness.SupervisorWorkflow)
	w.RegisterActivity(acts.RunSupervisor)
	go dispatch(ctx, repo, tc)
	log.Fatal(w.Run(worker.InterruptCh()))
}
func dispatch(ctx context.Context, repo *harness.Repository, tc client.Client) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			aggregate, _, err := repo.ClaimOutbox(ctx)
			if err != nil {
				log.Printf("claim outbox: %v", err)
				continue
			}
			if aggregate == "" {
				continue
			}
			_, err = tc.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: "supervisor-" + aggregate, TaskQueue: "harness-supervisor"}, "SupervisorWorkflow", harness.SupervisorInput{TaskID: aggregate})
			var already *serviceerror.WorkflowExecutionAlreadyStarted
			if err != nil && !errors.As(err, &already) {
				log.Printf("start workflow %s: %v", aggregate, err)
				continue
			}
			if err = repo.MarkOutboxPublished(ctx, aggregate); err != nil {
				log.Printf("mark outbox %s: %v", aggregate, err)
			}
		}
	}
}
