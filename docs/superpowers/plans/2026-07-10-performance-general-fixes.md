# Performance and General Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete SRV-0007, COMM-0003, and SYNC-0016 with a centralized post-list service, reliable desktop sharing, and corrected Warcraft page ordering.

**Architecture:** Move public post candidate queries and cache versioning into `PostListService`. Route all list-affecting writes through `PostMutationService`, which invalidates global or viewer cache versions only after a successful transaction. Keep HTTP validation and response DTO assembly in the API layer. Add a dependency-free desktop share utility and make the Warcraft change a template-only reorder.

**Tech Stack:** Go 1.24, Gin, GORM, Redis/miniredis, SQLite tests, Vue 3, TypeScript, Vitest, Vue Test Utils.

## Global Constraints

- Preserve current HTTP request and response formats.
- Preserve moderation, privacy, guild-access, and post-review rules.
- Do not add a Tauri plugin or new native permission.
- Do not redesign community cards or the Warcraft page.
- Do not modify unrelated dirty-worktree files.
- Write each regression test first and verify the expected failure before production changes.

---

## File Structure

- Create `server/internal/service/post_list.go`: normalized public-list query, candidate caching, cache versions, and invalidation.
- Create `server/internal/service/post_list_test.go`: key isolation, cache fallback, and volatile-sort tests.
- Create `server/internal/service/post_mutation.go`: transaction coordinator and post-list invalidation.
- Create `server/internal/service/post_mutation_test.go`: commit, rollback, global, viewer, and cache-error behavior.
- Modify `server/internal/api/server.go`: inject post list and mutation services.
- Modify `server/internal/api/post.go`: delegate public candidate queries, hydrate current rows, and migrate post mutations.
- Modify `server/internal/api/moderator.go`: migrate review, delete, hide, pin, feature, and edit approval mutations.
- Modify `server/internal/api/admin.go`: migrate bulk disable and delete operations.
- Modify `server/internal/api/safety.go`: migrate viewer block/hide changes and reported-post deletion invalidation.
- Modify `server/internal/api/content_moderation.go`: route automatic post deletion through the mutation service.
- Modify `server/internal/api/account_deletion.go`: use the centralized global invalidation path.
- Delete `server/internal/api/cache_helpers.go` after its final callers are removed.
- Create `server/internal/api/post_cache_invalidation_test.go`: representative API-level cache invalidation regressions.
- Create `client/src/utils/share.ts`: public URL, Web Share, clipboard, and DOM fallback logic.
- Create `client/src/__tests__/share.test.ts`: desktop share utility tests.
- Modify `client/src/views/community/PostDetail.vue`: detail-page share action.
- Modify `client/src/views/community/CommunityMain.vue`: post-card share action with event propagation protection.
- Modify `client/src/i18n/locales/zh-CN/community.ts`: share success/failure text.
- Modify `client/src/i18n/locales/en-US/community.ts`: matching English share text.
- Create `client/src/__tests__/communityShare.test.ts`: detail/list integration assertions.
- Modify `client/src/views/WarcraftFeatures.vue`: reorder feature and plugin sections.
- Create `client/src/__tests__/warcraftFeaturesOrder.test.ts`: section-order regression test.
- Modify `server/tasks/SRV-0007.md`, `client/src/views/community/tasks/COMM-0003.md`, `client/src/views/tasks/SYNC-0016.md`, and `TASK_LIST.md`: mark completed only after verification.

---

### Task 1: Add Post List Candidate Service

**Files:**
- Create: `server/internal/service/post_list_test.go`
- Create: `server/internal/service/post_list.go`

**Interfaces:**
- Produces: `PostListQuery`, `PostListCandidatePage`, `NewPostListService`, `Candidates`, `InvalidateGlobal`, `InvalidateViewer`.
- Consumes: `cache.Cache`, `model.Post`, `model.UserBlock`, `model.UserHiddenContent`, `gorm.DB`.

- [ ] **Step 1: Write failing cache-key and cacheability tests**

