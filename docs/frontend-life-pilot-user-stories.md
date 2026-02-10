# Life-Pilot Frontend User Stories and Implementation Tasks

## 1. Planning Model

- Story format: `As a <role>, I want <goal>, so that <value>.`
- Each story includes acceptance criteria and implementation tasks.
- Prioritization:
  - `P0` must-have for first usable release.
  - `P1` high-value follow-up.
  - `P2` improvement/nice-to-have.

## 2. Epic 0 - Frontend Foundation (`P0`)

## Story 0.1 - App Shell and Routing
As a user, I want a consistent app shell with navigation so that I can move between core features.

Acceptance criteria:
- `/app/*` routes load without 404 on refresh.
- Sidebar or top-nav links to all planned sections.
- Unknown routes show a not-found screen inside shell.

Implementation tasks:
- Initialize Vite React TypeScript app with Bun.
- Install and configure shadcn/ui + Tailwind.
- Add React Router route tree and layout shell.
- Add not-found route and loading skeleton.

## Story 0.2 - Embedded Asset Serving
As an operator, I want frontend assets embedded in Go binary so that deployment is single-artifact.

Acceptance criteria:
- Running compiled binary serves working frontend at `/app`.
- Static assets are served with proper content-type and cache behavior.
- SPA fallback serves `index.html` for deep links.

Implementation tasks:
- Add `web/dist` embed package in Go.
- Implement dedicated frontend handler (`pkg/web`).
- Wire handler into main router (`/app/*` + assets).
- Add startup validation for embedded files.

## Story 0.3 - Typed API Client
As a developer, I want typed API access so that frontend changes are safe and predictable.

Acceptance criteria:
- Shared API client wraps fetch and handles errors consistently.
- Response schemas validated via zod for primary endpoints.
- Query/mutation hooks used in at least one feature page.

Implementation tasks:
- Build `lib/api/client.ts` and error model.
- Add zod schemas for automation and project list payloads.
- Integrate React Query provider and devtools toggle.

## 3. Epic 1 - Dashboard (`P0`)

## Story 1.1 - Daily Overview
As a user, I want a dashboard snapshot so that I can quickly decide what to do next.

Acceptance criteria:
- Dashboard shows inbox, next tasks, active projects, and automation counts.
- Data refreshes on manual reload and periodic background refetch.
- Error state shows clear recovery action.

Implementation tasks:
- Add `/api/dashboard` backend endpoint.
- Create dashboard cards and loading skeletons.
- Add stale-while-revalidate query behavior.

## Story 1.2 - Activity Feed
As a user, I want recent runs/changes so that I can verify automation outcomes.

Acceptance criteria:
- Feed lists latest automation run statuses.
- Failed runs are visually distinct and clickable.

Implementation tasks:
- Add backend endpoint for recent activity.
- Build feed component and status badge mapping.

## 4. Epic 2 - Tasks and Inbox (`P0`)

## Story 2.1 - Inbox Triage
As a user, I want to process inbox items quickly so that they become actionable or archived.

Acceptance criteria:
- Inbox list is sortable by created time.
- User can convert inbox item to next action/project/reference.
- Bulk select + bulk operation works.

Implementation tasks:
- Add tasks API list and patch endpoints.
- Build inbox table with row selection.
- Add batch mutation flow with optimistic updates.

## Story 2.2 - Next Actions Board
As a user, I want tasks grouped by context/status so that I can work in the right mode.

Acceptance criteria:
- Tasks can be filtered by context, priority, and status.
- Task detail panel shows metadata and linked project.
- Marking a task done updates list without full refresh.

Implementation tasks:
- Build filter bar + query params sync.
- Add side panel with edit form.
- Add status update mutation and local cache patching.

## 5. Epic 3 - Projects (`P0`)

## Story 3.1 - Project List
As a user, I want to view active projects with health indicators so that I know what needs attention.

Acceptance criteria:
- List shows status, due/review date, open task count.
- Overdue review projects are highlighted.

Implementation tasks:
- Add `GET /api/projects` enhancements.
- Build project list with badges and health logic.

## Story 3.2 - Project Detail
As a user, I want one page for project context so that I can plan execution.

Acceptance criteria:
- Detail view includes project metadata and related tasks.
- User can navigate to linked vault note preview.

Implementation tasks:
- Add `GET /api/projects/{id}` endpoint.
- Build detail screen + related-task section.

## 6. Epic 4 - Calendar (`P1`)

## Story 4.1 - Week/Month Planner
As a user, I want calendar views of commitments so that I can align work with time.

Acceptance criteria:
- User can switch between week and month views.
- Calendar includes tasks with due dates and external events.
- Clicking an item opens details.

Implementation tasks:
- Add calendar aggregation endpoint.
- Build calendar grid with reusable event cards.
- Add date range navigation and deep-link query params.

## 7. Epic 5 - Vault Preview (`P1`)

## Story 5.1 - Read-Only Vault Browser
As a user, I want to inspect vault notes in-app so that I do not need to switch tools for quick checks.

