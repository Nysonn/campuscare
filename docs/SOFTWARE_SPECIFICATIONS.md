# CampusCare — Software Specifications Document

**Document Version:** 1.0  
**Date:** April 29, 2026  
**Status:** Approved  
**Prepared by:** Nysonn (Lead Engineer)  
**Production URL:** https://campuscare.me

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Project Overview](#2-project-overview)
3. [Stakeholders and User Roles](#3-stakeholders-and-user-roles)
4. [System Architecture](#4-system-architecture)
5. [Technology Stack](#5-technology-stack)
6. [Functional Requirements](#6-functional-requirements)
   - 6.1 Authentication & Authorization
   - 6.2 Student Registration & Profiles
   - 6.3 Counselor Registration & Profiles
   - 6.4 Fundraising Campaigns
   - 6.5 Contributions & Donations
   - 6.6 General Donation Pool
   - 6.7 Counselor Booking System
   - 6.8 AI Mental Health Chatbot
   - 6.9 Peer Sponsorship System
   - 6.10 Behaviour Tracking
   - 6.11 Self-Evaluation (Mental Health Assessment)
   - 6.12 Admin Panel
   - 6.13 Wallet & Disbursement Management
   - 6.14 Email Notification System
   - 6.15 Audit Logging
7. [Non-Functional Requirements](#7-non-functional-requirements)
8. [Database Design](#8-database-design)
9. [API Specification](#9-api-specification)
10. [Frontend Application](#10-frontend-application)
11. [Third-Party Integrations](#11-third-party-integrations)
12. [Security Model](#12-security-model)
13. [Scheduled Jobs & Background Processing](#13-scheduled-jobs--background-processing)
14. [Deployment & Infrastructure](#14-deployment--infrastructure)
15. [Constraints & Assumptions](#15-constraints--assumptions)
16. [Glossary](#16-glossary)

---

## 1. Executive Summary

CampusCare is a full-stack web platform designed to address the mental health and financial welfare needs of university students, with an initial focus on students in Uganda. The platform provides a unified environment where students can access professional counseling, receive peer mentorship, raise funds for personal or academic emergencies, track personal wellness habits, and interact with an AI-powered mental health chatbot.

The system is built around three distinct user roles — students, counselors, and administrators — each with a tailored dashboard and a defined set of permissions. All critical actions produce email notifications, and a background scheduler handles time-sensitive workflows such as session reminders and daily habit coaching.

CampusCare is deployed as a Progressive Web Application (PWA), making it accessible on both desktop browsers and mobile devices without requiring a native app installation.

---

## 2. Project Overview

### 2.1 Purpose

University students face significant mental health challenges including academic stress, financial hardship, social isolation, and anxiety. CampusCare provides a centralized digital platform to:

- Connect students with verified professional counselors for online and in-person sessions.
- Enable peer-to-peer emotional support through a structured sponsorship model.
- Allow students to raise emergency funds through community-backed campaigns.
- Support mental wellness through habit tracking and regular self-assessment.
- Provide immediate, AI-assisted guidance through a safety-aware mental health chatbot.

### 2.2 Scope

The platform encompasses:

- A **RESTful backend API** written in Go, backed by a PostgreSQL relational database.
- A **single-page frontend application** built with React and TypeScript, deployed as a PWA on Firebase Hosting.
- A **background scheduler** for time-sensitive automated tasks (reminders, habit emails).
- Integration with **third-party services** for real-time messaging, video calling, email delivery, cloud media storage, and AI inference.
- A full **admin control panel** for platform governance.

### 2.3 Out of Scope

- Native mobile applications (iOS/Android).
- In-platform payment processing or payment gateway integration (contribution status is manually managed or confirmed externally via mobile money and card references).
- Multi-tenancy or multi-institution deployment.
- SMS notification channels.

---

## 3. Stakeholders and User Roles

### 3.1 Student

A registered university student who is the primary beneficiary of the platform. Students can:

- Create and manage fundraising campaigns.
- Make contributions to other students' campaigns or the general pool.
- Browse and book sessions with verified counselors.
- Interact with the AI mental health chatbot.
- Become a sponsor or seek a sponsor for peer support.
- Track personal behaviour goals with daily logging.
- Complete mental health self-evaluations.

### 3.2 Counselor

A verified mental health professional who offers services through the platform. Counselors can:

- Register and maintain a professional profile including credentials and license documentation.
- Receive and respond to booking requests (accept or decline).
- Conduct online sessions via Google Meet or physical sessions at a defined location.
- View their upcoming and past appointment schedule.

Counselors must pass an admin verification step before they appear in the public listing or receive booking requests.

### 3.3 Administrator

A platform operator with elevated access. Administrators can:

- Review and approve or reject student campaigns and campaign bank account details.
- Verify or reject counselor applications.
- Manage user accounts (suspend or reactivate).
- Oversee all bookings and contributions.
- Manage the general donation pool (view donations, disburse to campaigns, record withdrawals).
- Monitor sponsor relationships and the overall platform health dashboard.

### 3.4 Anonymous Visitor (Unauthenticated)

Any visitor to the public-facing site. Anonymous visitors can:

- Browse the landing page and platform overview.
- View the public campaign listing and individual campaign detail pages.
- Make contributions to campaigns or the general pool without creating an account.

---

## 4. System Architecture

### 4.1 Architectural Overview

CampusCare follows a **client-server architecture** with a clear separation between the frontend SPA and the backend REST API. The two communicate exclusively over HTTPS using JSON.

```
Browser / PWA Client
        │
        │  HTTPS / REST JSON
        ▼
  Go / Gin REST API  (port 8080)
        │
        ├── PostgreSQL Database
        ├── Background Scheduler (goroutine)
        ├── Email Service (SMTP, async)
        ├── Groq API (LLM inference)
        ├── Google Calendar API
        ├── Stream Chat REST API
        └── Cloudinary (via frontend SDK)
```

### 4.2 Backend Structure

```
campuscare/
├── cmd/server/main.go          — Entry point, router setup, dependency wiring
├── internal/
│   ├── audit/                  — Audit log utility
│   ├── calendar/               — Google Calendar & Meet integration
│   ├── chatbot/                — AI chatbot service, moderation, prompts
│   ├── config/                 — Environment configuration loader
│   ├── db/                     — Database connection pool (pgx)
│   ├── handlers/               — HTTP request handlers per domain
│   ├── mail/                   — Mailer and HTML email templates
│   ├── middleware/              — Auth, RBAC, last-active middleware
│   ├── reminder/               — Background scheduler
│   ├── services/               — Session and password services
│   └── stream/                 — Stream Chat API client
├── migrations/                 — Ordered SQL migration files (up/down)
├── Dockerfile                  — Multi-stage: migrate + dev targets
└── docker-compose.yml          — Orchestrates migration and API services
```

### 4.3 Frontend Structure

```
campuscare_frontend/
├── src/
│   ├── api/                    — Typed API client modules per domain
│   ├── components/
│   │   ├── campaign/           — Campaign cards and modals
│   │   ├── chat/               — Floating chat, video call, ringtone hook
│   │   ├── counselor/          — Counselor profile modal
│   │   ├── donate/             — General pool donation modal
│   │   ├── layout/             — Header, footer, dashboard layout, protected routes
│   │   ├── seo/                — react-helmet-async SEO component
│   │   ├── sponsor/            — Become sponsor modal
│   │   └── ui/                 — Design system components
│   ├── context/                — DarkModeContext
│   ├── pages/
│   │   ├── admin/              — Admin dashboard and management pages
│   │   ├── counselor/          — Counselor dashboard and profile
│   │   └── student/            — All student-facing pages
│   ├── store/                  — Redux store, auth slice
│   ├── types/index.ts          — Shared TypeScript type definitions
│   └── utils/                  — Sitemap generator utility
└── public/                     — Static assets, robots.txt, sitemap.xml
```

### 4.4 Data Flow: Session Authentication

1. Client submits credentials via `POST /login`.
2. Backend validates credentials, creates a session record in the `sessions` table, and sets an `HttpOnly` session cookie.
3. All subsequent authenticated requests carry the cookie automatically.
4. Middleware validates the session on every protected route and injects `user_id` and role into the request context.
5. On `POST /logout`, the session record is deleted and the cookie is cleared.

---

## 5. Technology Stack

### 5.1 Backend

| Component        | Technology                                  | Version  |
|------------------|---------------------------------------------|----------|
| Language         | Go                                          | 1.25     |
| Web Framework    | Gin                                         | 1.12     |
| Database Driver  | pgx (PostgreSQL)                            | 5.8      |
| Migrations       | golang-migrate (via Dockerfile)             | —        |
| Query Builder    | sqlc (code generation)                      | —        |
| Password Hashing | bcrypt (golang.org/x/crypto)                | —        |
| UUID             | google/uuid                                 | 1.6      |
| Config           | godotenv                                    | 1.5      |
| Scheduler        | Native goroutines + time.Ticker             | —        |
| Live Reload      | Air                                         | —        |

### 5.2 Frontend

| Component         | Technology                             | Version  |
|-------------------|----------------------------------------|----------|
| Language          | TypeScript                             | 5.9      |
| UI Framework      | React                                  | 19.2     |
| Build Tool        | Vite                                   | 7.3      |
| Styling           | Tailwind CSS                           | 4.2      |
| Routing           | React Router DOM                       | 7.13     |
| State Management  | Redux Toolkit + React Redux            | 2.11     |
| Server State      | TanStack React Query                   | 5.90     |
| Icons             | Lucide React                           | 0.577    |
| SEO               | react-helmet-async                     | 3.0      |
| PWA               | vite-plugin-pwa                        | 1.2      |
| Real-time Chat    | stream-chat                            | 9.38     |
| Video Calling     | @stream-io/video-react-sdk             | 1.34     |

### 5.3 Infrastructure & Services

| Concern                  | Solution                          |
|--------------------------|-----------------------------------|
| Database                 | PostgreSQL                        |
| Database timezone        | Africa/Kampala (UTC+3)            |
| Containerization         | Docker (multi-stage Dockerfile)   |
| Frontend Hosting         | Firebase Hosting                  |
| Media Storage            | Cloudinary                        |
| Email Delivery           | SMTP (custom Mailer)              |
| LLM / AI Inference       | Groq API                          |
| Real-time Messaging      | Stream Chat (stream-io)           |
| Video Calls              | Stream Video (stream-io)          |
| Calendar & Video Links   | Google Calendar API + Google Meet |

---

## 6. Functional Requirements

### 6.1 Authentication & Authorization

#### 6.1.1 Registration

- Students register at `POST /register` with email, password, role (`student`), and full profile data.
- Counselors register at `POST /register` with email, password, role (`counselor`), professional details, and licence URL.
- Upon successful registration, a welcome email is dispatched to the provided address.
- A `consent_given` boolean field is recorded at registration time to track informed consent.

#### 6.1.2 Login

- Users authenticate at `POST /login` with email and password.
- On success, a server-side session record is created with a configurable TTL (via `SESSION_TTL_HOURS` environment variable).
- An `HttpOnly` session cookie is set in the response.
- The `last_active_at` timestamp on the user record is updated on each authenticated request.

#### 6.1.3 Logout

- `POST /logout` deletes the current session record from the database and instructs the client to clear the session cookie.

#### 6.1.4 Password Reset

- `POST /forgot-password` accepts an email address, generates a secure time-limited token (1-hour expiry), stores it in the `password_reset_tokens` table, and sends a password reset email containing a link to `{FRONTEND_URL}/reset-password?token=<token>`.
- `POST /reset-password` accepts the token and the new password. The token is validated for existence, expiry, and prior use. On success, the `used_at` timestamp is set and the password hash is updated.

#### 6.1.5 Role-Based Access Control (RBAC)

- Every protected route is guarded by both the `AuthRequired` middleware (session validation) and a `RequireRole` middleware that enforces the permitted role.
- Three roles are enforced at the route level: `student`, `counselor`, and `admin`.
- The admin group is registered as a separate router group with its own middleware chain, making it independent of the shared authenticated group.

#### 6.1.6 User Status

- Users may be in one of two statuses: `active` or `suspended`.
- Suspended users are blocked from API access by the auth middleware.
- Only administrators can change a user's status.

---

### 6.2 Student Registration & Profiles

#### 6.2.1 Profile Fields

| Field          | Type    | Description                              |
|----------------|---------|------------------------------------------|
| first_name     | Text    | Student's first name                     |
| last_name      | Text    | Student's last name                      |
| display_name   | Text    | Public-facing name (may differ from real)|
| bio            | Text    | Short self-description                   |
| university     | Text    | Enrolled institution                     |
| course         | Text    | Programme of study                       |
| year           | Text    | Current year of study                    |
| location       | Text    | City/district                            |
| avatar_url     | Text    | Cloudinary-hosted profile photo URL      |
| is_anonymous   | Boolean | Hides real name on public content        |

#### 6.2.2 Profile Updates

- Authenticated students can update their profile fields via `PATCH /profile`.
- Avatar images are uploaded directly to Cloudinary from the frontend; only the resulting URL is sent to the backend.

#### 6.2.3 Anonymous Mode

- When `is_anonymous` is `true`, the student's real name is replaced with their `display_name` on all public-facing content (campaigns, contributions, sponsor listings).

---

### 6.3 Counselor Registration & Profiles

#### 6.3.1 Profile Fields

| Field                | Type    | Description                                |
|----------------------|---------|--------------------------------------------|
| full_name            | Text    | Legal full name                            |
| specialization       | Text    | Area of counseling expertise               |
| bio                  | Text    | Professional background description        |
| phone                | Text    | Contact telephone number                   |
| avatar_url           | Text    | Profile photo URL (Cloudinary)             |
| location             | Text    | City/district of practice                  |
| age                  | Integer | Age (nullable)                             |
| years_of_experience  | Text    | Descriptor of experience level             |
| licence_url          | Text    | URL to uploaded professional licence scan  |
| verification_status  | Enum    | `pending` / `approved` / `rejected`        |

#### 6.3.2 Verification Workflow

1. Counselor submits registration form including licence URL.
2. Profile is created with `verification_status = 'pending'`.
3. Administrator reviews the application in the Admin Panel.
4. Admin approves or rejects via `PUT /admin/counselors/:id/verify`.
5. On approval, an email notification is sent to the counselor.
6. Only counselors with `verification_status = 'approved'` appear in the student-facing counselor listing and are bookable.

---

### 6.4 Fundraising Campaigns

#### 6.4.1 Campaign Lifecycle

```
[Created by Student] → status: 'pending'
         ↓
  [Admin Review]
         ↓
 Approved: status: 'approved'  ──→  [Publicly visible, accepting contributions]
 Rejected: status: 'rejected'  ──→  [Not visible]
         ↓
  [Admin: Verify Bank Account]
         ↓
  account_status: 'verified'
```

#### 6.4.2 Campaign Fields

| Field                    | Description                                           |
|--------------------------|-------------------------------------------------------|
| title                    | Short campaign headline                               |
| description              | Full narrative of the need                            |
| target_amount            | Fundraising goal in UGX (Ugandan Shillings)           |
| current_amount           | Total received so far (auto-incremented on contribution) |
| category                 | Topic classification (e.g., medical, tuition, food)   |
| urgency_level            | `normal` / `urgent` / `critical`                      |
| beneficiary_type         | `self` / `other individual` / `organisation`          |
| beneficiary_name         | Name if not self                                      |
| beneficiary_org_name     | Organisation name if applicable                       |
| verification_contact_name| Name of a person who can verify the need              |
| verification_contact_info| Phone or email of the verification contact            |
| bank_name                | Recipient bank name                                   |
| account_number           | Bank account number for fund transfer                 |
| account_holder_name      | Name on the bank account                              |
| account_status           | `unverified` / `verified`                             |
| is_anonymous             | Whether to hide the creator's real identity           |

#### 6.4.3 Attachments

- Each campaign can have multiple file attachments (images, PDFs) uploaded via Cloudinary.
- Attachments include a `file_url` and an optional descriptive `label`.
- Attachments are deleted when the parent campaign is soft-deleted.

#### 6.4.4 Campaign Operations

- `POST /campaigns` — Create campaign (student only).
- `PUT /campaigns/:id` — Update own campaign (student only, own campaigns).
- `DELETE /campaigns/:id` — Soft-delete own campaign (student only).
- `GET /campaigns/mine` — Retrieve own campaigns with full detail (student only).
- `GET /campaigns` — Public listing of approved campaigns.
- `GET /campaigns/:id` — Public detail view of a single campaign.
- `PUT /admin/campaigns/:id` — Approve or reject a campaign (admin only).
- `DELETE /admin/campaigns/:id` — Hard-delete any campaign (admin only).
- `GET /admin/campaigns/accounts` — List campaigns with unverified bank accounts (admin only).
- `PUT /admin/campaigns/:id/account` — Mark campaign bank account as verified (admin only).

On campaign approval, a confirmation email is sent to the student who created the campaign.

---

### 6.5 Contributions & Donations

#### 6.5.1 Contribution Model

Contributions are not tied to a user account — any visitor may contribute to a campaign without registering.

| Field           | Description                                         |
|-----------------|-----------------------------------------------------|
| campaign_id     | The target campaign                                 |
| donor_name      | Optional name of the donor                          |
| donor_email     | Required for receipt delivery                       |
| donor_phone     | Optional contact number                             |
| message         | Optional personal message                           |
| is_anonymous    | Hides donor identity on public campaign page        |
| payment_method  | `mtn_momo` / `airtel_money` / `visa`                |
| amount          | Donation amount in UGX                              |
| status          | `pending` / `success` / `failed`                   |

#### 6.5.2 Contribution Flow

1. Visitor submits contribution form on the campaign detail page via `POST /contributions`.
2. Record is created with status `pending`.
3. On confirmed success status, `current_amount` on the campaign is incremented.
4. A donation receipt email is sent to the `donor_email` address.

#### 6.5.3 Admin Export

- `GET /admin/contributions/export` — Administrators can export all contribution records in CSV format for financial reconciliation.

---

### 6.6 General Donation Pool

The general pool allows donors to contribute to CampusCare without specifying a particular campaign. Funds in the pool are managed by administrators.

#### 6.6.1 Donation Fields

| Field           | Description                                    |
|-----------------|------------------------------------------------|
| donor_name      | Donor's display name                           |
| donor_email     | Required for receipt                           |
| donor_phone     | Optional                                       |
| amount          | Amount in UGX                                  |
| message         | Optional donor message                         |
| payment_method  | `mtn_momo` / `airtel_money` / `visa`           |
| is_anonymous    | Hides identity in admin view                   |
| status          | Defaults to `success`                          |

- `POST /donate/general` — Public endpoint, no authentication required.
- `GET /admin/general-pool` — Admin view of all pool donations.

---

### 6.7 Counselor Booking System

#### 6.7.1 Booking Lifecycle

```
[Student creates booking] → status: 'pending'
          ↓
[Counselor reviews request]
          ↓
  Accepted → status: 'accepted'
           ├── Online: Google Meet link created + stored
           └── Email sent to both parties with session details
  Declined → status: 'declined'
           └── Email sent to student
          ↓
[30-minute pre-session email reminder]
  (sent automatically by scheduler)
```

#### 6.7.2 Booking Fields

| Field           | Description                                            |
|-----------------|--------------------------------------------------------|
| student_id      | Booking student                                        |
| counselor_id    | Target counselor                                       |
| type            | `online` / `physical`                                  |
| start_time      | Session start (RFC3339 timestamp)                      |
| end_time        | Session end (RFC3339 timestamp)                        |
| location        | Physical address (for in-person sessions)              |
| google_event_id | Google Calendar event ID (populated on acceptance)     |
| status          | `pending` / `accepted` / `declined`                   |
| reminder_sent   | Boolean flag preventing duplicate reminder emails      |

#### 6.7.3 Overlap Prevention

- On booking creation, the backend checks for any existing booking for the same counselor where the requested time window overlaps with an existing `pending` or `accepted` booking.
- If a conflict is detected, the API returns HTTP 409 Conflict.
- The check uses a database-level index on `(counselor_id, start_time, end_time)` for performance.

#### 6.7.4 Google Meet Integration

- When a counselor accepts an online booking, the system calls the Google Calendar API to:
  1. Create a calendar event for the session duration.
  2. Add both the student and counselor as attendees.
  3. Enable Google Meet conference link generation.
- The returned Meet link is stored in `google_event_id` and included in both parties' acceptance emails.

#### 6.7.5 Booking Endpoints

- `POST /bookings` — Create booking request (student only).
- `PUT /bookings/:id/status` — Accept or decline booking (counselor only, own bookings).
- `GET /bookings/mine` — Student's booking history.
- `GET /bookings/counselor` — Counselor's booking queue (filterable by status).
- `GET /counselors` — List all approved counselors (student only).
- `GET /counselors/:id` — Get single approved counselor profile (student only).
- `GET /admin/bookings` — Full bookings list (admin only).

---

### 6.8 AI Mental Health Chatbot

#### 6.8.1 Overview

The chatbot provides an always-available, text-based mental health support interface for students. It is powered by the Groq LLM API and operates within a strict safety framework designed to handle sensitive mental health disclosures.

#### 6.8.2 Access

- Available exclusively to authenticated students.
- Usage is tracked per user in the `chatbot_usage` table with a count and `last_reset` timestamp to support rate limiting.

#### 6.8.3 Conversation Context

- The 10 most recent messages in the student's conversation history are retrieved from the `chatbot_messages` table and included as context in every API call to Groq, enabling coherent multi-turn conversations.

#### 6.8.4 Crisis Detection (Pre-Request)

Before any message is sent to the LLM, the content is scanned for a predefined list of crisis keywords covering suicidal ideation and self-harm:

- Trigger terms include: `suicide`, `suicidal`, `kill myself`, `want to die`, `end my life`, `self harm`, `self-harm`, `cutting myself`, `hurt myself`, `overdose`, and related phrases.
- If a crisis keyword is detected, the LLM is bypassed entirely. A pre-written safe response is returned immediately, directing the student to campus counselors, local emergency services, and trusted contacts.
- The crisis event is recorded in the `crisis_flags` table.

#### 6.8.5 Response Mode Selection

For non-crisis messages, the system analyzes the content to determine an appropriate response strategy before calling the LLM:

| Mode                    | Trigger Conditions                                      | Behavior                                                             |
|-------------------------|---------------------------------------------------------|----------------------------------------------------------------------|
| `clarify`               | Message is ≤8 words and matches a vague emotion signal  | LLM is instructed to ask one clarifying question before giving advice|
| `referral_first`        | Message contains persistent or severe distress signals  | LLM is instructed to prioritise recommending human support first     |
| `low_confidence_support`| Message contains low-confidence signals                 | LLM is instructed to avoid prescriptive advice; offer validation only|
| `normal`                | Default, none of the above                              | Standard supportive response mode                                    |

#### 6.8.6 Post-Response Moderation

- After the LLM generates its reply, the response text is scanned with the same crisis-detection algorithm.
- If crisis content is found in the AI's reply, it is replaced with the safe pre-written crisis response before being returned to the client.

#### 6.8.7 Message Storage

- All non-crisis messages (both user input and assistant reply) are persisted to the `chatbot_messages` table.
- The API response includes a `crisis_flagged` boolean so the frontend can apply distinct visual styling for crisis responses.

#### 6.8.8 Chat History Endpoint

- `GET /chatbot/history` — Returns the full ordered message history for the authenticated student.

---

### 6.9 Peer Sponsorship System

#### 6.9.1 Overview

The sponsorship feature enables students to opt into a peer support role as a sponsor, offering emotional guidance and encouragement to a fellow student (the sponsee). Communication between sponsor and sponsee happens via a private, real-time chat channel powered by Stream Chat. Video calling is available via Stream Video SDK.

#### 6.9.2 Sponsorship Lifecycle

```
[Student registers as sponsor] → sponsor_profiles record created
          ↓
[Another student browses sponsors and sends request]
          ↓
  Sponsor receives request → sponsor_requests record, email notification
          ↓
[Sponsor accepts/declines]
  Accepted → sponsorships record created
           ├── Stream Chat channel created (private, 2 members)
           └── Email sent to sponsee
  Declined → Email sent to sponsee
          ↓
[Active sponsorship: real-time chat + video calling]
          ↓
[Either party terminates the sponsorship]
  → Stream Chat channel soft-deleted
  → sponsorships.terminated_at set
  → Email notifications to both parties
```

#### 6.9.3 Sponsor Profile Fields

| Field        | Description                                              |
|--------------|----------------------------------------------------------|
| user_id      | Reference to the sponsoring student                      |
| what_i_offer | Text description of the support the sponsor provides     |
| is_active    | Whether the sponsor is currently accepting requests      |

#### 6.9.4 Constraints

- A sponsor can have **one active sponsorship** at a time. The `UNIQUE(sponsor_id, sponsee_id)` constraint and active-sponsorship checks enforce this.
- A student cannot send multiple requests to the same sponsor (enforced by `UNIQUE(requester_id, sponsor_id)` on `sponsor_requests`).
- A student cannot be both a sponsor and a sponsee simultaneously.

#### 6.9.5 Real-Time Chat Integration (Stream Chat)

- The backend generates a Stream Chat JWT for each user via `GET /stream/token`.
- When a sponsorship is created, the backend creates a private `messaging`-type channel on Stream Chat using the sponsorship UUID as the channel ID.
- When a sponsorship is terminated, the channel is soft-deleted via the Stream Chat API.
- Message notifications: when a user sends a message to an inactive partner, `POST /sponsorships/notify-message` can be called to trigger an email nudge. This is rate-limited by `last_message_notified_at` on the `sponsorships` table.

#### 6.9.6 Sponsor Endpoints

- `POST /sponsors/me` — Opt in as a sponsor.
- `DELETE /sponsors/me` — Opt out of sponsorship listing.
- `GET /sponsors/me/status` — Check own sponsor status.
- `GET /sponsors` — Browse available sponsors (student only).
- `POST /sponsors/:id/request` — Send a sponsorship request.
- `GET /sponsors/incoming-requests` — View requests received as a sponsor.
- `GET /sponsors/my-requests` — View requests sent as a potential sponsee.
- `PUT /sponsor-requests/:id` — Accept or decline a request.
- `DELETE /sponsor-requests/:id` — Cancel a pending outgoing request.
- `GET /sponsorships/mine` — Get active sponsorship details.
- `DELETE /sponsorships/mine` — Terminate active sponsorship.
- `POST /sponsorships/notify-message` — Trigger email notification to inactive partner.
- `GET /stream/token` — Get Stream Chat JWT for the authenticated user.

---

### 6.10 Behaviour Tracking

#### 6.10.1 Overview

Students can define personal behaviour goals with a start and end date. Each goal has a direction of either `build` (establishing a positive habit) or `quit` (stopping a negative one). Students log each day whether they followed through (`did_it: true/false`).

#### 6.10.2 Goal Model

| Field      | Description                                   |
|------------|-----------------------------------------------|
| title      | Short name for the habit goal                 |
| direction  | `build` (forming) or `quit` (breaking)        |
| start_date | Goal tracking start date (YYYY-MM-DD)         |
| end_date   | Goal tracking end date (YYYY-MM-DD)           |
| status     | `active` / `completed`                        |

#### 6.10.3 Constraints

- A student may have only **one active goal** at a time. Attempting to create a second active goal returns HTTP 409 Conflict.
- Log entries use an `INSERT … ON CONFLICT DO UPDATE` (upsert) pattern, allowing students to revise the same day's entry.
- The end date must be strictly after the start date.

#### 6.10.4 Goal Statistics

The `GET /behaviour/goals/:id/stats` endpoint computes the following in a single database query:

| Metric         | Description                                              |
|----------------|----------------------------------------------------------|
| total_days     | Span of the goal period in calendar days                 |
| days_logged    | Number of days where an entry exists                     |
| days_succeeded | Count of days where `did_it = true`                      |
| success_rate   | `days_succeeded / days_logged × 100` (percentage)        |

#### 6.10.5 Behaviour Endpoints

- `POST /behaviour/goals` — Create a new goal.
- `GET /behaviour/goals/current` — Get the current active goal with all log entries.
- `GET /behaviour/goals` — Get all goals (history view).
- `POST /behaviour/goals/:id/logs` — Log a day's result.
- `POST /behaviour/goals/:id/complete` — Manually mark goal as completed.
- `GET /behaviour/goals/:id/stats` — Get goal statistics.

---

### 6.11 Self-Evaluation (Mental Health Assessment)

#### 6.11.1 Overview

The self-evaluation feature provides a structured, AI-assisted mental health check-in. Students answer eight questions covering key wellness dimensions. The system scores the responses and generates a personalized feedback message and actionable recommendations.

#### 6.11.2 Question Generation

- Each call to `GET /evaluations/questions` invokes the Groq API to generate a fresh set of 8 contextually varied questions, preventing repetition across sessions.
- A randomized theme instruction and numeric seed are injected into the prompt to increase variation.
- If Groq is unavailable or returns malformed output, the endpoint gracefully degrades to a predefined static set of 8 validated fallback questions.

#### 6.11.3 Wellness Dimensions Assessed

1. Sleep quality
2. Overall mood
3. Academic stress
4. Social connection
5. Ability to focus
6. Physical activity
7. Anxiety levels
8. General wellbeing

#### 6.11.4 Scoring Model

- Each question offers four options scored 1 (worst) through 4 (best).
- Total possible score range: **8 – 32**.

| Score Range | Category         | Meaning                                        |
|-------------|------------------|------------------------------------------------|
| 8 – 13      | Needs Support    | Student is struggling significantly             |
| 14 – 19     | Moderate Concern | Notable challenges present                     |
| 20 – 25     | Doing Well       | Managing adequately                            |
| 26 – 32     | Thriving         | Strong overall wellbeing                       |

#### 6.11.5 AI-Generated Outputs

On submission, two additional Groq calls are made concurrently:

1. **Personalized Feedback** — A warm, empathetic 2–3 sentence message tailored to the student's specific score pattern, generated by providing per-question scores to the LLM.
2. **Recommendations** — Three concise, actionable steps tailored to the score category.

Both calls include static fallbacks for when Groq is unavailable.

#### 6.11.6 Evaluation Endpoints

- `GET /evaluations/questions` — Fetch 8 evaluation questions.
- `POST /evaluations` — Submit answers and receive score, category, message, and recommendations.
- `GET /evaluations/history` — Retrieve the most recent 20 evaluation results for the authenticated student.

---

### 6.12 Admin Panel

#### 6.12.1 Dashboard

`GET /admin/dashboard` returns a high-level platform summary:

| Metric       | Description                                  |
|--------------|----------------------------------------------|
| users        | Total registered users                       |
| campaigns    | Total active (non-deleted) campaigns         |
| bookings     | Total bookings across all statuses           |
| total_raised | Sum of `current_amount` across all campaigns |

#### 6.12.2 User Management

- `GET /admin/users` — Paginated user list, filterable by `role` and `status`. Returns 20 records per page.
- `PUT /admin/users/:id/status` — Set a user's status to `active` or `suspended`.

#### 6.12.3 Campaign Management

- `GET /admin/campaigns` — List campaigns in `pending` status awaiting review.
- `PUT /admin/campaigns/:id` — Approve or reject a pending campaign. Approval triggers a notification email to the student.
- `DELETE /admin/campaigns/:id` — Soft-delete any campaign.
- `GET /admin/campaigns/accounts` — List campaigns with `account_status = 'unverified'` pending bank account review.
- `PUT /admin/campaigns/:id/account` — Mark a campaign's bank account as `verified`.

#### 6.12.4 Counselor Management

- `GET /admin/counselors` — List counselors filterable by verification status (`pending`, `approved`, `rejected`, `all`).
- `PUT /admin/counselors/:id/verify` — Approve or reject a counselor registration. Approval triggers a notification email.

#### 6.12.5 Booking Management

- `GET /admin/bookings` — Read-only list of all bookings across all users, including student and counselor names.

#### 6.12.6 Contribution Management

- `GET /admin/contributions` — Full list of all campaign contributions.
- `GET /admin/contributions/export` — CSV export of all contribution records for financial reconciliation.

#### 6.12.7 Sponsor Management

- `GET /admin/sponsors` — List all active sponsors with current sponsee relationship status.

#### 6.12.8 General Pool Management

- `GET /admin/general-pool` — List all general pool donations.

---

### 6.13 Wallet & Disbursement Management

The wallet module allows administrators to manage the general donation pool as a financial ledger.

#### 6.13.1 Balance Calculation

`GET /admin/wallet/balance` calculates and returns:

- **Total in**: Sum of all successful general pool donations.
- **Total disbursed**: Sum of all disbursements to campaigns.
- **Total withdrawn**: Sum of all pool withdrawals.
- **Available balance**: `Total in − Total disbursed − Total withdrawn`.

#### 6.13.2 Disbursements

- `GET /admin/wallet/campaigns` — List approved campaigns eligible for disbursement.
- `POST /admin/wallet/disburse` — Record a disbursement from the pool to a specific campaign. This creates a record in `pool_disbursements` and increments the campaign's `current_amount`.

#### 6.13.3 Withdrawals

- `POST /admin/wallet/withdraw` — Record a cash withdrawal from the pool to a bank account or mobile money number. Fields: `destination_type`, `destination_name`, `account_number`, `amount`, `note`.
- `GET /admin/wallet/disbursements` — List all disbursement records.
- `GET /admin/wallet/withdrawals` — List all withdrawal records.

---

### 6.14 Email Notification System

All emails are dispatched asynchronously using a non-blocking `SendAsync` method backed by a goroutine, ensuring email failures never impact the HTTP response cycle.

#### 6.14.1 Email Templates

All templates are HTML-formatted and include the CampusCare branding.

| Template                          | Trigger                                                      |
|-----------------------------------|--------------------------------------------------------------|
| Welcome                           | Student or counselor registration                            |
| Password Reset                    | Forgot password request                                      |
| Booking Accepted (Student)        | Counselor accepts booking                                    |
| Booking Accepted (Counselor)      | Counselor accepts their own booking                          |
| Booking Declined (Student)        | Counselor declines booking                                   |
| Session Reminder (Student)        | 30 minutes before an accepted session                        |
| Session Reminder (Counselor)      | 30 minutes before an accepted session                        |
| Campaign Approved                 | Admin approves a student campaign                            |
| Counselor Approved                | Admin approves a counselor registration                      |
| Donation Receipt                  | Successful contribution to any campaign                      |
| Sponsor Request Received          | Sponsor receives a new request                               |
| Sponsor Request Accepted          | Sponsee's request is accepted                                |
| Sponsor Request Declined          | Sponsee's request is declined                                |
| Sponsorship Terminated (Sponsee)  | Sponsor terminates the relationship                          |
| Sponsorship Terminated (Sponsor)  | Sponsee terminates the relationship                          |
| New Sponsor Registration          | Student opts in as a sponsor                                 |
| Sponsor Chat Notification         | Partner has sent a message while recipient is inactive       |
| Habit Goal Created                | Student creates a new behaviour goal                         |
| Daily Motivation                  | Sent once daily to students with active goals                |
| Missed Habit Notification         | Sent when a student has not logged for 2 consecutive days    |

---

### 6.15 Audit Logging

All significant mutations within the system are recorded in the `audit_logs` table.

| Field       | Description                                              |
|-------------|----------------------------------------------------------|
| actor_id    | UUID of the authenticated user who performed the action  |
| action      | String identifier (e.g., `CREATE_BOOKING`, `UPDATE_BOOKING`) |
| entity_type | Type of the affected resource (e.g., `booking`)          |
| entity_id   | UUID of the specific affected record                     |
| metadata    | JSONB payload capturing the request body or relevant context |
| created_at  | Timestamp of the action                                  |

Audit records are currently used for internal governance and forensic auditing. They are not exposed through any public-facing API endpoint.

---

## 7. Non-Functional Requirements

### 7.1 Performance

- API response time for standard data retrieval endpoints must be under 300ms at the 95th percentile under normal load.
- Database queries on high-traffic tables (campaigns, bookings, contributions) are supported by defined composite indexes.
- Email delivery is fully asynchronous and must not block HTTP responses.
- Groq API calls may take up to 10 seconds; the frontend must display loading indicators for all AI-powered operations (chatbot, evaluation).

### 7.2 Availability

- The backend API targets 99.5% uptime, excluding planned maintenance windows.
- The Docker service is configured with `restart: unless-stopped` to automatically recover from unexpected crashes.
- The frontend, hosted on Firebase, inherits Firebase's CDN-backed availability guarantees.

### 7.3 Scalability

- The PostgreSQL connection pool (via pgx) manages concurrency at the database layer.
- The API is stateless with respect to user data (state is held in the database), enabling horizontal scaling by running multiple container instances behind a load balancer if needed.

### 7.4 Security

Detailed in Section 12. Key requirements:

- Passwords must never be stored in plaintext; bcrypt hashing is mandatory.
- All authenticated routes require a valid server-issued session cookie.
- CORS is locked to a whitelist of approved origins.
- Request bodies are capped at 50 MB to prevent resource exhaustion.
- Crisis content in chatbot responses must be intercepted and rewritten before delivery.

### 7.5 Accessibility

- The frontend is keyboard-navigable.
- All images used for functional purposes include descriptive alt text.
- Color contrast meets WCAG 2.1 AA standards for body text and interactive elements.
- Dark mode is supported system-wide via a React context provider.

### 7.6 Progressive Web App (PWA)

- The frontend is PWA-compliant with a web manifest and service worker generated by `vite-plugin-pwa`.
- The application is installable on Android and desktop devices.
- A PWA update prompt is shown to returning users when a new version of the service worker is detected.

### 7.7 Search Engine Optimization

- All public pages use `react-helmet-async` for dynamic `<title>` and `<meta>` tags.
- A `sitemap.xml` and `robots.txt` are published under the `/public` path.

### 7.8 Timezone Handling

- All timestamps are stored with timezone awareness (`TIMESTAMPTZ`).
- The PostgreSQL server timezone is set to `Africa/Kampala` (UTC+3) to align with the primary user geography.

---

## 8. Database Design

### 8.1 Entity Summary

| Table                   | Purpose                                                       |
|-------------------------|---------------------------------------------------------------|
| users                   | Core identity records for all user roles                      |
| student_profiles        | Extended attributes for students                              |
| counselor_profiles      | Extended attributes and credentials for counselors            |
| campaigns               | Student fundraising campaigns                                 |
| campaign_attachments    | Supporting file attachments per campaign                      |
| contributions           | Donations made to specific campaigns                          |
| general_pool_donations  | Unattributed donations to the platform pool                   |
| pool_disbursements      | Admin-recorded transfers from pool to campaigns               |
| pool_withdrawals        | Admin-recorded cash withdrawals from the pool                 |
| bookings                | Counseling session requests and confirmed appointments        |
| sessions                | Server-side authentication sessions                           |
| audit_logs              | Immutable record of all significant system actions            |
| chatbot_messages        | Conversation history per user for the AI chatbot              |
| chatbot_usage           | Per-user message count and reset timestamp                    |
| crisis_flags            | Records of crisis-keyword detections in chatbot input         |
| sponsor_profiles        | Students registered as peer sponsors                          |
| sponsor_requests        | Pending sponsorship connection requests                       |
| sponsorships            | Active sponsor-sponsee pairings with Stream channel reference |
| behaviour_goals         | Student habit goals with direction, dates, and status         |
| behaviour_logs          | Daily did-it log entries per goal                             |
| self_evaluations        | Mental health self-assessment records with score and answers  |
| password_reset_tokens   | Secure time-limited tokens for password reset flow            |

### 8.2 Soft Deletion

The following tables implement soft deletion via a `deleted_at TIMESTAMPTZ` column:
`users`, `student_profiles`, `counselor_profiles`, `campaigns`, `bookings`.

All queries against soft-deleted tables include a `WHERE deleted_at IS NULL` filter unless explicitly querying deleted records.

### 8.3 Key Indexes

| Table        | Index Columns                        | Purpose                                    |
|--------------|--------------------------------------|--------------------------------------------|
| campaigns    | (created_at DESC)                    | Paginated public listing sorted by recency |
| contributions| (campaign_id)                        | Fast contribution aggregation per campaign |
| bookings     | (counselor_id, start_time, end_time) | Efficient booking overlap detection        |
| sessions     | (user_id)                            | Fast session lookup on auth middleware      |
| general_pool | (created_at DESC)                    | Admin view sorted by recency               |
| password_reset_tokens | (token)                   | Fast token lookup on reset requests        |

### 8.4 Enumerated Types

| Type                   | Values                               |
|------------------------|--------------------------------------|
| user_role              | student, counselor, admin            |
| campaign_status        | pending, approved, rejected          |
| booking_status         | pending, accepted, declined          |
| session_type           | online, physical                     |
| payment_method         | mtn_momo, airtel_money, visa         |
| contribution_status    | pending, success, failed             |
| sponsor_request_status | pending, accepted, declined          |

---

## 9. API Specification

### 9.1 Base URL

```
Production:  https://campuscare.me/api  (or direct backend URL)
Development: http://localhost:8080
```

### 9.2 Authentication

All protected endpoints require a valid session cookie set by `POST /login`. There is no Bearer token or API key authentication for client-facing routes.

### 9.3 Response Format

All responses use `application/json`. Error responses follow the structure:

```json
{
  "error": "Human-readable error message"
}
```

Successful responses return either a resource object or a confirmation object:

```json
{
  "message": "Action completed",
  "id": "<uuid>"
}
```

### 9.4 Endpoint Catalogue

#### Public Endpoints (No Authentication)

| Method | Path                    | Description                               |
|--------|-------------------------|-------------------------------------------|
| GET    | /health                 | Health check (database ping)              |
| POST   | /register               | Register student or counselor             |
| POST   | /login                  | Authenticate and create session           |
| POST   | /logout                 | Destroy current session                   |
| POST   | /forgot-password        | Request password reset email              |
| POST   | /reset-password         | Complete password reset                   |
| GET    | /campaigns              | Public list of approved campaigns         |
| GET    | /campaigns/:id          | Public detail of a single campaign        |
| POST   | /contributions          | Make a contribution to a campaign         |
| POST   | /donate/general         | Donate to the general pool                |

#### Authenticated — Student

| Method | Path                               | Description                              |
|--------|------------------------------------|------------------------------------------|
| GET    | /profile                           | Get own profile                          |
| PATCH  | /profile                           | Update own profile                       |
| POST   | /campaigns                         | Create a campaign                        |
| PUT    | /campaigns/:id                     | Update own campaign                      |
| DELETE | /campaigns/:id                     | Delete own campaign                      |
| GET    | /campaigns/mine                    | List own campaigns                       |
| POST   | /chatbot                           | Send message to AI chatbot               |
| GET    | /chatbot/history                   | Retrieve chat history                    |
| POST   | /bookings                          | Create a booking request                 |
| GET    | /bookings/mine                     | List own bookings                        |
| GET    | /counselors                        | Browse approved counselors               |
| GET    | /counselors/:id                    | Get a counselor profile                  |
| POST   | /sponsors/me                       | Register as a sponsor                    |
| DELETE | /sponsors/me                       | Opt out of sponsorship                   |
| GET    | /sponsors/me/status                | Check own sponsor status                 |
| GET    | /sponsors                          | Browse sponsors                          |
| POST   | /sponsors/:id/request              | Send sponsorship request                 |
| GET    | /sponsors/incoming-requests        | View incoming sponsor requests           |
| GET    | /sponsors/my-requests              | View outgoing sponsorship requests       |
| PUT    | /sponsor-requests/:id              | Accept or decline a request              |
| DELETE | /sponsor-requests/:id              | Cancel own outgoing request              |
| GET    | /sponsorships/mine                 | Get active sponsorship                   |
| DELETE | /sponsorships/mine                 | Terminate active sponsorship             |
| POST   | /sponsorships/notify-message       | Notify partner of new message            |
| GET    | /stream/token                      | Get Stream Chat JWT                      |
| POST   | /behaviour/goals                   | Create a behaviour goal                  |
| GET    | /behaviour/goals/current           | Get active goal with logs                |
| GET    | /behaviour/goals                   | Get all goals                            |
| POST   | /behaviour/goals/:id/logs          | Log a day                                |
| POST   | /behaviour/goals/:id/complete      | Complete a goal                          |
| GET    | /behaviour/goals/:id/stats         | Get goal statistics                      |
| GET    | /evaluations/questions             | Get evaluation questions                 |
| POST   | /evaluations                       | Submit evaluation answers                |
| GET    | /evaluations/history               | Get evaluation history                   |

#### Authenticated — Counselor

| Method | Path                              | Description                              |
|--------|-----------------------------------|------------------------------------------|
| GET    | /profile                          | Get own profile                          |
| PATCH  | /profile                          | Update own profile                       |
| PUT    | /bookings/:id/status              | Accept or decline a booking              |
| GET    | /bookings/counselor               | List own booking queue                   |

#### Authenticated — Admin

| Method | Path                                 | Description                            |
|--------|--------------------------------------|----------------------------------------|
| GET    | /admin/dashboard                     | Platform statistics                    |
| GET    | /admin/users                         | Paginated user list                    |
| PUT    | /admin/users/:id/status              | Update user status                     |
| GET    | /admin/campaigns                     | List pending campaigns                 |
| PUT    | /admin/campaigns/:id                 | Approve or reject campaign             |
| DELETE | /admin/campaigns/:id                 | Delete campaign                        |
| GET    | /admin/campaigns/accounts            | List campaigns with unverified accounts|
| PUT    | /admin/campaigns/:id/account         | Verify campaign bank account           |
| GET    | /admin/bookings                      | All bookings                           |
| GET    | /admin/contributions                 | All contributions                      |
| GET    | /admin/contributions/export          | Export contributions as CSV            |
| GET    | /admin/sponsors                      | List all sponsors                      |
| GET    | /admin/general-pool                  | List all pool donations                |
| GET    | /admin/counselors                    | List counselors by verification status |
| PUT    | /admin/counselors/:id/verify         | Approve or reject counselor            |
| GET    | /admin/wallet/balance                | Get pool balance                       |
| GET    | /admin/wallet/campaigns              | List eligible campaigns for disbursement|
| POST   | /admin/wallet/disburse               | Disburse from pool to campaign         |
| POST   | /admin/wallet/withdraw               | Record pool withdrawal                 |
| GET    | /admin/wallet/disbursements          | List all disbursements                 |
| GET    | /admin/wallet/withdrawals            | List all withdrawals                   |

---

## 10. Frontend Application

### 10.1 Routing Structure

The frontend uses React Router DOM v7 with nested layouts grouped by user role.

#### Public Routes

| Path                   | Page                    |
|------------------------|-------------------------|
| `/`                    | Landing Page            |
| `/campaigns`           | All Campaigns           |
| `/campaigns/:id`       | Campaign Detail         |
| `/login`               | Login                   |
| `/register/student`    | Student Registration    |
| `/register/counselor`  | Counselor Registration  |
| `/forgot-password`     | Forgot Password         |
| `/reset-password`      | Reset Password          |

#### Student Routes (Protected, role: student)

| Path                           | Page                         |
|--------------------------------|------------------------------|
| `/student/dashboard`           | Student Dashboard            |
| `/student/campaigns`           | My Campaigns                 |
| `/student/campaigns/new`       | Create Campaign              |
| `/student/campaigns/:id/edit`  | Edit Campaign                |
| `/student/bookings`            | My Bookings                  |
| `/student/bookings/new`        | Book a Counselor             |
| `/student/sponsors`            | Sponsors (Browse & Manage)   |
| `/student/sponsors/become`     | Become a Sponsor             |
| `/student/behaviour`           | Behaviour Tracking           |
| `/student/evaluation`          | Self-Evaluation              |
| `/student/profile`             | Student Profile              |

#### Counselor Routes (Protected, role: counselor)

| Path                    | Page                  |
|-------------------------|-----------------------|
| `/counselor/dashboard`  | Counselor Dashboard   |
| `/counselor/profile`    | Counselor Profile     |

#### Admin Routes (Protected, role: admin)

| Path                       | Page                          |
|----------------------------|-------------------------------|
| `/admin/dashboard`         | Admin Dashboard               |
| `/admin/users`             | User Management               |
| `/admin/counselors`        | Counselor Verification        |
| `/admin/campaigns`         | Campaign Moderation           |
| `/admin/bookings`          | Booking Overview              |
| `/admin/contributions`     | Contributions Overview        |
| `/admin/sponsors`          | Sponsor Management            |
| `/admin/wallet`            | Wallet Management             |

### 10.2 State Management

- **Redux Toolkit (`authSlice`)** — Manages global authentication state (current user profile, initialization flag). The `SessionHydrator` component calls `GET /profile` on app mount to rehydrate state from the server-side session cookie.
- **TanStack React Query** — Used for all server data fetching, caching, and background refresh. Default stale time is 60 seconds with a single retry on failure.
- **React Router DOM** — Client-side routing with nested layouts and `<Outlet>` composition.
- **React Context** — `DarkModeContext` for system-wide theme state.

### 10.3 UI Design System

CampusCare uses a custom component library built on Tailwind CSS. Core components include:

| Component      | Description                                              |
|----------------|----------------------------------------------------------|
| Button         | Variants: primary, secondary, danger, ghost              |
| Input          | Styled text input with label and error state             |
| Textarea       | Multi-line input                                         |
| Modal          | Accessible overlay dialog                                |
| Toast          | Ephemeral notification messages                          |
| Spinner        | Loading indicator                                        |
| Avatar         | Circular profile image with fallback initials            |
| Badge          | Status labels with color coding                          |
| ProgressBar    | Fundraising goal progress visualization                  |
| DarkModeToggle | Theme switcher component                                 |

### 10.4 Real-Time Features

#### Sponsor Chat

- When an authenticated student has an active sponsorship, a `FloatingChat` component renders in the student layout.
- The component initializes the Stream Chat client using the JWT from `GET /stream/token`.
- Messages are sent and received in real time via the Stream Chat WebSocket connection.
- Incoming call notifications use a dedicated `IncomingCallCard` component with a ringtone hook.
- Video calls are initiated via the Stream Video React SDK, rendered in a `VideoCallModal`.

### 10.5 PWA Behavior

- Workbox-generated service worker handles asset pre-caching for offline capability.
- A `PWAUpdatePrompt` component monitors for service worker updates and prompts users to refresh when a new version is available.
- The web manifest defines app name, icons, theme color, and display mode for home screen installation.

---

## 11. Third-Party Integrations

### 11.1 Groq API (LLM Inference)

- **Used by:** AI Chatbot, Self-Evaluation question generation, personalized feedback generation, recommendation generation.
- **Auth:** API key via environment variable.
- **Calls are made server-side** from the Go backend; API keys are never exposed to the browser.
- All Groq calls include a configurable temperature parameter. The chatbot uses default temperature; evaluation question generation uses `temperature: 1.0` for maximum variety; recommendation generation uses `temperature: 0.7`.

### 11.2 Stream Chat (Real-Time Messaging)

- **Used by:** Sponsor-to-sponsee private messaging.
- **Backend:** Custom lightweight Stream Chat REST API client (`internal/stream/client.go`) implementing JWT generation, user upsert, channel creation, and channel deletion. No third-party Stream server SDK is used.
- **Frontend:** Official `stream-chat` npm package handles WebSocket connection and message rendering.
- **Auth:** Backend-generated HS256 JWTs signed with the Stream API secret are issued to clients.

### 11.3 Stream Video (Video Calling)

- **Used by:** Live video sessions within the sponsorship feature.
- **Frontend:** `@stream-io/video-react-sdk` provides pre-built call UI components including the incoming call card and video call modal.

### 11.4 Google Calendar API & Google Meet

- **Used by:** Automatic Google Meet link generation for online counseling sessions.
- **Auth:** Service account credentials via `google.golang.org/api`.
- **Flow:** When a counselor accepts an online booking, a calendar event is created via the API and both participants are added as attendees. The Google Meet link auto-generated by the conference data is extracted and stored.

### 11.5 Cloudinary

- **Used by:** Uploading and hosting profile avatars, campaign attachments, counselor licence documents.
- **Integration:** Upload happens directly from the frontend browser to Cloudinary. Only the resulting CDN URL is passed to the backend.
- **API module:** `src/api/cloudinary.ts` wraps the Cloudinary upload API.

### 11.6 Firebase Hosting

- **Used by:** Static hosting of the compiled frontend SPA and PWA assets.
- **Configuration:** `firebase.json` defines hosting rules. `dist/` is the build output directory.

---

## 12. Security Model

### 12.1 Authentication Security

- Passwords are hashed using **bcrypt** before storage. Plaintext passwords are never logged or persisted.
- Sessions are stored server-side in the `sessions` table. The session cookie is `HttpOnly`, preventing JavaScript access.
- Session TTL is enforced on every request. Expired sessions are treated as unauthenticated.
- Password reset tokens are cryptographically random, stored hashed, single-use (marked with `used_at`), and expire after 1 hour.

### 12.2 Authorization Security

- Every protected API route applies both `AuthRequired` and `RequireRole` middleware. There is no path through the API that bypasses both checks.
- Role is stored in the `users` table and is read fresh from the database on every authenticated request. It is not encoded in a client-supplied token and cannot be spoofed.
- The `UpdateLastActive` middleware tracks session activity for audit and inactive-user detection purposes.

### 12.3 Input Validation

- All request bodies are bound and validated using Gin's binding infrastructure (`BindJSON`).
- UUID parameters are explicitly parsed before use; invalid UUIDs return HTTP 400 immediately.
- Enum values (booking status, user status, goal direction) are validated against an allowed list before database operations.
- Request body size is capped at **50 MB** at the router level to prevent large payload attacks.

### 12.4 CORS Policy

CORS is restricted to the following allowed origins:

- `https://campuscare.me` (production)
- `http://localhost:5173` (local development)
- Additional local network development addresses

Only the HTTP methods `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS` are permitted. Credentials (cookies) are allowed in cross-origin requests from whitelisted origins.

### 12.5 AI Content Safety

- The chatbot applies two layers of content moderation: a deterministic keyword-based pre-check before the LLM call, and a post-generation check on the AI's response.
- Crisis responses bypass the LLM entirely to ensure a consistent, safe, and human-authored message is always delivered.
- AI-generated evaluation feedback is labeled as supportive guidance only. The system prompt explicitly prohibits medical advice.

### 12.6 Data Privacy

- The `is_anonymous` flag on student profiles and campaigns allows students to participate in public-facing features without exposing their real identity.
- Contributions support anonymous posting, honoring donor privacy preferences.
- Chatbot conversation history is scoped per user; no cross-user message access is possible.
- The `consent_given` field on the users table records that each registered user has acknowledged the platform's terms and privacy policy.

### 12.7 Soft Deletion

- User and campaign records are never hard-deleted from the system except via explicit admin action.
- Soft-deleted records are excluded from all standard queries via `deleted_at IS NULL` filters, while remaining available for audit purposes.

---

## 13. Scheduled Jobs & Background Processing

The background scheduler is initialized at server startup as a dedicated goroutine. It runs a tick loop at a **1-minute interval** and executes the following jobs:

### 13.1 Session Reminder Emails

**Frequency:** Every tick (1 minute).

**Logic:** Queries all `accepted` bookings where `start_time` falls within a 29–31 minute window from the current time AND `reminder_sent = FALSE`. For each qualifying booking:

1. Sends a 30-minute reminder email to the student.
2. Sends a 30-minute reminder email to the counselor.
3. Sets `reminder_sent = TRUE` on the booking to prevent duplicate sends.

### 13.2 Daily Habit Motivation Emails

**Frequency:** Once per calendar day when the local hour is ≥ 08:00.

**Logic:** Queries all `active` behaviour goals where `last_motivation_sent` is either null or prior to today. For each:

1. Computes the number of days where `did_it = TRUE` as a success count.
2. Sends a personalized daily motivation email to the student including their goal title, direction, and accumulated success days.
3. Updates `last_motivation_sent` to today's date.

### 13.3 Missed Habit Notification Emails

**Frequency:** Once per calendar day when the local hour is ≥ 08:00 (run together with motivation emails).

**Logic:** Queries all `active` goals that started at least 2 days ago, where there are no log entries for both yesterday and the day before, and where `missed_notified_date` is not today. For each:

1. Sends a re-engagement email encouraging the student to return to their habit.
2. Updates `missed_notified_date` to today's date.

### 13.4 Idle Session Cleanup (Implicit)

Expired sessions are filtered out at the middleware level on every request. Periodic cleanup of the `sessions` table may be added as a future background job.

---

## 14. Deployment & Infrastructure

### 14.1 Backend Deployment

The backend is packaged via a multi-stage Dockerfile with two targets:

#### `migrate` stage

- Runs `golang-migrate` to apply all pending SQL migration files in `migrations/` against the configured `DATABASE_URL`.
- Designed to run as a one-shot initialization container.

#### `dev` stage

- Runs the Go API server using `air` for live reload in development.
- In production, the binary is compiled and run directly.

The `docker-compose.yml` orchestrates the two services:

- `migrate` runs first and must complete successfully before `api` starts.
- `api` maps host port 8080 to container port 8080.
- Both services share the `.env` file for configuration.

### 14.2 Frontend Deployment

- The frontend is compiled to static assets using `vite build`.
- Assets are deployed to **Firebase Hosting** via the Firebase CLI.
- Hosting configuration is defined in `firebase.json` with the `.firebaserc` file pointing to the Firebase project.

### 14.3 Environment Configuration

All runtime configuration is loaded from environment variables (`.env` file in development).

| Variable             | Description                                        |
|----------------------|----------------------------------------------------|
| APP_PORT             | HTTP port the API listens on (default 8080)        |
| DATABASE_URL         | PostgreSQL connection string                       |
| SESSION_TTL_HOURS    | Number of hours before a session expires           |
| STREAM_API_KEY       | Stream Chat API key                                |
| STREAM_API_SECRET    | Stream Chat API secret                             |
| FRONTEND_URL         | Base URL of the frontend (for email links)         |
| SMTP credentials     | Email delivery configuration                       |
| GROQ_API_KEY         | Groq LLM API key                                   |
| Google credentials   | Google Calendar service account credentials        |

### 14.4 DNS

- Production frontend: `https://campuscare.me`
- Backend API is accessible internally via Docker network or via a configured subdomain.

---

## 15. Constraints & Assumptions

### 15.1 Technical Constraints

- Payment processing is **not integrated**. Contributions and pool donations record the declared payment method and amount; actual fund settlement is handled offline via MTN Mobile Money, Airtel Money, or Visa, with status updated manually or via external webhook by an administrator.
- The Groq API is a **synchronous external dependency**. If Groq is unavailable, evaluation questions fall back to static content, and chatbot responses will return an error. The system is designed to degrade gracefully for evaluation; chatbot failures surface as HTTP 500 errors.
- Google Calendar integration requires **pre-configured service account credentials** with appropriate Calendar API permissions. If not configured, online booking acceptance will succeed but Meet links will not be generated.
- The Stream Chat integration uses a **custom REST client** (not the official Stream server SDK). Any Stream Chat API contract changes must be handled by updating `internal/stream/client.go`.

### 15.2 Operational Assumptions

- Only one admin account exists at platform launch. The initial admin is seeded via migration `000002_seed_admin.up.sql`.
- Currency throughout the system is **Ugandan Shillings (UGX)**. Amounts are stored as `BIGINT` representing whole shilling values.
- Campaign `target_amount` and `current_amount` are not validated against each other in the backend; a campaign may receive more contributions than its target.
- The platform assumes genuine, good-faith use by all parties. There is no automated fraud detection beyond the admin review workflow.
- Email deliverability depends on the configured SMTP provider. Transient SMTP failures will result in undelivered emails with no retry mechanism in the current implementation.

---

## 16. Glossary

| Term              | Definition                                                                 |
|-------------------|----------------------------------------------------------------------------|
| Sponsee           | A student who has been accepted by a sponsor for peer support              |
| Sponsor           | A student who has volunteered to provide peer emotional support             |
| Sponsorship       | An active 1-to-1 peer support connection between a sponsor and a sponsee   |
| Campaign          | A student-created fundraising request for emergency financial support      |
| Contribution      | A donation made to a specific campaign by any visitor                      |
| General Pool      | A collective fund managed by admins, not tied to any single campaign       |
| Disbursement      | An admin-recorded transfer from the general pool to a specific campaign    |
| Withdrawal        | An admin-recorded cash draw from the general pool to an external account   |
| Behaviour Goal    | A student-defined habit goal with a defined period and daily logging       |
| Crisis Flag       | A database record created when crisis keywords are detected in chatbot input|
| Session           | A server-side authentication token stored in a database-backed cookie      |
| Verification      | The admin review process that approves or rejects counselor registrations  |
| RBAC              | Role-Based Access Control — permission model enforced at the API route level|
| UGX               | Ugandan Shilling, the primary currency unit used throughout the platform    |
| Stream Chat       | Third-party real-time messaging service (stream.io) used for sponsor chat  |
| Groq              | AI inference API provider powering the chatbot and self-evaluation features|
| PWA               | Progressive Web App — a web application installable on device home screens |
| Soft Delete       | Marking records as deleted via a `deleted_at` timestamp rather than removal|
| MTN MoMo          | MTN Mobile Money, a mobile money payment method available in Uganda        |
| Meet Link         | Google Meet conference URL auto-generated when an online booking is accepted|

---

*End of CampusCare Software Specifications Document — Version 1.0*
