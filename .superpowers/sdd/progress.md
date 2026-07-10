# Performance and General Fixes Progress

- Branch: `fix/performance-general-issues`
- Worktree: `C:\Users\rog\WorkSpace\projects\github_repo\RPBox\.codex-tmp\performance-general-issues`
- Merge base: `2625457`

## Task Status

- [ ] Task 1: Add Post List Candidate Cache Service
- [ ] Task 2: Add Transactional Post Mutation Service
- [ ] Task 3: Inject Services and Refactor Public Post Listing
- [ ] Task 4: Migrate All List-Affecting Mutations
- [ ] Task 5: Add Desktop Post Sharing
- [ ] Task 6: Move Warcraft Shortcuts Before Plugin Installation
- [ ] Task 7: Final Verification and Task Completion

## Review Notes

- Minor findings: none
- Deferred baseline issue: repository-wide `vue-tsc --noEmit` failures predate this branch.
- Client verification uses explicit Vitest and Vite Node entrypoints because workspace-level
  command discovery is unreliable on this Windows worktree.
