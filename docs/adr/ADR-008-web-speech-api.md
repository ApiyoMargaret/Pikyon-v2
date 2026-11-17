# ADR-008: Use the browser-native Web Speech API for speech-to-text

**Status:** Accepted

## Context
Pikyon lets users narrate memories by voice, then summarizes or polishes the transcript with AI. Server-side transcription services charge per minute, conflicting with the no-paid-infrastructure constraint.

## Decision
Use the browser-native Web Speech API for transcription, entirely client-side. Transcript text is then sent to Gemini for summarize/polish processing. When the API is unavailable (Firefox, Safari), show the plain text editor as the fallback.

## Consequences
**Positive:** Free, no per-minute cost, no API key or backend integration for the transcription step.
**Negative:** Chrome/Edge only. The fallback preserves the full authoring flow for other browsers.
**Rejected alternative:** OpenAI Whisper API — more accurate and cross-browser, but costs per minute of audio and is ruled out by the no-paid-infrastructure decision.
**Related:** ADR-006, ADR-015.
