# TASKS.md

Phases are done in order. Mark each task `[x]` when finished. Rules and model details are in `CLAUDE.md`.

---

## Phase 0. Project skeleton

- [x] **0.1** Go module, directory layout (per `CLAUDE.md`), `.env.example`, `.gitignore`, `.dockerignore` (exclude `node_modules`, `web/dist`, `.venv`, `data/`).
- [x] **0.2** `internal/config`: read and validate env, load `*time.Location` from `TZ_NAME`.
- [x] **0.3** zap logger (dev/prod). fiber app with request-id, recover, access log and a JSON error handler (error format from `CLAUDE.md`).
- [x] **0.4** `sqlc.yaml` (pgx/v5, `emit_json_tags`, uuid → `github.com/google/uuid`).
- [x] **0.5** Vue 3 + Vite + TS project in `web/`, created and managed with **bun**: Tailwind + shadcn-vue init, vue-router, Pinia, vue-query, vee-validate + zod, ESLint, Vitest. Commit `bun.lock`.
- [x] **0.6** `web/embed.go` (`//go:embed all:dist`), SPA fallback, `/api/health` endpoint (also pings the DB and the face-service).
- [x] **0.7** `cmd/app` subcommands: `serve`, `migrate up|down|status` (embedded goose).
- [x] **0.8** face-service skeleton managed with **uv**: `uv init`, FastAPI + uvicorn + pytest added via `uv add`, only `GET /health` for now (DeepFace is added in phase 5). Commit `uv.lock`.
- [x] **0.9** Dockerfiles: root multi-stage `Dockerfile` (bun → go → slim runtime) and `face-service/Dockerfile` (uv), per `CLAUDE.md` → Dockerfiles. Use layer caching for `bun install` and `uv sync`.
- [x] **0.10** `compose.yaml` with `postgres`, `face-service`, `app`, `tools` and the `test` profile services (`go-test`, `face-test`, `web-test`), per `CLAUDE.md` → compose.yaml requirements. Postgres init script creates the `app_test` database.
- [x] **0.11** `Makefile` with the compose-based targets from `CLAUDE.md` → Commands.

**Done when:** `make up` brings all three services up healthy, `http://localhost:8080/` serves the Vue page, `/api/health` returns `ok` (DB and face-service reachable), and `make test` passes. The face-service has no published port.

---

## Phase 1. SQL models and login+password authentication

- [x] **1.1** `migrations/00001_init.sql`: `pgcrypto`, all tables, CHECK, UNIQUE, FK constraints and indexes (`CLAUDE.md` → Data model). Include the down migration.
- [x] **1.2** CRUD queries in `queries/`: users, faculties, groups, locations, schedules, sessions, audit_logs. `make sqlc` runs without errors.
- [x] **1.3** `internal/audit`: `Write(ctx, tx, Entry)` helper. Before/after diff for `changes`, with sensitive fields (`password_hash`, `face_embedding`) stripped.
- [x] **1.4** Passwords: bcrypt hash/verify. Django-style random password generator (`crypto/rand`, 12 chars, no ambiguous chars like `0 O l 1 I`).
- [x] **1.5** Sessions: on login, a 32-byte random token. DB stores its sha256 hash, the browser gets an httpOnly cookie. Middleware: `RequireAuth`, `RequireRole(...)`. Expired sessions are cleaned up.
- [x] **1.6** Endpoints: `POST /api/auth/login`, `POST /api/auth/logout`, `GET /api/auth/me`, `POST /api/auth/change-password`. If `must_change_password=true`, only change-password and logout are allowed.
- [x] **1.7** Login rate limit (per IP + username, e.g. 5 attempts / 5 minutes). `login` and `login_failed` are audited. Users with `is_active=false` cannot log in.
- [x] **1.8** `createadmin --username X` command: generates a password and prints it once. Must work via `make createadmin`.
- [x] **1.9** No student login for now (it will come via HEMIS in phase 6). Only when `DEV_STUDENT_PASSWORD_LOGIN=true` can a student log in with username/password (for testing phases 4–5). A `seed` command creates a test faculty, group, teacher and students.
- [x] **1.10** Frontend: login page (zod form), change-password page, auth store, route guards, role-based redirect, 3 empty layouts.
- [x] **1.11** Tests: password generator, session middleware, role checks, login rate limit. DB-backed tests run in `go-test` against `app_test` (migrate up before the suite, clean tables between tests).

**Done when:** on the compose stack, an admin created with `make createadmin` can log in at `localhost:8080`, the layout matching their role opens, logout deletes the session, and `make test` passes.

