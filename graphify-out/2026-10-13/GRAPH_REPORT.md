# Graph Report - pc  (2026-10-13)

## Corpus Check
- 415 files · ~137,261 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 12 file(s) not represented in the graph (top: (none) 7, .lock 2, .example 1)

## Summary
- 2980 nodes · 9381 edges · 125 communities (113 shown, 12 thin omitted)
- Extraction: 97% EXTRACTED · 3% INFERRED · 0% AMBIGUOUS · INFERRED: 324 edges (avg confidence: 0.86)
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
- Queries
- OAuth
- UsersPage.vue
- compilerOptions
- rules_test.go
- Handler
- SidebarMenuButton.vue
- Handler
- tsconfig.json
- face-service
- pc
- New
- reka-ui
- me
- admin.ts
- select/index.ts
- newHemisEnv
- UserDetailPage.vue
- AttendanceReport.vue
- @vueuse/core
- Calendar.vue
- AttemptsPage.vue
- Attendance
- Checkbox.vue
- nullUUID
- faceGuard.test.ts
- HEMIS integration (phase 6)
- FaceCapture.vue
- context.Context
- newAttemptEnv
- Data model
- CurrentUser
- LocationsPage.vue
- AuditPage.vue
- github.com/jackc/pgx/v5/pgtype.Date
- github.com/google/uuid.UUID
- OptUUID
- GroupsPage.vue
- PointError
- test_api.py
- devDependencies
- engine.py
- testing.T
- time.Time
- TestAdminEnrollsStudentPhoto
- RadioGroup.vue
- FakeFaculties
- ThemeToggle.vue
- env
- StudentHomePage.vue
- newAdminEnv
- Actor
- verify
- main.py
- Point
- @tanstack/vue-query
- Engine
- Config
- newSyncEnv
- newOAuthEnv
- VerifyPage.vue
- FacultiesPage.vue
- SidebarProvider.vue
- StudentProfilePage.vue
- AppSidebar.vue
- RoleLayout.vue
- form.ts
- uz.ts
- ChangePasswordPage.vue
- scripts
- FaceCapture.test.ts
- LocationMap.vue
- fakes.py
- vue
- @lucide/vue
- fetch_photos.sh
- TeacherGroupPage.vue
- PasswordDialog.vue
- PopoverContent.vue
- vite.config.ts
- .validate
- test_model.py
- Sidebar.vue
- session.ts
- teacher.ts
- SheetContent.vue
- NewServer
- AssignmentsPage.vue
- PaginationBar.vue
- shifts.go
- dialog/index.ts
- TASKS.md
- SelectField.vue
- AssignmentDetailPage.vue
- VerifyPage.test.ts
- mediapipe.ts
- web_src_components_ui_button_index_button
- badge/index.ts
- student.ts
- applyApiError
- useListQuery.ts
- useRowSelection
- Fake
- LoginPage.vue
- verify.ts
- LocationStatusCard.vue
- Queries
- NativeSelect.vue
- Changes
- BarChart.vue
- StudentHomePage.test.ts
- Local
- SchedulesPage.test.ts
- ReportsPage.vue

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
- `Resource owner response and mapping` --references--> `login()`  [INFERRED]
  CLAUDE.md → web/src/api/auth.ts
- `Face-service contract` --references--> `lifespan()`  [INFERRED]
  CLAUDE.md → face-service/app/main.py
- `Phase 5. FaceID + location verification` --references--> `lifespan()`  [INFERRED]
  TASKS.md → face-service/app/main.py
- `HEMIS login flow` --references--> `login()`  [INFERRED]
  CLAUDE.md → web/src/api/auth.ts
- `Phase 1. SQL models and login+password authentication` --references--> `login()`  [INFERRED]
  TASKS.md → web/src/api/auth.ts

## Import Cycles
- None detected.

## Communities (125 total, 12 thin omitted)

### Community 0 - "server.go"
Cohesion: 0.11
Nodes (21): createUserBody, nameBody, changePasswordRequest, loginRequest, kind, stepPlan, apiError, envelope (+13 more)

### Community 1 - "package.json"
Cohesion: 0.09
Nodes (22): clsx, eslint, @eslint/js, eslint-plugin-vue, globals, jsdom, @mediapipe/tasks-vision, shadcn-vue (+14 more)

