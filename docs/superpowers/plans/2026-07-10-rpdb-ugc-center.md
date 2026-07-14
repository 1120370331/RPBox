# RP 数据库玩家共创中心 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a production-ready V1 player-driven RP item, transmog, and guide sharing center across the Go API and Vue desktop client.

**Architecture:** Add an independent `rpdb_*` domain beside the existing TRP3 item market. A shared work table stores common UGC metadata, while references, media, transmog slots, guide steps, comments, interactions, lists, tags, revisions, and verification feedback remain focused tables. The client uses dedicated RPDB API types and four primary views: discovery, detail, editor, and collection lists.

**Tech Stack:** Go 1.21, Gin, GORM, PostgreSQL, SQLite tests, Vue 3, TypeScript, Pinia-compatible client state, Vue Router, Vitest, Remix Icon.

## Global Constraints

- Preserve the existing TRP3 `items` domain and its API behavior.
- Preserve user changes currently present in `model.go` and `database.go`.
- Do not return Base64 media in RPDB list responses.
- Use existing JWT, user display, upload, notification, safety, moderation, `RDialog`, and `RToast` infrastructure.
- Use ASCII quotes in JSON and TypeScript source.
- Public reads return only `published + approved + public` works.
- Major edits to published works create reviewable revisions.
- Complete V1 excludes automatic Wowhead ingestion, personalized recommendation, reminders, and live addon synchronization.

---

### Task 1: RPDB Domain Models and Migration

**Files:**
- Create: `server/internal/model/rpdb.go`
- Modify: `server/internal/database/database.go`
- Test: `server/internal/model/rpdb_test.go`

**Interfaces:**
- Produces: `RPDBWork`, `RPDBReference`, `RPDBMedia`, `RPDBTransmogSlot`, `RPDBGuideStep`, `RPDBTag`, `RPDBLike`, `RPDBFavorite`, `RPDBComment`, `RPDBCommentLike`, `RPDBList`, `RPDBListEntry`, `RPDBRevision`, `RPDBVerification`, and `RPDBSet` models.

- [ ] Write model tests covering table names, unique reference constraints, one favorite per user/work, one list entry per list/work, and comment reply relationships.
- [ ] Run `go test ./internal/model -run RPDB -count=1` and confirm the new tests fail before the models exist.
- [ ] Implement focused models in `rpdb.go` with explicit `TableName()` methods and indexed status/filter fields.
- [ ] Add all RPDB models to `AutoMigrate` without altering existing migration behavior.
- [ ] Add PostgreSQL indexes for public listing, references, media ordering, list membership, and verification feedback.
- [ ] Re-run `go test ./internal/model -run RPDB -count=1`.

### Task 2: Public Discovery and Detail API

**Files:**
- Create: `server/internal/api/rpdb_query.go`
- Create: `server/internal/api/rpdb_response.go`
- Modify: `server/internal/api/routes.go`
- Test: `server/internal/api/rpdb_query_test.go`

**Interfaces:**
- Produces:
  - `GET /api/v1/rpdb/works`
  - `GET /api/v1/rpdb/works/:id`
  - `GET /api/v1/rpdb/sets`
  - `GET /api/v1/rpdb/sets/:id`
- Supports optional authentication so list/detail responses can include viewer interaction state.

- [ ] Write API tests for public visibility, work-type filtering, search, tags, availability, validation status, sorting, pagination, and complete detail composition.
- [ ] Register public RPDB routes outside the mandatory JWT group.
- [ ] Implement a query builder that always applies public moderation predicates before optional filters.
- [ ] Return lightweight cards from list endpoints and load references, approved media, transmog slots, guide steps, tags, author display, and viewer state only in detail.
- [ ] Run `go test ./internal/api -run RPDBQuery -count=1`.

### Task 3: Authoring, Media, and Revision API

**Files:**
- Create: `server/internal/api/rpdb_authoring.go`
- Create: `server/internal/api/rpdb_validation.go`
- Modify: `server/internal/api/routes.go`
- Test: `server/internal/api/rpdb_authoring_test.go`

**Interfaces:**
- Produces:
  - `POST /api/v1/rpdb/works`
  - `PUT /api/v1/rpdb/works/:id`
  - `DELETE /api/v1/rpdb/works/:id`
  - `POST /api/v1/rpdb/works/:id/media`
  - `DELETE /api/v1/rpdb/works/:id/media/:mediaId`
  - `GET /api/v1/rpdb/my/works`
- Accepts one transaction payload containing work fields plus references, tags, transmog slots, or guide steps.

- [ ] Write failing tests for draft creation, submission, duplicate external reference warnings, direct draft edits, revision creation for published works, ownership checks, and transactional child replacement.
- [ ] Implement strict enum, length, URL scheme, coordinate, and content-type validation.
- [ ] Store uploaded media through the existing upload pipeline and keep external media URLs explicit.
- [ ] Ensure normal user submissions enter review while moderator submissions can publish immediately.
- [ ] Run `go test ./internal/api -run RPDBAuthoring -count=1`.

### Task 4: Community Interactions and Collection Lists

**Files:**
- Create: `server/internal/api/rpdb_interactions.go`
- Create: `server/internal/api/rpdb_lists.go`
- Modify: `server/internal/api/routes.go`
- Test: `server/internal/api/rpdb_interactions_test.go`
- Test: `server/internal/api/rpdb_lists_test.go`

**Interfaces:**
- Produces like, favorite, comment, reply, verification feedback, list CRUD, list entry updates, and list export endpoints.