---

## Phase 2. Admin panel

- [x] **2.1** **Dashboard:** stat cards (students, teachers, groups, today's attendance: full / partial / absent) and an attendance chart for the last 7 days.
- [x] **2.2** **Faculties:** CRUD (table, search, pagination).
- [x] **2.3** **Groups:** CRUD, assign faculty and teacher, list of students in the group.
- [x] **2.4** **Users:**
    - list with filters by role, faculty, group, active/inactive, plus search;
    - create admin/teacher: a password is generated and shown ONCE in a dialog (with a copy button), `must_change_password=true`;
    - edit, deactivate, regenerate password;
    - student profile: passport fields are revealed only via a separate "Ko'rsatish" (Show) button (`view_sensitive` audit);
    - upload a photo for a student → face-service `/embed` → embedding is stored (manual enrollment until HEMIS exists). The face-service is built in phase 5, so this button is wired up after task 5.3.
- [x] **2.5** **Locations:** CRUD. Form: name, at least 3 points in `lat,lng` format (points can be added/removed), `tolerance_meters`, map link. The polygon is shown on a map (Leaflet + OSM, installed via npm, no CDN).
- [x] **2.6** **Schedules:** CRUD. Form: group OR student (chosen via radio, never both), location, date range, weekdays (Mon–Sat). Overlapping schedules produce a validation error.
- [x] **2.7** **Audit:** list (filter by actor, action, entity, date range) and entry details (before/after diff view).
- [x] **2.8** Attendance report for all students for the admin (reuses the teacher report component, wired up after task 3.3).
- [x] **2.9** Tests: schedule CHECK and overlap validation, polygon parsing, audit entries being written.

**Done when:** the admin manages every entity through the UI and every change shows up in the audit log.

---

## Phase 3. Teacher panel

- [x] **3.1** **My groups:** only groups with `teacher_id = me`, student count, today's attendance percentage.
- [x] **3.2** **Group page:** student list with each student's status today (full / partial / absent, shift).
- [x] **3.3** **Attendance report:** period picker (day / week / month / custom range), group × date matrix (each cell shows status and shift), per-student totals (expected days, full, partial, absent, percentage).
- [x] **3.4** **Student details:** steps and attempts by day (time, result flags, distance, error, selfie if still available).
- [x] **3.5** **Excel export:** attendance report as xlsx (excelize) with Uzbek headers. `export` is audited.
- [x] **3.6** **Assignments:** create/edit/delete for a group (title, description, file, deadline). Deletion is rejected if submissions exist.
- [x] **3.7** **Submissions:** per assignment, lists of students who submitted and who did not, file download, grading (1.0–5.0 in 0.5 steps) with feedback. Excel export of grades.
- [x] **3.8** Tests: accessing another teacher's group returns `404`, expected-days calculation in reports (weekdays, date range boundaries).

**Done when:** a teacher sees only their own groups, reports are correct and export to Excel, and submissions can be graded.

---

## Phase 4. Student pages

- [x] **4.1** **Home (mobile-first):** today's schedule (location name, map link), current shift/step and its time window, status of today's steps (done / pending / missed).
- [x] **4.2** **My attendance:** monthly calendar view with each day's status.
- [x] **4.3** **Assignments:** the group's assignments, time left until the deadline, file upload. After the deadline the upload button is disabled (and the backend rejects it too). Re-upload is allowed while ungraded. Grade and feedback are shown.
- [x] **4.4** **Profile:** personal info (without passport fields), photo, group, faculty.
- [x] **4.5** Tests: submission rejected after the deadline, `404` for another group's assignment, re-upload rejected after grading.

**Done when:** a dev-mode student sees their schedule and assignments and can upload a submission.

---

## Phase 5. FaceID + location verification

- [x] **5.1** **face-service:** extend the phase-0 skeleton. Add DeepFace and its CPU dependencies with `uv add` (pinned in `uv.lock`, CPU-only TensorFlow). Load the model once in `lifespan` with a warm-up, plus `/health`, `/embed`, `/verify` (`CLAUDE.md` → Face-service contract), `X-Internal-Token` check, semaphore-limited concurrency. Update `face-service/Dockerfile` with the system libraries OpenCV needs. Model weights are cached in the `deepface-weights` volume.
- [x] **5.2** face-service tests (`make test-face`): the model is not reloaded (build is called exactly once even after several requests), no-face and multiple-faces cases, bad token. Heavy model tests may be marked and run separately so `make test` stays fast.
- [x] **5.3** **Go `internal/face` client:** timeout (e.g. 15 s), typed errors, a clear error when face-service is down ("Yuzni tekshirish xizmati vaqtincha ishlamayapti"). Admin enrollment (2.4) gets wired to this client.
- [x] **5.4** **`internal/attendance`:** shift/step calculation with full table-driven tests (boundaries: 07:59, 08:00, 09:59, 10:00, 12:59, 13:00, 17:59, 18:00).
- [x] **5.5** **`internal/geo`:** point-in-polygon (ray casting) and distance to the polygon edge (for tolerance), with tests.
- [x] **5.6** **Attempt endpoint:** `POST /api/student/attendance/attempt` (multipart: `image`, `latitude`, `longitude`, `accuracy`). Flow per `CLAUDE.md` → Attendance rules. Transaction with `FOR UPDATE`, every attempt is recorded and audited (`attendance_attempt`). Students without an embedding get a clear error ("Yuz namunasi mavjud emas, adminga murojaat qiling").
- [x] **5.7** **Frontend verification flow:**
    - geolocation permission (`enableHighAccuracy`), accuracy is displayed;
    - front camera (`getUserMedia`, `facingMode: user`), oval face guide, capture photo;
    - submit, then a result screen showing which check failed and what to do (in Uzbek);
    - remaining attempts count;
    - clear messages for denied permissions or non-HTTPS contexts.
- [x] **5.8** **`internal/jobs`:** daily selfie cleanup (`SELFIE_RETENTION_DAYS=30`), run by a ticker inside `serve`, results logged.
- [x] **5.9** Fill in the attempt details on the teacher/admin side with the data from 5.6 (distance/threshold, antispoof_score, selfie).
- [x] **5.10** Integration test with a mocked face-service covering the full flow: success, location failure, liveness failure, shift switch rejected, attempt limit, two concurrent requests.
- [x] **5.11** End-to-end check on the compose stack with the real face-service: enroll a test student from a photo, then call the attempt endpoint with a matching photo and with a photo of a photo (anti-spoofing must reject it).

**Done when:** on the compose stack, a student confirms a step on their phone according to their schedule, results land in the correct shift/step, and they show up in the teacher's report.

---

## Phase 6. HEMIS integration

> Full contract, field mapping and flow: `CLAUDE.md` → HEMIS integration.

- [x] **6.1** **Migration** (new file): `users.student_id_number TEXT UNIQUE NULL`, `hemis_raw JSONB NULL`, `hemis_synced_at TIMESTAMPTZ NULL`, `face_enroll_error TEXT NOT NULL DEFAULT ''`. Add the matching sqlc queries (upsert faculty/group/user by `hemis_id`).
- [x] **6.2** **Config:** `HEMIS_BASE_URL`, `HEMIS_TIMEOUT`, `HEMIS_IMAGE_HOSTS`, `HEMIS_ALLOWED_STATUS_CODES` in `config`, `.env.example` and `compose.yaml`.
- [x] **6.3** **`internal/hemis` client:**
    - `Login(ctx, login int64, password string) (token string, err error)` → `POST /auth/login`, `login` sent as a JSON number;
    - `Me(ctx, token string) (*Profile, error)` → `GET /account/me` with Bearer token;
    - envelope check (`HTTP 200 && success == true`), typed errors: `ErrInvalidCredentials`, `ErrUnavailable`, `ErrUnexpectedResponse`;
    - defensive decoding: every nested object nullable, `birth_date` from unix seconds;
    - no request/response bodies, passwords or tokens in logs.
- [x] **6.4** **Fixtures + client tests:** `internal/hemis/testdata/` with anonymized `login_ok.json`, `login_fail.json`, `me_ok.json`, `me_nulls.json` (nested objects null), `me_expelled.json` (`studentStatus.code != 11`). Tests use `httptest`: success, wrong password, 500, timeout, malformed JSON, null nested objects.
- [x] **6.5** **Sync service** (`service/hemis_sync.go`): maps `Profile` → faculty, group, user per the mapping table, all in one transaction. Never touches `groups.teacher_id`, `passport_number`, `is_active`, or the embedding. Audits create/update with only the changed fields (passport fields masked). Unit tests for the mapping (including nulls) and for "second login updates changed fields only".
- [x] **6.6** **Login routing** in `POST /api/auth/login` per `CLAUDE.md` → Login routing: local admin/teacher first, then all-digits → HEMIS, otherwise invalid. Add validation that admin/teacher usernames cannot be all digits. Status-code check, `is_active` check, session creation, `last_login_at`, audit with reasons (`hemis_invalid_credentials`, `hemis_unavailable`, `hemis_status`).
- [x] **6.7** **Face enrollment worker** (`internal/jobs`): bounded queue, 1–2 workers, started by `serve`. Downloads the photo (host allowlist, timeout, 5 MB, `image/*`), stores it under `avatars/`, calls face-service `/embed`, sets the embedding fields or `face_enroll_error`. Enqueued after login when the embedding is null or `hemis_img_url` changed. On startup, enqueue students that have a photo URL but no embedding and no error.
- [x] **6.8** **Frontend:**
    - login form: one "Login" field + password, hint text "Talabalar HEMIS login va parolini kiritadi";
    - error messages from the backend shown as-is (Uzbek);
    - student home: "Yuz namunasi tayyorlanmoqda" / "Yuz namunasini yaratib bo'lmadi, adminga murojaat qiling" states; the verification button is disabled until an embedding exists;
    - admin user list: show `face_enroll_error`, "HEMIS'dan oxirgi sinxronlash" timestamp, and a "Yuzni qayta yaratish" (re-enroll) action.
- [x] **6.9** **hemis-mock** compose service (profile `hemis-mock`): a tiny server that serves the fixtures (`/rest/v1/auth/login`, `/rest/v1/account/me`) with a fixed test login/password, so the full flow can be tested locally. Document how to switch `HEMIS_BASE_URL` to it.
- [x] **6.10** **Integration tests** (`go-test`, mocked HEMIS + mocked face-service): first login creates faculty/group/user and enqueues enrollment; second login with a changed group moves the student; expelled status is rejected; HEMIS down returns the "unavailable" message; deactivated student is rejected; admin with an all-digit username cannot be created; no password/token appears in logs or audit rows.
- [x] **6.11** Remove or hard-disable `DEV_STUDENT_PASSWORD_LOGIN` in prod: `serve` refuses to start if `APP_ENV=prod` and the flag is `true`.

**Done when:** on the compose stack (with `hemis-mock`, and once manually against the real HEMIS), a student logs in with HEMIS credentials, their profile, group and faculty appear in the admin panel, their face embedding is created in the background, and they can complete an attendance attempt.

---

## Phase 7. HEMIS OAuth login for teachers

> Full flow, mapping and rules: `CLAUDE.md` → HEMIS OAuth for teachers. Before starting, get the real OAuth URLs, client credentials and one anonymized sample of the resource-owner response from the user.

- [x] **7.1** **Migration** (new file): `users.hemis_login TEXT UNIQUE NULL`. Replace `UNIQUE (hemis_id)` with `UNIQUE (role, hemis_id)` and update the phase-6 student upsert queries to match on `(role='student', hemis_id)`.
- [x] **7.2** **Config:** `HEMIS_OAUTH_CLIENT_ID`, `HEMIS_OAUTH_CLIENT_SECRET`, `HEMIS_OAUTH_REDIRECT_URI`, `HEMIS_TEACHER_AUTHORIZE_URL`, `HEMIS_TEACHER_ACCESS_TOKEN_URL`, `HEMIS_TEACHER_RESOURCE_OWNER_URL`, `HEMIS_TEACHER_AUTO_CREATE` in `config`, `.env.example` and `compose.yaml`. `serve` fails to start if OAuth is enabled but any of them is missing. The secret is never logged.
- [x] **7.3** **`internal/hemis/oauth.go`:**
    - `AuthorizeURL(state string) string` (`client_id`, `redirect_uri`, `response_type=code`, `state`, all URL-encoded via `url.Values`);
    - `ExchangeCode(ctx, code) (accessToken string, err error)`: form-encoded POST, `grant_type=authorization_code`, error if `access_token` is missing;
    - `TeacherProfile(ctx, accessToken) (*TeacherProfile, error)`: Bearer GET, defensive decoding, `birth_date` as string or number;
    - typed errors (`ErrOAuthToken`, `ErrOAuthProfile`, `ErrUnavailable`), 15 s timeouts, no bodies/codes/tokens in logs.
- [x] **7.4** **Shared photo downloader:** extract the phase-6 HEMIS photo download (host allowlist, http/https only, 5 MB streamed limit, jpeg/png/webp content type) into one function used by both students and teachers.
- [x] **7.5** **Teacher sync service** (`service/hemis_teacher.go`): find by `hemis_login` → role/active checks → auto-create (if enabled) → field sync per the mapping, never overwriting a non-empty value with an empty one → audit with masked passport fields. Unit tests for every branch.
- [x] **7.6** **Endpoints:**
    - `GET /api/auth/hemis/teacher/start`: generate state, set the `hemis_oauth_state` cookie, 302 to `AuthorizeURL`;
    - `GET /auth/callback`: registered before the SPA fallback; validate state, handle `?error=`, exchange code, fetch profile, sync, create session, set `last_login_at`, audit `login` (method `hemis_oauth`), 302 to `/teacher`; every failure → 302 `/login?error=<code>` plus audit `login_failed` with the reason;
    - rate limit the callback per IP.
- [x] **7.7** **Admin panel:** "HEMIS login" field on the teacher create/edit form (unique, optional) to link existing teachers, and a "HEMIS orqali" badge plus last sync time in the user list.
- [x] **7.8** **Frontend login page:**
    - a separate "HEMIS orqali kirish (O'qituvchi)" button below the login form, doing a full-page navigation to `/api/auth/hemis/teacher/start` (no fetch, no client-side URL building);
    - read `?error=` and show the Uzbek message for each code from `CLAUDE.md`, then clear the query param;
    - loading state on the button after click.
- [x] **7.9** **hemis-mock:** extend the phase-6 mock with `/oauth/authorize` (immediately 302 to `redirect_uri` with a fixed `code` and the received `state`, or `error=access_denied` for a special client_id), `/oauth/access-token`, and `/oauth/api/user` serving `teacher_me_ok.json`.
- [x] **7.10** **Tests** (`go-test`, `httptest` + mock): happy path creates the teacher and session; second login updates fields only; missing/mismatched state rejected; `?error=access_denied` handled; token exchange failure; missing `login` in profile; linked user is a student → rejected; inactive teacher rejected; auto-create off → `hemis_not_registered`; same numeric `hemis_id` for a student and a teacher can coexist; the secret, code and tokens never appear in logs or audit rows.

> Status: built and verified on compose with `hemis-mock` (full redirect round trip, session, `/teacher`). **Still open:** one manual run against the real HEMIS, confirming that it echoes `state`, and replacing `internal/hemis/testdata/teacher_me_ok.json` (a fabricated sample built from the fields in CLAUDE.md) with an anonymized real response.

**Done when:** on the compose stack (with `hemis-mock`, and once manually against the real HEMIS), clicking "HEMIS orqali kirish (O'qituvchi)" goes to HEMIS, comes back to `/auth/callback`, logs the teacher in, and lands on `/teacher`. A linked teacher sees their assigned groups.

---

## Phase 8. Production readiness

- [x] **8.1** `compose.prod.yaml` override: `restart: unless-stopped`, resource limits for face-service, no published Postgres port, `APP_ENV=prod`, `COOKIE_SECURE=true`, log rotation.
  - Done: `compose.prod.yaml`. Postgres port unpublished and test-DB init script removed, `restart: unless-stopped`, log rotation (10m x 5), memory limits everywhere and a CPU limit for face-service, `APP_ENV=prod`, `COOKIE_SECURE=true`. Config now refuses prod with an insecure cookie or `FACE_SERVICE_TOKEN=change-me`. The app got a healthcheck.
- [x] **8.2** Reverse proxy (caddy or nginx as a compose service) + HTTPS. Camera and geolocation only work over HTTPS.
  - Done: Caddy service (`infra/caddy/Caddyfile`), automatic Let's Encrypt for `DOMAIN`, HTTP to HTTPS redirect, HSTS, Permissions-Policy for camera/geolocation. New `TRUSTED_PROXIES` makes the app use the real client IP from X-Forwarded-For (login rate limit and audit); a spoofed header is ignored.
- [x] **8.3** Postgres backup script and `UPLOAD_DIR` backup.
  - Done: `make backup` (pg_dump custom format + uploads tgz, verified, atomic, pruned after `BACKUP_KEEP_DAYS`) and `make restore` (needs `CONFIRM=yes`, can restore into another database name). Schedule it with cron on the host.
- [x] **8.4** Load test: N concurrent attempts (face-service queueing on CPU), tune `FACE_CONCURRENCY` if needed.
  - Done: `make loadtest` (concurrent /verify). Result on a 12-core machine: ~1.4 attempts/s, `FACE_CONCURRENCY` 2 / 4 / 6 give the same throughput, so the default 2 stays. With 4 cores: ~0.45/s. See "Production" in CLAUDE.md.