### Community 2 - "newTeacherEnv"
Cohesion: 0.15
Nodes (22): rawResult, TestE2ERealFaceService(), env, tEnv, TestStudentCannotSeeOtherGroupsAssignment(), TestStudentDeadlineEnforcedByServer(), TestStudentProfileHidesPassportAndServesPhoto(), TestStudentSubmissionLifecycle() (+14 more)

### Community 3 - "SchedulesPage.vue"
Cohesion: 0.06
Nodes (26): clientValidate(), dayKeys, deleteError, editing, fieldErrors, form, formError, formOpen (+18 more)

### Community 4 - "components.json"
Cohesion: 0.09
Nodes (22): aliases, components, composables, lib, ui, utils, font, iconLibrary (+14 more)

### Community 5 - "CLAUDE.md — Internship Monitoring Platform"
Cohesion: 0.11
Nodes (17): API conventions, Architecture, Assignment rules, Attendance rules (the most important business logic), CLAUDE.md — Internship Monitoring Platform, Code rules, Commands, compose.yaml requirements (+9 more)

### Community 6 - "dependencies"
Cohesion: 0.10
Nodes (20): dependencies, class-variance-authority, clsx, leaflet, @lucide/vue, @mediapipe/tasks-vision, pinia, reka-ui (+12 more)

### Community 7 - "compilerOptions"
Cohesion: 0.12
Nodes (16): compilerOptions, allowImportingTsExtensions, erasableSyntaxOnly, lib, module, moduleDetection, noEmit, noFallthroughCasesInSwitch (+8 more)

### Community 8 - "Queries"
Cohesion: 0.11
Nodes (18): ListAssignmentRosterParams, Submission, ListAssignmentRosterRow, Queries, Assignment, AssignmentInput, AssignmentService, GradesExport (+10 more)

### Community 9 - "OAuth"
Cohesion: 0.09
Nodes (18): facultiesPage, flex, flexID, groupsPage, HTTP, meJSON, OAuth, OAuthConfig (+10 more)

### Community 10 - "UsersPage.vue"
Cohesion: 0.06
Nodes (28): actionError, active, activeOpts, created, { defineField, handleSubmit, errors, setErrors, resetForm, isSubmitting }, extra, facultyId, facultyOptions (+20 more)

### Community 11 - "compilerOptions"
Cohesion: 0.14
Nodes (13): @vue/tsconfig/tsconfig.dom.json, compilerOptions, allowArbitraryExtensions, erasableSyntaxOnly, noFallthroughCasesInSwitch, noUnusedLocals, noUnusedParameters, paths (+5 more)

### Community 12 - "rules_test.go"
Cohesion: 0.11
Nodes (17): Days, Schedule, Status, DateRange(), Expected(), Totals, StatusFromSteps(), date() (+9 more)

### Community 13 - "Handler"
Cohesion: 0.16
Nodes (3): Handler, Body(), IDParam()

### Community 14 - "SidebarMenuButton.vue"
Cohesion: 0.09
Nodes (15): SidebarMenuButtonVariants, delegatedProps, { isMobile, state }, props, props, SidebarMenuButtonProps, emits, forwarded (+7 more)

### Community 16 - "tsconfig.json"
Cohesion: 0.40
Nodes (4): compilerOptions, paths, files, references

### Community 20 - "New"
Cohesion: 0.09
Nodes (19): Error, UserResponse, Forbidden(), New(), Unauthorized(), TestTokenHash(), HashToken(), NewToken() (+11 more)

### Community 21 - "reka-ui"
Cohesion: 0.05
Nodes (40): reka-ui, emits, forwarded, props, delegatedProps, emits, forwarded, props (+32 more)

### Community 22 - "me"
Cohesion: 0.20
Nodes (6): Handler, parseAssignmentForm(), Handler, me(), New(), sendXLSX()

### Community 23 - "admin.ts"
Cohesion: 0.11
Nodes (20): attempts, crud(), Dashboard, faculties, Faculty, Group, GroupDetail, GroupInput (+12 more)

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
Cohesion: 0.06
Nodes (40): fetchAttempts(), fetchReport(), student, { data, isPending, error }, from, props, search, to (+32 more)

### Community 28 - "@vueuse/core"
Cohesion: 0.06
Nodes (24): @vueuse/core, delegatedProps, props, delegatedProps, forwardedProps, props, delegatedProps, forwardedProps (+16 more)