Acceptance criteria:
- Folder tree and file list render GTD structure.
- Selecting a note shows markdown preview.
- Frontmatter metadata is visible in summary panel.

Implementation tasks:
- Add safe file-tree and file-read APIs.
- Implement split-pane tree + preview UI.
- Add markdown renderer and frontmatter block.

## 8. Epic 6 - Automations UX (`P0`)

## Story 6.1 - List and Control Automations
As a user, I want to inspect and control automations so that I can trust and tune scheduled actions.

Acceptance criteria:
- Automation table shows action, schedule, next run, enabled status.
- Run-now and pause/resume work from UI.
- Last status shown inline.

Implementation tasks:
- Connect existing `/automations` APIs to React Query hooks.
- Build automations table with row actions.
- Add optimistic toggle for enabled state.

## Story 6.2 - Create/Edit Automation
As a user, I want a guided form so that I can safely define schedules and payloads.

Acceptance criteria:
- Form supports `cron`, `interval`, `oneshot`.
- Schedule validation errors are shown before submit.
- Payload JSON editor validates syntax.

Implementation tasks:
- Build create/edit modal with schema validation.
- Add helper text and examples per schedule type.
- Add JSON editor component (or textarea + validator).

## Story 6.3 - Run History
As a user, I want recent runs and errors so that I can debug failures.

Acceptance criteria:
- Run history lists status, started/finished times, error message.
- User can filter failed-only.

Implementation tasks:
- Add `GET /api/automations/{id}/runs`.
- Build timeline list with status chips and details drawer.

## 9. Epic 7 - AI Chat (`P1`)

## Story 7.1 - Chat Workspace
As a user, I want to ask for planning help in chat so that I can delegate routine GTD reasoning.

Acceptance criteria:
- Chat supports streaming or progressive message updates.
- User can copy generated action plan into inbox/tasks.
- Failure and retry states are clear.

Implementation tasks:
- Add `POST /api/chat` endpoint contract.
- Build chat thread UI with input composer.
- Add action buttons to convert outputs into tasks.

## Story 7.2 - Workflow Commands
As a user, I want command shortcuts so that common actions are one click.

Acceptance criteria:
- Quick actions exist for daily summary, triage inbox, weekly review.
- Command execution status appears in chat timeline.

Implementation tasks:
- Add quick command bar and command schema.
- Wire command actions to backend APIs.

## 10. Epic 8 - Settings and Diagnostics (`P1`)

## Story 8.1 - Integration Health
As an operator, I want to see provider status so that I can diagnose configuration issues quickly.

Acceptance criteria:
- Page shows status for AI provider, Gmail, Calendar, Drive, Git sync.
- Missing env/config issues include actionable hints.

Implementation tasks:
- Add `GET /api/settings/health`.
- Build status cards with red/amber/green states.

## 11. Epic 9 - Quality and Release (`P0`)

## Story 9.1 - Accessibility Baseline
As a user, I want keyboard and screen-reader support so that core flows are usable without a mouse.

Acceptance criteria:
- Primary navigation and forms keyboard accessible.
- Interactive elements have labels and visible focus states.

Implementation tasks:
- Add accessibility checklist to PR template.
- Run axe checks on key routes.

## Story 9.2 - E2E Smoke Coverage
As a team, we want smoke tests so that release confidence is measurable.

Acceptance criteria:
- CI runs smoke tests for:
  - open dashboard,
  - create automation,
  - run automation now,
  - view tasks.

Implementation tasks:
- Add Playwright with Bun test scripts.
- Add CI job and artifacts for failures.

## 12. Feature-by-Feature Implementation Order

1. Epic 0: Foundation and embedded serving.
2. Epic 6: Automations UX (already aligned with current backend direction).
3. Epic 2: Tasks and inbox.
4. Epic 3: Projects.
5. Epic 1: Dashboard.
6. Epic 5: Vault preview.
7. Epic 4: Calendar.
8. Epic 7: Chat.
9. Epic 8 + 9: Diagnostics, accessibility, E2E hardening.

## 13. Suggested Initial Milestone Split

## Milestone A - Vertical Slice (2-3 weeks)
- App shell, embedded handler, automations list/create/run-now, basic dashboard cards.

## Milestone B - Core GTD Workflow (2-4 weeks)
- Inbox triage, next actions board, projects list/detail.

## Milestone C - Planning and Intelligence (3-5 weeks)
- Calendar, vault preview, chat workflows, quality hardening.

## 14. Ready-to-Start Task Checklist

- [ ] Scaffold `web/` app with Bun + Vite + shadcn.
- [ ] Add Go embed static handler and `/app` route.
- [ ] Add frontend build step to Makefile/CI.
- [ ] Add API client + query provider.
- [ ] Implement automations pages against current endpoints.
- [ ] Add backend endpoints required for dashboard/tasks/projects.
- [ ] Add Playwright smoke tests before wider rollout.
