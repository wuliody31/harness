import http from 'node:http';
import { randomUUID } from 'node:crypto';

const port = Number(process.env.PORT || 3001);
const actors = new Map();
const tasks = new Map();
const messages = new Map();
const now = () => new Date().toISOString();
function seed() {
  const a1 = { id: randomUUID(), tenant_id: 'default', name: 'Supervisor', kind: 'AGENT', capabilities: ['planning', 'routing'], load: 1 };
  const a2 = { id: randomUUID(), tenant_id: 'default', name: 'DeepSeek Worker 01', kind: 'AGENT', capabilities: ['analysis', 'code'], load: 2 };
  const a3 = { id: randomUUID(), tenant_id: 'default', name: 'DeepSeek Worker 02', kind: 'AGENT', capabilities: ['tests', 'docs'], load: 1 };
  [a1, a2, a3].forEach(a => actors.set(a.id, a));
  const task = { id: randomUUID(), tenant_id: 'default', title: '支付服务错误处理改造', status: 'IN_PROGRESS', assignee_actor_id: a1.id, created_at: now(), subtasks: [
    { id: randomUUID(), title: '分析异常路径', status: 'DONE', assignee_actor_id: a2.id },
    { id: randomUUID(), title: '补充回归测试', status: 'IN_PROGRESS', assignee_actor_id: a3.id }
  ] };
  tasks.set(task.id, task);
  messages.set(task.id, [{ sender: 'Supervisor', body: '已将补充回归测试分配给 DeepSeek Worker 02。' }, { sender: 'DeepSeek Worker 02', body: '测试环境已就绪，等待分析产物。' }]);
}
seed();
function send(res, status, value) { res.writeHead(status, { 'content-type': 'application/json', 'access-control-allow-origin': '*', 'access-control-allow-headers': 'content-type' }); res.end(JSON.stringify(value)); }
async function readBody(req) { let raw = ''; for await (const chunk of req) raw += chunk; return raw ? JSON.parse(raw) : {}; }
const server = http.createServer(async (req, res) => {
  if (req.method === 'OPTIONS') { res.writeHead(204, { 'access-control-allow-origin': '*', 'access-control-allow-methods': 'GET,POST,OPTIONS', 'access-control-allow-headers': 'content-type' }); return res.end(); }
  const path = new URL(req.url, 'http://localhost').pathname;
  try {
    if (path === '/healthz') return send(res, 200, { status: 'ok', service: 'node-api' });
    if (path === '/api/actors' && req.method === 'GET') return send(res, 200, [...actors.values()]);
    if (path === '/api/actors' && req.method === 'POST') { const input = await readBody(req); const actor = { id: randomUUID(), tenant_id: input.tenant_id || 'default', name: input.name, kind: input.kind || 'AGENT', capabilities: input.capabilities || [], load: 0 }; actors.set(actor.id, actor); return send(res, 201, actor); }
    if (path === '/api/tasks' && req.method === 'GET') return send(res, 200, [...tasks.values()]);
    if (path === '/api/tasks' && req.method === 'POST') { const input = await readBody(req); const task = { id: randomUUID(), tenant_id: input.tenant_id || 'default', title: input.title, status: 'OPEN', assignee_actor_id: [...actors.keys()][0] || '', created_at: now(), subtasks: [] }; tasks.set(task.id, task); messages.set(task.id, []); return send(res, 201, task); }
    const match = path.match(/^\/api\/tasks\/([^/]+)(?:\/(messages|supervise))?$/);
    if (match) { const task = tasks.get(match[1]); if (!task) return send(res, 404, { error: 'task not found' }); if (match[2] === 'messages' && req.method === 'GET') return send(res, 200, messages.get(task.id) || []); if (match[2] === 'supervise' && req.method === 'POST') { task.status = 'IN_PROGRESS'; if (!task.subtasks.length) task.subtasks = [{ id: randomUUID(), title: '分析异常路径', status: 'OPEN', assignee_actor_id: [...actors.keys()][1] || '' }, { id: randomUUID(), title: '补充回归测试', status: 'OPEN', assignee_actor_id: [...actors.keys()][2] || '' }]; return send(res, 202, { workflow_id: 'node-supervisor-' + task.id, task }); } if (req.method === 'GET') return send(res, 200, task); }
    send(res, 404, { error: 'not found' });
  } catch (error) { send(res, 400, { error: error.message }); }
});
server.listen(port, () => console.log('Harness Node API listening on http://localhost:' + port));
