"""A tiny stand-in for the HEMIS student API, for local development and smoke tests.

Speaks the shape the app expects:
  GET  /rest/v1/data/group-list  paged group list (needs GROUPS_TOKEN)
  GET  /uploads/<file>       files from /srv/uploads
Student OAuth (same flow, profile = the first account in accounts.json):
  GET  /student/authorize, POST /student/access-token, GET /student/user
Teacher OAuth (authorization-code flow):
  GET  /oauth/authorize      -> 302 to redirect_uri with a fixed code and the received state
                                (client_id "denied-client" -> error=access_denied instead)
  POST /oauth/access-token   -> {"access_token": ...} for the fixed code and the right client_secret
  GET  /oauth/api/user       Bearer token -> teacher_me_ok.json
Accounts come from accounts.json.
"""
import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlencode, urlparse, urlunparse

ACCOUNTS = json.load(open(os.environ.get("ACCOUNTS", "/srv/accounts.json")))
UPLOADS = os.environ.get("UPLOADS", "/srv/uploads")
GROUPS_TOKEN = os.environ.get("GROUPS_TOKEN", "")
OAUTH_SECRET = os.environ.get("OAUTH_CLIENT_SECRET", "")
OAUTH_CODE = "mock-code"
OAUTH_TOKEN = "mock-teacher-token"
STUDENT_CODE = "mock-student-code"
STUDENT_TOKEN = "mock-student-token"
TEACHER = os.environ.get("TEACHER_ME", "/srv/teacher_me_ok.json")
# 450 demo groups (so pagination with limit=200 spans three pages); id 778 matches the demo student's group
GROUPS = [{"id": 778, "name": "101-guruh"}] + [{"id": 10000 + i, "name": f"Demo guruh {i:03d}"} for i in range(1, 450)]


class Handler(BaseHTTPRequestHandler):
    def _json(self, status, body):
        data = json.dumps(body).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def _redirect(self, location):
        self.send_response(302)
        self.send_header("Location", location)
        self.send_header("Content-Length", "0")
        self.end_headers()

    def do_POST(self):
        for path, code, token in (("/oauth/access-token", OAUTH_CODE, OAUTH_TOKEN), ("/student/access-token", STUDENT_CODE, STUDENT_TOKEN)):
            if self.path != path:
                continue
            form = parse_qs(self.rfile.read(int(self.headers.get("Content-Length", 0))).decode())
            ok = (
                form.get("grant_type") == ["authorization_code"]
                and form.get("code") == [code]
                and OAUTH_SECRET != ""
                and form.get("client_secret") == [OAUTH_SECRET]
            )
            if not ok:
                return self._json(400, {"error": "invalid_grant"})
            return self._json(200, {"access_token": token, "token_type": "Bearer"})
        self._json(404, {"success": False})

    def do_GET(self):
        route = urlparse(self.path)
        if route.path in ("/oauth/authorize", "/student/authorize"):
            code = STUDENT_CODE if route.path.startswith("/student") else OAUTH_CODE
            q = parse_qs(route.query)
            redirect = (q.get("redirect_uri") or [""])[0]
            if not redirect:
                return self._json(400, {"error": "invalid_request"})
            target = urlparse(redirect)
            params = parse_qs(target.query)
            extra = {"error": "access_denied"} if q.get("client_id") == ["denied-client"] else {"code": code}
            pairs = [(k, v[0]) for k, v in params.items()] + list(extra.items())
            if "state" in q:  # echo the state back, as OAuth2 requires
                pairs.append(("state", q["state"][0]))
            return self._redirect(urlunparse(target._replace(query=urlencode(pairs))))
        if route.path == "/oauth/api/user":
            if self.headers.get("Authorization") != f"Bearer {OAUTH_TOKEN}":
                return self._json(401, {"error": "invalid_token"})
            return self._json(200, json.load(open(TEACHER)))
        if route.path == "/student/user":
            if self.headers.get("Authorization") != f"Bearer {STUDENT_TOKEN}":
                return self._json(401, {"error": "invalid_token"})
            profile = next(iter(ACCOUNTS.values()))["profile"]
            summary = {k: profile.get(k) for k in ("id", "student_id_number", "phone", "passport_pin", "passport_number")}
            # like the real HEMIS: a summary on top, the full record under "data"
            return self._json(200, {**summary, "login": profile.get("student_id_number"), "type": "student", "data": profile})
        if self.path.startswith("/rest/v1/data/group-list"):
            if not GROUPS_TOKEN or self.headers.get("Authorization") != f"Bearer {GROUPS_TOKEN}":
                return self._json(401, {"success": False, "error": "Unauthorized", "code": 401, "data": None})
            q = parse_qs(urlparse(self.path).query)
            limit = max(1, min(200, int(q.get("limit", ["200"])[0])))
            page = max(1, int(q.get("page", ["1"])[0]))
            items = GROUPS[(page - 1) * limit : page * limit]
            pages = (len(GROUPS) + limit - 1) // limit
            return self._json(200, {"success": True, "error": None, "code": 0, "data": {
                "items": items, "pagination": {"totalCount": len(GROUPS), "pageSize": limit, "pageCount": pages, "page": page}}})
        if self.path.startswith("/uploads/"):
            name = os.path.basename(self.path)
            path = os.path.join(UPLOADS, name)
            if not os.path.isfile(path):
                return self._json(404, {})
            data = open(path, "rb").read()
            self.send_response(200)
            self.send_header("Content-Type", "image/jpeg")
            self.send_header("Content-Length", str(len(data)))
            self.end_headers()
            return self.wfile.write(data)
        self._json(404, {"success": False})

    def log_message(self, fmt, *args):  # no credentials in logs: log only method and path
        print("hemis-mock", self.command, self.path.split("?")[0], flush=True)


if __name__ == "__main__":
    ThreadingHTTPServer(("0.0.0.0", 9000), Handler).serve_forever()
