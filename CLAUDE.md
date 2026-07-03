# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

IM-Claude is the IM (Instant Messaging) capability for the 数莲 PaaS platform, built on top of **OpenIM v3.8**. It supports single chat, group chat, structured message cards, bot integration (Webhook), and compliance features (sensitive word filtering, audit logging). Clients: iOS, Android, Web.

Source of truth for requirements: `docs/prd.md`

## Tech Stack (per PRD)

| Layer | Technology |
|---|---|
| IM Engine | OpenIM v3.8 (Go) |
| App Client | iOS 12+ / Android 5+ |
| Web Client | Existing PaaS frontend framework |
| Database | MySQL 8.0, Redis 7.0, MongoDB 7.0 |
| Message Queue | Kafka 3.5.1 |
| File Storage | MinIO / S3-compatible |
| Deployment | Docker Compose (dev), K8s Helm (prod) |

## Architecture

Three-layer design:

1. **Client** (App / Web) — UI rendering, WebSocket connection management
2. **IM Server** (OpenIM) — message routing, persistence, offline push, multi-device sync
3. **Business System** — external integration via API / Webhook

Communication: WebSocket (real-time), HTTP (offline message pull), Webhook (bot events)

## Project Modules (from PRD)

- **Basic System** — splash screen, login, global search, cache management, user status, tenant switching
- **Session & Chat** — conversation list, single/group chat, full message types (text, image, video, file, voice, card, SSE streaming)
- **Group Management** — CRUD groups, role-based permissions (owner/admin/member), mute, join requests
- **Contacts** — org tree, member search, profile cards
- **Bot & Compliance** (Web admin) — system/user bots, sensitive words, rate limiting, audit log

## Key Conventions

- All session and message data is synced across all clients in real time
- Role system: group owner > admin > member (strict permission hierarchy)
- Message sending goes through sensitive-word check before delivery
- Tenancy: all tables include a tenant ID field; data is tenant-isolated
- i18n: client-side key-based translation; server returns no static text
- Capacity limits per PRD §5.4 (e.g., max 5 pinned conversations, 2000 members/group, 5000 chars/message)

## Design Files

UI mockups: `docs/MasterGo/` (PNG exports from MasterGo)

## Getting Started

Development environment uses Docker Compose for middleware:
- MySQL 8.0, Redis 7.0, MongoDB 7.0
- Kafka 3.5.1
- MinIO (S3-compatible object storage)
- OpenIM Server v3.8

Start the local dev environment via `docker-compose up` (once compose file is created).
