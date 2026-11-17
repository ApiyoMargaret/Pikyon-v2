# ADR-006: Use Google Gemini API for AI features

**Status:** Accepted

## Context
Pikyon needs AI for auto-captioning media, mood detection, tag suggestion, and summarizing/polishing speech transcripts — all under a strict no-paid-infrastructure MVP constraint.

## Decision
Use the Google Gemini API (Vision + text generation) for all AI-driven features. The model name is supplied through `GEMINI_MODEL` and must be a model currently listed as free-tier in AI Studio; it is never hardcoded.

## Consequences
**Positive:** Genuinely free tier (15 req/min, 1500/day) sufficient for MVP usage; native multimodal support covers caption-from-image without a second service; one API covers captioning, mood, tags and text polishing.
**Negative:** Free-tier ceiling could be hit with growth — mitigated by the 20/hour/user AI endpoint limit, well below the Gemini ceiling; model availability changes, so the model is configuration, not code.
**Mitigation:** All AI work runs in a goroutine behind a job record, so a slow or unavailable Gemini call never blocks an HTTP request and can be retried by the user.
**Related:** ADR-008, ADR-011; `docs/planning/04-system-architecture.md` §4.7.
