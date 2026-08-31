# ADR-015: Inter-only typography system

**Status:** Accepted

## Context
Original UI/UX plan (Section 8, early draft) specified Playfair Display (serif, "editorial memoir" feel) for headings paired with DM Mono for metadata. When the homepage brief was benchmarked against Ente Photos, Linear, and Notion — the actual visual reference target — those products use a single neutral sans-serif system (Inter/Geist) throughout. Running two typography personalities (warm-editorial serif vs. clean-neutral-SaaS) across the same app reads as inconsistent.

## Decision
Adopt Inter exclusively across the entire application — headings, body, and UI text. Drop Playfair Display and DM Mono.

## Consequences
**Positive:** Visual consistency matching the chosen benchmark products; simpler font-loading (one typeface family, fewer weights to ship); avoids the "two different apps stitched together" feeling.
**Negative:** Loses some of the distinct "personal memoir" warmth the serif choice was originally meant to convey — compensated for through layout, copy voice, and the neutral-with-single-accent color system rather than typography.
