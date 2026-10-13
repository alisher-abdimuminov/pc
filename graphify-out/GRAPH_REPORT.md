# Graph Report - pc  (2026-10-13)

## Corpus Check
- 419 files · ~147,851 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 13 file(s) not represented in the graph (top: (none) 8, .lock 2, .example 1)

## Summary
- 3004 nodes · 9435 edges · 101 communities (89 shown, 12 thin omitted)
- Extraction: 97% EXTRACTED · 3% INFERRED · 0% AMBIGUOUS · INFERRED: 329 edges (avg confidence: 0.86)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- server.go
- package.json
- newTeacherEnv
- SchedulesPage.vue
- components.json
- CLAUDE.md — Internship Monitoring Platform
- dependencies
- compilerOptions
- AssignmentService
- OAuth
- UsersPage.vue
- compilerOptions
- Expected
- Actor
- SidebarMenuButton.vue
- Handler
- tsconfig.json
- face-service
- pc
- Handler
- reka-ui
- .upsertStudent
- NewServer
- select/index.ts
- newHemisEnv
- UserDetailPage.vue
- AttendanceReport.vue
- @vueuse/core
- Calendar.vue
- AttemptsPage.vue
- Config
- StudentAssignmentsPage.vue
- New
- faceGuard.test.ts
- AuthService
- FaceCapture.vue
- context.Context
- newAttemptEnv
- NotFound
- avatar/index.ts
- Totals
- .OAuthCallback
- github.com/jackc/pgx/v5/pgtype.Date
- github.com/google/uuid.NullUUID
- NewErrorHandler
- GroupsPage.vue
- PointError
- test_api.py
- devDependencies
- main.py
- testing.T
- time.Time
- mockFace
- Load
- LoginLimiter
- FileService
- newEnv
- StudentHomePage.vue
- admin_test.go
- Write
- verify
- loadtest.py
- Point
- vitest
- Engine
- HemisPhotoService
- newSyncEnv
- newOAuthEnv
- backup.sh
- restore.sh
- SidebarProvider.vue
- VerifyPage.vue
- uz.ts
- LoginPage.vue
- scripts
- FaceCapture.test.ts
- LocationMap.vue
- conftest.py
- vue
- @lucide/vue
- fetch_photos.sh
- PopoverContent.vue
- vite.config.ts
- Queries
- test_model.py
- Sidebar.vue
- SheetContent.vue
- Storage
- AssignmentsPage.vue
- shifts.go
- dialog/index.ts
- SelectField.vue
- Fake
- verify.ts
- LocationStatusCard.vue
- github.com/google/uuid.UUID
- NativeSelect.vue
- Changes
- .Open

## God Nodes (most connected - your core abstractions)
1. `vue` - 155 edges
2. `cn()` - 114 edges
3. `reka-ui` - 89 edges
4. `Actor()` - 79 edges
5. `@vueuse/core` - 62 edges
6. `@lucide/vue` - 47 edges
7. `uz` - 47 edges
8. `New()` - 46 edges
9. `Write()` - 44 edges
10. `NewServer()` - 44 edges

## Surprising Connections (you probably didn't know these)
- `Face-service contract` --references--> `lifespan()`  [INFERRED]
  CLAUDE.md → face-service/app/main.py
- `Phase 5. FaceID + location verification` --references--> `lifespan()`  [INFERRED]
  TASKS.md → face-service/app/main.py
- `Resource owner response and mapping` --references--> `login()`  [INFERRED]
  CLAUDE.md → web/src/api/auth.ts
- `HEMIS login flow` --references--> `login()`  [INFERRED]
  CLAUDE.md → web/src/api/auth.ts
- `Phase 1. SQL models and login+password authentication` --references--> `login()`  [INFERRED]
  TASKS.md → web/src/api/auth.ts

## Import Cycles
- None detected.

## Communities (101 total, 12 thin omitted)

### Community 0 - "server.go"
Cohesion: 0.12
Nodes (18): createUserBody, nameBody, changePasswordRequest, loginRequest, kind, stepPlan, apiError, envelope (+10 more)

### Community 1 - "package.json"
Cohesion: 0.09
Nodes (22): clsx, eslint, @eslint/js, eslint-plugin-vue, globals, jsdom, @mediapipe/tasks-vision, shadcn-vue (+14 more)

