# CLAUDE.md — Internship Monitoring Platform

A system for monitoring student internships (practice). The **admin** manages the whole system, the **teacher** monitors their own groups and gives assignments, and the **student** confirms they are at the internship site via FaceID + geolocation.

Phases and tasks live in `TASKS.md`. Always work on the tasks of the current phase. Do not move on to the next phase on your own.

## Architecture

Two processes:

1. **Go binary** (`cmd/app`): HTTP API + embedded Vue SPA + goose migrations. This is the only publicly exposed process.
2. **face-service** (`face-service/`, Python FastAPI + DeepFace): never exposed publicly. Go calls it over a private network (`face-service:8001` inside compose). It is stateless: it stores nothing and never connects to the DB.

```
browser ──> app (Go, fiber) ──> postgres
                   └──────────> face-service:8001 (internal network only)
```

**Everything runs and is tested through Docker Compose, locally too, from phase 0 onward.** Do not rely on host-installed Postgres, Go, Python or bun to verify work. A task is only done when it works via `docker compose`.

## Stack

| Layer                  | Technology                                                        |
| ---------------------- | ----------------------------------------------------------------- |
| HTTP                   | fiber v2                                                          |
| DB driver              | pgx v5 (pgxpool)                                                  |
| Migrations             | goose (SQL files, embedded in the binary via `go:embed`)          |
| Queries                | sqlc (`pgx/v5` engine)                                            |
| Logging                | zap (prod: JSON, dev: console)                                    |
| Excel                  | excelize                                                          |
| Passwords              | bcrypt (cost 12)                                                  |
| Face                   | FastAPI + DeepFace (CPU), uvicorn                                 |
| Python package manager | **uv** (`pyproject.toml` + `uv.lock`, never pip/requirements.txt) |
| Frontend               | Vue 3 + Vite + TypeScript, vue-router, Pinia, @tanstack/vue-query |
| JS package manager     | **bun** (`bun.lock`, never npm/yarn/pnpm)                         |
| Containers             | Docker Compose (`compose.yaml`) for dev, tests and prod           |
| UI                     | shadcn-vue + Tailwind CSS                                         |
| Forms                  | vee-validate + zod (`@vee-validate/zod`)                          |

## Directory layout

```
cmd/app/main.go            # subcommands: serve, migrate, createadmin, seed
internal/
  config/                  # env -> struct
  db/                      # sqlc-generated code (NEVER edit by hand)
  http/
    server.go              # fiber app, routing, SPA fallback
    middleware/            # auth, role, request-id, recover, logger
    handlers/{admin,teacher,student,auth}/
  service/                 # business logic (handler -> service -> db)
  attendance/              # shift/step calculation, attempt flow
  geo/                     # point-in-polygon, distance (haversine)
  face/                    # face-service HTTP client
  storage/                 # file storage interface (local disk implementation)
  audit/                   # audit write helper
  export/                  # excel export
  hemis/                   # HEMIS client (phase 6)
  jobs/                    # background jobs (selfie cleanup)
migrations/                # goose: 00001_init.sql, ...
queries/                   # *.sql for sqlc
sqlc.yaml
web/                       # Vue project (bun)
  embed.go                 # //go:embed all:dist
  package.json
  bun.lock
face-service/
  app/main.py
  tests/
  pyproject.toml           # uv project
  uv.lock
  Dockerfile
Dockerfile                 # multi-stage: bun build web -> go build -> slim runtime
compose.yaml               # postgres, face-service, app (+ test profile)
.env.example
Makefile
```

## Commands

All Makefile targets are thin wrappers around `docker compose`.

```
make up              # docker compose up -d --build  (postgres + face-service + app)
make down            # docker compose down
make logs            # docker compose logs -f app face-service
make ps              # docker compose ps
make migrate         # docker compose exec app /app migrate up
make createadmin     # docker compose exec app /app createadmin --username admin
make seed            # docker compose exec app /app seed
make sqlc            # docker compose run --rm tools sqlc generate
make test            # all tests below
make test-go         # docker compose --profile test run --rm go-test       (go vet + go test ./... against test DB)
make test-face       # docker compose --profile test run --rm face-test     (uv run pytest)
make test-web        # docker compose --profile test run --rm web-test      (bun run lint && bun run test && bun run build)
make web-dev         # cd web && bun run dev  (optional host hot-reload; proxies /api -> localhost:8080)
```

Host tooling is optional. If used, it must be the same tools: `bun install`, `bun run dev`, `uv sync`, `uv run ...`.

`make createadmin` creates an admin. Like Django, it generates a random password and prints it once.

### compose.yaml requirements

