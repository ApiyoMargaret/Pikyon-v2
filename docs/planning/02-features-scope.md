# Section 2: Features & Scope

## 2.1 Feature Categories

```
Pikyon Features
├── Authentication & Security
├── Memory Management
├── Story Writing & AI Enrichment
├── Media Handling
├── Privacy & Access Control
├── Sharing & Notifications
├── Search & Discovery
├── Social Media Integration
└── User Profile & Settings
```

## 2.2 Authentication & Security

| Feature | Description | Priority |
|---|---|---|
| Email/Password registration | Sign up with email and password | Must Have |
| Email verification | Verification email sent on registration | Must Have |
| Google OAuth | One-click sign in with Google | Must Have |
| 2FA | TOTP two-factor authentication | Must Have |
| JWT + Refresh tokens | Stateless auth with rotation | Must Have |
| Password reset | Reset via email link | Must Have |
| Session management | View/revoke active sessions | Must Have |
| Account deletion | Permanently delete account and data | Must Have |
| Rate limiting | Limit failed login/PIN attempts | Must Have |

## 2.3 Memory Management

| Feature | Description | Priority |
|---|---|---|
| Create/edit/delete memory | Full CRUD, soft delete to trash (30-day grace) | Must Have |
| Restore from trash | Recover within 30 days | Must Have |
| Bulk trash / empty trash | Batch operations, max 50 items per request | Must Have |
| Memory date, location, tags | Metadata per memory | Must Have |
| Visibility toggle | Private / Public | Must Have |
| PIN lock | Per-memory 4-digit PIN | Must Have |
| Timeline view | Grouped by month/year | Must Have |

## 2.4 Story Writing & AI Enrichment

| Feature | Description | Priority |
|---|---|---|
| Write story manually | Plain text editor, fully user-authored | Must Have |
| AI caption from media | Gemini Vision analyzes upload, generates description | Must Have |
| AI tag suggestion | AI suggests relevant tags | Must Have |
| AI mood detection | Detects emotional tone | Must Have |
| Speech-to-text | Web Speech API (free, browser-native, Chrome/Edge) | Must Have |
| AI summarize mode | Narration → short summary | Must Have |
| AI polish mode | Narration → grammar-fixed, filler-removed journal entry | Must Have |
| AI social captions | Twitter/Instagram/LinkedIn captions | Must Have |
| User always edits AI output | Nothing saves without user review | Must Have |
| AI is always optional | User may ignore AI entirely | Must Have |

> AI features are **manually triggered** (user clicks "Analyze with AI"), never automatic on upload — this was an explicit product decision to keep AI assistive, not forced.

## 2.5 Media Handling

| Feature | Description | Priority |
|---|---|---|
| Photo upload | JPG, PNG, HEIC, WebP | Must Have |
| Video upload | MP4, MOV (100MB raw cap) | Must Have |
| Document upload | PDF | Must Have |
| Audio upload | MP3, WAV | Must Have |
| Auto compression (images) | browser-image-compression, silent | Must Have |
| Video compression | Server-side Go worker, staging bucket | Sprint 2 |
| Secure streaming | 5-minute pre-signed URLs | Must Have |
| Media never downloadable | Always streamed | Must Have |
| Thumbnail generation | Videos and documents | Must Have |
| Backend-verified metadata | Real file size/mime type fetched from Supabase Storage, never trusted from client | Must Have |

## 2.6 Privacy & Access Control

| Feature | Description | Priority |
|---|---|---|
| Private by default | All new memories start private | Must Have |
| PIN lock per memory | Independent of account password | Must Have |
| RLS enforcement | Database-level user isolation | Must Have |
| No URL guessing | Signed URLs, random object keys | Must Have |

## 2.7 Sharing & Notifications

| Feature | Description | Priority |
|---|---|---|
| Share via email | Recipient must have a Pikyon account | Must Have |
| Non-registered recipient | Invited to register, share honored after signup | Must Have |
| Share expiry / max views | Owner-controlled | Must Have |
| Revoke share link | Anytime | Must Have |
| View log | Who viewed, when | Must Have |
| Reactions | Heart, wow, sad — no re-sharing | Must Have |
| In-app + email notifications | Views, reactions, storage warnings | Must Have |

## 2.8 Search & Discovery

| Feature | Description | Priority |
|---|---|---|
| Keyword search | Full-text search (title, story, tags, location) | Must Have |
| Filters | Date range, media type, mood, visibility | Must Have |
| Cursor-based pagination | All list/search endpoints | Must Have |
| Semantic search | AI meaning-based search | Future Sprint |

## 2.9 Social Media Integration

| Feature | Description | Priority |
|---|---|---|
| AI-generated captions | Twitter, Instagram, LinkedIn | Must Have |
| Copy caption | One click | Must Have |
| Share to platform | WhatsApp, Twitter, Facebook, LinkedIn | Must Have |

## 2.10 User Profile & Settings

| Feature | Description | Priority |
|---|---|---|
| Profile view/edit | Name, avatar | Must Have |
| Change password | With session invalidation | Must Have |
| Manage 2FA / PIN | Enable, disable, remove | Must Have |
| Storage usage | Breakdown by media type | Must Have |
| Notification preferences | Configurable | Must Have |
| Active sessions | View and revoke | Must Have |

## 2.11 MVP vs Future

| Feature | MVP | Future |
|---|---|---|
| All features above | ✅ | — |
| Semantic/vector search | — | ✅ |
| Multi-language support | — | ✅ |
| Mobile native app | — | ✅ |
| Face recognition | — | ✅ |
| Collaborative memories | — | ✅ |
| Memory books/albums | — | ✅ |
| Redis-backed rate limiting/caching | — | ✅ |
| Custom domain + strict cross-site cookies | — | ✅ |

## 2.12 User Story Format

```
As a [type of user]
I want to [perform an action]
So that [I achieve a goal]

Acceptance Criteria:
  - Criterion 1
  - Criterion 2
```