### Community 2 - "newTeacherEnv"
Cohesion: 0.13
Nodes (23): rawResult, TestE2ERealFaceService(), env, tEnv, TestStudentCannotSeeOtherGroupsAssignment(), TestStudentDeadlineEnforcedByServer(), TestStudentProfileHidesPassportAndServesPhoto(), TestStudentSubmissionLifecycle() (+15 more)

### Community 3 - "SchedulesPage.vue"
Cohesion: 0.06
Nodes (30): Schedule, clientValidate(), dayKeys, deleteError, editing, fieldErrors, form, formError (+22 more)

### Community 4 - "components.json"
Cohesion: 0.09
Nodes (22): aliases, components, composables, lib, ui, utils, font, iconLibrary (+14 more)

### Community 5 - "CLAUDE.md — Internship Monitoring Platform"
Cohesion: 0.05
Nodes (43): API conventions, Architecture, Assignment rules, assignments, attendance_attempts, Attendance rules (the most important business logic), attendance_steps, attendances (+35 more)

### Community 6 - "dependencies"
Cohesion: 0.10
Nodes (20): dependencies, class-variance-authority, clsx, leaflet, @lucide/vue, @mediapipe/tasks-vision, pinia, reka-ui (+12 more)

### Community 7 - "compilerOptions"
Cohesion: 0.12
Nodes (16): compilerOptions, allowImportingTsExtensions, erasableSyntaxOnly, lib, module, moduleDetection, noEmit, noFallthroughCasesInSwitch (+8 more)

### Community 8 - "AssignmentService"
Cohesion: 0.25
Nodes (7): Assignment, AssignmentInput, AssignmentService, mustRoster(), AssignmentDTO, SubmissionsView, Upload

### Community 9 - "OAuth"
Cohesion: 0.07
Nodes (21): Client, EmbedResult, VerifyResult, FacultiesHTTP, facultiesPage, flex, flexID, GroupsHTTP (+13 more)

### Community 10 - "UsersPage.vue"
Cohesion: 0.02
Nodes (101): audit, open, from, page, props, to, copied, open (+93 more)

### Community 11 - "compilerOptions"
Cohesion: 0.14
Nodes (13): @vue/tsconfig/tsconfig.dom.json, compilerOptions, allowArbitraryExtensions, erasableSyntaxOnly, noFallthroughCasesInSwitch, noUnusedLocals, noUnusedParameters, paths (+5 more)

### Community 12 - "Expected"
Cohesion: 0.21
Nodes (9): Days, Schedule, DateRange(), Expected(), date(), TestDateRangeAndTotals(), TestExpectedSundayNeverEvenIfFlagged(), TestExpectedWeekdaysAndBoundaries() (+1 more)

### Community 13 - "Actor"
Cohesion: 0.06
Nodes (25): Handler, styles, Attendance(), bptr(), cell(), Grades(), newStyles(), open() (+17 more)

### Community 14 - "SidebarMenuButton.vue"
Cohesion: 0.09
Nodes (15): SidebarMenuButtonVariants, delegatedProps, { isMobile, state }, props, props, SidebarMenuButtonProps, emits, forwarded (+7 more)

### Community 16 - "tsconfig.json"
Cohesion: 0.40
Nodes (4): compilerOptions, paths, files, references

### Community 20 - "Handler"
Cohesion: 0.22
Nodes (8): UserResponse, actorOf(), Handler, meta(), toResponse(), CurrentUser(), SessionID(), Actor

### Community 21 - "reka-ui"
Cohesion: 0.05
Nodes (40): reka-ui, emits, forwarded, props, delegatedProps, emits, forwarded, props (+32 more)

### Community 22 - ".upsertStudent"
Cohesion: 0.09
Nodes (15): insertAttempt(), pgFloat(), placeholderPNG(), TestPlaceholderIsAValidPNG(), FailureSummary(), failureSummary(), nullUUIDValue(), uuidNull() (+7 more)

### Community 23 - "NewServer"
Cohesion: 0.16
Nodes (22): Services, Deps, FacultySource, GroupSource, New(), accessLog(), healthHandler(), mountSPA() (+14 more)

### Community 24 - "select/index.ts"
Cohesion: 0.05
Nodes (26): emits, forwarded, props, delegatedProps, emits, forwarded, props, delegatedProps (+18 more)

