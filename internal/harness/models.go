package harness

import "time"

type Actor struct {
	ID           string   `json:"id"`
	TenantID     string   `json:"tenant_id"`
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	Capabilities []string `json:"capabilities"`
	Load         int      `json:"load"`
}
type Task struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	ParentTaskID    string    `json:"parent_task_id,omitempty"`
	Title           string    `json:"title"`
	Status          string    `json:"status"`
	AssigneeActorID string    `json:"assignee_actor_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	Subtasks        []string  `json:"subtasks,omitempty"`
}
type Message struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	TaskID           string    `json:"task_id"`
	SenderActorID    string    `json:"sender_actor_id,omitempty"`
	RecipientActorID string    `json:"recipient_actor_id,omitempty"`
	Type             string    `json:"type"`
	Body             string    `json:"body"`
	Sequence         int64     `json:"sequence"`
	CreatedAt        time.Time `json:"created_at"`
}
type ScoreDetail struct {
	ActorID         string `json:"actor_id"`
	CapabilityScore int    `json:"capability_score"`
	LoadScore       int    `json:"load_score"`
	Total           int    `json:"total"`
	Reason          string `json:"reason"`
}
type AssignmentProposal struct {
	ID        string        `json:"id"`
	TaskID    string        `json:"task_id"`
	ActorID   string        `json:"actor_id"`
	Scores    []ScoreDetail `json:"scores"`
	Status    string        `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
}
type SupervisorResult struct {
	Task      Task                 `json:"task"`
	Subtasks  []Task               `json:"subtasks"`
	Proposals []AssignmentProposal `json:"proposals"`
}
