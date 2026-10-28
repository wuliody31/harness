package harness

import (
	"context"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"time"
)

type SupervisorInput struct{ TaskID string }
type SupervisorActivities struct{ Repo *Repository }

func SupervisorWorkflow(ctx workflow.Context, in SupervisorInput) (SupervisorResult, error) {
	opts := workflow.ActivityOptions{StartToCloseTimeout: 2 * time.Minute, RetryPolicy: &temporal.RetryPolicy{InitialInterval: time.Second, BackoffCoefficient: 2, MaximumInterval: time.Minute, MaximumAttempts: 5}}
	ctx = workflow.WithActivityOptions(ctx, opts)
	var out SupervisorResult
	err := workflow.ExecuteActivity(ctx, "RunSupervisor", in).Get(ctx, &out)
	return out, err
}
func (a *SupervisorActivities) RunSupervisor(ctx context.Context, in SupervisorInput) (SupervisorResult, error) {
	return a.Repo.Supervise(ctx, in.TaskID)
}
