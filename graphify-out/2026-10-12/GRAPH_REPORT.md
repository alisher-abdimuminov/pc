# Graph Report - pc  (2026-10-12)

## Corpus Check
- 372 files · ~117,177 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 12 file(s) not represented in the graph (top: (none) 7, .lock 2, .example 1)

## Summary
- 2658 nodes · 8295 edges · 87 communities (77 shown, 10 thin omitted)
- Extraction: 97% EXTRACTED · 3% INFERRED · 0% AMBIGUOUS · INFERRED: 289 edges (avg confidence: 0.86)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- server.go
- package.json
- teacher_test.go
- SchedulesPage.vue
- components.json
- CLAUDE.md — Internship Monitoring Platform
- dependencies
- compilerOptions
- main.py
- nullUUID
- AuditPage.vue
- compilerOptions
- rules_test.go
- Handler
- SidebarMenuButton.vue
- server.py
- tsconfig.json
- face-service
- pc
- Handler
- AssignmentService
- SelectField.vue
- github.com/google/uuid.NullUUID
- reka-ui
- newHemisEnv
- UserDetailPage.vue
- AttendanceReport.vue
- calendar/index.ts
- Calendar.vue
- AttemptsPage.vue
- mockFace
- Deps
- Actor
- FaceCapture.vue
- Engine
- DialogContent.vue
- context.Context
- newAttemptEnv
- AssignmentsPage.vue
- github.com/jackc/pgx/v5/pgxpool.Pool
- VerifyPage.vue
- vue
- Client
- github.com/google/uuid.UUID
- Pagination.vue
- LocationsPage.vue
- PointError
- test_api.py
- TASKS.md
- engine.py
- testing.T
- New
- shifts.go
- RadioGroup.vue
- NewErrorHandler
- DatePicker.vue
- env
- StudentHomePage.vue
- admin_test.go
- AuthService
- Point
- @tanstack/vue-query
- conftest.py
- github.com/jackc/pgx/v5/pgtype.Date
- Fake
- NewLocal
- NotFound
- SidebarProvider.vue
- uz.ts
- time.Time
- UsersPage.vue
- HTTP
- .Open
- fakes.py
- button/index.ts
- fetch_photos.sh
- newSyncEnv
- SheetContent.vue
- test_model.py
- Sidebar.vue
- Queries
- @vueuse/core
- LoginPage.vue

## God Nodes (most connected - your core abstractions)
1. `vue` - 139 edges
2. `cn()` - 100 edges
3. `Actor()` - 72 edges
4. `reka-ui` - 72 edges
5. `@vueuse/core` - 53 edges
6. `New()` - 46 edges
7. `uz` - 45 edges
8. `Handler` - 42 edges
9. `Write()` - 40 edges
10. `NewServer()` - 40 edges

## Surprising Connections (you probably didn't know these)
- `Phase 1. SQL models and login+password authentication` --references--> `login()`  [INFERRED]
  TASKS.md → web/src/api/auth.ts
- `Face-service contract` --references--> `lifespan()`  [INFERRED]
  CLAUDE.md → face-service/app/main.py
- `Phase 5. FaceID + location verification` --references--> `lifespan()`  [INFERRED]
  TASKS.md → face-service/app/main.py
- `HEMIS login flow` --references--> `login()`  [INFERRED]
  CLAUDE.md → web/src/api/auth.ts
- `Phase 6. HEMIS integration` --references--> `login()`  [INFERRED]
  TASKS.md → web/src/api/auth.ts

## Import Cycles
- None detected.

## Communities (87 total, 10 thin omitted)

### Community 0 - "server.go"
Cohesion: 0.10
Nodes (26): createUserBody, nameBody, changePasswordRequest, loginRequest, kind, stepPlan, apiError, envelope (+18 more)

### Community 1 - "package.json"
Cohesion: 0.05
Nodes (35): class-variance-authority, clsx, eslint, @eslint/js, eslint-plugin-vue, globals, jsdom, leaflet (+27 more)

### Community 2 - "teacher_test.go"
Cohesion: 0.14
Nodes (22): rawResult, TestE2ERealFaceService(), env, tEnv, TestStudentCannotSeeOtherGroupsAssignment(), TestStudentDeadlineEnforcedByServer(), TestStudentProfileHidesPassportAndServesPhoto(), TestStudentSubmissionLifecycle() (+14 more)

### Community 3 - "SchedulesPage.vue"
Cohesion: 0.04
Nodes (51): crud(), Dashboard, DayStats, Group, GroupDetail, GroupInput, groups, Location (+43 more)

