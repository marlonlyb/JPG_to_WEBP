# Skill Registry

**Delegator use only.** Any agent that launches sub-agents reads this registry to resolve compact rules, then injects them directly into sub-agent prompts. Sub-agents do NOT read this registry or individual SKILL.md files.

## User Skills

| Trigger | Skill | Path |
|---------|-------|------|
| When building AI chat features - breaking changes from v4. | ai-sdk-5 | /home/marlon_ly/.claude/skills/ai-sdk-5/SKILL.md |
| When creating a pull request, opening a PR, or preparing changes for review. | branch-pr | /home/marlon_ly/.claude/skills/branch-pr/SKILL.md |
| When building REST APIs with Django - ViewSets, Serializers, Filters. | django-drf | /home/marlon_ly/.claude/skills/django-drf/SKILL.md |
| When writing C# code, .NET APIs, or Entity Framework models. | dotnet | /home/marlon_ly/.config/opencode/skills/dotnet/SKILL.md |
| When writing Go tests, using teatest, or adding test coverage. | go-testing | /home/marlon_ly/.claude/skills/go-testing/SKILL.md |
| When user asks to release, bump version, update homebrew, or publish a new version. | homebrew-release | /home/marlon_ly/.config/opencode/skills/homebrew-release/SKILL.md |
| When creating a GitHub issue, reporting a bug, or requesting a feature. | issue-creation | /home/marlon_ly/.claude/skills/issue-creation/SKILL.md |
| When user asks to create an epic, large feature, or multi-task initiative. | jira-epic | /home/marlon_ly/.config/opencode/skills/jira-epic/SKILL.md |
| When user asks to create a Jira task, ticket, or issue. | jira-task | /home/marlon_ly/.config/opencode/skills/jira-task/SKILL.md |
| When user says "judgment day", "judgment-day", "review adversarial", "dual review", "doble review", "juzgar", "que lo juzguen". | judgment-day | /home/marlon_ly/.claude/skills/judgment-day/SKILL.md |
| When working with Next.js - routing, Server Actions, data fetching. | nextjs-15 | /home/marlon_ly/.claude/skills/nextjs-15/SKILL.md |
| When writing E2E tests - Page Objects, selectors, MCP workflow. | playwright | /home/marlon_ly/.claude/skills/playwright/SKILL.md |
| When user wants to review PRs (even if first asking what's open), analyze issues, or audit PR/issue backlog. | pr-review | /home/marlon_ly/.config/opencode/skills/pr-review/SKILL.md |
| When writing Python tests - fixtures, mocking, markers. | pytest | /home/marlon_ly/.claude/skills/pytest/SKILL.md |
| When writing React components - no useMemo/useCallback needed. | react-19 | /home/marlon_ly/.claude/skills/react-19/SKILL.md |
| When writing Angular components, services, templates, or making architectural decisions about component placement. | scope-rule-architect-angular | /home/marlon_ly/.config/opencode/skills/angular/SKILL.md |
| When user asks to create a new skill, add agent instructions, or document patterns for AI. | skill-creator | /home/marlon_ly/.claude/skills/skill-creator/SKILL.md |
| When building a presentation, slide deck, course material, stream web, or talk slides. | stream-deck | /home/marlon_ly/.config/opencode/skills/stream-deck/SKILL.md |
| When styling with Tailwind - cn(), theme variables, no var() in className. | tailwind-4 | /home/marlon_ly/.claude/skills/tailwind-4/SKILL.md |
| When reviewing technical exercises, code assessments, candidate submissions, or take-home tests. | technical-review | /home/marlon_ly/.config/opencode/skills/technical-review/SKILL.md |
| When writing TypeScript code - types, interfaces, generics. | typescript | /home/marlon_ly/.claude/skills/typescript/SKILL.md |
| When using Zod for validation - breaking changes from v3. | zod-4 | /home/marlon_ly/.claude/skills/zod-4/SKILL.md |
| When managing React state with Zustand. | zustand-5 | /home/marlon_ly/.claude/skills/zustand-5/SKILL.md |

## Compact Rules

### ai-sdk-5
- Import `useChat` from `@ai-sdk/react`, not `ai`.
- Use `DefaultChatTransport({ api })` and `sendMessage()` instead of v4 input helpers.
- Render `message.parts`; do not assume `message.content` is a string.
- Handle text, image, tool-call, and tool-result parts explicitly.
- Server handlers should use `streamText()` and return `toDataStreamResponse()`.

### branch-pr
- Every PR MUST link an approved issue and include exactly one `type:*` label.
- Branch names MUST match `type/description` in lowercase.
- Use conventional commits and never add `Co-Authored-By` trailers.
- Use the PR template with issue link, summary, file table, and test plan.
- Ensure automated checks pass before merge.

### django-drf
- Prefer `ModelViewSet` plus action-specific serializers via `get_serializer_class()`.
- Put filtering in `FilterSet` classes, not ad-hoc query parsing.
- Keep serializer roles explicit: read, create, and update serializers.
- Use DRF permissions classes for authz logic instead of inline conditionals.
- Register routes through DRF routers for consistent REST endpoints.

### dotnet
- Use Minimal APIs with typed results for new endpoints.
- Prefer primary constructors for DI; avoid manual field assignment constructors.
- Follow Clean Architecture boundaries: Domain, Application, Infrastructure, WebApi.
- Configure EF Core with Fluent API and `ApplyConfigurationsFromAssembly()`.
- Use repositories only when they add real abstraction value over `DbContext`.

### go-testing
- Prefer table-driven tests for behavioral coverage.
- Test Bubbletea state transitions directly through model updates.
- Use `teatest` for interactive TUI flows when needed.
- Use golden files for view/output regressions.
- Compare expected errors explicitly instead of only checking non-nil.

### homebrew-release
- Match the project's required tag format before release.
- Build binaries or tarballs according to project release type.
- Create GitHub releases with correct assets and notes.
- Update Homebrew formula version and SHA256 values together.
- Commit and push repo and tap changes separately with conventional messages.

### issue-creation
- Always use the GitHub issue template; blank issues are disabled.
- Search for duplicates before creating a new issue.
- Issues start with `status:needs-review`; PRs require later `status:approved`.
- Route questions to Discussions, not Issues.
- Fill all required template fields, including repro or problem statement.

### jira-epic
- Use title format `[EPIC] Feature Name`.
- Include feature overview, grouped requirements, technical considerations, and checklist.
- Cover performance, data integration, and UI component concerns explicitly.
- Use Mermaid diagrams for architecture, flow, or state when useful.
- Write requirements as clear, testable bullets by functional area.

### jira-task
- Split multi-component work into separate tasks instead of one large ticket.
- For bugs, create sibling tasks per component; for features, use parent + child tasks.
- Parent feature tasks stay user-facing; child tasks hold technical details.
- Include dependencies (`blocked by`, `blocks`, `parent`) explicitly.
- List affected files, acceptance criteria, and testing notes in technical tasks.

### judgment-day
- Resolve and inject project standards before launching judges.
- Run two independent blind reviews in parallel; never let judges influence each other.
- Synthesize findings as confirmed, suspect, or contradictory.
- If confirmed issues exist, fix them and re-run both judges.
- Escalate after two fix iterations if both judges still do not pass.

### nextjs-15
- Use App Router file conventions under `app/`.
- Default to Server Components; add `"use client"` only when interactivity is required.
- Prefer Server Actions for mutations and revalidation.
- Fetch data in parallel with `Promise.all` and use Suspense for streaming.
- Export metadata from route files instead of using a head component.

### playwright
- If MCP tools are available, explore the UI with them before writing tests.
- Keep one spec file per page/feature; do not split by scenario unnecessarily.
- Prefer `getByRole`, then `getByLabel`, then `getByText`; use test ids last.
- Avoid CSS/id selectors unless unavoidable.
- Use Page Object models with shared base behavior for navigation and helpers.

### pr-review
- Always gather PR/issue metadata first, then read current code for context.
- Evaluate code quality, tests, breaking changes, conflicts, hygiene, and docs.
- Treat secrets, debug files, syntax issues, and unsafe configs as hard red flags.
- Use structured output separating ready, needs work, and do not merge.
- Do not approve changes without evidence from diffs and current code.

### pytest
- Organize tests with clear assertions and `pytest.raises` for failures.
- Use fixtures for reusable setup and teardown; share common fixtures in `conftest.py`.
- Prefer mocks/patches for external services and unstable dependencies.
- Use `@pytest.mark.parametrize` for matrix-style input coverage.
- Register and use markers consistently for slow or integration test groups.

### react-19
- Do not add `useMemo` or `useCallback` for routine memoization.
- Use named React imports; avoid default `React` imports.
- Prefer Server Components first and client components only when browser/state APIs are needed.
- Use `use()` for promises/context when appropriate.
- Treat `ref` as a normal prop; no `forwardRef` unless legacy interop requires it.

### scope-rule-architect-angular
- Use standalone components, `inject()`, signals, and modern control flow syntax.
- Keep one-feature code local; move code to shared only when used by 2+ features.
- Prefer reactive forms, typed APIs, and `OnPush` change detection.
- Avoid `any`, `ngOnInit`, and NgModule-based feature organization.
- Name directories by business capability, not by technical layer alone.

### skill-creator
- Create skills only for repeatable, non-trivial patterns.
- Use the standard `skills/{name}/SKILL.md` structure with concise frontmatter.
- Put reusable templates/schemas in `assets/` and local doc pointers in `references/`.
- Keep examples minimal and focus on critical rules over long explanation.
- Register new skills in the relevant agent instruction index after creation.

### stream-deck
- Build slide decks as static HTML/CSS/JS with no framework or build step.
- Use inline SVG for diagrams; do not rely on external image assets for core visuals.
- Keep slides within a 100dvh presentation viewport without vertical scrolling.
- Use the Kanagawa Blur palette and maintain minimum contrast on dark surfaces.
- Structure slides with module rails and consistent `data-index`, `data-module`, and tone metadata.

### tailwind-4
- Never use `var()` or hex colors directly in `className`.
- Use plain `className` for static styles and `cn()` only for conditional/merged classes.
- Put truly dynamic values in `style`, not generated Tailwind strings.
- Use CSS custom properties only for libraries that cannot consume Tailwind classes.
- Prefer semantic Tailwind utilities over ad-hoc arbitrary values.

### technical-review
- Review structure, key files, tests, and red flags before scoring.
- Score the six required factors with concrete evidence.
- Treat missing tests, secrets, leaked corp data, and security gaps as major concerns.
- Prefer concise markdown tables with strengths, concerns, and follow-up questions.
- Judge maintainability and judgment, not personal style preferences.

### typescript
- Define const objects first, then derive union-like types from them.
- Keep interfaces flat; extract nested structures into named interfaces.
- Never use `any`; prefer `unknown`, generics, and explicit type guards.
- Use utility types instead of duplicating shape variations.
- Use `import type` where imports are type-only.

### zod-4
- Use Zod 4 top-level validators like `z.email()`, `z.uuid()`, and `z.url()`.
- Replace `nonempty()` with `.min(1)` and use object `error` options for messages.
- Prefer `safeParse` when validation failures are expected control flow.
- Use discriminated unions for tagged result objects.
- Keep coercion, preprocess, and refine logic inside schemas when validation owns the rule.

### zustand-5
- Define stores with explicit interfaces and focused actions.
- Use selectors for single fields and `useShallow` for multi-field picks.
- Avoid selecting the whole store in components.
- Model async actions with loading/error state transitions.
- Use middleware like `persist` only when state truly must survive reloads.

## Project Conventions

| File | Path | Notes |
|------|------|-------|
| None | — | No project-level convention files detected in the repository root. |