### Community 29 - "Calendar.vue"
Cohesion: 0.04
Nodes (28): emit, label, open, pick(), props, value, [DefineMonthTemplate, ReuseMonthTemplate], [DefineYearTemplate, ReuseYearTemplate] (+20 more)

### Community 30 - "AttemptsPage.vue"
Cohesion: 0.06
Nodes (25): formatDate(), formatDateTime(), formatTime(), pct(), blockedIds, blockError, confirmBlock, extra (+17 more)

### Community 31 - "Attendance"
Cohesion: 0.18
Nodes (10): styles, Attendance(), bptr(), cell(), Grades(), newStyles(), open(), sample() (+2 more)

### Community 32 - "Checkbox.vue"
Cohesion: 0.33
Nodes (4): delegatedProps, emits, forwarded, props

### Community 33 - "nullUUID"
Cohesion: 0.07
Nodes (22): Faculty, Location, auditDTO(), facultyDTO(), ListParams, locationDTO(), newPage(), nullUUID() (+14 more)

### Community 34 - "faceGuard.test.ts"
Cohesion: 0.15
Nodes (17): deg(), evaluateFace(), GuideKey, headPose(), LIMITS, Pose, STABLE_MS, StableTimer (+9 more)

### Community 35 - "HEMIS integration (phase 6)"
Cohesion: 0.17
Nodes (11): Face enrollment from the HEMIS photo, Field mapping (`me.data` → our tables), Find or create, Flow, HEMIS API, HEMIS integration (phase 6), HEMIS login flow, HEMIS OAuth for teachers (phase 7) (+3 more)

### Community 36 - "FaceCapture.vue"
Cohesion: 0.19
Nodes (15): emit, loading, message, progress, props, show(), start(), state (+7 more)

### Community 37 - "context.Context"
Cohesion: 0.04
Nodes (33): CountGroupsParams, CreateFacultyParams, CreateGroupParams, CreateHemisStudentParams, CreateHemisTeacherParams, CreateLocationParams, DeleteUserSessionsExceptParams, GetGroupForTeacherParams (+25 more)

### Community 38 - "newAttemptEnv"
Cohesion: 0.14
Nodes (26): aEnv, attemptResp, mockFace, files(), TestRunCreatesVariedAttemptsForTheSeededStudents(), TestRunIsIdempotentAndResettable(), newAttemptEnv(), strs() (+18 more)

### Community 39 - "Data model"
Cohesion: 0.15
Nodes (13): assignments, attendance_attempts, attendance_steps, attendances, audit_logs, Data model, faculties, groups (+5 more)

### Community 40 - "CurrentUser"
Cohesion: 0.25
Nodes (3): New(), CurrentUser(), Handler

### Community 41 - "LocationsPage.vue"
Cohesion: 0.09
Nodes (20): parsePoints(), { defineField, handleSubmit, errors, setErrors, resetForm, isSubmitting }, deleteError, editing, formError, formOpen, [name, nameAttrs], onSubmit (+12 more)

### Community 42 - "AuditPage.vue"
Cohesion: 0.11
Nodes (15): audit, AuditEntry, action, actionOpts, actorId, detail, diffRows, entity (+7 more)

### Community 43 - "github.com/jackc/pgx/v5/pgtype.Date"
Cohesion: 0.08
Nodes (19): Attendance, CountSchedulesParams, CreateScheduleParams, DayAttendanceStatsRow, GetAttemptOwnerRow, GetTodayAttendanceParams, ListAttendanceSummaryParams, ListAttendanceSummaryRow (+11 more)

### Community 44 - "github.com/google/uuid.UUID"
Cohesion: 0.04
Nodes (63): AttendanceAttempt, AttendanceStep, CountAttemptsAdminParams, CountAuditLogsParams, CountTeacherAssignmentsParams, CountUsersParams, CreateAssignmentParams, CreateAttemptParams (+55 more)

### Community 46 - "GroupsPage.vue"
Cohesion: 0.09
Nodes (21): { defineField, handleSubmit, errors, setErrors, resetForm, isSubmitting }, deleteError, editing, extra, facultyFilter, [facultyId], facultyOptions, fOpts (+13 more)