### Community 4 - "components.json"
Cohesion: 0.09
Nodes (22): aliases, components, composables, lib, ui, utils, font, iconLibrary (+14 more)

### Community 5 - "CLAUDE.md — Internship Monitoring Platform"
Cohesion: 0.05
Nodes (36): API conventions, Architecture, Assignment rules, assignments, attendance_attempts, Attendance rules (the most important business logic), attendance_steps, attendances (+28 more)

### Community 6 - "dependencies"
Cohesion: 0.05
Nodes (36): dependencies, class-variance-authority, clsx, leaflet, @lucide/vue, @mediapipe/tasks-vision, pinia, reka-ui (+28 more)

### Community 7 - "compilerOptions"
Cohesion: 0.12
Nodes (16): compilerOptions, allowImportingTsExtensions, erasableSyntaxOnly, lib, module, moduleDetection, noEmit, noFallthroughCasesInSwitch (+8 more)

### Community 8 - "main.py"
Cohesion: 0.13
Nodes (10): Face-service contract, embed(), face_error_handler(), health(), http_error_handler(), lifespan(), make_engine(), read_image() (+2 more)

### Community 9 - "nullUUID"
Cohesion: 0.08
Nodes (23): Schedule, auditDTO(), dateStr(), ListParams, newPage(), nullUUID(), scheduleDTO(), textPtr() (+15 more)

### Community 10 - "AuditPage.vue"
Cohesion: 0.07
Nodes (18): audit, AuditEntry, emit, multiple, props, action, actionOpts, actorId (+10 more)

### Community 11 - "compilerOptions"
Cohesion: 0.14
Nodes (13): @vue/tsconfig/tsconfig.dom.json, compilerOptions, allowArbitraryExtensions, erasableSyntaxOnly, noFallthroughCasesInSwitch, noUnusedLocals, noUnusedParameters, paths (+5 more)

### Community 12 - "rules_test.go"
Cohesion: 0.11
Nodes (17): Days, Schedule, Status, DateRange(), Expected(), Totals, StatusFromSteps(), date() (+9 more)

### Community 13 - "Handler"
Cohesion: 0.06
Nodes (17): Handler, Attendance(), cell(), Grades(), write(), Handler, parseAssignmentForm(), Handler (+9 more)

### Community 14 - "SidebarMenuButton.vue"
Cohesion: 0.09
Nodes (15): SidebarMenuButtonVariants, delegatedProps, { isMobile, state }, props, props, SidebarMenuButtonProps, emits, forwarded (+7 more)

### Community 16 - "tsconfig.json"
Cohesion: 0.40
Nodes (4): compilerOptions, paths, files, references

### Community 20 - "Handler"
Cohesion: 0.22
Nodes (8): Handler, UserResponse, actorOf(), meta(), toResponse(), CurrentUser(), SessionID(), Actor

### Community 21 - "AssignmentService"
Cohesion: 0.21
Nodes (10): Assignment, AssignmentInput, AssignmentService, GradesExport, mustRoster(), rosterEntries(), uuidPtr(), AssignmentDTO (+2 more)

### Community 22 - "SelectField.vue"
Cohesion: 0.22
Nodes (4): emit, onChange(), Option, props

### Community 23 - "github.com/google/uuid.NullUUID"
Cohesion: 0.05
Nodes (44): AttendanceAttempt, CountAttemptsAdminParams, CountAuditLogsParams, CountGroupsParams, CountUsersParams, CreateAttemptParams, CreateAuditLogParams, CreateGroupParams (+36 more)

### Community 24 - "reka-ui"
Cohesion: 0.05
Nodes (36): reka-ui, emits, forwarded, props, props, delegatedProps, emits, forwarded (+28 more)

### Community 25 - "newHemisEnv"
Cohesion: 0.27
Nodes (15): hEnv, aliProfile(), newHemisEnv(), TestDevStudentPasswordLoginWorksAlongsideHemis(), TestHemisAdoptsSeededStudentAndProtectsUsernames(), TestHemisFirstLoginCreatesStudentFacultyAndGroup(), TestHemisInactiveStudentCannotLogIn(), TestHemisOutageIsReportedClearlyAndNotCountedAgainstTheUser() (+7 more)

### Community 26 - "UserDetailPage.vue"
Cohesion: 0.03
Nodes (64): vee-validate, @vee-validate/zod, zod, getDashboard(), errorMessage(), { data, isPending }, del, del (+56 more)

### Community 27 - "AttendanceReport.vue"
Cohesion: 0.07
Nodes (35): Attempt, AttemptDay, DayStatus, fetchAttempts(), fetchReport(), Report, ReportCell, ReportStudent (+27 more)

