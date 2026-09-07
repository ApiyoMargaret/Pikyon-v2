# Section 8: UI/UX Plan (Rewritten — based on real design references)


## 8.1 Design System: Revised

```
Two deliberate modes by CONTEXT, not a user-facing toggle:

  Marketing site (Homepage, Login, Register, public pages):
    LIGHT mode — white/cream backgrounds, dark navy text

  Authenticated app (Dashboard, Memory flip-card, Create/Edit,
  Settings, all post-login screens):
    DARK mode — near-black background, purple/indigo accents

Primary accent color (both modes): purple/indigo, ~#5B3FF5
  — REPLACES the earlier amber (#D4956A) direction entirely.

Typography: Inter only (ADR-015, unchanged) — bold/black
  weights for headlines, regular/medium for body and UI.

Components: shadcn/ui (ADR-014, unchanged).

Shape language: heavily rounded — pill-shaped buttons and nav,
  rounded-corner cards, soft drop shadows on light-mode cards.
  This is a shift from the earlier "subtle borders over shadows"
  direction — shadows are used deliberately here for a softer,
  warmer feel consistent with the real references.

Photography: REAL photography is now core to the visual identity
  (hero, login backdrop, footer illustration) — not abstract
  gradients or geometric placeholders as earlier drafts used.
```

## 8.2 Page Inventory — Revised

```
MARKETING (light mode):
  Homepage, Login, Register, Verify Email, Forgot/Reset Password

AUTHENTICATED APP (dark mode):
  Dashboard (calendar-card grid — see 8.5)
  Create/Edit Memory (compose flow — see 8.6)
  Timeline, Search Results, Shared Memory, Profile/Settings,
  Trash, Notifications

RETIRED:
  MemoryDetailPage as a separate route no longer exists — the
  dashboard's flip-card interaction (8.5) fully replaces it.
  This is a genuine structural simplification: one surface for
  viewing a complete memory instead of a card-then-detail-page
  handoff. (File structure note: frontend/src/pages/
  MemoryDetailPage.tsx should be removed from the scaffold or
  repurposed into the flip-card component itself.)
```

## 8.3 Homepage Structure — Revised

