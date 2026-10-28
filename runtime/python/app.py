import json
import os
from http.server import BaseHTTPRequestHandler, HTTPServer

class Handler(BaseHTTPRequestHandler):
    def send_json(self, status, value):
        body = json.dumps(value).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)
    def do_GET(self):
        if self.path == "/healthz": self.send_json(200, {"status": "ok", "runtime": "python"})
        else: self.send_json(404, {"error": "not found"})
    def do_POST(self):
        if self.path != "/v1/plan": self.send_json(404, {"error": "not found"}); return
        size = int(self.headers.get("Content-Length", 0)); payload = json.loads(self.rfile.read(size) or b"{}")
        self.send_json(200, {"schema_version": 1, "proposal": "DEFERRED", "task_id": payload.get("task_id"), "reason": "DeepSeek adapter boundary is ready; no provider call in local slice."})
    def log_message(self, *_): pass

if __name__ == "__main__": HTTPServer((os.getenv("HOST", "0.0.0.0"), int(os.getenv("PORT", "8090"))), Handler).serve_forever()
