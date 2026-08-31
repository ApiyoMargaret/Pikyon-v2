# Section 1: Project Overview

## 1.1 Project Name
**Pikyon** — Intelligent Digital Memory Platform

## 1.2 Project Description
Pikyon is an intelligent digital memory platform that turns static photo collections into dynamic personal narratives. By combining secure cloud storage with AI-driven content enrichment, rich storytelling tools, and privacy-first sharing, Pikyon transforms raw media files into curated, living archives.

## 1.3 Problem Statement
Every year, we capture thousands of photos and videos, yet we keep losing the actual stories behind them. Basic cloud storage acts like a cold filing cabinet — it saves where and when a photo was taken, but completely forgets the emotion and reason behind the memory. Meanwhile, social media pushes us to post intimate life events publicly just to show them to friends and family. This leaves people stuck between hiding their memories in forgotten folders or oversharing them online. Without an easy way to organize and tell these stories, our most meaningful moments get lost in digital clutter. Pikyon was built to change that — giving people a private, intelligent space where memories are not just stored, but truly preserved and told.

## 1.4 Project Goals

| # | Goal | Priority |
|---|---|---|
| 1 | Give users a secure private space to store and organize memories | Must Have |
| 2 | Enable rich storytelling attached to every memory | Must Have |
| 3 | Use AI to help users describe and enrich their memories | Must Have |
| 4 | Allow controlled sharing with other registered users via email | Must Have |
| 5 | Provide smart search and discovery of memories | Must Have |
| 6 | Deliver a beautiful, intuitive and responsive UI | Must Have |
| 7 | Ensure world-class security and data privacy | Must Have |
| 8 | Support semantic/vector search | Future Sprint |
| 9 | Support multiple languages | Future Sprint |
| 10 | Mobile app (iOS/Android) | Future Sprint |

## 1.5 Target Users

| User Type | Description |
|---|---|
| Primary | Individuals who want to preserve personal memories with stories |
| Secondary | Families sharing private memories with each other |
| Tertiary | Creatives who use journaling and storytelling as a habit |

## 1.6 Project Scope

### In Scope (MVP)
- User authentication (Email + Google OAuth + 2FA)
- Memory upload (photos, videos, documents, audio)
- Client-side automatic media compression before upload (images); server-side compression for video (Sprint 2)
- Story writing — fully user-authored OR AI-assisted, always the user's choice
- Speech-to-text story enrichment via Web Speech API (free, browser-native)
- AI auto-caption and smart tag suggestion (Gemini Vision, manually triggered)
- Mood detection on memories
- Private and public memory sections with PIN lock
- Controlled sharing via email link (Resend)
- Smart keyword search and discovery
- Social media caption generation
- Responsive web application (English only)

### Out of Scope (MVP)
- Mobile native app
- Multi-language support
- Semantic/vector search
- Music/audio soundtrack integration
- Video editing features
- Face recognition
- Apple OAuth
- Custom domain (budget constraint — revisit post-MVP)

## 1.7 Success Metrics

| Metric | Target |
|---|---|
| User registers and uploads first memory | Under 3 minutes |
| AI caption generation response time | Under 5 seconds |
| Media upload with compression | Under 30 seconds for 50MB file |
| Page load time | Under 2 seconds |
| API response time | Under 500ms for 95% of requests |
| Uptime | 99.5% |

## 1.8 Project Timeline (High Level)

| Phase | What | Duration |
|---|---|---|
| Sprint 0 | Planning, setup, architecture | 1 week |
| Sprint 1 | Auth & database foundation | 1 week |
| Sprint 2 | Memory CRUD + media upload | 1 week |
| Sprint 3 | AI features + speech-to-text | 1 week |
| Sprint 4 | Sharing + notifications | 1 week |
| Sprint 5 | Search, PIN lock, 2FA & settings | 1 week |
| Sprint 6 | Testing, CI/CD & hardening | 1 week |
| Sprint 7 | Polish & launch | 1 week |

> See `10-sprint-plan.md` for the full breakdown and rationale behind sprint sequencing changes.