```
1. NAV — floating pill-shaped nav, not a full-width sticky bar.
   Logo mark (left) + 4 links (Features, Security, plus items
   TBD per real Pikyon scope) + solid "Go to App" pill button
   (right). White/light pill floating over the page background.

2. HERO — symmetrical layout:
   - Real photography flanking the headline on both left and
     right (people, warm/candid, not stock-generic)
   - Two-tone bold headline: dark navy line + purple-accent line
     (e.g. dark: "Every memory," accent: "properly preserved")
   - One-line subheading — MUST reflect Pikyon's real value prop
     (story-first, private, AI-enriched) — NOT "end-to-end
     encrypted, cross-platform, open-source" (that copy belongs
     to the reference app, not Pikyon — see architecture note
     below)
   - Dual CTA: solid dark "Sign up" pill + light-purple ghost
     "Login" pill, side by side
   - Row of 4 rounded, slightly-rotated photo cards beneath the
     CTA row

   ARCHITECTURE ALIGNMENT NOTE: subheading copy must not claim
   "end-to-end encrypted" — Pikyon's actual security model is
   RLS + 5-minute pre-signed URLs + optional PIN lock (see
   07-security-plan.md). Accurate subheading language: something
   like "Private by default. AI-enriched. Yours to share, on
   your terms."

3. FEATURE CAROUSEL — a curved, semi-circular 3D carousel of
   photo-backed feature cards that rotates continuously and
   pauses when a card is centered/facing the user, giving time
   to read. This REPLACES the earlier "3 alternating full-width
   sections" plan entirely — genuinely more distinctive.

   Implementation note: real 3D transform carousel (CSS
   transform-style: preserve-3d + rotateY on a circular track,
   or Framer Motion useAnimationFrame loop / a library like
   Swiper's 3D coverflow). This is non-trivial and should be
   scheduled as a dedicated Sprint 7 (Polish) task, not assumed
   as page-scaffolding-level effort in Sprint 1-2.

   Card content — Pikyon's REAL features, not the reference
   app's claims:
     Story-First Writing    — AI-assisted or fully your own words
     AI Enrichment           — captions, mood, tags on demand
     PIN-Locked Memories     — a second lock, independent of login
     Controlled Sharing      — email-based, revocable, expiring
     Secure Streaming        — 5-minute signed URLs, never public

4. SOCIAL PROOF — DECIDED: keep this section live from launch,
   using the hanging-tag-card visual mechanic (cards suspended
   from a rope/bar, alternating purple/light-blue), but as a
   FOUNDER'S NOTE, not fabricated testimonials.

   Content: real, honest cards written in the founder's own
   voice — e.g. "Why I built this," a note on the story-first
   philosophy, a note on privacy-by-default — NOT invented user
   names, photos, star ratings, or usage claims ("trusted by
   500+ users" is explicitly rejected — untrue at launch and
   misleading to visitors). Headline changes from "trusted by
   over 500+ users" to something honest, e.g. "Why Pikyon exists"
   or "A note from the person building this."

   Individual cards get swapped for genuine user testimonials
   organically as real ones come in post-launch — the mechanic
   stays, the content evolves from founder-voice to user-voice
   over time, never fabricated in either state.

   Cards animate with a subtle swing/sway (Framer Motion,
   small rotate oscillation e.g. [-2deg, 2deg], spring easing,
   staggered per card so they don't move in unison — either an
   idle continuous sway or a settle-after-scroll-into-view swing).

5. TRUST & SECURITY — 3-card grid, content must stay accurate
   to our real architecture:
     Zero Public Buckets      (5-min signed URLs only)
     Scoped Access Control    (per-memory PIN)
     Strict Data Isolation    (Postgres RLS)

6. FAQ — real objections, no fabricated content

7. FINAL CTA banner

8. FOOTER — solid purple/indigo block (not white, contrast
   with the rest of the light-mode page), white card floating
   inside it with a friendly illustrated icon at top-center.
   Column structure TRIMMED to match what Pikyon actually has
   at launch (the reference's "Open Source" and "Compare vs
   Google Photos/iCloud/Dropbox" columns don't apply — Pikyon
   is not open source and has no competitor-comparison content
   planned):
     Product    - Features, Security, Pricing
     Company    - About, Contact
     Legal      - Privacy, Terms
     Support    - Help, Articles
   Social icons + language selector + copyright row at the
   bottom, matching the reference's bottom bar pattern.
```

## 8.4 Login / Register — Revised

```
Card-over-photo-collage pattern (not a solid neutral background
as originally specced — that earlier "avoid photo backdrops,
they read as dated 2018 SaaS" guidance is REVISED: the deciding
factor is execution quality, not the concept itself. A soft,
scattered polaroid-style photo collage behind a clean, minimal
white card works well and stays thematically tied to "photos" -
it does not read as dated when done this way).

Structure:
  - Scattered, slightly-rotated real photos as a soft background
    collage (light mode, low visual noise so the card stays the
    clear focal point)
  - Centered white card, rounded corners, soft shadow
  - "Welcome back" headline + "Sign in to continue your story"
    subtext (subtext phrasing intentionally ties to Pikyon's
    story-first identity)
  - Google OAuth button FIRST, above the divider (revised order
    — originally specced as a secondary option below email/pass)
  - "OR" divider
  - Email field, Password field — inline labels ABOVE each input
    (revised from floating-label pattern originally specced)
  - Solid purple "Sign in" pill button, full-width
  - Footer link: "New here? Create one" (register), or the
    reverse phrasing on the Register screen
```

## 8.5 Dashboard — Fully Revised (Calendar-Card Flip System)

This is the single biggest structural change from the original plan. The masonry/uniform-grid photo-thumbnail dashboard is REPLACED entirely by a torn-calendar-page card metaphor.