```go
func TestPostListQueryCacheKeySeparatesFilters(t *testing.T) {
    base := PostListQuery{
        ViewerID: 7,
        Page: 1,
        PageSize: 12,
        SortBy: "created_at",
        Order: "desc",
        Status: "published",
    }

    pageTwo := base
    pageTwo.Page = 2
    category := base
    category.Category = "event"
    pinned := true
    pinnedQuery := base
    pinnedQuery.IsPinned = &pinned

    keys := map[string]struct{}{
        base.cacheSuffix(): {},
        pageTwo.cacheSuffix(): {},
        category.cacheSuffix(): {},
        pinnedQuery.cacheSuffix(): {},
    }
    if len(keys) != 4 {
        t.Fatalf("expected isolated keys, got %d", len(keys))
    }
}

func TestPostListQueryBypassesVolatileSorts(t *testing.T) {
    for _, sortBy := range []string{"like_count", "view_count"} {
        query := PostListQuery{SortBy: sortBy}
        if query.Cacheable() {
            t.Fatalf("%s must bypass candidate cache", sortBy)
        }
    }
}
```

- [ ] **Step 2: Run tests and verify RED**

Run:

```powershell
cd server
go test ./internal/service -run 'TestPostListQuery' -count=1
```

Expected: compilation fails because `PostListQuery` does not exist.

- [ ] **Step 3: Implement normalized query and candidate service**

```go
type PostListQuery struct {
    ViewerID  uint
    Page      int
    PageSize  int
    SortBy    string
    Order     string
    Search    string
    AuthorName string
    Region    string
    Address   string
    TagID     string
    AuthorID  string
    Status    string
    Category  string
    IsPinned  *bool
}

type PostListCandidatePage struct {
    IDs   []uint `json:"ids"`
    Total int64  `json:"total"`
}

type PostListService struct {
    db    *gorm.DB
    cache cache.Cache
}

func NewPostListService(db *gorm.DB, cacheClient cache.Cache) *PostListService {
    return &PostListService{db: db, cache: cacheClient}
}

func (q PostListQuery) Cacheable() bool {
    return q.SortBy != "like_count" && q.SortBy != "view_count"
}
```

Implement `cacheSuffix` with `url.Values`, explicit schema `candidate-v1`, normalized lower bounds for page/page size, normalized order, and `cache.HashKey(values.Encode())`.

Implement `Candidates(ctx, query)`:

1. Build the filtered public-post query with `published`, `approved`, and `is_public=true`.
2. Exclude blocked authors and viewer-hidden posts.
3. Apply category, search, author name, region, address, tag, author, and pinned filters.
4. Count total.
5. Select ordered IDs only.
6. For cacheable queries, combine global and viewer versions in the key and use `cache.Fetch`.
7. On any version/fetch error, execute the database loader and return its result.

- [ ] **Step 4: Add miniredis behavior tests**

Test that:

- two identical calls execute the loader once;
- a global invalidation changes the effective key;
- invalidating viewer 7 does not change viewer 8's effective key;
- a stopped miniredis instance returns database results without error.

- [ ] **Step 5: Run service tests and verify GREEN**

Run:

```powershell
cd server
go test ./internal/service -run 'TestPostList' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```powershell
git add server/internal/service/post_list.go server/internal/service/post_list_test.go
git commit -m "feat: add post list candidate cache service"
```

---

### Task 2: Add Transactional Post Mutation Service

**Files:**
- Create: `server/internal/service/post_mutation_test.go`
- Create: `server/internal/service/post_mutation.go`

**Interfaces:**
- Consumes: `PostListService`.
- Produces: `NewPostMutationService`, `Global`, `Viewer`.

- [ ] **Step 1: Write failing transaction tests**

```go
func TestPostMutationGlobalInvalidatesAfterCommit(t *testing.T) {
    db := newPostServiceTestDB(t)
    lists, versions := newVersionTrackingPostListService(t, db)
    mutations := NewPostMutationService(db, lists)

    err := mutations.Global(context.Background(), func(tx *gorm.DB) error {
        return tx.Create(&model.Post{Title: "published"}).Error
    })
    if err != nil {
        t.Fatal(err)
    }
    if versions.globalBumps != 1 {
        t.Fatalf("expected one global bump, got %d", versions.globalBumps)
    }
}