- **postgres** (16-alpine): named volume `pgdata`, healthcheck (`pg_isready`), published only on `127.0.0.1:5432` for local DB tools. An init script also creates the `app_test` database used by Go tests.
- **face-service**: built from `face-service/Dockerfile` (uv-based). Listens on `0.0.0.0:8001` inside the container but has **no published ports** (internal network only). Named volume `deepface-weights` mounted at `/root/.deepface` so model weights download once. Healthcheck on `/health` with a long `start_period` (model loading).
- **app**: built from the root `Dockerfile`. `depends_on` postgres and face-service with `condition: service_healthy`. Published on `8080:8080`. Named volume `uploads` at `/data/uploads`. `env_file: .env`. Entrypoint runs `migrate up` then `serve`.
- **tools** (profile `tools`): Go image with sqlc and goose for code generation.
- **go-test / face-test / web-test** (profile `test`): run the test suites inside containers. `go-test` uses `DATABASE_URL` pointing at `app_test` and a mocked face-service (no real model needed).

### Dockerfiles

- Root `Dockerfile`, multi-stage: `oven/bun` stage runs `bun install --frozen-lockfile && bun run build` → `golang` stage copies `web/dist` and builds a static binary (`CGO_ENABLED=0`) → runtime `gcr.io/distroless/static` (or `alpine` with `tzdata`) with `TZ_NAME=Asia/Tashkent`. Final image contains only the binary.
- `face-service/Dockerfile`: `python:3.11-slim` + uv (copy from `ghcr.io/astral-sh/uv`), `uv sync --frozen --no-dev` in a cached layer before copying the code, system libs required by OpenCV (`libgl1`, `libglib2.0-0`), runs `uv run uvicorn app.main:app --host 0.0.0.0 --port 8001 --workers 1`.

## Configuration (env)

```
APP_ENV=dev|prod
HTTP_ADDR=:8080
DATABASE_URL=postgres://app:app@postgres:5432/app?sslmode=disable
TZ_NAME=Asia/Tashkent
UPLOAD_DIR=/data/uploads
FACE_SERVICE_URL=http://face-service:8001
FACE_SERVICE_TOKEN=...            # shared secret for internal requests
SESSION_TTL=168h
COOKIE_SECURE=true
MAX_ATTEMPTS_PER_STEP=5
GPS_MAX_ACCURACY_M=100
SELFIE_RETENTION_DAYS=30
DEV_STUDENT_PASSWORD_LOGIN=false  # dev only, until HEMIS login exists
HEMIS_BASE_URL=https://student.samdpi.uz/rest/v1   # phase 6
HEMIS_TIMEOUT=10s
HEMIS_IMAGE_HOSTS=hemis.samdpi.uz                  # allowlist for photo downloads
HEMIS_ALLOWED_STATUS_CODES=11                       # studentStatus codes allowed to log in (11 = O'qimoqda)
# phase 7: teacher OAuth (URLs are examples, confirm the real ones)
HEMIS_OAUTH_CLIENT_ID=...
HEMIS_OAUTH_CLIENT_SECRET=...
HEMIS_OAUTH_REDIRECT_URI=http://localhost:8080/auth/callback
HEMIS_TEACHER_AUTHORIZE_URL=https://hemis.samdpi.uz/oauth/authorize
HEMIS_TEACHER_ACCESS_TOKEN_URL=https://hemis.samdpi.uz/oauth/access-token
HEMIS_TEACHER_RESOURCE_OWNER_URL=https://hemis.samdpi.uz/oauth/api/user
HEMIS_TEACHER_AUTO_CREATE=true                      # create a teacher on first OAuth login if no linked user exists
TRUSTED_PROXIES=                                    # reverse proxy CIDRs whose X-Forwarded-For is trusted (prod: compose network)
```

## Data model

General rules:

- PK: `id UUID DEFAULT gen_random_uuid()`.
- All timestamps are `TIMESTAMPTZ`, dates are `DATE`. Business logic runs in `Asia/Tashkent` time.
- Text columns are `TEXT NOT NULL DEFAULT ''` (to avoid `pgtype.Text` in sqlc). FKs and genuinely optional values may be NULL.
- Every table has `created_at`. Editable tables also have `updated_at`.
- Users are never deleted. They are deactivated with `is_active=false`.

### faculties

`id, name, hemis_id BIGINT UNIQUE NULL, created_at`

### groups

`id, name, hemis_id BIGINT UNIQUE NULL, faculty_id → faculties ON DELETE SET NULL, teacher_id → users ON DELETE SET NULL, created_at`

### users

