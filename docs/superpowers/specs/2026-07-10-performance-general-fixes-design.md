# RPBox Performance and General Fixes Design

**Date:** 2026-07-10

**Scope:** SRV-0007, COMM-0003, SYNC-0016

## Goals

This change set resolves three task-card items that do not require new product policy:

1. Make community post lists refresh immediately after every operation that changes list membership, ordering, filtering, or viewer visibility.
2. Add desktop post sharing with a reliable copy-link fallback.
3. Move Warcraft feature shortcuts before plugin installation without changing directory or installation behavior.

The work must preserve the current API contracts, moderation rules, post visibility rules, and visual language.

## Root Cause

The post list currently caches a complete response under a global versioned key. Cache invalidation is performed by scattered handler calls to `bumpPostListCache`. Several list-changing paths do not call it, including initial moderation review, moderator deletion/hiding/pinning/featuring, direct user block changes, administrative bulk post operations, and tag mutations.

The complete-response cache also contains volatile counters and viewer-dependent data. Invalidating the global namespace for every interaction would create unnecessary churn, while not invalidating it leaves counts and volatile sorts stale.

Desktop post pages have no shared sharing abstraction. Mobile already detects native sharing and falls back to the clipboard, but the desktop client does not expose equivalent behavior.

The Warcraft page already has independent, lockable sections. The shortcut section is simply rendered after the plugin installation section.

## Server Architecture

### PostListService

Create `server/internal/service/post_list.go` with a `PostListService` that owns:

- the GORM database handle;
- the cache interface;
- cache key generation;
- global and viewer cache-version lookup;
- candidate-list fetch and fallback behavior;
- global and viewer invalidation.

The service caches only a candidate page:

```go
type PostListCandidatePage struct {
    IDs   []uint
    Total int64
}
```

The API continues to own HTTP parameter parsing and response formatting. The service receives a normalized query value that includes page, page size, status, category, search text, author, tag, pinned state, region, address, and viewer ID.

Cache keys include:

- global cache version;
- viewer cache version when the result is viewer-dependent;
- every normalized filter, pagination, and ordering value;
- an explicit schema version.

Queries sorted by `like_count` or `view_count` bypass candidate caching because their ordering changes too frequently. Guild-only and self-management views retain their existing direct-database behavior unless tests demonstrate that candidate caching is safe.

After candidate IDs are loaded, the API hydrates the current post rows, author data, tags, counters, and image URLs in the candidate order. This keeps volatile display fields current while preserving the performance benefit of caching the expensive filtered ID query.

### PostMutationService

Create `server/internal/service/post_mutation.go` with a `PostMutationService` that owns the database handle and `PostListService`.

It exposes transaction coordinators with explicit invalidation scope:

```go
func (s *PostMutationService) Global(
    ctx context.Context,
    mutate func(tx *gorm.DB) error,
) error

func (s *PostMutationService) Viewer(
    ctx context.Context,
    viewerID uint,
    mutate func(tx *gorm.DB) error,
) error
```

The coordinator:

1. runs the mutation in a database transaction;
2. returns immediately on rollback or database failure;
3. invalidates the required cache namespace only after commit;
4. logs cache invalidation failure without changing the successful API result.

All existing post-list-affecting writes move behind this service:

- create, direct update, delete, and approved edit application;
- initial publish approval or rejection;
- moderator delete, hide, pin, and feature actions;
- administrative disable or bulk delete;
- post tag addition and removal;
- report moderation that deletes a post;
- local post hiding;
- user block and unblock visibility changes.

Viewer-only visibility operations invalidate only that viewer's namespace. Structural operations invalidate the global namespace.

Likes, favorites, comments, and views do not invalidate the stable candidate cache. Their current counts are loaded during hydration. Volatile count sorts bypass caching.

### Dependency Injection

`Server` receives initialized `PostListService` and `PostMutationService` fields in `NewServer`. Tests may construct both services with SQLite and a fake or miniredis-backed cache.

The old API-level `bumpPostListCache` helper is removed after all call sites migrate.

## Cache Failure Behavior

Cache reads, version reads, writes, and invalidations are optional optimizations:

- a version or fetch error falls back to the database;
- a cache write error does not fail the list request;
- an invalidation error is logged after a successful database commit;
- a database error never invalidates a cache namespace;
- Redis unavailability never blocks post creation, moderation, editing, deletion, blocking, or listing.

## Desktop Sharing

Create `client/src/utils/share.ts` with:

- public post URL construction using the existing public-site URL convention;
- `shareRouteLink`, which prefers `navigator.share`;
- clipboard fallback through `navigator.clipboard.writeText`;
- a DOM copy fallback when the Clipboard API is unavailable;
- a result value that distinguishes `shared` from `copied`.

Add a share action to:

- `client/src/views/community/PostDetail.vue`;
- each standard post card in `client/src/views/community/CommunityMain.vue`.

The card button stops click propagation so sharing does not open the post. Success text distinguishes an opened share sheet from a copied link. Failure uses the existing Toast mechanism and does not alter navigation.

No new Tauri plugin or native permission is required.

## Warcraft Shortcut Ordering

In `client/src/views/WarcraftFeatures.vue`:

- render the feature shortcut section immediately after directory selection;
- render plugin installation after shortcuts;
- update animation delays to match visual order;
- preserve the existing `hasWowPath` lock, button disabled states, directory setup flow, plugin checks, progress, installation, update, and uninstall logic.

No styling redesign is included.

## Testing

### Server

Add service tests covering:

- distinct candidate keys for pagination and every supported filter;
- global invalidation after successful structural mutation;
- viewer-only invalidation after block, unblock, or local hide;
- no invalidation after transaction rollback;
- database fallback when cache version lookup or fetch fails;
- current counters after a cached candidate page is hydrated;
- cache bypass for `like_count` and `view_count` sorting.

Add API regression coverage for representative mutation paths:

- publish approval;
- moderator pin or feature;
- moderator hide or delete;
- user block and unblock;
- tag addition or removal.

### Client

Add utility tests covering:

- Web Share API success;
- clipboard fallback;
- DOM copy fallback;
- public post URL construction;
- failure propagation.

Add component-level assertions that:

- detail and list share controls invoke the sharing utility without unintended navigation;
- the Warcraft section heading order is directory, shortcuts, plugin installation.

## Acceptance Criteria

- A newly approved post is visible on the next list request.
- Edits, deletion, hiding, pinning, featuring, tags, and bulk administration are reflected on the next list request.
- Blocking, hiding, and unblocking affect only the requesting viewer's cached results.
- Filter, page, sort, author, category, tag, location, and pinned queries cannot share a candidate cache entry accidentally.
- Redis failure does not break reads or mutations.
- Desktop users can share or copy a public post link from detail and list views.
- Warcraft shortcuts appear before plugin installation while all lock and installation behavior remains unchanged.
- Targeted tests, full Go tests, client tests, type checks, and production builds pass.

## Non-Goals

- No new community business rules or moderation policy.
- No redesign of community cards or the Warcraft page.
- No native desktop share plugin.
- No caching for guild-only management views or volatile count-based ordering.
- No unrelated refactoring of current dirty worktree files.