### Community 25 - "newHemisEnv"
Cohesion: 0.27
Nodes (15): hEnv, aliProfile(), newHemisEnv(), TestDevStudentPasswordLoginWorksAlongsideHemis(), TestHemisAdoptsSeededStudentAndProtectsUsernames(), TestHemisFirstLoginCreatesStudentFacultyAndGroup(), TestHemisInactiveStudentCannotLogIn(), TestHemisOutageIsReportedClearlyAndNotCountedAgainstTheUser() (+7 more)

### Community 26 - "UserDetailPage.vue"
Cohesion: 0.05
Nodes (41): errorMessage(), block, del, sync, del, sync, del, del (+33 more)

### Community 27 - "AttendanceReport.vue"
Cohesion: 0.05
Nodes (52): Attempt, AttemptDay, DayStatus, fetchAttempts(), fetchReport(), Report, ReportCell, ReportStudent (+44 more)

### Community 28 - "@vueuse/core"
Cohesion: 0.05
Nodes (28): @vueuse/core, delegatedProps, props, delegatedProps, forwardedProps, props, delegatedProps, forwardedProps (+20 more)

### Community 29 - "Calendar.vue"
Cohesion: 0.04
Nodes (28): emit, label, open, pick(), props, value, [DefineMonthTemplate, ReuseMonthTemplate], [DefineYearTemplate, ReuseYearTemplate] (+20 more)

### Community 30 - "AttemptsPage.vue"
Cohesion: 0.05
Nodes (27): AdminAttempt, search, cls, labels, BadgeVariants, blockedIds, blockError, confirmBlock (+19 more)

### Community 31 - "Config"
Cohesion: 0.14
Nodes (21): createAdmin(), resetPassword(), seed(), seedDemo(), main(), serve(), usage(), Config (+13 more)

### Community 32 - "StudentAssignmentsPage.vue"
Cohesion: 0.15
Nodes (12): clockOffset(), formatTimeLeft(), busy, { data, isPending, error }, errors, offset, okId, onFile() (+4 more)

### Community 33 - "New"
Cohesion: 0.16
Nodes (11): New(), auditDTO(), Report, ReportService, todayStats(), AuditDTO, AuditFilter, GroupStudentToday (+3 more)

### Community 34 - "faceGuard.test.ts"
Cohesion: 0.15
Nodes (17): deg(), evaluateFace(), GuideKey, headPose(), LIMITS, Pose, STABLE_MS, StableTimer (+9 more)

### Community 35 - "AuthService"
Cohesion: 0.18
Nodes (6): TestHashAndCheck(), CheckPassword(), Client, AuthService, limiterKey(), LoginResult

### Community 36 - "FaceCapture.vue"
Cohesion: 0.14
Nodes (18): emit, loading, message, progress, props, show(), start(), state (+10 more)

### Community 37 - "context.Context"
Cohesion: 0.04
Nodes (24): Attendance, CountGroupsParams, CreateLocationParams, DeleteUserSessionsExceptParams, GetGroupForTeacherParams, ListExpiredSelfiesParams, ListExpiredSelfiesRow, ListFacultiesParams (+16 more)

### Community 38 - "newAttemptEnv"
Cohesion: 0.16
Nodes (25): aEnv, attemptResp, files(), TestRunCreatesVariedAttemptsForTheSeededStudents(), TestRunIsIdempotentAndResettable(), newAttemptEnv(), strs(), TestAdminAttemptsListAndFilters() (+17 more)

### Community 39 - "NotFound"
Cohesion: 0.26
Nodes (10): Error, Forbidden(), New(), NotFound(), Unauthorized(), TestTokenHash(), HashToken(), NewToken() (+2 more)

### Community 40 - "avatar/index.ts"
Cohesion: 0.18
Nodes (7): class-variance-authority, props, delegatedProps, props, props, props, AvatarVariants

### Community 41 - "Totals"
Cohesion: 0.18
Nodes (8): Status, Totals, StatusFromSteps(), TestStatusFromSteps(), statusText(), ReportCell, MonthView, ReportStudent

### Community 43 - "github.com/jackc/pgx/v5/pgtype.Date"
Cohesion: 0.08
Nodes (19): CountSchedulesParams, CreateScheduleParams, DayAttendanceStatsRow, EnsureAttendanceParams, GetTodayAttendanceParams, ListAttendanceSummaryParams, ListAttendanceSummaryRow, ListOverlapCandidatesParams (+11 more)

### Community 44 - "github.com/google/uuid.NullUUID"
Cohesion: 0.04
Nodes (51): AttendanceAttempt, CountAttemptsAdminParams, CountAuditLogsParams, CountUsersParams, CreateAttemptParams, CreateAuditLogParams, CreateFacultyParams, CreateGroupParams (+43 more)