- Auth: `username TEXT UNIQUE NULL, password_hash TEXT NULL, role TEXT CHECK (role IN ('admin','teacher','student')), is_active BOOL DEFAULT true, must_change_password BOOL DEFAULT false, last_login_at TIMESTAMPTZ NULL`
- HEMIS: `hemis_id BIGINT UNIQUE NULL, first_name, second_name, third_name, full_name, short_name, image, hemis_img_url, birth_date, phone, passport_pin, passport_number, gender, payment_form, level, semester, gpa, address, country, province, district`
- `full_name` and `short_name` are plain columns that come from HEMIS. They are NOT generated columns.
- `group_id → groups SET NULL, faculty_id → faculties SET NULL`
- Face: `face_embedding REAL[] NULL, face_model TEXT DEFAULT '', face_enrolled_at TIMESTAMPTZ NULL`
- `created_at, updated_at`
- `passport_pin` and `passport_number` are returned only by admin endpoints. Viewing them writes a `view_sensitive` audit entry.
- Added in phase 6 (new migration): `student_id_number TEXT UNIQUE NULL, hemis_raw JSONB NULL, hemis_synced_at TIMESTAMPTZ NULL, face_enroll_error TEXT DEFAULT ''`. See "HEMIS integration".
- Changed in phase 7 (new migration): add `hemis_login TEXT UNIQUE NULL` (teacher identity from HEMIS OAuth). Replace `UNIQUE (hemis_id)` with `UNIQUE (role, hemis_id)`, because HEMIS student IDs and employee IDs come from different tables and can collide. See "HEMIS OAuth for teachers".

### sessions

`id TEXT PK (sha256 hash of the token), user_id → users ON DELETE CASCADE, ip INET, user_agent, created_at, expires_at`. The cookie holds only the raw token: `httpOnly, SameSite=Lax, Secure (prod)`.

### locations

`id, name, polygon JSONB NOT NULL` (`[[lat,lng],[lat,lng],...]`, at least 3 points), `tolerance_meters INT DEFAULT 30, location_url TEXT, created_at, updated_at`.
In the admin form, points are entered as `39.667365,66.878348`. The backend parses them and stores JSONB.

### schedules

`id, name, start_date DATE, end_date DATE, user_id → users SET NULL, group_id → groups SET NULL, location_id → locations SET NULL, monday..saturday BOOL, created_at, updated_at`

- `CHECK ((user_id IS NULL) <> (group_id IS NULL))`: a schedule is assigned to a group OR to a single student, never both.
- `CHECK (end_date >= start_date)`.
- Service layer: a schedule cannot be created if it overlaps (date range and weekdays) with an existing schedule for the same student or group.
- A student's schedule for today: their personal schedule (`user_id`) first, otherwise their group's schedule (`group_id`).

### attendances

`id, student_id → users RESTRICT, schedule_id → schedules RESTRICT, shift SMALLINT CHECK (shift IN (1,2)), date DATE, created_at, UNIQUE (student_id, schedule_id, date)`

### attendance_steps

`id, attendance_id → attendances CASCADE, step SMALLINT CHECK (step IN (1,2,3)), success BOOL DEFAULT false, created_at, UNIQUE (attendance_id, step)`

### attendance_attempts

`id, step_id → attendance_steps CASCADE, image TEXT (file path, '' after retention cleanup), ip_address INET, user_agent, latitude DOUBLE PRECISION, longitude DOUBLE PRECISION, gps_accuracy DOUBLE PRECISION, location_id → locations SET NULL, face_verified, location_verified, liveness_verified, success BOOL, error TEXT, face_distance DOUBLE PRECISION NULL, face_threshold DOUBLE PRECISION NULL, antispoof_score DOUBLE PRECISION NULL, attempted_at TIMESTAMPTZ`

### assignments

`id, teacher_id → users RESTRICT, group_id → groups RESTRICT, title, description, file TEXT, deadline TIMESTAMPTZ, created_at, updated_at`

### submissions

`id, assignment_id → assignments CASCADE, student_id → users RESTRICT, file TEXT, submitted_at, grade NUMERIC(2,1) NULL CHECK (grade BETWEEN 1 AND 5), feedback TEXT, graded_at TIMESTAMPTZ NULL, UNIQUE (assignment_id, student_id)`

### audit_logs

`id, actor_id → users SET NULL, actor_role, action TEXT (create|update|delete|login|login_failed|logout|view_sensitive|export|grade|attendance_attempt), entity_type, entity_id UUID NULL, changes JSONB (before/after), ip INET, user_agent, created_at`
Indexes: `(entity_type, entity_id)`, `(actor_id, created_at DESC)`, `(created_at DESC)`.

## Attendance rules (the most important business logic)

Shifts and steps (Asia/Tashkent, half-open intervals `[start, end)`):