### Community 48 - "test_api.py"
Cohesion: 0.18
Nodes (18): png_bytes(), post_embed(), post_verify(), test_concurrency_is_limited_by_the_semaphore(), test_embed_multiple_faces(), test_embed_no_face(), test_embed_returns_vector(), test_embed_without_anti_spoofing_ignores_liveness() (+10 more)

### Community 49 - "devDependencies"
Cohesion: 0.12
Nodes (16): devDependencies, eslint, @eslint/js, eslint-plugin-vue, globals, jsdom, @types/leaflet, @types/node (+8 more)

### Community 50 - "engine.py"
Cohesion: 0.12
Nodes (8): decode_image(), EmbedResult, FaceError, InvalidEmbedding, InvalidImage, MultipleFaces, NoFace, VerifyResult

### Community 51 - "testing.T"
Cohesion: 0.08
Nodes (40): TestGeneratePassword(), placeholderPNG(), TestPlaceholderIsAValidPNG(), TestAttendanceCellsAndTotals(), TestAttendanceFreezesHeaderAndNames(), TestAttendanceHasATitleBlockAndAHeaderWithWeekdays(), TestAttendanceStatusColoursAndGrayOffDays(), New() (+32 more)

### Community 52 - "time.Time"
Cohesion: 0.07
Nodes (24): New(), TestStudentTodayView(), CleanupSelfies(), TestCleanupSelfiesRemovesOldFilesButKeepsRows(), tstz(), Report, ReportService, todayStats() (+16 more)

### Community 53 - "TestAdminEnrollsStudentPhoto"
Cohesion: 0.33
Nodes (5): tEnv, jsonHas(), newMockFace(), TestAdminEnrollsStudentPhoto(), toJSON()

### Community 54 - "RadioGroup.vue"
Cohesion: 0.20
Nodes (7): delegatedProps, emits, forwarded, props, delegatedProps, forwardedProps, props

### Community 55 - "FakeFaculties"
Cohesion: 0.10
Nodes (16): Client, EmbedResult, VerifyResult, FacultiesHTTP, Faculty, GroupsHTTP, unwrapNet(), FakeFaculties (+8 more)

### Community 56 - "ThemeToggle.vue"
Cohesion: 0.30
Nodes (10): options, { theme, setTheme }, applyTheme(), initialTheme(), readStored(), _resetThemeForTests(), setTheme(), systemPrefersDark() (+2 more)

### Community 57 - "env"
Cohesion: 0.22
Nodes (14): errCode(), env, newEnv(), newEnvDeps(), TestGeneratedPasswordCanSkipTheForcedChange(), TestLoginFailures(), TestLoginMeLogout(), TestLoginRateLimit() (+6 more)

### Community 58 - "StudentHomePage.vue"
Cohesion: 0.07
Nodes (38): clockOffset(), formatTimeLeft(), businessMinutes(), hourMarks(), layoutSteps(), markerTop(), PX_PER_MIN, Rect (+30 more)

### Community 59 - "newAdminEnv"
Cohesion: 0.31
Nodes (8): adminEnv, newAdminEnv(), TestAdminCRUDIsAudited(), TestDashboardCounts(), TestLocationPolygonValidation(), TestScheduleDBConstraints(), TestScheduleValidationAndOverlap(), TestUserAdminActions()

### Community 60 - "Actor"
Cohesion: 0.06
Nodes (39): ensureUser(), Validation(), Write(), TestHashAndCheck(), CheckPassword(), GeneratePassword(), HashPassword(), ValidateNewPassword() (+31 more)

### Community 61 - "verify"
Cohesion: 0.24
Nodes (6): embed(), face_error_handler(), health(), http_error_handler(), read_image(), verify()

### Community 62 - "main.py"
Cohesion: 0.13
Nodes (4): Face-service contract, lifespan(), make_engine(), Phase 5. FaceID + location verification

### Community 63 - "Point"
Cohesion: 0.19
Nodes (21): Point, Result, ParsePoint(), ParsePoints(), TestParsePoint(), TestParsePoints(), Check(), Contains() (+13 more)

### Community 64 - "@tanstack/vue-query"
Cohesion: 0.08
Nodes (15): @tanstack/vue-query, vitest, @vue/test-utils, AdminAttempt, options, api, api, api (+7 more)