### Community 45 - "NewErrorHandler"
Cohesion: 0.18
Nodes (4): codeForStatus(), messageForStatus(), NewErrorHandler(), Dist()

### Community 46 - "GroupsPage.vue"
Cohesion: 0.04
Nodes (51): attempts, AuditEntry, crud(), Dashboard, DayStats, faculties, Faculty, Group (+43 more)

### Community 48 - "test_api.py"
Cohesion: 0.22
Nodes (17): png_bytes(), post_embed(), post_verify(), test_concurrency_is_limited_by_the_semaphore(), test_embed_multiple_faces(), test_embed_no_face(), test_embed_returns_vector(), test_embed_without_anti_spoofing_ignores_liveness() (+9 more)

### Community 49 - "devDependencies"
Cohesion: 0.12
Nodes (16): devDependencies, eslint, @eslint/js, eslint-plugin-vue, globals, jsdom, @types/leaflet, @types/node (+8 more)

### Community 50 - "main.py"
Cohesion: 0.09
Nodes (10): decode_image(), EmbedResult, FaceError, InvalidEmbedding, InvalidImage, MultipleFaces, NoFace, VerifyResult (+2 more)

### Community 51 - "testing.T"
Cohesion: 0.07
Nodes (44): OAuthConfig, TestLoginLimiter(), NewLoginLimiter(), TestAttendanceCellsAndTotals(), TestAttendanceFreezesHeaderAndNames(), TestAttendanceHasATitleBlockAndAHeaderWithWeekdays(), TestAttendanceStatusColoursAndGrayOffDays(), New() (+36 more)

### Community 52 - "time.Time"
Cohesion: 0.15
Nodes (15): Clock(), User, CleanupSelfies(), tstz(), tsPtr(), buildStudentAssignment(), StudentService, assignmentRow (+7 more)

### Community 53 - "mockFace"
Cohesion: 0.22
Nodes (6): mockFace, tEnv, jsonHas(), newMockFace(), TestAdminEnrollsStudentPhoto(), toJSON()

### Community 54 - "Load"
Cohesion: 0.33
Nodes (7): Load(), setBase(), TestLoadDefaults(), TestLoadValidation(), TestOAuthConfig(), TestProdWithSaneSettingsLoads(), TestTrustedProxies()

### Community 55 - "LoginLimiter"
Cohesion: 0.17
Nodes (7): Faculty, LoginLimiter, FakeFaculties, NewFakeFaculties(), newFacultySyncEnv(), TestFacultySyncCreatesRenamesAdoptsAndKeepsGroups(), TestFacultySyncFailuresAndAccess()

### Community 56 - "FileService"
Cohesion: 0.40
Nodes (4): Handler, New(), FileService, NewFileService()

### Community 57 - "newEnv"
Cohesion: 0.16
Nodes (17): errCode(), env, newEnv(), newEnvDeps(), TestGeneratedPasswordCanSkipTheForcedChange(), TestLoginFailures(), TestLoginMeLogout(), TestLoginRateLimit() (+9 more)

### Community 58 - "StudentHomePage.vue"
Cohesion: 0.12
Nodes (26): businessMinutes(), clock(), hourMarks(), layoutSteps(), markerTop(), PX_PER_MIN, Rect, stepLook (+18 more)

### Community 59 - "admin_test.go"
Cohesion: 0.33
Nodes (8): adminEnv, newAdminEnv(), TestAdminCRUDIsAudited(), TestDashboardCounts(), TestLocationPolygonValidation(), TestScheduleDBConstraints(), TestScheduleValidationAndOverlap(), TestUserAdminActions()

### Community 60 - "Write"
Cohesion: 0.05
Nodes (42): Validation(), Write(), TestGeneratePassword(), GeneratePassword(), HashPassword(), ValidGrade(), finite(), facultyDTO() (+34 more)

### Community 61 - "verify"
Cohesion: 0.24
Nodes (6): embed(), face_error_handler(), health(), http_error_handler(), read_image(), verify()

### Community 62 - "loadtest.py"
Cohesion: 0.13
Nodes (5): call(), main(), one(), multipart(), read()

### Community 63 - "Point"
Cohesion: 0.19
Nodes (21): Point, Result, ParsePoint(), ParsePoints(), TestParsePoint(), TestParsePoints(), Check(), Contains() (+13 more)