func TestPostMutationDoesNotInvalidateAfterRollback(t *testing.T) {
    db := newPostServiceTestDB(t)
    lists, versions := newVersionTrackingPostListService(t, db)
    mutations := NewPostMutationService(db, lists)

    expected := errors.New("rollback")
    err := mutations.Global(context.Background(), func(tx *gorm.DB) error {
        return expected
    })
    if !errors.Is(err, expected) {
        t.Fatalf("expected rollback error, got %v", err)
    }
    if versions.globalBumps != 0 {
        t.Fatalf("rollback must not invalidate")
    }
}
```

Add viewer-scope and cache-invalidation-error tests.

- [ ] **Step 2: Run tests and verify RED**

Run:

```powershell
cd server
go test ./internal/service -run 'TestPostMutation' -count=1
```

Expected: compilation fails because `PostMutationService` does not exist.

- [ ] **Step 3: Implement transaction coordinators**

```go
type PostMutationService struct {
    db    *gorm.DB
    lists *PostListService
}

func NewPostMutationService(db *gorm.DB, lists *PostListService) *PostMutationService {
    return &PostMutationService{db: db, lists: lists}
}

func (s *PostMutationService) Global(
    ctx context.Context,
    mutate func(tx *gorm.DB) error,
) error {
    if err := s.db.WithContext(ctx).Transaction(mutate); err != nil {
        return err
    }
    if err := s.lists.InvalidateGlobal(ctx); err != nil {
        log.Printf("[Cache] post list global invalidation failed: %v", err)
    }
    return nil
}

func (s *PostMutationService) Viewer(
    ctx context.Context,
    viewerID uint,
    mutate func(tx *gorm.DB) error,
) error {
    if err := s.db.WithContext(ctx).Transaction(mutate); err != nil {
        return err
    }
    if err := s.lists.InvalidateViewer(ctx, viewerID); err != nil {
        log.Printf("[Cache] post list viewer invalidation failed viewer=%d: %v", viewerID, err)
    }
    return nil
}
```

Nil cache/list services must be safe and keep database mutations successful.

- [ ] **Step 4: Run tests and verify GREEN**

Run:

```powershell
cd server
go test ./internal/service -run 'TestPostMutation' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add server/internal/service/post_mutation.go server/internal/service/post_mutation_test.go
git commit -m "feat: centralize post mutation invalidation"
```

---

### Task 3: Inject Services and Refactor Public Post Listing

**Files:**
- Modify: `server/internal/api/server.go`
- Modify: `server/internal/api/post.go`
- Create: `server/internal/api/post_cache_invalidation_test.go`

**Interfaces:**
- Consumes: `service.PostListService`, `service.PostMutationService`.
- Produces: `Server.postLists`, `Server.postMutations`, live hydration by candidate ID order.

- [ ] **Step 1: Write failing API list-cache test**

Create a test that:

1. inserts two published posts;
2. calls `listPosts` once to populate candidate cache;
3. updates the first post's counter directly;
4. calls `listPosts` again;
5. asserts candidate order is cached but the returned counter is current.

Also assert `sort=like_count` executes the live query and reflects changed ordering immediately.

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```powershell
cd server
go test ./internal/api -run 'TestPostListCandidateCache' -count=1
```

Expected: the complete cached response returns a stale counter or stale volatile ordering.

- [ ] **Step 3: Inject services**

Add fields:

```go
postLists     *service.PostListService
postMutations *service.PostMutationService
```

Initialize them after Redis setup:

```go
postLists := service.NewPostListService(database.DB, cacheClient)
postMutations := service.NewPostMutationService(database.DB, postLists)
```

- [ ] **Step 4: Split list loading**

In `post.go`:

- keep `postListParams` for HTTP parsing;
- add conversion to `service.PostListQuery`;
- route global public lists through `s.postLists.Candidates`;
- add `hydratePostList(ctx, ids, total)` that selects current rows, author data, cover metadata, and preserves ID order;
- keep guild and self views on `loadPostListDirect`;
- remove complete-response cache logic and its `cache` import.

Use an order map:

```go
position := make(map[uint]int, len(ids))
for index, id := range ids {
    position[id] = index
}
sort.Slice(posts, func(i, j int) bool {
    return position[posts[i].ID] < position[posts[j].ID]
})
```

- [ ] **Step 5: Run focused and package tests**

Run:

```powershell
cd server
go test ./internal/api -run 'TestPostListCandidateCache' -count=1
go test ./internal/service ./internal/api -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```powershell
git add server/internal/api/server.go server/internal/api/post.go server/internal/api/post_cache_invalidation_test.go
git commit -m "refactor: hydrate post lists from cached candidates"
```