### Community 28 - "calendar/index.ts"
Cohesion: 0.07
Nodes (20): delegatedProps, forwardedProps, props, delegatedProps, forwardedProps, props, props, delegatedProps (+12 more)

### Community 29 - "Calendar.vue"
Cohesion: 0.06
Nodes (14): [DefineMonthTemplate, ReuseMonthTemplate], [DefineYearTemplate, ReuseYearTemplate], delegatedProps, emits, formatter, forwarded, placeholder, props (+6 more)

### Community 30 - "AttemptsPage.vue"
Cohesion: 0.06
Nodes (31): attempts, cls, labels, props, extra, from, gOpts, groupId (+23 more)

### Community 31 - "mockFace"
Cohesion: 0.22
Nodes (6): mockFace, tEnv, jsonHas(), newMockFace(), TestAdminEnrollsStudentPhoto(), toJSON()

### Community 32 - "Deps"
Cohesion: 0.18
Nodes (8): Deps, GroupSource, accessLog(), healthHandler(), mountSPA(), GroupSyncService, NewGroupSyncService(), Dist()

### Community 33 - "Actor"
Cohesion: 0.08
Nodes (32): Validation(), Write(), TestHashAndCheck(), CheckPassword(), HashPassword(), Actor(), ValidGrade(), limiterKey() (+24 more)

### Community 34 - "FaceCapture.vue"
Cohesion: 0.05
Nodes (47): emit, loading, message, progress, props, show(), start(), state (+39 more)

### Community 36 - "DialogContent.vue"
Cohesion: 0.25
Nodes (6): delegatedProps, emits, forwarded, props, delegatedProps, props

### Community 37 - "context.Context"
Cohesion: 0.05
Nodes (21): Attendance, CreateFacultyParams, CreateLocationParams, CreateSessionParams, DeleteUserSessionsExceptParams, ListExpiredSelfiesParams, ListExpiredSelfiesRow, ListFacultiesParams (+13 more)

### Community 38 - "newAttemptEnv"
Cohesion: 0.18
Nodes (22): aEnv, attemptResp, newAttemptEnv(), strs(), TestAdminAttemptsListAndFilters(), TestAttemptConcurrentFailuresRespectTheLimit(), TestAttemptConcurrentSuccessIsAppliedOnce(), TestAttemptFaceServiceDownConsumesNothing() (+14 more)

### Community 39 - "AssignmentsPage.vue"
Cohesion: 0.05
Nodes (35): ListQuery, Page, Assignment, GroupStudentToday, RosterEntry, SubmissionsView, teacher, TeacherGroup (+27 more)

### Community 40 - "github.com/jackc/pgx/v5/pgxpool.Pool"
Cohesion: 0.15
Nodes (26): Services, Options, setup(), Service, New(), New(), NewServer(), NewAssignmentService() (+18 more)

### Community 41 - "VerifyPage.vue"
Cohesion: 0.05
Nodes (46): AttemptResult, student, StudentAssignment, Submission, Today, TodayStep, GeoState, useGeoStatus() (+38 more)

### Community 42 - "vue"
Cohesion: 0.04
Nodes (51): vue, props, props, props, props, props, props, props (+43 more)

### Community 43 - "Client"
Cohesion: 0.38
Nodes (4): Client, EmbedResult, VerifyResult, unwrapNet()

### Community 44 - "github.com/google/uuid.UUID"
Cohesion: 0.05
Nodes (38): AttendanceStep, CountTeacherAssignmentsParams, CreateAssignmentParams, CreateSubmissionParams, EnsureStepParams, GetAssignmentForTeacherParams, GetAttemptOwnerRow, GetStepParams (+30 more)

### Community 45 - "Pagination.vue"
Cohesion: 0.40
Nodes (4): delegatedProps, emits, forwarded, props

### Community 46 - "LocationsPage.vue"
Cohesion: 0.09
Nodes (20): formatDateTime(), parsePoints(), pct(), { defineField, handleSubmit, errors, setErrors, resetForm, isSubmitting }, deleteError, editing, formError, formOpen (+12 more)

### Community 48 - "test_api.py"
Cohesion: 0.20
Nodes (17): png_bytes(), post_embed(), post_verify(), test_concurrency_is_limited_by_the_semaphore(), test_embed_multiple_faces(), test_embed_no_face(), test_embed_returns_vector(), test_embed_without_anti_spoofing_ignores_liveness() (+9 more)

### Community 49 - "TASKS.md"
Cohesion: 0.20
Nodes (8): Phase 0. Project skeleton, Phase 1. SQL models and login+password authentication, Phase 2. Admin panel, Phase 3. Teacher panel, Phase 4. Student pages, Phase 6. HEMIS integration, Phase 7. Production readiness, Profile