### Community 65 - "Engine"
Cohesion: 0.15
Nodes (5): Engine, fake(), fake_client(), _token(), test_health_is_unavailable_until_loaded()

### Community 66 - "Config"
Cohesion: 0.06
Nodes (29): serve(), Deps, TestLoginLimiter(), LoginLimiter, NewLoginLimiter(), Config, Load(), setBase() (+21 more)

### Community 67 - "newSyncEnv"
Cohesion: 0.15
Nodes (14): Group, syncEnv, FakeGroups, NewFakeGroups(), TestFakeGroups(), counts(), newSyncEnv(), TestConcurrentSyncsDoNotDuplicateGroups() (+6 more)

### Community 68 - "newOAuthEnv"
Cohesion: 0.20
Nodes (14): oEnv, fieldsOf(), newOAuthEnv(), TestOAuthAutoCreateOff(), TestOAuthCallbackIsRateLimitedPerIP(), TestOAuthCallbackNotServedBySPA(), TestOAuthErrorCodes(), TestOAuthHappyPathCreatesTeacherAndSession() (+6 more)

### Community 69 - "VerifyPage.vue"
Cohesion: 0.12
Nodes (19): geoErrorMessage(), again(), cameraActive, captureKey, { data: today, isPending }, Fix, geoError, gpsLimit (+11 more)

### Community 70 - "FacultiesPage.vue"
Cohesion: 0.11
Nodes (13): { defineField, handleSubmit, errors, setErrors, resetForm, isSubmitting }, deleteError, editing, formError, formOpen, [name, nameAttrs], qc, rows (+5 more)

### Community 71 - "SidebarProvider.vue"
Cohesion: 0.16
Nodes (15): emits, isMobile, open, openMobile, props, setOpen(), setOpenMobile(), state (+7 more)

### Community 72 - "StudentProfilePage.vue"
Cohesion: 0.17
Nodes (8): getDashboard(), { data, isPending }, { data, isPending, error }, initials, router, rows, session, { data, isPending }

### Community 73 - "AppSidebar.vue"
Cohesion: 0.11
Nodes (4): route, router, session, { setOpenMobile }

### Community 74 - "RoleLayout.vue"
Cohesion: 0.22
Nodes (4): current, hideOnPhone, props, route

### Community 75 - "form.ts"
Cohesion: 0.14
Nodes (15): changePassword(), CurrentUser, json(), logout(), me(), Role, api(), ApiError (+7 more)

### Community 76 - "uz.ts"
Cohesion: 0.18
Nodes (12): NavLink, links, links, links, uz, sid, session, adminChildren (+4 more)

### Community 77 - "ChangePasswordPage.vue"
Cohesion: 0.14
Nodes (11): vee-validate, @vee-validate/zod, zod, [confirmPassword, confirmAttrs], { defineField, handleSubmit, errors, setErrors, isSubmitting }, formError, [newPassword, newAttrs], [oldPassword, oldAttrs] (+3 more)

### Community 78 - "scripts"
Cohesion: 0.33
Nodes (6): scripts, build, dev, lint, preview, test

### Community 79 - "FaceCapture.test.ts"
Cohesion: 0.14
Nodes (11): detector, frame(), getUserMedia, goodFace, identity, lib, queue, run() (+3 more)

### Community 80 - "LocationMap.vue"
Cohesion: 0.40
Nodes (3): leaflet, el, props

### Community 81 - "fakes.py"
Cohesion: 0.13
Nodes (4): FaceNotDetected, FakeDeepFace, FakeSpoofModel, SpoofDetected

### Community 82 - "vue"
Cohesion: 0.04
Nodes (57): class-variance-authority, vue, props, props, delegatedProps, props, props, props (+49 more)

### Community 83 - "@lucide/vue"
Cohesion: 0.07
Nodes (29): @lucide/vue, Props, ButtonVariants, delegatedProps, forwardedProps, props, delegatedProps, forwardedProps (+21 more)

### Community 86 - "TeacherGroupPage.vue"
Cohesion: 0.17
Nodes (7): { data, isPending }, id, route, { data, isPending }, id, tab, tabs

### Community 87 - "PasswordDialog.vue"
Cohesion: 0.22
Nodes (4): open, copied, open, props

### Community 88 - "PopoverContent.vue"
Cohesion: 0.10
Nodes (12): emits, forwarded, props, props, delegatedProps, emits, forwarded, props (+4 more)