- [ ] Write tests for idempotent like/favorite operations and exact counter changes.
- [ ] Write tests for comment replies, author notifications, hidden/blocked user filtering, and comment deletion ownership.
- [ ] Write tests for automatic default-list creation, status updates, duplicate prevention, JSON/CSV/TomTom export, and missing-coordinate reporting.
- [ ] Implement all writes in transactions and keep counters recalculable.
- [ ] Run `go test ./internal/api -run 'RPDB(Interaction|List)' -count=1`.

### Task 5: Moderation, Reports, and Account Lifecycle

**Files:**
- Create: `server/internal/api/rpdb_moderation.go`
- Modify: `server/internal/api/routes.go`
- Modify: `server/internal/api/moderator.go`
- Modify: `server/internal/api/safety.go`
- Modify: `server/internal/api/account_deletion.go`
- Test: `server/internal/api/rpdb_moderation_test.go`

**Interfaces:**
- Produces review queues for works, media, and revisions plus hide/delete/verify/merge actions.
- Adds `rpdb_work`, `rpdb_comment`, and `rpdb_media` report targets.

- [ ] Write tests for moderator authorization, approval/rejection, revision application with base-version conflict detection, report target validation, and author anonymization during account deletion.
- [ ] Add RPDB counts to moderator statistics and write admin action logs for every moderation mutation.
- [ ] Notify contributors when submissions or revisions are reviewed.
- [ ] Run `go test ./internal/api -run RPDBModeration -count=1`.

### Task 6: Client API, Navigation, and Shared RPDB Components

**Files:**
- Create: `client/src/api/rpdb.ts`
- Create: `client/src/components/rpdb/RPDBWorkCard.vue`
- Create: `client/src/components/rpdb/RPDBFilterBar.vue`
- Create: `client/src/components/rpdb/RPDBMediaGallery.vue`
- Create: `client/src/components/rpdb/RPDBVerificationBadge.vue`
- Create: `client/src/i18n/locales/zh-CN/rpdb.ts`
- Modify: `client/src/i18n/locales/zh-CN/index.ts`
- Modify: `client/src/i18n/locales/zh-CN/nav.ts`
- Modify: `client/src/router.ts`
- Modify: `client/src/components/AppLayout.vue`
- Test: `client/src/components/rpdb/RPDBWorkCard.test.ts`

**Interfaces:**
- Produces strongly typed request/response functions for every V1 endpoint.
- Adds `/rpdb`, `/rpdb/create`, `/rpdb/:id`, `/rpdb/:id/edit`, and `/rpdb/lists`.

- [ ] Write component tests for content type, verification, author, media fallback, and interaction counts.
- [ ] Add a first-class RP database navigation item and route cache mapping.
- [ ] Implement shared cards, filters, verification badge, and media gallery using existing theme variables and Remix Icons.
- [ ] Run `npm test -- --run src/components/rpdb/RPDBWorkCard.test.ts`.

### Task 7: Discovery, Detail, Editor, and Lists Views

**Files:**
- Create: `client/src/views/rpdb/RPDBMain.vue`
- Create: `client/src/views/rpdb/RPDBDetail.vue`
- Create: `client/src/views/rpdb/RPDBEditor.vue`
- Create: `client/src/views/rpdb/RPDBLists.vue`
- Create: `client/src/views/rpdb/RPDBPreview.vue`
- Test: `client/src/views/rpdb/RPDBMain.test.ts`
- Test: `client/src/views/rpdb/RPDBEditor.test.ts`

**Interfaces:**
- Consumes: `client/src/api/rpdb.ts` and shared RPDB components.
- Produces the complete desktop RPDB user workflow.

- [ ] Implement discovery with featured strip, responsive card grid, sticky search/filter toolbar, empty states, loading states, and pagination.
- [ ] Implement detail with media stage, structured metadata, referenced objects, transmog slot grid or guide steps, related content, comments, and collection actions.
- [ ] Implement a seven-step editor for all three work types with draft persistence, validation summary, media management, preview, and submit-for-review.
- [ ] Implement list management, state changes, progress summary, and JSON/CSV/TomTom export.
- [ ] Use `RDialog` for destructive confirmation and `RToast` for async results.
- [ ] Run focused Vitest suites and `npm run build`.

### Task 8: Moderator and User Surface Integration

**Files:**
- Modify: `client/src/api/moderator.ts`
- Modify: `client/src/views/moderator/ModeratorMain.vue`
- Modify: `client/src/views/user/UserProfile.vue`
- Modify: `client/src/views/library/Favorites.vue`
- Test: relevant existing moderator and user view tests, or add focused RPDB tests where no harness exists.

**Interfaces:**
- Adds RPDB review tabs, user contribution sections, and RPDB favorites.

- [ ] Add work, media, and revision queues with preview and approve/reject actions.
- [ ] Add RPDB works to public user profiles without mixing them with TRP3 market items.
- [ ] Add RPDB favorites to the existing library surface.
- [ ] Verify navigation return behavior and notification deep links.

### Task 9: Final Verification and Visual Acceptance

**Files:**
- Modify only files required by defects found during verification.

- [ ] Run `gofmt` on changed Go files.
- [ ] Run `go test ./...` from `server`.
- [ ] Run `npm test -- --run` from `client`.
- [ ] Run `npm run build` from `client`.
- [ ] Start the client development server and inspect discovery, detail, editor, and lists at desktop and narrow viewports.
- [ ] Run Lighthouse accessibility and best-practices audit against the RPDB discovery page.
- [ ] Confirm there are no `#REF`-style broken links, blank required states, console errors, overflow, overlapping text, or unhandled API failures.
- [ ] Update `server/tasks/RPDB-0001.md` and `TASK_LIST.md` only after all acceptance criteria pass.