### Community 50 - "engine.py"
Cohesion: 0.12
Nodes (8): decode_image(), EmbedResult, FaceError, InvalidEmbedding, InvalidImage, MultipleFaces, NoFace, VerifyResult

### Community 51 - "testing.T"
Cohesion: 0.08
Nodes (42): Changes(), equal(), HasChanges(), decode(), TestChangesDiffOnlyChangedFields(), TestChangesSensitiveChangeIsInvisible(), TestChangesStripsSensitive(), toMap() (+34 more)

### Community 52 - "New"
Cohesion: 0.24
Nodes (8): New(), Report, ReportService, todayStats(), GroupStudentToday, TeacherGroupDetail, TeacherGroupDTO, TodayStats

### Community 53 - "shifts.go"
Cohesion: 0.22
Nodes (13): StepState, Window, Clock(), minutesOfDay(), ShiftWindows(), StateOf(), at(), TestClock() (+5 more)

### Community 54 - "RadioGroup.vue"
Cohesion: 0.20
Nodes (7): delegatedProps, emits, forwarded, props, delegatedProps, forwardedProps, props

### Community 55 - "NewErrorHandler"
Cohesion: 0.50
Nodes (3): codeForStatus(), messageForStatus(), NewErrorHandler()

### Community 56 - "DatePicker.vue"
Cohesion: 0.18
Nodes (6): emit, label, open, pick(), props, value

### Community 57 - "env"
Cohesion: 0.24
Nodes (12): errCode(), env, newEnv(), newEnvDeps(), TestGeneratedPasswordCanSkipTheForcedChange(), TestLoginFailures(), TestLoginMeLogout(), TestLoginRateLimit() (+4 more)

### Community 58 - "StudentHomePage.vue"
Cohesion: 0.11
Nodes (30): clockOffset(), formatTimeLeft(), businessMinutes(), clock(), hourMarks(), layoutSteps(), markerTop(), PX_PER_MIN (+22 more)

### Community 59 - "admin_test.go"
Cohesion: 0.33
Nodes (8): adminEnv, newAdminEnv(), TestAdminCRUDIsAudited(), TestDashboardCounts(), TestLocationPolygonValidation(), TestScheduleDBConstraints(), TestScheduleValidationAndOverlap(), TestUserAdminActions()

### Community 62 - "AuthService"
Cohesion: 0.14
Nodes (12): TestLoginLimiter(), LoginLimiter, NewLoginLimiter(), Config, Client, New(), New(), AuthService (+4 more)

### Community 63 - "Point"
Cohesion: 0.19
Nodes (21): Point, Result, ParsePoint(), ParsePoints(), TestParsePoint(), TestParsePoints(), Check(), Contains() (+13 more)

### Community 64 - "@tanstack/vue-query"
Cohesion: 0.07
Nodes (24): @tailwindcss/vite, @tanstack/vue-query, @vitejs/plugin-vue, vitest, @vue/test-utils, AdminAttempt, options, options (+16 more)

### Community 65 - "conftest.py"
Cohesion: 0.24
Nodes (3): fake(), fake_client(), _token()

### Community 66 - "github.com/jackc/pgx/v5/pgtype.Date"
Cohesion: 0.08
Nodes (19): CountSchedulesParams, CreateScheduleParams, DayAttendanceStatsRow, EnsureAttendanceParams, GetTodayAttendanceParams, ListAttendanceSummaryParams, ListAttendanceSummaryRow, ListOverlapCandidatesParams (+11 more)

### Community 67 - "Fake"
Cohesion: 0.27
Nodes (4): fakeAccount, Fake, NewFake(), TestFakeBehavesLikeTheClient()

### Community 68 - "NewLocal"
Cohesion: 0.19
Nodes (7): TestRunNeedsTheSeededStudents(), NewLocal(), TestLocalRejectsTraversal(), TestLocalSaveOpenDelete(), Pool(), FileResult, Local

### Community 69 - "NotFound"
Cohesion: 0.26
Nodes (10): Error, Forbidden(), New(), NotFound(), Unauthorized(), TestTokenHash(), HashToken(), NewToken() (+2 more)

### Community 71 - "SidebarProvider.vue"
Cohesion: 0.16
Nodes (15): emits, isMobile, open, openMobile, props, setOpen(), setOpenMobile(), state (+7 more)

### Community 73 - "uz.ts"
Cohesion: 0.04
Nodes (38): @lucide/vue, vue-router, NavLink, route, router, session, { setOpenMobile }, links (+30 more)