| shift | step 1      | step 2      | step 3      |
| ----- | ----------- | ----------- | ----------- |
| 1     | 08:00–10:00 | 10:00–12:00 | 12:00–13:00 |
| 2     | 13:00–14:00 | 14:00–16:00 | 16:00–18:00 |

This table lives in exactly one place in `internal/attendance` as a constant and is fully unit-tested.

Attempt flow (`POST /api/student/attendance/attempt`):

1. Determine `shift` and `step` from server time. Before 08:00 or at/after 18:00, reject (no attempt is recorded).
2. Find the student's active schedule for today (date range + weekday flag). If none, reject.
3. Get or create the `attendance`. One attendance per day: if the existing attendance's `shift` differs from the current shift, reject.
4. Get or create the `attendance_step`. If the step is already `success`, reject. If the attempt count has reached `MAX_ATTEMPTS_PER_STEP`, reject.
5. Save the selfie. Check location (polygon + `tolerance_meters`, `gps_accuracy <= GPS_MAX_ACCURACY_M`). Call face-service `/verify` (anti-spoofing on).
6. Record all results (every flag, distance, threshold, error) in `attendance_attempts`. `success = face && liveness && location`. On success, set the step to `success=true`.
7. Steps 3–6 run in one transaction. The face-service call happens before the transaction starts. The attendance row is locked with `SELECT ... FOR UPDATE` to handle concurrent requests.

Daily status: 3/3 steps = **full**, 1–2 = **partial**, 0 or no attendance on a scheduled working day = **absent**. Expected days in reports are computed from the schedule's weekdays and date range.

Important: time always comes from the server. Never trust client-supplied time.

## Face-service contract

- The model and detector are loaded ONCE at startup (`DeepFace.build_model` + one warm-up call inside the FastAPI `lifespan`). Loading the model per request is forbidden.
- Dependencies are managed with uv only (`uv add`, `uv sync`, `uv run`). Tests use pytest, run via `uv run pytest`.
- CPU server: `uvicorn --workers 1`. Inference runs via `run_in_threadpool`, with concurrency limited by an `asyncio.Semaphore` (env `FACE_CONCURRENCY`, default 2).
- Defaults: `FACE_MODEL=Facenet512`, `FACE_DETECTOR=retinaface`, `FACE_METRIC=cosine`. The threshold comes from DeepFace's default for that model/metric and can be overridden via env.
- Every request is checked for the `X-Internal-Token` header.

Endpoints:

- `GET /health` → `{status, model, detector}`
- `POST /embed` (multipart `image`, `anti_spoofing: bool=false`) → `{embedding: float[], model, face_count}`. Used for enrollment (HEMIS photo), so anti-spoofing is off.
- `POST /verify` (multipart `image`, `embedding` as a JSON string) → `{face_found, is_real, antispoof_score, distance, threshold, verified}`. Anti-spoofing is always on.
- Errors: no face or more than one face in the image → `422` with a specific `code` (`no_face`, `multiple_faces`).

## HEMIS integration (phase 6)

Students never get a local password. They log in with their HEMIS credentials. Our backend verifies them against HEMIS, syncs the profile and opens OUR OWN session. HEMIS is the source of truth for student profile data.

### HEMIS API

All responses use the envelope `{"success": bool, "error": string|null, "data": ..., "code": int}`. Treat a call as successful only if HTTP status is 200 AND `success == true`.

**Login:** `POST {HEMIS_BASE_URL}/auth/login`

```json
// request (login is sent as a JSON NUMBER, parsed from the digits the user typed)
{"login": 320261100667, "password": "..."}

// response
{"success": true, "error": null, "data": {"token": "<jwt>", "refresh_token": "<opaque>"}, "code": 200}
```

**Profile:** `GET {HEMIS_BASE_URL}/account/me` with `Authorization: Bearer <token>`. Response `data` shape (values anonymized):

```json
{
  "id": 17694,
  "first_name": "ALI", "second_name": "VALIYEV", "third_name": "OLIM O'G'LI",
  "full_name": "VALIYEV ALI OLIM O'G'LI", "short_name": "VALIYEV A. O.",
  "student_id_number": "320261100000",
  "passport_pin": "00000000000000",
  "image": "https://hemis.samdpi.uz/static/crop/.../320_320_90_xxx.jpg",
  "birth_date": 1199145600,
  "phone": "+998900000000",
  "gender": {"code": "12", "name": "Ayol"},
  "paymentForm": {"code": "12", "name": "To'lov-shartnoma"},
  "studentStatus": {"code": "11", "name": "O'qimoqda"},
  "group": {"id": 980, "name": "...", "educationLang": {...}},
  "faculty": {"id": 1, "name": "Tillar", "code": "320-101", ...},
  "level": {"code": "11", "name": "1-kurs"},
  "semester": {"id": 2471, "code": "11", "name": "1-semestr", ...},
  "avg_gpa": "0.00",
  "address": "...",
  "country": {"code": "UZ", "name": "O'zbekiston"},
  "province": {"code": "1718", "name": "Samarqand viloyati"},
  "district": {"code": "1718215", "name": "Kattaqo'rg'on tumani"},
  "specialty": {...}, "educationForm": {...}, "educationType": {...}, ...
}
```

