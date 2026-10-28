package harness

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	_ "github.com/jackc/pgx/v5/stdlib"
	"time"
)

type Repository struct{ DB *sql.DB }

func OpenRepository(ctx context.Context, url string) (*Repository, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return &Repository{DB: db}, nil
}
func (r *Repository) Close() error { return r.DB.Close() }
func (r *Repository) CreateActor(ctx context.Context, a Actor) error {
	caps, _ := json.Marshal(a.Capabilities)
	_, err := r.DB.ExecContext(ctx, "INSERT INTO actors(id,tenant_id,name,kind,capabilities) VALUES($1,$2,$3,$4,$5)", a.ID, a.TenantID, a.Name, a.Kind, caps)
	return err
}
func (r *Repository) CreateTask(ctx context.Context, t Task) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "INSERT INTO tasks(id,tenant_id,parent_task_id,title,status,assignee_actor_id,created_at) VALUES($1,$2,NULLIF($3,''),$4,$5,NULLIF($6,''),$7)", t.ID, t.TenantID, t.ParentTaskID, t.Title, t.Status, t.AssigneeActorID, t.CreatedAt); err != nil {
		return err
	}
	p, _ := json.Marshal(t)
	if _, err = tx.ExecContext(ctx, "INSERT INTO outbox_events(id,tenant_id,aggregate_id,event_type,payload) VALUES($1,$2,$3,'task.created',$4)", NewID(), t.TenantID, t.ID, p); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) GetTask(ctx context.Context, id string) (Task, error) {
	var t Task
	err := r.DB.QueryRowContext(ctx, "SELECT id,tenant_id,COALESCE(parent_task_id::text,''),title,status,COALESCE(assignee_actor_id::text,''),created_at FROM tasks WHERE id=$1", id).Scan(&t.ID, &t.TenantID, &t.ParentTaskID, &t.Title, &t.Status, &t.AssigneeActorID, &t.CreatedAt)
	if err != nil {
		return t, err
	}
	rows, err := r.DB.QueryContext(ctx, "SELECT id FROM tasks WHERE parent_task_id=$1 ORDER BY created_at", id)
	if err != nil {
		return t, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid string
		_ = rows.Scan(&cid)
		t.Subtasks = append(t.Subtasks, cid)
	}
	return t, rows.Err()
}
func (r *Repository) ListMessages(ctx context.Context, id string) ([]Message, error) {
	rows, err := r.DB.QueryContext(ctx, "SELECT id,tenant_id,task_id,COALESCE(sender_actor_id::text,''),COALESCE(recipient_actor_id::text,''),type,body,sequence,created_at FROM messages WHERE task_id=$1 ORDER BY sequence", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		if err = rows.Scan(&m.ID, &m.TenantID, &m.TaskID, &m.SenderActorID, &m.RecipientActorID, &m.Type, &m.Body, &m.Sequence, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (r *Repository) CreateMessage(ctx context.Context, m Message) error {
	var seq int64
	if err := r.DB.QueryRowContext(ctx, "SELECT COALESCE(MAX(sequence),0)+1 FROM messages WHERE task_id=$1", m.TaskID).Scan(&seq); err != nil {
		return err
	}
	_, err := r.DB.ExecContext(ctx, "INSERT INTO messages(id,tenant_id,task_id,sender_actor_id,recipient_actor_id,type,body,sequence,created_at) VALUES($1,$2,$3,NULLIF($4,''),NULLIF($5,''),$6,$7,$8,$9)", m.ID, m.TenantID, m.TaskID, m.SenderActorID, m.RecipientActorID, m.Type, m.Body, seq, m.CreatedAt)
	return err
}
func (r *Repository) ListActors(ctx context.Context, tenant string) ([]Actor, error) {
	rows, err := r.DB.QueryContext(ctx, "SELECT id,tenant_id,name,kind,capabilities,load FROM actors WHERE tenant_id=$1 ORDER BY id", tenant)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Actor{}
	for rows.Next() {
		var a Actor
		var caps []byte
		if err = rows.Scan(&a.ID, &a.TenantID, &a.Name, &a.Kind, &caps, &a.Load); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(caps, &a.Capabilities)
		out = append(out, a)
	}
	return out, rows.Err()
}
func (r *Repository) Supervise(ctx context.Context, id string) (SupervisorResult, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return SupervisorResult{}, err
	}
	defer tx.Rollback()
	var p Task
	err = tx.QueryRowContext(ctx, "SELECT id,tenant_id,COALESCE(parent_task_id::text,''),title,status,COALESCE(assignee_actor_id::text,''),created_at FROM tasks WHERE id=$1 FOR UPDATE", id).Scan(&p.ID, &p.TenantID, &p.ParentTaskID, &p.Title, &p.Status, &p.AssigneeActorID, &p.CreatedAt)
	if err != nil {
		return SupervisorResult{}, err
	}
	var count int
	_ = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks WHERE parent_task_id=$1", id).Scan(&count)
	if count == 0 {
		actors, err := r.ListActors(ctx, p.TenantID)
		if err != nil {
			return SupervisorResult{}, err
		}
		for _, suffix := range []string{"analysis", "execution"} {
			child := Task{ID: NewID(), TenantID: p.TenantID, ParentTaskID: p.ID, Title: p.Title + " (" + suffix + ")", Status: "OPEN", CreatedAt: time.Now().UTC()}
			scores := []ScoreDetail{}
			best := ScoreDetail{Total: -1}
			for _, a := range actors {
				capScore := 35
				if len(a.Capabilities) == 0 {
					capScore = 10
				}
				loadScore := 25 - a.Load
				if loadScore < 0 {
					loadScore = 0
				}
				v := ScoreDetail{ActorID: a.ID, CapabilityScore: capScore, LoadScore: loadScore, Total: capScore + loadScore, Reason: "deterministic score"}
				scores = append(scores, v)
				if v.Total > best.Total {
					best = v
				}
			}
			child.AssigneeActorID = best.ActorID
			if child.AssigneeActorID != "" {
				if _, err = tx.ExecContext(ctx, "UPDATE actors SET load=load+1 WHERE id=$1", child.AssigneeActorID); err != nil {
					return SupervisorResult{}, err
				}
			}
			if _, err = tx.ExecContext(ctx, "INSERT INTO tasks(id,tenant_id,parent_task_id,title,status,assignee_actor_id,created_at) VALUES($1,$2,$3,$4,$5,NULLIF($6,''),$7)", child.ID, child.TenantID, child.ParentTaskID, child.Title, child.Status, child.AssigneeActorID, child.CreatedAt); err != nil {
				return SupervisorResult{}, err
			}
			raw, _ := json.Marshal(scores)
			if _, err = tx.ExecContext(ctx, "INSERT INTO assignment_proposals(id,task_id,actor_id,scores,status,created_at) VALUES($1,$2,NULLIF($3,''),$4,'AUTO_APPROVED',$5)", NewID(), child.ID, child.AssigneeActorID, raw, time.Now().UTC()); err != nil {
				return SupervisorResult{}, err
			}
		}
		_, err = tx.ExecContext(ctx, "UPDATE tasks SET status='IN_PROGRESS' WHERE id=$1", id)
		if err != nil {
			return SupervisorResult{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return SupervisorResult{}, err
	}
	return r.SupervisorResult(ctx, id)
}
func (r *Repository) SupervisorResult(ctx context.Context, id string) (SupervisorResult, error) {
	p, err := r.GetTask(ctx, id)
	if err != nil {
		return SupervisorResult{}, err
	}
	out := SupervisorResult{Task: p, Subtasks: []Task{}, Proposals: []AssignmentProposal{}}
	for _, cid := range p.Subtasks {
		t, err := r.GetTask(ctx, cid)
		if err != nil {
			return out, err
		}
		out.Subtasks = append(out.Subtasks, t)
		rows, err := r.DB.QueryContext(ctx, "SELECT id,task_id,COALESCE(actor_id::text,''),scores,status,created_at FROM assignment_proposals WHERE task_id=$1", cid)
		if err != nil {
			return out, err
		}
		for rows.Next() {
			var x AssignmentProposal
			var raw []byte
			if err = rows.Scan(&x.ID, &x.TaskID, &x.ActorID, &raw, &x.Status, &x.CreatedAt); err != nil {
				rows.Close()
				return out, err
			}
			_ = json.Unmarshal(raw, &x.Scores)
			out.Proposals = append(out.Proposals, x)
		}
		rows.Close()
	}
	return out, nil
}
func (r *Repository) ClaimOutbox(ctx context.Context) (string, []byte, error) {
	var id string
	var payload []byte
	err := r.DB.QueryRowContext(ctx, "UPDATE outbox_events SET claimed_at=now() WHERE id=(SELECT id FROM outbox_events WHERE published_at IS NULL AND (claimed_at IS NULL OR claimed_at < now()-$1::interval) ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1) RETURNING aggregate_id,payload", "1 minute").Scan(&id, &payload)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, nil
	}
	return id, payload, err
}
func (r *Repository) MarkOutboxPublished(ctx context.Context, aggregate string) error {
	_, err := r.DB.ExecContext(ctx, "UPDATE outbox_events SET published_at=now(),claimed_at=NULL WHERE aggregate_id=$1 AND published_at IS NULL", aggregate)
	return err
}