### Community 74 - "time.Time"
Cohesion: 0.17
Nodes (10): User, StudentService, ProfileDTO, StudentAssignment, StudentAssignments, StudentSubmission, TodaySchedule, TodayStep (+2 more)

### Community 77 - "UsersPage.vue"
Cohesion: 0.03
Nodes (66): faculties, Faculty, open, copied, open, props, emits, forwarded (+58 more)

### Community 79 - "HTTP"
Cohesion: 0.16
Nodes (10): flex, flexID, GroupsHTTP, groupsPage, HTTP, meJSON, Ref, refJSON (+2 more)

### Community 80 - ".Open"
Cohesion: 0.24
Nodes (8): Handler, New(), downloadName(), FileService, TestMaxSizeAndMIME(), IsImage(), MaxSize(), MIMEForExt()

### Community 81 - "fakes.py"
Cohesion: 0.13
Nodes (4): FaceNotDetected, FakeDeepFace, FakeSpoofModel, SpoofDetected

### Community 83 - "button/index.ts"
Cohesion: 0.08
Nodes (26): Props, ButtonVariants, delegatedProps, forwardedProps, props, delegatedProps, forwardedProps, props (+18 more)

### Community 86 - "newSyncEnv"
Cohesion: 0.14
Nodes (13): Group, syncEnv, FakeGroups, NewFakeGroups(), TestFakeGroups(), counts(), newSyncEnv(), TestConcurrentSyncsDoNotDuplicateGroups() (+5 more)

### Community 88 - "SheetContent.vue"
Cohesion: 0.22
Nodes (7): delegatedProps, emits, forwarded, props, SheetContentProps, delegatedProps, props

### Community 91 - "test_model.py"
Cohesion: 0.33
Nodes (6): client(), embed(), photo(), test_replayed_photo_is_rejected_by_anti_spoofing(), test_same_person_verifies_and_other_person_does_not(), verify()

### Community 92 - "Sidebar.vue"
Cohesion: 0.09
Nodes (14): emits, forwarded, props, props, delegatedProps, props, props, delegatedProps (+6 more)

### Community 93 - "Queries"
Cohesion: 0.08
Nodes (28): createAdmin(), ensureUser(), resetPassword(), seed(), seedDemo(), main(), serve(), usage() (+20 more)

### Community 100 - "@vueuse/core"
Cohesion: 0.05
Nodes (28): @vueuse/core, delegatedProps, emits, forwarded, props, delegatedProps, forwardedProps, props (+20 more)

### Community 101 - "LoginPage.vue"
Cohesion: 0.08
Nodes (27): pinia, changePassword(), CurrentUser, json(), login(), logout(), me(), Role (+19 more)

## Knowledge Gaps
- **726 isolated node(s):** `face-service`, `fetch_photos.sh script`, `pc`, `Queries`, `apiError` (+721 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 976 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **10 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `vue` connect `vue` to `package.json`, `SchedulesPage.vue`, `AuditPage.vue`, `SidebarMenuButton.vue`, `reka-ui`, `UserDetailPage.vue`, `AttendanceReport.vue`, `calendar/index.ts`, `Calendar.vue`, `AttemptsPage.vue`, `FaceCapture.vue`, `DialogContent.vue`, `AssignmentsPage.vue`, `VerifyPage.vue`, `Pagination.vue`, `LocationsPage.vue`, `RadioGroup.vue`, `DatePicker.vue`, `StudentHomePage.vue`, `@tanstack/vue-query`, `SidebarProvider.vue`, `uz.ts`, `UsersPage.vue`, `button/index.ts`, `SheetContent.vue`, `Sidebar.vue`, `@vueuse/core`, `LoginPage.vue`?**
  _High betweenness centrality (0.088) - this node is a cross-community bridge._
- **Why does `CLAUDE.md — Internship Monitoring Platform` connect `CLAUDE.md — Internship Monitoring Platform` to `main.py`?**
  _High betweenness centrality (0.028) - this node is a cross-community bridge._
- **What connects `face-service`, `fetch_photos.sh script`, `pc` to the rest of the system?**
  _726 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `server.go` be split into smaller, more focused modules?**
  _Cohesion score 0.10341402855369336 - nodes in this community are weakly interconnected._
- **Should `package.json` be split into smaller, more focused modules?**
  _Cohesion score 0.05365853658536585 - nodes in this community are weakly interconnected._
- **Should `teacher_test.go` be split into smaller, more focused modules?**
  _Cohesion score 0.13968253968253969 - nodes in this community are weakly interconnected._
- **Should `SchedulesPage.vue` be split into smaller, more focused modules?**
  _Cohesion score 0.043233082706766915 - nodes in this community are weakly interconnected._