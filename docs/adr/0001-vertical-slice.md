# ADR 0001: 垂直切片 MVP

先实现任务、Actor、消息和 Supervisor 的最薄闭环；Temporal、PostgreSQL、DeepSeek、OPA 和 OTel 通过契约边界接入。模型只生成结构化提案，任何副作用操作必须经过策略检查。

下一步把当前内存 store 替换为 PostgreSQL repository，接入 Temporal workflow、LiteLLM/DeepSeek adapter、Outbox publisher 和 OTel instrumentation。
