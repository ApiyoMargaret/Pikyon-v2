# ADR-011: Client-side image compression, server-side video compression (revised)

**Status:** Accepted (revised during planning review)

## Context
Large media uploads need compression before storage to control Supabase Storage usage against the free-tier 1GB cap. The original plan used FFmpeg.wasm client-side for both images and video. During architecture review this was identified as unsafe: FFmpeg.wasm requires WebAssembly threads and heavy memory allocation, and mobile Safari's memory cap routinely crashes the tab on high-resolution video.

## Decision
- **Images:** `browser-image-compression`, client-side, silent — safe on all browsers, no WebAssembly.
- **Video (MVP):** raw upload with a 100MB cap; no client-side compression.
- **Video (Sprint 2+):** server-side Go worker transcodes from a staging bucket to the permanent bucket; the user sees a "Processing video…" status.

## Consequences
**Positive:** Eliminates the mobile browser crash risk; image compression remains automatic and free; the video path has a documented upgrade route.
**Negative:** Videos ship uncompressed at MVP launch within the 100MB cap — an accepted, documented limitation, not a silent gap.
**Related:** ADR-017; `docs/planning/04-system-architecture.md` §4.6.