### Community 64 - "vitest"
Cohesion: 0.06
Nodes (24): vitest, @vue/test-utils, User, TodayStep, options, options, { theme, setTheme }, applyTheme() (+16 more)

### Community 66 - "HemisPhotoService"
Cohesion: 0.21
Nodes (7): TeacherOAuth, TestPhotoFetcherRules(), PhotoFetcher, NewPhotoFetcher(), NewHemisPhotoService(), HemisPhotoService, teacherOAuth

### Community 67 - "newSyncEnv"
Cohesion: 0.14
Nodes (14): Group, syncEnv, FakeGroups, NewFakeGroups(), TestFakeGroups(), counts(), newSyncEnv(), TestConcurrentSyncsDoNotDuplicateGroups() (+6 more)

### Community 68 - "newOAuthEnv"
Cohesion: 0.21
Nodes (14): oEnv, fieldsOf(), newOAuthEnv(), TestOAuthAutoCreateOff(), TestOAuthCallbackIsRateLimitedPerIP(), TestOAuthCallbackNotServedBySPA(), TestOAuthErrorCodes(), TestOAuthHappyPathCreatesTeacherAndSession() (+6 more)

### Community 71 - "SidebarProvider.vue"
Cohesion: 0.16
Nodes (15): emits, isMobile, open, openMobile, props, setOpen(), setOpenMobile(), state (+7 more)

### Community 75 - "VerifyPage.vue"
Cohesion: 0.04
Nodes (61): pinia, Phase 0. Project skeleton, Phase 1. SQL models and login+password authentication, Phase 2. Admin panel, Phase 3. Teacher panel, Phase 4. Student pages, Phase 5. FaceID + location verification, Phase 6. HEMIS integration (+53 more)

### Community 76 - "uz.ts"
Cohesion: 0.04
Nodes (41): @tanstack/vue-query, vue-router, groups, teacher, NavLink, route, router, session (+33 more)

### Community 77 - "LoginPage.vue"
Cohesion: 0.06
Nodes (32): vee-validate, @vee-validate/zod, zod, date, emit, props, set(), time (+24 more)

### Community 78 - "scripts"
Cohesion: 0.33
Nodes (6): scripts, build, dev, lint, preview, test

### Community 79 - "FaceCapture.test.ts"
Cohesion: 0.15
Nodes (10): detector, frame(), getUserMedia, goodFace, identity, lib, queue, run() (+2 more)

### Community 80 - "LocationMap.vue"
Cohesion: 0.40
Nodes (3): leaflet, el, props

### Community 81 - "conftest.py"
Cohesion: 0.09
Nodes (7): fake(), fake_client(), _token(), FaceNotDetected, FakeDeepFace, FakeSpoofModel, SpoofDetected

### Community 82 - "vue"
Cohesion: 0.05
Nodes (50): vue, props, props, props, props, props, props, props (+42 more)

### Community 83 - "@lucide/vue"
Cohesion: 0.07
Nodes (29): @lucide/vue, Props, ButtonVariants, delegatedProps, forwardedProps, props, delegatedProps, forwardedProps (+21 more)

### Community 88 - "PopoverContent.vue"
Cohesion: 0.10
Nodes (12): emits, forwarded, props, props, delegatedProps, emits, forwarded, props (+4 more)

### Community 90 - "Queries"
Cohesion: 0.14
Nodes (13): ensureUser(), Queries, Schedule, ensureSchedule(), dateStr(), scheduleDTO(), ScheduleService, CreateUserInput (+5 more)

### Community 91 - "test_model.py"
Cohesion: 0.29
Nodes (6): client(), embed(), photo(), test_replayed_photo_is_rejected_by_anti_spoofing(), test_same_person_verifies_and_other_person_does_not(), verify()

### Community 92 - "Sidebar.vue"
Cohesion: 0.09
Nodes (15): emits, forwarded, props, props, delegatedProps, props, props, props (+7 more)

### Community 95 - "SheetContent.vue"
Cohesion: 0.22
Nodes (7): delegatedProps, emits, forwarded, props, SheetContentProps, delegatedProps, props

### Community 96 - "Storage"
Cohesion: 0.13
Nodes (18): Options, Summary, ensureLocation(), reset(), Run(), setup(), Service, New() (+10 more)

