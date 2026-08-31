# Section 8: UI/UX Plan

## 8.1 Design System

```
Typography:   Inter (all weights) — no serif, ADR-015
Icons:        Lucide (via shadcn)
Components:   shadcn/ui on Tailwind CSS — ADR-014
Palette:      Neutral slate grays, high-contrast text
              bg: neutral-50 (light) / neutral-950 (dark)
              borders: neutral-200 / neutral-800
              One accent color for primary actions only
              (amber #D4956A, used sparingly, not as a dominant theme)
Elevation:    Subtle borders preferred over drop shadows
Motion:       Framer Motion, 0.15–0.3s, purposeful only, never decorative
```

> **Design direction changelog**: earlier exploratory mockups (dark-cinematic-amber palette, then an archival/contact-sheet concept) were rejected as reading generic/AI-templated. Benchmarked instead against Ente Photos, Linear, and Notion — settled on a neutral, high-contrast, restrained system with a single sparing accent color. Full production visual fidelity to be built as real React/Tailwind components, not chat-tool mockups.

## 8.2 Page Inventory

Homepage, Login, Register, Verify Email, Forgot/Reset Password, Dashboard, Memory Detail (Reader View), Create/Edit Memory, Timeline, Search Results, Shared Memory, Profile/Settings, Trash, Notifications.

## 8.3 Homepage Structure

```
1. Nav — sticky, minimal: logo, Features/Privacy/Pricing, Sign In CTA
2. Hero — benefit headline + subtext + dual CTA, REAL browser-framed
   dashboard screenshot below (not abstract art — this literally
   cannot be finalized until the dashboard UI exists)
3. Feature sections — 3 alternating full-width (text one side,
   UI preview the other):
     Private Storage & Signed URLs
     AI Story & Speech Polish
     PIN Locks & Recipient Sharing
4. Comparison table — Generic Cloud Storage vs Pikyon
5. Trust & Security — 3-card grid:
     Zero Public Buckets (5-min signed URLs only)
     Scoped Access Control (per-memory PIN — corrected from an
       earlier draft that mistakenly listed "Local Encryption",
       which isn't part of our actual architecture)
     Strict Data Isolation (Postgres RLS)
6. FAQ — real objections, no fabricated content
7. Final CTA banner
8. Footer — multi-column: Product / Security / Legal / Copyright
```

## 8.4 Dashboard Structure

```
Sidebar (collapsible): Library, Timeline, Private, PIN Locked,
  Shared, Trash, storage meter pinned at bottom
Top bar: wide centered search with inline filter tags
  (type:image, mood:joyful, after:2026-01-01 — maps directly to
  the query params in 06-api-design.md §6.10), Upload button, avatar
Grid: sticky date headers ("July 2026"), uniform 4-column
  responsive grid — NOT masonry (reconsidered: real photo apps
  use consistent grids because they scan faster than variable
  heights)
Card: hover → top-left checkbox (multi-select), bottom-left media
  type badge, lock icon if PIN-protected
Selection bar: floating top bar on selection — Share, Add to PIN
  Vault, Delete, Deselect
Upload feedback: floating bottom-right progress drawer (Google
  Drive/Photos pattern) — itemized per-file progress, Supabase
  completion status, retry on error. Needed because our upload
  architecture is direct-to-Supabase via pre-signed URLs
  (06-api-design.md §6.7), so per-file state must be tracked
  client-side.
```

## 8.5 Memory Detail (Reader View)

```
Desktop: two-column modal, 65/35 split
  Left  (65%): media viewport — carousel/waveform/video,
                object-fit: contain
  Right (35%): title, date, location, AI mood/tag badges,
                PIN-lock status, shared recipient avatars,
                full story text
Mobile: stacked, media on top, story panel below
```
> Revised from an earlier vaguer "full-screen takeover" note into this specific two-column spec.

## 8.6 Create/Edit Memory
Story field offers two clearly visible, non-forced paths at all times:
```
[ Write it myself ]     [ Use AI assistance ]
```
AI assistance sub-paths: caption-from-media, speech-to-text (summarize or polish). User always reviews/edits before saving — nothing auto-saves from AI output.

## 8.7 Authentication Screen

```
Centered card, ~400px, solid neutral background (no photo backdrop —
  reconsidered from an earlier split-screen-with-photo pattern,
  which reads as a dated 2018-era SaaS template)
Logo top-center → "Welcome back" → subtext
Primary: Google OAuth single-click button
Divider: "or continue with email"
Email + password (floating labels), "Forgot password?" right-aligned
Submit button, footer link "Don't have an account? Sign up"
```

## 8.8 Search Bar Behavior
Inline filter-tag syntax (`type:image`, `mood:joyful`, `after:2026-01-01`) or a quick filter-pill dropdown — must match the exact query parameters defined in the backend API spec, not a separate ad-hoc filter vocabulary.

## 8.9 Stack Additions

```
ADR-014: shadcn/ui — free, MIT licensed, component source copied
  into frontend/src/components/ui/, styled per-project, no runtime
  dependency cost.

ADR-015: Inter-only typography — replaces an earlier Playfair
  Display + DM Mono "editorial memoir" direction, which conflicted
  with the neutral-SaaS benchmark (Ente/Linear) once the homepage
  brief was finalized. Two typography personalities can't coexist
  across the same app.
```

## 8.10 Design Process Note
Chat-based inline mockup tools (both the internal sketch widget and a hand-built HTML preview) were used for early exploration but consistently fell short of production visual quality. **Final visual design will be built as real React + Tailwind + shadcn components during implementation, not as static previews** — the dashboard and memory detail screens should be treated as living UI from Sprint 1 onward, refined in place rather than mocked separately first.