Nested objects may be `null`. Decode defensively: a missing or null nested object maps to `''`, never panics. Go structs for the response live in `internal/hemis/types.go`.

### Field mapping (`me.data` → our tables)

| Our column                                                           | HEMIS source                                     | Notes                                        |
| -------------------------------------------------------------------- | ------------------------------------------------ | -------------------------------------------- |
| `users.hemis_id`                                                     | `id`                                             | upsert key                                   |
| `users.student_id_number`                                            | `student_id_number`                              | also stored as `username`                    |
| `users.username`                                                     | `student_id_number`                              |                                              |
| `first_name`, `second_name`, `third_name`, `full_name`, `short_name` | same names                                       | copied as-is                                 |
| `passport_pin`                                                       | `passport_pin`                                   |                                              |
| `passport_number`                                                    | (not provided)                                   | never overwritten by sync                    |
| `hemis_img_url`                                                      | `image`                                          |                                              |
| `image`                                                              | downloaded copy of `image`                       | `avatars/...` in storage                     |
| `birth_date`                                                         | `birth_date` (unix seconds)                      | converted to `YYYY-MM-DD` in UTC             |
| `phone`                                                              | `phone`                                          |                                              |
| `gender`                                                             | `gender.name`                                    |                                              |
| `payment_form`                                                       | `paymentForm.name`                               |                                              |
| `level`                                                              | `level.name`                                     |                                              |
| `semester`                                                           | `semester.name`                                  |                                              |
| `gpa`                                                                | `avg_gpa`                                        |                                              |
| `address`                                                            | `address`                                        |                                              |
| `country`, `province`, `district`                                    | `country.name`, `province.name`, `district.name` |                                              |
| `role`                                                               | always `student`                                 |                                              |
| `hemis_raw`                                                          | whole `data` object                              | for future fields; never returned by any API |
| `hemis_synced_at`                                                    | now                                              |                                              |
| `faculties` (upsert by `hemis_id`)                                   | `faculty.id`, `faculty.name`                     |                                              |
| `groups` (upsert by `hemis_id`)                                      | `group.id`, `group.name`, faculty from above     | `teacher_id` is NEVER touched by sync        |
| `users.group_id`, `users.faculty_id`                                 | the upserted group/faculty                       |                                              |

Fields managed only by us and never overwritten by sync: `is_active`, `role` of existing non-student users, `face_embedding` (except via re-enrollment below), `passport_number`, `groups.teacher_id`.

### Login routing (single login form)

1. Look up a local user by `username` with `role IN ('admin','teacher')` and a non-null `password_hash`. If found, use local bcrypt auth. HEMIS is never called.
2. Otherwise, if the login is all digits (`^\d+$`), use HEMIS auth.
3. Otherwise, return "Login yoki parol noto'g'ri".

Admin/teacher usernames must NOT be all digits (validated on create/update) so the two paths can never collide. Teachers can additionally use HEMIS OAuth (phase 7), which is a separate button and never goes through this form. `DEV_STUDENT_PASSWORD_LOGIN` adds a local-password path for students and must be `false` in prod.

### HEMIS login flow

1. Rate limit (same limiter as local login).
2. `POST /auth/login`. On HTTP 401/422 or `success=false`: return "Login yoki parol noto'g'ri" and audit `login_failed` (reason `hemis_invalid_credentials`). On timeout/5xx/network error: return "HEMIS tizimi vaqtincha javob bermayapti, keyinroq urinib ko'ring" and audit `login_failed` (reason `hemis_unavailable`).
3. `GET /account/me` with the token.
4. If `studentStatus.code` is not in `HEMIS_ALLOWED_STATUS_CODES`: reject with "Siz o'qiyotgan talabalar ro'yxatida emassiz" (audit `login_failed`, reason `hemis_status`).
5. In ONE transaction: upsert faculty → upsert group → upsert user (by `hemis_id`). Audit `create` or `update` for the user (actor = the student, `changes` = changed fields only, passport fields masked).
6. If an existing user is `is_active=false`, reject after the sync ("Hisobingiz bloklangan").
7. Create our session, set `last_login_at`, audit `login`.
8. If `face_embedding IS NULL` or `hemis_img_url` changed: enqueue face enrollment (see below). Login does NOT wait for it.

