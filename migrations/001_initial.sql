CREATE TABLE IF NOT EXISTS actors (id uuid PRIMARY KEY, tenant_id text NOT NULL, name text NOT NULL, kind text NOT NULL, capabilities jsonb NOT NULL DEFAULT '[]', created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS tasks (id uuid PRIMARY KEY, tenant_id text NOT NULL, parent_task_id uuid NULL, title text NOT NULL, status text NOT NULL, assignee_actor_id uuid NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS messages (id uuid PRIMARY KEY, tenant_id text NOT NULL, task_id uuid NOT NULL, sender_actor_id uuid NULL, recipient_actor_id uuid NULL, type text NOT NULL, body text NOT NULL, sequence bigint NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS outbox_events (id uuid PRIMARY KEY, tenant_id text NOT NULL, aggregate_id uuid NOT NULL, event_type text NOT NULL, payload jsonb NOT NULL, published_at timestamptz NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS assignment_proposals (id uuid PRIMARY KEY, task_id uuid NOT NULL, actor_id uuid NULL, scores jsonb NOT NULL, status text NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
ALTER TABLE outbox_events ADD COLUMN IF NOT EXISTS claimed_at timestamptz NULL;
CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox_events (created_at) WHERE published_at IS NULL;