---

### Task 4: Migrate All List-Affecting Mutations

**Files:**
- Modify: `server/internal/api/post.go`
- Modify: `server/internal/api/moderator.go`
- Modify: `server/internal/api/admin.go`
- Modify: `server/internal/api/safety.go`
- Modify: `server/internal/api/content_moderation.go`
- Modify: `server/internal/api/account_deletion.go`
- Delete: `server/internal/api/cache_helpers.go`
- Create: `server/internal/api/user_cache_helpers.go`
- Modify: `server/internal/api/post_cache_invalidation_test.go`

**Interfaces:**
- Consumes: `Server.postMutations`.
- Produces: no direct `bumpPostListCache` callers.

- [ ] **Step 1: Add failing representative API tests**

Add tests that populate a list cache, perform the real handler action, and assert the next list response changes for:

- initial post approval;
- moderator pin;
- moderator hide;
- moderator delete;
- tag add/remove;
- user block/unblock.

For viewer isolation, cache list responses for users 1 and 2, block an author as user 1, and assert only user 1's result changes.

- [ ] **Step 2: Run tests and verify RED**

Run:

```powershell
cd server
go test ./internal/api -run 'TestPostCacheInvalidation' -count=1
```

Expected: at least review, moderator actions, tag changes, and direct block changes return stale results.

- [ ] **Step 3: Migrate global mutations**

Replace direct transactions and trailing cache bumps with:

```go
err := s.postMutations.Global(c.Request.Context(), func(tx *gorm.DB) error {
    // Existing database mutation, using tx for every statement.
    return nil
})
```

Migrate:

- `createPost`, direct `updatePost`, `deletePost`, `addPostTag`, `removePostTag`;
- `reviewPost`, approved `reviewPostEdit`;
- `deletePostByMod`, `hidePostByMod`, `pinPost`, `featurePost`;
- `disableUserPosts`, `deleteUserPosts`;
- automatic moderation and reported-post deletion;
- account deletion when owned posts are removed.

Keep image cleanup and notifications outside the transaction when they are side effects that must not be rolled back.

- [ ] **Step 4: Migrate viewer mutations**

Wrap direct block/unblock and local post hide operations with:

```go
err := s.postMutations.Viewer(c.Request.Context(), userID, func(tx *gorm.DB) error {
    return upsertUserBlockRecord(tx, userID, blockedUserID, reason)
})
```

When one request both changes global post structure and viewer visibility, perform the database transaction once and explicitly invalidate both scopes after commit through a dedicated `GlobalAndViewer` method added with tests.

- [ ] **Step 5: Remove legacy helper**

Verify:

```powershell
rg -n 'bumpPostListCache|postListCacheName' server/internal/api server/internal/service
```

Expected: no API-level legacy references.

Delete `server/internal/api/cache_helpers.go` after retaining the unrelated user profile helper in a correctly named file such as `server/internal/api/user_cache_helpers.go`.

- [ ] **Step 6: Run focused and full server tests**

Run:

