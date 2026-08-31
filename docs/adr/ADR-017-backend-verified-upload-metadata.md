# ADR-017: Backend fetches real upload metadata, never trusts client-reported values

**Status:** Accepted

## Context
The original `POST /media/confirm` design accepted `object_key`, `file_size`, and `mime_type` directly from the client request body and trusted them when creating the `media_files` record. This allows a malicious client to spoof file size (corrupting storage-usage tracking) or inject a mismatched `object_key` pointing at a different object than what was actually uploaded, since nothing verified the claimed values against reality.

## Decision
`POST /media/confirm` now accepts only `upload_id` and `object_key` from the client. The backend:
1. Verifies the `upload_id` belongs to the authenticated user via the `pending_uploads` record.
2. Verifies the submitted `object_key` matches the one originally issued — cannot be substituted.
3. Fetches the **real** file metadata (Content-Length, Content-Type, dimensions) directly from the Supabase Storage API.
4. Creates the `media_files` record using only backend-verified values.

## Consequences
**Positive:** Closes a metadata-spoofing vulnerability; storage usage tracking and media type records are always accurate; object keys can't be substituted to point at another user's file.
**Negative:** One additional Supabase Storage API call per confirmed upload (negligible latency cost).
**Related:** Works together with the `pending_uploads` orphan-cleanup cron (see `05-database-design.md`), which handles the case where confirmation never arrives at all.
