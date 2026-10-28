# Harness

企业内部 AI Harness：Go 控制面与 Temporal 负责任务状态和恢复，React + Node.js 提供简洁清晰的操作界面与本地 API。

## 技术结构

- `client/`：React 19 + Vite 前端，任务调度、Supervisor 结果、Actor 负载和任务消息。
- `server/`：Node.js 原生 HTTP API，提供本地演示数据与任务/Actor 操作。
- `cmd/control-plane/`：Go + PostgreSQL + Temporal 生产控制面。
- `cmd/workflow-worker/`：Temporal Supervisor Workflow Worker 与 PostgreSQL Outbox dispatcher。
- `runtime/python/`：Python Agent Runtime。

## 本地开发

```powershell
pnpm install
pnpm dev
```

前端 `http://localhost:5173`，Node API `http://localhost:3001`。前端 Vite 将 `/api` 与 `/healthz` 代理到 Node API。Node API 使用内存演示数据；Go 控制面仍使用 PostgreSQL 和 Temporal。

## 前端构建

```powershell
pnpm --filter harness-console exec vite build
```

## 完整 Go/Temporal 环境

```powershell
docker compose -f deploy/compose/docker-compose.yml up --build
```

Control Plane `http://localhost:8080`，Temporal UI `http://localhost:8088`，PostgreSQL `localhost:5432`。