### Community 97 - "AssignmentsPage.vue"
Cohesion: 0.04
Nodes (32): Assignment, GroupStudentToday, RosterEntry, SubmissionsView, TeacherGroup, TeacherGroupDetail, TodayStats, emit (+24 more)

### Community 99 - "shifts.go"
Cohesion: 0.25
Nodes (12): StepState, Window, minutesOfDay(), ShiftWindows(), StateOf(), at(), TestClock(), TestStateOf() (+4 more)

### Community 100 - "dialog/index.ts"
Cohesion: 0.06
Nodes (23): emits, forwarded, props, props, delegatedProps, emits, forwarded, props (+15 more)

### Community 102 - "SelectField.vue"
Cohesion: 0.22
Nodes (4): emit, onChange(), Option, props

### Community 112 - "Fake"
Cohesion: 0.27
Nodes (4): fakeAccount, Fake, NewFake(), TestFakeBehavesLikeTheClient()

### Community 114 - "verify.ts"
Cohesion: 0.18
Nodes (14): Today, accuracyLevel(), cameraErrorMessage(), canVerify(), failureGuidance(), Guidance, MAX_CAPTURE_PX, scaleToFit() (+6 more)

### Community 115 - "LocationStatusCard.vue"
Cohesion: 0.21
Nodes (10): failed, hint, poor, props, { state, accuracy, refresh }, title, GeoState, useGeoStatus() (+2 more)

### Community 116 - "github.com/google/uuid.UUID"
Cohesion: 0.05
Nodes (34): AttendanceStep, CountTeacherAssignmentsParams, CreateAssignmentParams, CreateSubmissionParams, EnsureStepParams, GetAssignmentForTeacherParams, GetAttemptOwnerRow, GetStepParams (+26 more)

### Community 117 - "NativeSelect.vue"
Cohesion: 0.20
Nodes (6): delegatedProps, emit, modelValue, props, props, props

### Community 118 - "Changes"
Cohesion: 0.36
Nodes (8): Changes(), equal(), HasChanges(), decode(), TestChangesDiffOnlyChangedFields(), TestChangesSensitiveChangeIsInvisible(), TestChangesStripsSensitive(), toMap()

### Community 121 - ".Open"
Cohesion: 0.17
Nodes (7): downloadName(), TestMaxSizeAndMIME(), IsImage(), MaxSize(), MIMEForExt(), FileResult, Local

## Knowledge Gaps
- **815 isolated node(s):** `face-service`, `fetch_photos.sh script`, `pc`, `backup.sh script`, `restore.sh script` (+810 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 1111 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **12 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `vue` connect `vue` to `package.json`, `SchedulesPage.vue`, `UsersPage.vue`, `SidebarMenuButton.vue`, `reka-ui`, `select/index.ts`, `UserDetailPage.vue`, `AttendanceReport.vue`, `@vueuse/core`, `Calendar.vue`, `AttemptsPage.vue`, `StudentAssignmentsPage.vue`, `FaceCapture.vue`, `avatar/index.ts`, `GroupsPage.vue`, `StudentHomePage.vue`, `vitest`, `SidebarProvider.vue`, `VerifyPage.vue`, `uz.ts`, `LoginPage.vue`, `LocationMap.vue`, `@lucide/vue`, `PopoverContent.vue`, `Sidebar.vue`, `SheetContent.vue`, `AssignmentsPage.vue`, `dialog/index.ts`, `LocationStatusCard.vue`, `NativeSelect.vue`?**
  _High betweenness centrality (0.079) - this node is a cross-community bridge._
- **Why does `lifespan()` connect `main.py` to `VerifyPage.vue`, `CLAUDE.md — Internship Monitoring Platform`?**
  _High betweenness centrality (0.048) - this node is a cross-community bridge._
- **What connects `face-service`, `fetch_photos.sh script`, `pc` to the rest of the system?**
  _815 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `server.go` be split into smaller, more focused modules?**
  _Cohesion score 0.11757641265837987 - nodes in this community are weakly interconnected._
- **Should `package.json` be split into smaller, more focused modules?**
  _Cohesion score 0.09420289855072464 - nodes in this community are weakly interconnected._
- **Should `newTeacherEnv` be split into smaller, more focused modules?**
  _Cohesion score 0.12941176470588237 - nodes in this community are weakly interconnected._
- **Should `SchedulesPage.vue` be split into smaller, more focused modules?**
  _Cohesion score 0.05689900426742532 - nodes in this community are weakly interconnected._