### Community 90 - ".validate"
Cohesion: 0.23
Nodes (7): Schedule, dateStr(), parseDate(), scheduleDTO(), ScheduleService, ScheduleDTO, ScheduleFilter

### Community 91 - "test_model.py"
Cohesion: 0.39
Nodes (5): embed(), photo(), test_replayed_photo_is_rejected_by_anti_spoofing(), test_same_person_verifies_and_other_person_does_not(), verify()

### Community 92 - "Sidebar.vue"
Cohesion: 0.09
Nodes (15): emits, forwarded, props, props, delegatedProps, props, props, props (+7 more)

### Community 93 - "session.ts"
Cohesion: 0.24
Nodes (7): pinia, links, logoutApi, mountAt(), Page, app, useSessionStore

### Community 94 - "teacher.ts"
Cohesion: 0.22
Nodes (8): Assignment, GroupStudentToday, RosterEntry, SubmissionsView, teacher, TeacherGroup, TeacherGroupDetail, TodayStats

### Community 95 - "SheetContent.vue"
Cohesion: 0.22
Nodes (7): delegatedProps, emits, forwarded, props, SheetContentProps, delegatedProps, props

### Community 96 - "NewServer"
Cohesion: 0.05
Nodes (62): Services, createAdmin(), resetPassword(), seed(), seedDemo(), main(), usage(), Options (+54 more)

### Community 97 - "AssignmentsPage.vue"
Cohesion: 0.06
Nodes (25): date, emit, props, set(), time, search, search, deleteError (+17 more)

### Community 98 - "PaginationBar.vue"
Cohesion: 0.20
Nodes (3): emit, multiple, props

### Community 99 - "shifts.go"
Cohesion: 0.24
Nodes (13): StepState, Window, Clock(), minutesOfDay(), ShiftWindows(), StateOf(), at(), TestClock() (+5 more)

### Community 100 - "dialog/index.ts"
Cohesion: 0.06
Nodes (23): emits, forwarded, props, props, delegatedProps, emits, forwarded, props (+15 more)

### Community 101 - "TASKS.md"
Cohesion: 0.20
Nodes (10): Phase 0. Project skeleton, Phase 1. SQL models and login+password authentication, Phase 2. Admin panel, Phase 3. Teacher panel, Phase 4. Student pages, Phase 6. HEMIS integration, Phase 7. HEMIS OAuth login for teachers, Phase 8. Production readiness (+2 more)

### Community 102 - "SelectField.vue"
Cohesion: 0.22
Nodes (4): emit, onChange(), Option, props

### Community 103 - "AssignmentDetailPage.vue"
Cohesion: 0.20
Nodes (8): { data, isPending }, fieldErrors, form, formError, GRADE_OPTIONS, grading, id, qc

### Community 104 - "VerifyPage.test.ts"
Cohesion: 0.22
Nodes (4): AttemptResult, api, FaceCaptureStub, seen

### Community 105 - "mediapipe.ts"
Cohesion: 0.38
Nodes (3): Landmark, FaceDetector, loadFaceLandmarker()

### Community 106 - "web_src_components_ui_button_index_button"
Cohesion: 0.33
Nodes (4): from, page, props, to

### Community 107 - "badge/index.ts"
Cohesion: 0.33
Nodes (3): cls, labels, BadgeVariants

### Community 108 - "student.ts"
Cohesion: 0.19
Nodes (12): Attempt, AttemptDay, DayStatus, Report, ReportCell, ReportStudent, StepAttempts, StudentAttempts (+4 more)

### Community 109 - "applyApiError"
Cohesion: 0.40
Nodes (5): applyApiError(), onSubmit, onSubmit, onSubmit, save

### Community 110 - "useListQuery.ts"
Cohesion: 0.50
Nodes (3): ListQuery, Page, useListQuery()

### Community 112 - "Fake"
Cohesion: 0.27
Nodes (4): fakeAccount, Fake, NewFake(), TestFakeBehavesLikeTheClient()

### Community 113 - "LoginPage.vue"
Cohesion: 0.12
Nodes (14): vue-router, { defineField, handleSubmit, errors, setErrors, isSubmitting }, formError, hemisLoading, oauthError, onSubmit, [password, passwordAttrs], route (+6 more)