The HEMIS password, token and refresh_token are never stored, never logged and never written to audit. They live only for the duration of the request. HTTP request/response bodies to HEMIS are never logged.

### Face enrollment from the HEMIS photo

- Runs in a background worker (`internal/jobs`, bounded queue, 1–2 workers) so login stays fast.
- Download `hemis_img_url` only if its host is in `HEMIS_IMAGE_HOSTS`, with a timeout and a 5 MB limit, content-type `image/*`. Save to storage as `avatars/...` and set `users.image`.
- Call face-service `/embed` (anti-spoofing off). On success: set `face_embedding`, `face_model`, `face_enrolled_at`, clear `face_enroll_error`. On failure (`no_face`, `multiple_faces`, download error): keep the old embedding if any and set `face_enroll_error` to the error code.
- If an embedding already exists and the photo URL did not change, do nothing.
- The student home page shows "Yuz namunasi tayyorlanmoqda" while the embedding is null and `face_enroll_error` is empty, and "Yuz namunasini yaratib bo'lmadi, adminga murojaat qiling" when it is set. Admins see `face_enroll_error` in the user list and can enroll manually (task 2.4).

### Testing HEMIS

- `internal/hemis` is behind an interface. Tests use an `httptest` server that serves fixtures from `internal/hemis/testdata/` (`login_ok.json`, `login_fail.json`, `me_ok.json`, `me_nulls.json`, `me_expelled.json`).
- Fixtures must contain only fake, anonymized data. Never commit real student names, PINs, phone numbers or tokens.
- In compose dev, `HEMIS_BASE_URL` may point at a small mock service (profile `hemis-mock`) that serves the same fixtures, so login can be tested without a real HEMIS account.

## HEMIS OAuth for teachers (phase 7)

Teachers can sign in with "HEMIS orqali kirish (O'qituvchi)" using the standard OAuth2 authorization-code flow. This is for **teachers only**. Students keep the login/password flow from phase 6. Admins and teachers can still use local username/password.

The legacy Django implementation (`HemisClient`, `download_profile_image`, `sync_hemis_user`) is the behavioral reference. Port its behavior to Go with the fixes below. Do not copy its `print` statements: secrets, codes and tokens must never be logged.

### Flow

```
Login page button  ──> GET /api/auth/hemis/teacher/start
                         sets state cookie, 302 ──> {HEMIS_TEACHER_AUTHORIZE_URL}?client_id=..&redirect_uri=..&response_type=code&state=..
HEMIS login page   ──> 302 ──> {HEMIS_OAUTH_REDIRECT_URI} = /auth/callback?code=..&state=..
GET /auth/callback (handled by Go, NOT the SPA)
  1. verify state == state cookie, then delete the cookie
  2. POST {HEMIS_TEACHER_ACCESS_TOKEN_URL} (form-encoded):
       client_id, client_secret, code, redirect_uri, grant_type=authorization_code
     -> {"access_token": "..."}   (missing access_token = error)
  3. GET {HEMIS_TEACHER_RESOURCE_OWNER_URL} with Authorization: Bearer <access_token>
  4. find/create + sync the teacher (below), create OUR session cookie, audit `login`
  5. 302 -> /teacher
  on any error: 302 -> /login?error=<code>, audit `login_failed` with the reason
```

- The authorize URL is built by the backend (`/start`). The frontend button is a plain full-page navigation to `/api/auth/hemis/teacher/start`. `client_secret` never reaches the browser.
- **state** (CSRF protection, missing in the Python code): 32 random bytes, base64url, stored in an httpOnly cookie `hemis_oauth_state` (`SameSite=Lax`, `Path=/auth/callback`, 10 minutes). A missing or mismatched state is rejected with `error=hemis_state`. Confirm that HEMIS echoes `state` back. If it does not, stop and report, and do not silently drop the check.
- `/auth/callback` is a Go route registered before the SPA fallback. It is the only non-`/api` server route besides static assets.
- Timeouts: 15 s per HEMIS call. HEMIS error responses are logged only as status code + error code, never as bodies.
- Error codes for `/login?error=`: `hemis_state`, `hemis_denied` (HEMIS redirected with `error=` instead of `code`), `hemis_token` (code exchange failed), `hemis_profile` (resource owner call failed), `hemis_not_teacher` (linked user is not a teacher), `hemis_inactive` (`is_active=false`), `hemis_not_registered` (no linked user and auto-create is off), `hemis_unavailable` (network/timeout/5xx). The login page maps each code to an Uzbek message.

### Resource owner response and mapping