```
LAYOUT (dark mode):
  Sidebar (left, dark, ~220px):
    Logo mark + wordmark (top)
    Solid purple "Upload" pill button
    Nav: Private, Public, Shared, Favorites, Trash
      (DECIDED: Favorites ADDED as a new lightweight MVP feature
      — a single is_favorite boolean toggle per memory, no new
      table needed. Albums is DEFERRED to a future sprint —
      a real feature requiring its own schema (albums +
      album_memories join table) and sharing implications,
      too large to fold into scope without dedicated planning.
      See schema/API additions below.)
    Storage meter (bottom): percentage bar + "X GB of Y GB used"
      — maps directly to GET /users/me/storage response shape
      (06-api-design.md 6.5), real data, not mockup
    Settings (bottom, separated)

  Top bar:
    Wide search input with Pikyon's real inline filter-tag
      syntax (type:image, mood:joyful, after:2026-01-01 — per
      06-api-design.md 6.10 query params), not plain-text-only
      search as shown in the reference
    Notification bell (with unread dot)
    User avatar + name + dropdown

  Main content:
    "Your Memories" heading
    Grid/List view toggle + sort dropdown (top-right)
    Sticky month headers ("October 2026")
    Grid of CALENDAR CARDS, chronological, grouped by month

CALENDAR CARD (front face — memory NOT yet flipped):
  Torn-notepad-page visual: spiral binding illustration at top,
  month abbreviation ("OCT") in small caps, large bold day
  number, memory title below, small torn-paper texture at the
  bottom edge of the card.

  ADDITIONS beyond the reference (Pikyon-specific, since these
  differentiators weren't shown in the original mock):
    - Small AI mood/tag indicator (colored dot or tiny chip)
      near the title, surfacing AI enrichment on the main grid
      view, not hidden until the card is opened
    - Lock icon overlay + darkened/blurred card treatment for
      PIN-locked memories — this state must exist even though
      the reference didn't show it, since PIN lock is a core
      Pikyon feature (02-features-scope.md 2.6)
    - Public/Private badge, small, consistent with the badge
      language used elsewhere in the app

CARD INTERACTION — flip reveals the FULL memory (no separate
detail page/route, per explicit product decision):
  1. User clicks a card
  2. Flip animation plays (Framer Motion 3D flip, ~400-600ms)
  3a. IF unlocked/public: back face reveals the complete memory
      — media (streamed via a freshly-fetched 5-minute signed
      URL, requested on click, not prefetched for all visible
      cards — one memory viewed at a time keeps this consistent
      with the zero-direct-serving media architecture in
      04-system-architecture.md 4.5), full story text, AI mood/
      tag badges, all on that single back face — this literally
      is the memory detail view, not a link to one
  3b. IF PIN-locked: back face shows a PIN entry prompt INSTEAD
      of memory content. Correct PIN submitted -> content swaps
      into that same back face (no second flip, no separate
      modal taking over the screen). Incorrect PIN -> inline
      error, same as the PINInput component already specced.
  4. Timing note: the flip animation (fixed duration) and the
     signed-URL fetch (network-dependent duration) are two
     separate timers. If the fetch hasn't resolved by the time
     the flip completes, the back face shows a brief shimmer/
     loading state rather than a blank or broken card.

  Click again / close button -> flips back to the grid.
```

## 8.6 Create/Edit Memory — Unchanged in Concept, Reconfirmed as a Distinct Stage

