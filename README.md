# CampusCare API

CampusCare is a RESTful backend API for a university mental health and student support platform. It provides counselor booking, peer fundraising campaigns, an AI-powered mental health chatbot, contribution tracking, and a full admin management layer.

Developed by Nyson — nyson.me


## Table of Contents

- [Overview](#overview)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Environment Variables](#environment-variables)
- [Running the Project](#running-the-project)
- [Database and Migrations](#database-and-migrations)
- [API Reference](#api-reference)
- [Authentication and Sessions](#authentication-and-sessions)
- [Roles and Permissions](#roles-and-permissions)
- [Chatbot](#chatbot)
- [Postman Collection](#postman-collection)
- [Makefile Commands](#makefile-commands)
- [Adding a New Migration](#adding-a-new-migration)


## Overview

CampusCare serves three types of users:

- **Students** — Can register, chat with the AI support assistant, create fundraising campaigns, make and track contributions, and book sessions with counselors.
- **Counselors** — Can accept or decline booking requests from students.
- **Admins** — Have full visibility over users, campaigns, bookings, contributions, crisis flags, and audit logs.

The API is built in Go using the Gin framework, backed by a PostgreSQL database hosted on NeonDB, and runs entirely inside Docker during development.


## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.25 |
| HTTP Framework | Gin |
| Database | PostgreSQL (NeonDB) |
| Database Driver | pgx v5 |
| Migrations | golang-migrate |
| Authentication | HTTP-only cookie sessions |
| AI Chatbot | Groq API (llama-3.3-70b-versatile) |
| Email | Resend API via internal mailer |
| Dev Hot Reload | Air |
| Containerisation | Docker + Docker Compose |


## Project Structure

```
campuscare/
  cmd/
    server/
      main.go              Entry point. All routes are registered here.

  internal/
    audit/
      logger.go            Audit log writer.
    calendar/
      google.go            Google Calendar integration for bookings.
    chatbot/
      groq.go              Groq API client (CallGroq function).
      memory.go            Stores and retrieves conversation history per user.
      moderation.go        Crisis detection (DetectCrisis) and safe rewrite logic.
      prompts.go           System prompt that defines the chatbot's behaviour and scope.
      service.go           ChatbotHandler, Service, Ask logic, CrisisResponse.
    config/
      config.go            Loads environment variables into a Config struct.
    db/
      db.go                Opens and returns a pgxpool connection.
    handlers/
      auth.go              Register, Login, Logout handlers.
      booking.go           Create booking, UpdateStatus (counselor).
      campaign.go          Create, Update, Delete, Approve, ListPending, PublicList.
      contribution.go      Create, Simulate contributions. ExportContributions, AnonymizeUser (admin).
      admin.go             Dashboard, ListUsers, UpdateUserStatus, DeleteCampaign,
                           ListBookings, ListContributions, ListCrisisFlags, AuditLogs.
    mail/
      mailer.go            Resend mailer setup.
      templates.go         Email body templates.
    middleware/
      auth.go              AuthRequired — validates session cookie and attaches user_id to context.
      rbac.go              RequireRole — enforces role-based access control.
    services/
      password.go          HashPassword and CheckPassword using bcrypt.
      session.go           Session creation, lookup, and deletion.

  migrations/
    000001_init_schema     Core tables: users, profiles, campaigns, contributions, bookings, sessions, audit_logs.
    000002_seed_admin      Seeds the default admin account.
    000003_chatbot_tables  chatbot_messages and chatbot_usage tables.
    000004_crisis          crisis_flags table.
    000005_user_status     Adds the status column to the users table.

  campus_care.json         Postman collection with all endpoints documented.
  docker-compose.yml       Defines migrate and api services.
  Dockerfile               Multi-stage: migrate, dev (air), builder, production.
  Makefile                 Common commands.
  sqlc.yaml                SQLC config (for future query generation).
```


## Environment Variables

Create a `.env` file in the project root. Do not commit this file.

```
DATABASE_URL=postgres://user:password@host/dbname?sslmode=require
APP_PORT=8080
SESSION_TTL_HOURS=24
GROQ_API_KEY=your_groq_api_key
RESEND_API_KEY=your_resend_api_key
RESEND_FROM=CampusCare <onboarding@resend.dev>
```

Notes:
- `DATABASE_URL` must not be wrapped in quotes inside the `.env` file. Docker Compose reads it literally including any quote characters.
- `SESSION_TTL_HOURS` must be a valid integer. The app will fatal on startup if it is missing or non-numeric.
- `GROQ_API_KEY` is required for the chatbot to function. Obtain it from console.groq.com.
- `RESEND_API_KEY` is required for welcome emails and booking notifications.
- `RESEND_FROM` should use a sender verified in Resend for production. `onboarding@resend.dev` is suitable for initial testing.
- DNS is set to `8.8.8.8` and `8.8.4.4` in Docker Compose to ensure NeonDB (external cloud PostgreSQL) is reachable from inside containers.


## Running the Project

### Prerequisites

- Docker and Docker Compose installed
- A `.env` file configured as above

### Start development server

```bash
make dev
```

This command:
1. Builds both the `migrate` and `api` Docker images.
2. Runs the `migrate` container first, applying all pending migrations against the database.
3. Once migrations complete successfully, starts the `api` container with Air hot reload on port 8080.

The API will be available at `http://localhost:8080`.

### Stop all containers

```bash
make down
```

### Run without Docker (direct Go)

Ensure your `.env` file is present in the root directory, then:

```bash
make run
```

This runs `go run cmd/server/main.go` directly on the host machine. You will need Go 1.25 installed and a reachable database.

### Build production image

```bash
make docker-prod
```

This builds a minimal Alpine-based production image tagged `campuscare:prod` using the multi-stage Dockerfile. The binary is statically compiled with CGO disabled.


## Database and Migrations

The project uses golang-migrate for versioned SQL migrations. Migration files live in the `migrations/` directory and follow the naming convention:

```
000001_description.up.sql
000001_description.down.sql
```

Migrations run automatically on every `make dev`. The migrate container applies all pending `.up.sql` files in order against `DATABASE_URL`.

### Current migration sequence

| Version | Description |
|---|---|
| 000001 | Core schema — users, profiles, campaigns, contributions, bookings, sessions, audit_logs |
| 000002 | Seeds the default admin user |
| 000003 | Chatbot tables — chatbot_messages, chatbot_usage |
| 000004 | crisis_flags table |
| 000005 | Adds status column to the users table |

### Adding a new migration

Always increment the version number. Never edit an already-applied migration file.

```bash
# Create new migration files manually
touch migrations/000006_your_description.up.sql
touch migrations/000006_your_description.down.sql
```

Write your SQL in the `.up.sql` file and the rollback SQL in the `.down.sql` file. The next `make dev` will apply it automatically.

To run migrations manually:

```bash
make migrate-up
make migrate-down
```


## API Reference

All requests and responses use JSON unless stated otherwise. The base URL in development is `http://localhost:8080`.

### Authentication

| Method | Endpoint | Access | Description |
|---|---|---|---|
| POST | /register | Public | Register a new student or counselor |
| POST | /login | Public | Login and receive a session cookie |
| POST | /logout | Authenticated | Clear the session cookie |

Register request body:
```json
{
  "email": "alice@university.edu",
  "password": "SecurePass123!",
  "role": "student",
  "full_name": "Alice Nakamura",
  "consent": true
}
```

Role must be `student` or `counselor`. Admin accounts are pre-seeded via migration and cannot be self-registered.

On successful login, the server sets an HTTP-only cookie named `session_id`. All subsequent authenticated requests must send this cookie.

---

### Campaigns

| Method | Endpoint | Access | Description |
|---|---|---|---|
| GET | /campaigns | Public | List all approved campaigns |
| POST | /campaigns | Student | Create a campaign |
| PUT | /campaigns/:id | Student | Update own campaign |
| DELETE | /campaigns/:id | Student | Soft-delete own campaign |
| GET | /admin/campaigns | Admin | List pending campaigns for review |
| PUT | /admin/campaigns/:id | Admin | Approve or reject a campaign |

---

### Contributions

| Method | Endpoint | Access | Description |
|---|---|---|---|
| POST | /contributions | Public | Submit a contribution to a campaign |
| POST | /contributions/:id/simulate | Public | Simulate a payment status update (for testing) |

Payment method must be one of: `mtn_momo`, `airtel_money`, `visa`.

---

### Bookings

| Method | Endpoint | Access | Description |
|---|---|---|---|
| POST | /bookings | Student | Book a session with a counselor |
| PUT | /bookings/:id/status | Counselor | Accept or decline a booking |

Booking request body:
```json
{
  "counselor_id": "uuid",
  "type": "online",
  "start_time": "2026-03-10T09:00:00Z",
  "end_time": "2026-03-10T10:00:00Z",
  "location": "Room 4B"
}
```

`type` must be `online` or `physical`. The API checks for counselor availability overlap before confirming.

Status update body:
```json
{
  "status": "accepted"
}
```

`status` must be `accepted` or `declined`.

---

### Chatbot

| Method | Endpoint | Access | Description |
|---|---|---|---|
| POST | /chatbot | Student | Send a message to the AI support assistant |

Request body:
```json
{
  "message": "I have been feeling very stressed about my exams"
}
```

Response:
```json
{
  "reply": "...",
  "crisis_flagged": false
}
```

If `crisis_flagged` is `true`, the user's message was detected as a potential crisis situation. The reply will include emergency support guidance. The message is also stored in the `crisis_flags` table for admin review.

The chatbot maintains per-user conversation history (last 10 messages) for context.

---

### Admin

All admin endpoints require an active session with the `admin` role.

| Method | Endpoint | Description |
|---|---|---|
| GET | /admin/dashboard | Platform stats (user count, campaigns, bookings, total raised) |
| GET | /admin/users | List users, filter by role or status, paginated (20 per page) |
| PUT | /admin/users/:id/status | Set a user's status to `active` or `suspended` |
| DELETE | /admin/users/:id | Anonymize a user's PII (GDPR, irreversible) |
| DELETE | /admin/campaigns/:id | Force delete any campaign |
| GET | /admin/bookings | List all bookings with student and counselor names |
| GET | /admin/contributions | List all contributions |
| GET | /admin/contributions/export | Download all contributions as a CSV file |
| GET | /admin/crisis-flags | List all crisis-flagged chatbot messages |
| GET | /admin/audit | Paginated audit log (50 per page, use ?page=N) |


## Authentication and Sessions

Sessions are managed server-side using the `sessions` database table. When a user logs in:

1. A UUID session ID is generated and stored in the database with an expiry time.
2. The session ID is sent to the client as an HTTP-only cookie named `session_id`.
3. On each authenticated request, the `AuthRequired` middleware reads the cookie, looks up the session, and attaches the `user_id` to the Gin context.
4. On logout, the session row is deleted and the cookie is cleared.

Session duration is controlled by the `SESSION_TTL_HOURS` environment variable.


## Roles and Permissions

| Role | Capabilities |
|---|---|
| student | Register, login, create campaigns, make contributions, book counselor sessions, use chatbot |
| counselor | Login, accept or decline booking requests |
| admin | Full read access to all data, manage users, manage campaigns, view crisis flags and audit logs |

Role is set at registration and stored in the `users` table as a PostgreSQL enum. The `RequireRole` middleware reads the user's role from the database on each request and aborts with `403 Forbidden` if the role does not match.


## Chatbot

The chatbot is powered by the Groq API using the `llama-3.3-70b-versatile` model.

How it works:

1. The student sends a message to `POST /chatbot`.
2. The `DetectCrisis` function scans the message for keywords indicating self-harm or emergency. If triggered, a crisis response is returned immediately and the message is stored in `crisis_flags`.
3. If not a crisis, the last 10 messages for that user are loaded from the database to provide conversation context.
4. The system prompt (defined in `internal/chatbot/prompts.go`) is prepended to the message history and sent to Groq.
5. The response is checked again by the crisis detector. If the model's reply unexpectedly contains crisis content, it is rewritten by `SafeRewrite`.
6. Both the user message and assistant reply are stored in the database for future context.

The chatbot responds in the same language the student writes in. It covers: stress, anxiety, burnout, sleep, academic pressure, relationships, loneliness, homesickness, grief, self-esteem, financial stress, trauma awareness, anger, and substance use awareness. Off-topic questions receive a warm redirection.

To update the chatbot's behaviour, personality, scope, or formatting rules, edit `internal/chatbot/prompts.go`. No code changes are required — the prompt is injected at runtime.


## Postman Collection

The file `campus_care.json` in the project root is a Postman collection containing all endpoints with example request bodies and responses.

To use it:
1. Open Postman.
2. Click Import and select `campus_care.json`.
3. Set the `base_url` collection variable to `http://localhost:8080`.
4. Run Register and Login first — Postman will automatically store the `session_id` cookie for subsequent requests.


## Makefile Commands

| Command | Description |
|---|---|
| make dev | Build images, run migrations, start API with hot reload |
| make down | Stop and remove all containers |
| make run | Run the API directly on the host (no Docker) |
| make migrate-up | Apply pending migrations manually |
| make migrate-down | Roll back the last migration |
| make docker-build | Build Docker images without starting containers |
| make docker-up | Start containers in detached mode |
| make docker-logs | Tail the API container logs |
| make docker-restart | Restart only the API container |
| make docker-prod | Build a production-optimised image |