The exact teacher response shape must be captured once from the real HEMIS (anonymized) into `internal/hemis/testdata/teacher_me_ok.json` before the mapping is finalized. Fields known from the Python code:

| Our column                                | HEMIS field                                            | Notes                                                              |
| ----------------------------------------- | ------------------------------------------------------ | ------------------------------------------------------------------ |
| `users.hemis_login`                       | `login`                                                | identity key, required (missing = `hemis_profile` error)           |
| `users.hemis_id`                          | `employee_id` or `id` (confirm from the real response) |                                                                    |
| `full_name`                               | `name`                                                 |                                                                    |
| `first_name`, `second_name`, `third_name` | `firstname`, `surname`, `patronymic` if present        |                                                                    |
| `phone`                                   | `phone`                                                |                                                                    |
| `passport_number`                         | `passport_number`                                      |                                                                    |
| `passport_pin`                            | `passport_pin` if present                              |                                                                    |
| `birth_date`                              | `birth_date`                                           | normalize to `YYYY-MM-DD` (it may be a string or a unix timestamp) |
| `hemis_img_url` / `image`                 | `picture_full`                                         | downloaded with the same downloader as students                    |
| `hemis_raw`, `hemis_synced_at`            | whole response, now                                    |                                                                    |
| `role`                                    | `teacher`                                              |                                                                    |

All fields are optional except `login`. Missing fields map to `''` and never overwrite an existing non-empty value with an empty one.

### Find or create

1. Find a user by `hemis_login`.
2. If found and `role != 'teacher'`: reject (`hemis_not_teacher`). If `is_active = false`: reject (`hemis_inactive`).
3. If not found: if `HEMIS_TEACHER_AUTO_CREATE=true`, create a teacher (`username = NULL`, `password_hash = NULL`, `must_change_password = false`); otherwise reject (`hemis_not_registered`).
4. Sync profile fields (mapping above) in one transaction with an audit `create`/`update` entry (passport fields masked).
5. Admins can link an existing local teacher by setting `hemis_login` in the teacher form. A teacher can then use both local password and HEMIS OAuth.
6. Never touch `role`, `is_active`, `password_hash`, or `groups.teacher_id`. A new OAuth teacher sees no groups until an admin assigns them.
7. Teachers do not need a face embedding. Download the photo only (same host allowlist, 5 MB limit, jpeg/png/webp), synchronously inside the callback but outside the DB transaction. A photo failure is logged and does not block login.

Unlike the Python code, the HEMIS photo is not re-downloaded if `hemis_img_url` is unchanged and `image` is set. No unusable-password trick is needed: OAuth-only teachers have `password_hash = NULL`, which the local login path already rejects.

## Roles and permissions

- **admin**: everything. Only admins create teachers/admins/little admins and generate their passwords.
- **little** (little admin): works in the admin panel but only in the sections granted to it (`users.permissions`, e.g. `locations:view`, `locations:create`, `schedules:view`). Granting an action also grants what it depends on (see `internal/permission`). Never available to it, whatever it holds: creating accounts, regenerating passwords, passport data, editing permissions, and touching admin/teacher/little accounts (it manages students only). Permissions are checked on every request from the database, so a change applies at once.
- **teacher**: only groups where `groups.teacher_id = me`, their students, attendance, assignments and submissions. Requesting another group's ID returns `404` (not 403).
- **student**: only their own schedule, own attendance, their group's assignments and their own submissions.
- Permissions are always enforced on the backend. Hiding things in the frontend is only for convenience.

## Files

- `storage.Storage` interface (`Save, Open, Delete`). For now, local disk under `UPLOAD_DIR`.
- Path: `{kind}/{yyyy}/{mm}/{uuid}.{ext}` (kind: `selfies`, `avatars`, `assignments`, `submissions`).
- Files are never served statically. They are served only through the permission-checking `GET /api/files/...` endpoint.
- Limits: selfie ≤ 5 MB (jpeg/png/webp). Assignment and submission files ≤ 20 MB (pdf, doc, docx, xls, xlsx, ppt, pptx, zip, rar, jpg, png). MIME type is detected from content, never trusted from the extension.
- A daily job in `jobs` deletes selfie files older than `SELFIE_RETENTION_DAYS` and sets `attendance_attempts.image = ''`. The attempt row itself is kept.

## Assignment rules

- After the deadline, a submission can be neither created nor updated (server time).
- Before the deadline, and only while ungraded, a student may re-upload their file (the old file is deleted).
- Grade: 1.0–5.0 with one decimal place (e.g. 4.5). Grading sets `graded_at` and writes an audit entry.

## API conventions