```
This is explicitly a SEPARATE lifecycle stage from viewing —
confirmed during design review: the calendar-card/flip system
in 8.5 only applies to memories that are ALREADY complete
(uploaded, story written, saved). A freshly-started memory
(upload in progress, description not yet written) lives in its
own compose flow and is NOT represented as a calendar card until
it's finished and saved.

Compose flow (dark mode, consistent with the rest of the
authenticated app):
  Story field offers two clearly visible, non-forced paths:
    [ Write it myself ]     [ Use AI assistance ]
  AI assistance sub-paths: caption-from-media, speech-to-text
  (summarize or polish). User always reviews/edits before saving
  — nothing auto-saves from AI output (unchanged from original
  plan, 02-features-scope.md 2.4).

  Media upload, tags, visibility toggle, PIN-lock toggle — same
  fields as originally specced, now styled to match the dark
  authenticated-app system rather than the earlier neutral plan.

  Upload progress: floating bottom-right progress drawer
  (Google Drive/Photos pattern, unchanged from original plan)
  — still necessary given the direct-to-Supabase pre-signed
  upload architecture (04-system-architecture.md 4.5).

  On save/completion -> memory becomes a calendar card, appears
  in the dashboard grid under its correct month group.
```

## 8.7 Search Bar Behavior — Unchanged
Inline filter-tag syntax (type:image, mood:joyful,
after:2026-01-01) or a quick filter-pill dropdown, matching
06-api-design.md 6.10 query params exactly. Confirmed still
wanted — the reference dashboard's plain search bar does not
reflect this; Pikyon's implementation should still include it.

## 8.8 Product Decisions (Resolved)

```
1. Favorites — ADDED to MVP scope. Requires:
   - Schema: ALTER TABLE memories ADD COLUMN is_favorite BOOLEAN
     NOT NULL DEFAULT FALSE; (05-database-design.md needs this
     migration added — new migration file, sequential number
     after the existing 10)
   - API: PATCH /memories/:id already supports partial updates
     (06-api-design.md 6.6) — is_favorite becomes a valid field
     on that existing endpoint, no new endpoint required. Add a
     GET /memories?favorite=true filter option for the Favorites
     nav view.
   - Albums — DEFERRED to a future sprint, out of MVP scope
     (02-features-scope.md 2.11 future list). Not implemented,
     not in the sidebar.

2. Feature carousel (8.3.3) — CONFIRMED scheduled as a dedicated
   Sprint 7 (Polish) task, not expected in early sprint page
   scaffolding.

3. Social proof section (8.3.4) — CONFIRMED: stays live from
   launch as a founder's-note version of the hanging-tag-card
   mechanic. See 8.3.4 for full content direction.
```

## 8.9 Design Changelog (This Revision)

| Area | Original Plan | Revised Plan | Reason |
|---|---|---|---|
| Color system | Neutral slate, amber accent, single mode | Purple/indigo accent, TWO modes (light marketing / dark app) | Real design references provided, direct visual match required |
| Dashboard layout | Uniform 4-column grid, no masonry, separate detail page | Calendar-card grid with flip-to-reveal interaction, no separate detail page | Stronger, more thematic visual metaphor; explicit product decision to fold detail view into the flip |
| Hero visual | Browser-framed dashboard screenshot | Real photography flanking headline + photo card row | Matches provided references; more emotional, still product-honest |
| Feature section | 3 alternating full-width text+screenshot sections | Curved 3D rotating carousel, pause-on-center | Provided reference is more distinctive; flagged as non-trivial build effort |
| Social proof | Not originally planned as a section | Hanging-tag-card mechanic, content TBD (real testimonials vs founder's note) | Provided reference; fabricated content explicitly rejected per project's honesty standard |
| Login backdrop | "No photo backdrop, dated pattern" (explicit original guidance) | Photo-collage backdrop behind a minimal card — GUIDANCE REVERSED | Execution quality, not the concept, was the actual problem with earlier attempts |
| Login OAuth position | Below email/password fields | Above, first option before the divider | Matches provided reference |
| Footer | 4-column (Product/Security/Legal/Copyright) | 4-column, renamed/trimmed from a 6-column reference that included Open Source and Compare columns not applicable to Pikyon | Reference footer assumed a more mature, open-source, competitor-comparison product than Pikyon's actual MVP |
| Typography | Inter only (ADR-015) | Unchanged | Already correct, no conflict with new references |
| Component library | shadcn/ui (ADR-014) | Unchanged | Already correct, no conflict with new references |