```powershell
cd server
go test ./internal/service ./internal/api -count=1
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```powershell
git add server/internal/api server/internal/service
git commit -m "fix: invalidate post lists through mutation service"
```

---

### Task 5: Add Desktop Post Sharing

**Files:**
- Create: `client/src/__tests__/share.test.ts`
- Create: `client/src/utils/share.ts`
- Modify: `client/src/views/community/PostDetail.vue`
- Modify: `client/src/views/community/CommunityMain.vue`
- Modify: `client/src/i18n/locales/zh-CN/community.ts`
- Modify: `client/src/i18n/locales/en-US/community.ts`
- Create: `client/src/__tests__/communityShare.test.ts`

**Interfaces:**
- Produces: `buildPublicPostUrl`, `shareRouteLink`, `ShareRouteResult`.

- [ ] **Step 1: Write failing share utility tests**

```ts
it('uses Web Share when available', async () => {
  const share = vi.fn().mockResolvedValue(undefined)
  Object.defineProperty(navigator, 'share', { configurable: true, value: share })

  await expect(shareRouteLink({
    path: '/posts/42',
    title: 'Test post',
  })).resolves.toEqual({
    method: 'shared',
    url: 'https://totalrpbox.com/posts/42',
  })
})

it('copies the public URL when Web Share is unavailable', async () => {
  Object.defineProperty(navigator, 'share', { configurable: true, value: undefined })
  const writeText = vi.fn().mockResolvedValue(undefined)
  Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: { writeText },
  })

  const result = await shareRouteLink({ path: '/posts/42' })
  expect(writeText).toHaveBeenCalledWith('https://totalrpbox.com/posts/42')
  expect(result.method).toBe('copied')
})
```

Add DOM fallback and rejected-share tests.

- [ ] **Step 2: Run tests and verify RED**

Run:

```powershell
cd client
node node_modules/vitest/vitest.mjs run --config vite.config.ts src/__tests__/share.test.ts
```

Expected: module not found.

- [ ] **Step 3: Implement share utility**

```ts
export interface ShareRouteResult {
  method: 'shared' | 'copied'
  url: string
}

const PUBLIC_SITE_ORIGIN = 'https://totalrpbox.com'

export function buildPublicPostUrl(postId: number) {
  return `${PUBLIC_SITE_ORIGIN}/posts/${postId}`
}

export async function shareRouteLink(options: {
  path: string
  title?: string
  text?: string
}): Promise<ShareRouteResult> {
  const url = new URL(options.path, PUBLIC_SITE_ORIGIN).toString()
  if (navigator.share) {
    await navigator.share({ title: options.title, text: options.text, url })
    return { method: 'shared', url }
  }
  await copyTextToClipboard(url)
  return { method: 'copied', url }
}
```

Implement `copyTextToClipboard` with Clipboard API and textarea/`document.execCommand('copy')` fallback.

- [ ] **Step 4: Add detail and list integration tests**

Mock `@/utils/share` and assert:

- the detail share button calls `shareRouteLink` with `/posts/:id`;
- the list card share button calls the utility;
- the list card handler calls `event.stopPropagation`;
- the router is not pushed by the share click.

- [ ] **Step 5: Add UI controls**

In `PostDetail.vue`, add a share action beside like/favorite:

```vue
<button class="action-btn" type="button" @click="handleShare">
  <i class="ri-share-forward-line"></i>
  <span>{{ t('community.action.share') }}</span>
</button>
```

In `CommunityMain.vue`, add an icon button in the card footer:

```vue
<button
  type="button"
  class="card-share-btn"
  :title="t('community.action.share')"
  @click.stop="handleSharePost(post)"
>
  <i class="ri-share-forward-line"></i>
