"""Simple HTTP server that serves static files AND saves feedback to JSON."""
import json
import os
from http.server import HTTPServer, SimpleHTTPRequestHandler
from datetime import datetime

FEEDBACK_FILES = {
    "/api/feedback": "feedback.json",
    "/api/spec-feedback": "spec-feedback.json",
    "/api/plan-feedback": "plan-feedback.json",
}

class FeedbackHandler(SimpleHTTPRequestHandler):
    def _get_feedback_file(self):
        return FEEDBACK_FILES.get(self.path)

    def do_POST(self):
        fb_file = self._get_feedback_file()
        if fb_file:
            length = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(length)
            data = json.loads(body)

            feedback = {}
            if os.path.exists(fb_file):
                with open(fb_file, "r") as f:
                    feedback = json.load(f)

            section_id = data.get("section")
            feedback[section_id] = {
                "status": data.get("status", "pending"),
                "comment": data.get("comment", ""),
                "updated_at": datetime.now().isoformat(),
            }

            with open(fb_file, "w") as f:
                json.dump(feedback, f, ensure_ascii=False, indent=2)

            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Access-Control-Allow-Origin", "*")
            self.end_headers()
            self.wfile.write(json.dumps({"ok": True}).encode())
            return

        self.send_response(404)
        self.end_headers()

    def do_GET(self):
        fb_file = self._get_feedback_file()
        if fb_file:
            feedback = {}
            if os.path.exists(fb_file):
                with open(fb_file, "r") as f:
                    feedback = json.load(f)
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Access-Control-Allow-Origin", "*")
            self.end_headers()
            self.wfile.write(json.dumps(feedback, ensure_ascii=False).encode())
            return
        return super().do_GET()

    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.end_headers()

if __name__ == "__main__":
    server = HTTPServer(("0.0.0.0", 8899), FeedbackHandler)
    print("Server running at http://localhost:8899")
    server.serve_forever()