### Community 114 - "verify.ts"
Cohesion: 0.21
Nodes (12): accuracyLevel(), cameraErrorMessage(), canVerify(), failureGuidance(), Guidance, MAX_CAPTURE_PX, scaleToFit(), step() (+4 more)

### Community 115 - "LocationStatusCard.vue"
Cohesion: 0.21
Nodes (10): failed, hint, poor, props, { state, accuracy, refresh }, title, GeoState, useGeoStatus() (+2 more)

### Community 116 - "Queries"
Cohesion: 0.24
Nodes (5): CreateSubmissionParams, GetSubmissionForUpdateParams, ListStepsWithAttemptCountsRow, ReplaceSubmissionFileParams, Queries

### Community 117 - "NativeSelect.vue"
Cohesion: 0.20
Nodes (6): delegatedProps, emit, modelValue, props, props, props

### Community 118 - "Changes"
Cohesion: 0.36
Nodes (8): Changes(), equal(), HasChanges(), decode(), TestChangesDiffOnlyChangedFields(), TestChangesSensitiveChangeIsInvisible(), TestChangesStripsSensitive(), toMap()

### Community 119 - "BarChart.vue"
Cohesion: 0.29
Nodes (7): DayStats, max, PAD, props, segs, slot, y()

### Community 120 - "StudentHomePage.test.ts"
Cohesion: 0.29
Nodes (5): Today, TodayStep, api, step(), today()

### Community 122 - "SchedulesPage.test.ts"
Cohesion: 0.40
Nodes (4): Schedule, api, page(), render()

### Community 123 - "ReportsPage.vue"
Cohesion: 0.40
Nodes (4): groups, gOpts, groupId, options

## Knowledge Gaps
- **812 isolated node(s):** `face-service`, `fetch_photos.sh script`, `pc`, `Queries`, `apiError` (+807 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 1101 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **12 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `vue` connect `vue` to `package.json`, `SchedulesPage.vue`, `UsersPage.vue`, `SidebarMenuButton.vue`, `reka-ui`, `select/index.ts`, `UserDetailPage.vue`, `AttendanceReport.vue`, `@vueuse/core`, `Calendar.vue`, `AttemptsPage.vue`, `Checkbox.vue`, `FaceCapture.vue`, `LocationsPage.vue`, `AuditPage.vue`, `GroupsPage.vue`, `RadioGroup.vue`, `ThemeToggle.vue`, `StudentHomePage.vue`, `VerifyPage.vue`, `FacultiesPage.vue`, `SidebarProvider.vue`, `StudentProfilePage.vue`, `AppSidebar.vue`, `RoleLayout.vue`, `form.ts`, `ChangePasswordPage.vue`, `LocationMap.vue`, `@lucide/vue`, `TeacherGroupPage.vue`, `PasswordDialog.vue`, `PopoverContent.vue`, `Sidebar.vue`, `session.ts`, `SheetContent.vue`, `AssignmentsPage.vue`, `PaginationBar.vue`, `dialog/index.ts`, `AssignmentDetailPage.vue`, `VerifyPage.test.ts`, `web_src_components_ui_button_index_button`, `useListQuery.ts`, `useRowSelection`, `LoginPage.vue`, `LocationStatusCard.vue`, `NativeSelect.vue`, `BarChart.vue`, `ReportsPage.vue`?**
  _High betweenness centrality (0.084) - this node is a cross-community bridge._
- **Why does `shift()` connect `CLAUDE.md — Internship Monitoring Platform` to `AttendanceReport.vue`?**
  _High betweenness centrality (0.026) - this node is a cross-community bridge._
- **What connects `face-service`, `fetch_photos.sh script`, `pc` to the rest of the system?**
  _812 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `server.go` be split into smaller, more focused modules?**
  _Cohesion score 0.11362975879104911 - nodes in this community are weakly interconnected._
- **Should `package.json` be split into smaller, more focused modules?**
  _Cohesion score 0.09420289855072464 - nodes in this community are weakly interconnected._
- **Should `newTeacherEnv` be split into smaller, more focused modules?**
  _Cohesion score 0.14623655913978495 - nodes in this community are weakly interconnected._
- **Should `SchedulesPage.vue` be split into smaller, more focused modules?**
  _Cohesion score 0.06451612903225806 - nodes in this community are weakly interconnected._