</button>
```

Use the returned method to show either copied-link or opened-share Toast text.

- [ ] **Step 6: Run client tests and build**

Run:

```powershell
cd client
node node_modules/vitest/vitest.mjs run --config vite.config.ts src/__tests__/share.test.ts src/__tests__/communityShare.test.ts
node node_modules/vite/bin/vite.js build --config vite.config.ts
```

Expected: PASS.

- [ ] **Step 7: Commit**

```powershell
git add client/src/utils/share.ts client/src/__tests__/share.test.ts client/src/__tests__/communityShare.test.ts client/src/views/community/PostDetail.vue client/src/views/community/CommunityMain.vue client/src/i18n/locales/zh-CN/community.ts client/src/i18n/locales/en-US/community.ts
git commit -m "feat: add desktop post sharing"
```

---

### Task 6: Move Warcraft Shortcuts Before Plugin Installation

**Files:**
- Create: `client/src/__tests__/warcraftFeaturesOrder.test.ts`
- Modify: `client/src/views/WarcraftFeatures.vue`

**Interfaces:**
- Preserves: all existing component state and commands.
- Changes: rendered heading order and animation delays only.

- [ ] **Step 1: Write failing section-order test**

Mount `WarcraftFeatures.vue` with Tauri APIs mocked and assert:

```ts
const headings = wrapper.findAll('h2').map(node => node.text())
expect(headings).toEqual([
  '选择游戏目录',
  '功能快捷入口',
  '插件安装',
])
```

- [ ] **Step 2: Run test and verify RED**

Run:

```powershell
cd client
node node_modules/vitest/vitest.mjs run --config vite.config.ts src/__tests__/warcraftFeaturesOrder.test.ts
```

Expected: received order places `插件安装` before `功能快捷入口`.

- [ ] **Step 3: Reorder template blocks**

Move the entire `feature-row` section before `plugin-row`. Set shortcut delay to `2` and plugin delay to `3`. Do not change script logic or section internals.

- [ ] **Step 4: Run test and verify GREEN**

Run:

```powershell
cd client
node node_modules/vitest/vitest.mjs run --config vite.config.ts src/__tests__/warcraftFeaturesOrder.test.ts
node node_modules/vite/bin/vite.js build --config vite.config.ts
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add client/src/views/WarcraftFeatures.vue client/src/__tests__/warcraftFeaturesOrder.test.ts
git commit -m "fix: prioritize Warcraft feature shortcuts"
```

---

### Task 7: Final Verification and Task Completion

**Files:**
- Modify: `server/tasks/SRV-0007.md`
- Modify: `client/src/views/community/tasks/COMM-0003.md`
- Modify: `client/src/views/tasks/SYNC-0016.md`
- Modify: `TASK_LIST.md`

**Interfaces:**
- Produces: completed task documentation and verification evidence.

- [ ] **Step 1: Run formatting and static checks**

```powershell
gofmt -w server/internal/service/post_list.go server/internal/service/post_list_test.go server/internal/service/post_mutation.go server/internal/service/post_mutation_test.go server/internal/api/server.go server/internal/api/post.go server/internal/api/moderator.go server/internal/api/admin.go server/internal/api/safety.go server/internal/api/content_moderation.go server/internal/api/account_deletion.go server/internal/api/post_cache_invalidation_test.go
cd server
go vet ./...
```

Expected: no errors.

- [ ] **Step 2: Run complete server verification**

```powershell
cd server
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 3: Run complete client verification**

```powershell
cd client
node node_modules/vitest/vitest.mjs run --config vite.config.ts
node node_modules/vite/bin/vite.js build --config vite.config.ts
```

Expected: PASS.

`vue-tsc --noEmit` currently has repository-wide baseline failures unrelated to these tasks
(including missing `@` path mappings and historical type errors). Record its output as a
known baseline issue, but do not use it as a completion gate for this change.

- [ ] **Step 4: Review changed code**

Verify:

```powershell
git diff --check
rg -n 'bumpPostListCache|postListCacheName' server/internal/api server/internal/service
git status --short
```

Expected:

- no whitespace errors;
- no legacy post-list cache helper references;
- no unrelated files added to commits.

- [ ] **Step 5: Mark task cards DONE**

Change the status in all three task cards and `TASK_LIST.md` only after every verification command passes. Add a concise implementation note listing the relevant tests.

- [ ] **Step 6: Commit completion metadata**

```powershell
git add TASK_LIST.md server/tasks/SRV-0007.md client/src/views/community/tasks/COMM-0003.md client/src/views/tasks/SYNC-0016.md
git commit -m "docs: complete performance and general fixes"
```

- [ ] **Step 7: Final acceptance review**

Review the full diff for:

- stale-cache regressions;
- missed mutation paths;
- transaction boundary errors;
- viewer cache leakage;
- share-button navigation regressions;
- accidental Warcraft behavior changes;
- compatibility with existing dirty worktree changes.