- Prefixes: `/api/auth/*`, `/api/admin/*`, `/api/teacher/*`, `/api/student/*`, `/api/files/*`.
- JSON is `snake_case`. Lists: `?page=&page_size=&search=&sort=` → `{items, total, page, page_size}`.
- Error format: `{"error": {"code": "validation_error", "message": "...", "fields": {"name": "..."}}}`. `message` is in Uzbek and is shown to the user.
- Dates are `YYYY-MM-DD`, timestamps RFC3339.
- Every write action (create/update/delete/grade/export/login) is audited in the service layer, in the SAME transaction as the main change.

## Frontend conventions

- Use bun for everything in `web/` (`bun add`, `bun install`, `bun run ...`). Never create `package-lock.json`, `yarn.lock` or `pnpm-lock.yaml`.
- The UI is Uzbek only (Latin script). All UI strings live in one place, `web/src/locales/uz.ts`, so i18n can be added later.
- Layouts: `AdminLayout`, `TeacherLayout`, `StudentLayout`. The student pages are mobile-first.
- Every form: zod schema → `toTypedSchema` → vee-validate. Backend `fields` errors are mapped back onto the form.
- API calls live in `web/src/api/` as typed functions. Server state lives in vue-query. Pinia is only for the session/current user.
- Route guard: redirects by role based on `/api/auth/me`.
- The Vite dev server proxies `/api` to `:8080`. In prod, Go serves `web/dist`. Every GET path outside `/api/*` and `/auth/callback` falls back to `index.html`. `assets/*` gets long-lived cache headers.

## Code rules

- Code, identifiers and comments are in English. User-facing text is in Uzbek.
- `internal/db` is generated by sqlc and is never edited by hand. If a query is needed, add it to `queries/*.sql` and run `make sqlc`.
- Migrations change only by adding new files. Never edit an already applied migration.
- Handlers stay thin: parse → validate → service → respond. Business logic lives in `service/` and `attendance/`.
- Never log passwords, tokens, passport data or embeddings.
- At the end of every task that changes behavior: `make up` brings the stack up healthy and `make test` passes. At the end of every phase, also smoke-test the new features against the running compose stack (e.g. with `curl` against `localhost:8080`).
- When adding a new env variable, add it to `.env.example` and to `compose.yaml` in the same change.

## Production (phase 8)

`compose.prod.yaml` is an override: `docker compose -f compose.yaml -f compose.prod.yaml ...` (`make up-prod`, `down-prod`, `ps-prod`). `.env` needs a real `DOMAIN`, `POSTGRES_PASSWORD` and `FACE_SERVICE_TOKEN`; compose refuses to start without them, and the app refuses `APP_ENV=prod` with `COOKIE_SECURE=false` or the `change-me` token.

- **Public surface:** only Caddy (80/443). The app, Postgres and face-service publish nothing. Caddy gets and renews the Let's Encrypt certificate for `DOMAIN` (ports 80/443 must be reachable), redirects HTTP to HTTPS and sets HSTS and a Permissions-Policy that allows the camera and geolocation.
- **Client IP:** `TRUSTED_PROXIES` (default: the compose network `172.16.0.0/12`) is the only source of believed `X-Forwarded-For`. Without it the header is ignored. The login rate limit and audit log depend on the real IP.
- **Postgres password:** it is applied only when the data directory is first created. Changing `POSTGRES_PASSWORD` later needs `ALTER USER app PASSWORD ...`. An existing dev volume keeps `app/app`.
- **Backups:** `make backup` writes `db-<ts>.dump` and `uploads-<ts>.tgz` to `./backups` (verified, written atomically, older than `BACKUP_KEEP_DAYS` deleted). Run it from the host cron, e.g. `0 2 * * * cd /srv/pc && make backup`, and copy `./backups` off the machine. Restore: stop the app, then `make restore RESTORE_DB=db-<ts>.dump RESTORE_UPLOADS=uploads-<ts>.tgz CONFIRM=yes`. `TARGET_DB=app_check` restores into another database to inspect a backup without touching live data.
- **Capacity (`make loadtest`, concurrent `/verify`):** the face-service is the bottleneck. On a 12-core machine one inference uses every core and throughput is about 1.4 attempts/s regardless of `FACE_CONCURRENCY` (2, 4 and 6 measured the same), so the default 2 stays. Latency is the queue: p95 is about 3 s at 4 concurrent, 11 s at 16, and at 32 requests time out at the app's 15 s `FACE_TIMEOUT`. With a 4-core cap throughput drops to about 0.45/s and 8 concurrent already time out. Rule of thumb: about 0.12 attempts/s per core, so size `FACE_CPUS` (default 8) for the largest group that confirms at the same moment. The memory limit (4g) has headroom: the service used about 2 GB.

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

Rules:

- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
