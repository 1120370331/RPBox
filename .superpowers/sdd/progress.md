# Performance and General Fixes Progress

- Branch: `fix/performance-general-issues`
- Worktree: `C:\Users\rog\WorkSpace\projects\github_repo\RPBox\.codex-tmp\performance-general-issues`
- Merge base: `2625457`

## Task Status

- [x] Task 1: Add Post List Candidate Cache Service (`07e5fa9`)
- [x] Task 2: Add Transactional Post Mutation Service (`fc8da5d`)
- [x] Task 3: Inject Services and Refactor Public Post Listing (`24528e1`)
- [x] Task 4: Migrate All List-Affecting Mutations (`9b9a8e3`)
- [x] Task 5: Add Desktop Post Sharing (`8c25ccc`)
- [x] Task 6: Move Warcraft Shortcuts Before Plugin Installation (`84ecb81`)
- [x] Task 7: Final Verification and Task Completion (`93a2b26`)

## Review Notes

- Final review found a stale-candidate visibility boundary when Redis invalidation failed;
  live filter revalidation and temporary cache bypass were added in `93a2b26`.
- Task 1 verification: focused tests, full `internal/service` tests, `go vet
  ./internal/service`, and `git diff --check` passed.
- Task 2 verification: RED compile failure confirmed; focused mutation tests,
  full `internal/service` tests, `go vet ./internal/service`, and staged diff checks passed.
- Task 3 verification: stale full-response cache reproduced, candidate hydration test
  passed after refactor, and service/API tests plus `go vet` passed.
- Task 4 verification: server `go test ./... -count=1`, `go vet ./...`, and
  `git diff --check` passed before commit.
- Task 5 verification: share utility and component tests passed (10 tests);
  the client production build completed successfully.
- Task 6 verification: heading-order RED/GREEN test and client production build passed.
- Final verification: server tests and vet passed; client 56 tests and production build
  passed; no legacy post-list cache helper references remain.
- Deferred baseline issue: repository-wide `vue-tsc --noEmit` failures predate this branch.
- Client verification uses explicit Vitest and Vite Node entrypoints because workspace-level
  command discovery is unreliable on this Windows worktree.
