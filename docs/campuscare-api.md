# CampusCare API Documentation

**Base URL:** `https://campuscare-5zm2.onrender.com`

All request and response bodies use `application/json` unless noted otherwise.

---

## Table of Contents

1. [Authentication](#authentication)
   - [Register](#register)
   - [Login](#login)
   - [Logout](#logout)
2. [Campaigns](#campaigns)
   - [Create Campaign](#create-campaign)
   - [List Approved Campaigns](#list-approved-campaigns)
   - [Update Campaign](#update-campaign)
   - [Delete Campaign](#delete-campaign)
3. [Contributions](#contributions)
   - [Create Contribution](#create-contribution)
   - [Simulate Payment](#simulate-payment)
4. [Bookings](#bookings)
   - [Create Booking](#create-booking)
   - [Update Booking Status](#update-booking-status)
5. [Chatbot](#chatbot)
6. [Admin](#admin)
   - [Dashboard](#dashboard)
   - [List Users](#list-users)
   - [Update User Status](#update-user-status)
   - [List Unapproved Campaigns](#list-unapproved-campaigns)
   - [Approve / Reject Campaign](#approve--reject-campaign)
   - [Delete Campaign (Admin)](#delete-campaign-admin)
   - [List All Bookings](#list-all-bookings)
   - [List All Contributions](#list-all-contributions)
   - [Export Contributions](#export-contributions)

---

## Authentication

### Register

Create a new user account. Supported roles are `student` and `counselor`.

**`POST /register`**

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `email` | string | ✅ | User's email address |
| `password` | string | ✅ | Account password |
| `role` | string | ✅ | Either `"student"` or `"counselor"` |
| `full_name` | string | ✅ | User's full name |
| `consent` | boolean | ✅ | Must be `true` to proceed |

**Example — Register a Student**

```json
{
  "email": "leila.hassan@polytech.edu",
  "password": "Desert_Rose#2026",
  "role": "student",
  "full_name": "Leila Hassan",
  "consent": true
}
```

**Example — Register a Counselor**

```json
{
  "email": "d.kapoor@wellness-center.org",
  "password": "Zen_Master&2026!",
  "role": "counselor",
  "full_name": "Dr. Dev Kapoor",
  "consent": true
}
```

**Response `200 OK`**

```json
{
  "message": "Registered",
  "user_id": "b3750f8a-7511-49ac-94a6-b8a9c13eabd7"
}
```

---

### Login

Authenticate an existing user and start a session.

**`POST /login`**

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `email` | string | ✅ | Registered email address |
| `password` | string | ✅ | Account password |

**Example**

```json
{
  "email": "d.kapoor@wellness-center.org",
  "password": "Zen_Master&2026!"
}
```

**Response `200 OK`**

```json
{
  "message": "Logged in",
  "user_id": "7076f52b-89db-4922-9d12-bf8350c27d83"
}
```

> **Admin Login:** Use the same endpoint with admin credentials (`admin@university.edu`). The response structure is identical.

---

### Logout

End the current user session.

**`POST /logout`**

No request body required.

**Response `200 OK`**

```json
{
  "message": "Logged out"
}
```

---

## Campaigns

### Create Campaign

Submit a new campaign for admin approval. Requires an authenticated student session.

**`POST /campaigns`**

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `title` | string | ✅ | Campaign title |
| `description` | string | ✅ | Detailed description of the need |
| `target_amount` | integer | ✅ | Fundraising goal in UGX |
| `category` | string | ✅ | e.g. `"medical"`, `"education"` |
| `attachments` | array of strings | ❌ | URLs to supporting documents or images |

**Example**

```json
{
  "title": "Final Year Tuition for Sarah",
  "description": "Sarah is a brilliant Computer Science senior facing deregistration due to an outstanding balance of 1.2M UGX. Let's help her clear her finals and graduate this semester.",
  "target_amount": 1200000,
  "category": "education",
  "attachments": [
    "https://res.cloudinary.com/df3lhzzy7/image/upload/v1/campuscare/sarah_statement.pdf",
    "https://res.cloudinary.com/df3lhzzy7/image/upload/v1/campuscare/id_front.jpg"
  ]
}
```

**Response `200 OK`**

```json
{
  "campaign_id": "505a1165-9aff-4ed4-9685-508682b976dd",
  "message": "Campaign submitted for approval"
}
```

> Campaigns are not publicly visible until approved by an admin.

---

### List Approved Campaigns

Fetch all publicly visible, approved campaigns. No authentication required.

**`GET /campaigns`**

**Response `200 OK`**

```json
[
  {
    "id": "4229914d-c0a3-4c32-8e1a-1ece98367955",
    "title": "Emergency Rent for Musa",
    "description": "Musa lost his part-time job last month and is facing eviction from his hostel in 3 days. We are raising funds to cover his rent for this semester so he can focus on his upcoming mid-semester exams.",
    "target_amount": 850000,
    "current_amount": 150000,
    "created_at": "2026-03-04T16:34:04.406144Z"
  }
]
```

**Response Fields**

| Field | Type | Description |
|---|---|---|
| `id` | string | Unique campaign ID |
| `title` | string | Campaign title |
| `description` | string | Campaign details |
| `target_amount` | integer | Goal in UGX |
| `current_amount` | integer | Amount raised so far in UGX |
| `created_at` | string | ISO 8601 timestamp |

---

### Update Campaign

Edit an existing campaign. The campaign will be re-submitted for admin approval after updating.

**`PUT /campaigns/:campaign_id`**

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `title` | string | ✅ | Updated title |
| `description` | string | ✅ | Updated description |
| `target_amount` | integer | ✅ | Updated target in UGX |
| `category` | string | ✅ | Updated category |
| `attachments` | array of strings | ❌ | Pass an empty array to clear attachments |

**Example**

```json
{
  "title": "Medical Aid for John Doe (Updated)",
  "description": "Updated details: John's surgery is scheduled for March 20th. Funds needed urgently.",
  "target_amount": 6000000,
  "category": "medical",
  "attachments": []
}
```

**Response `200 OK`**

```json
{
  "message": "Campaign updated and pending approval"
}
```

---

### Delete Campaign

Delete a campaign. Only the campaign owner can delete their campaign.

**`DELETE /campaigns/:campaign_id`**

No request body required.

**Response `200 OK`**

```json
{
  "message": "Campaign deleted"
}
```

---

## Contributions

### Create Contribution

Submit a donation to an approved campaign. Authentication is not required for anonymous donations.

**`POST /contributions`**

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `campaign_id` | string | ✅ | ID of the campaign to donate to |
| `donor_name` | string | ✅ | Full name of the donor |
| `donor_email` | string | ✅ | Donor's email address |
| `donor_phone` | string | ✅ | Donor's phone number |
| `message` | string | ❌ | Optional message for the beneficiary |
| `is_anonymous` | boolean | ✅ | Set `true` to hide donor identity publicly |
| `payment_method` | string | ✅ | e.g. `"mobile_money"` |
| `amount` | integer | ✅ | Donation amount in UGX |

**Example**

```json
{
  "campaign_id": "4229914d-c0a3-4c32-8e1a-1ece98367955",
  "donor_name": "Jane Smith",
  "donor_email": "jane.smith@gmail.com",
  "donor_phone": "+256701234567",
  "message": "Wishing you a speedy recovery, John!",
  "is_anonymous": false,
  "payment_method": "mobile_money",
  "amount": 150000
}
```

**Response `200 OK`**

```json
{
  "contribution_id": "29138d40-968d-41f6-94df-1ec2eab9c986",
  "message": "Pending payment simulation"
}
```

> After creating a contribution, you must call the **Simulate Payment** endpoint to complete the transaction.

---

### Simulate Payment

Confirm or fail a pending payment for a contribution.

**`POST /contributions/:contribution_id/simulate`**

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `success` | boolean | ✅ | `true` to confirm payment, `false` to simulate failure |

**Example**

```json
{
  "success": true
}
```

**Response `200 OK`**

```json
{
  "status": "success"
}
```

---

## Bookings

### Create Booking

Book a counseling session with a counselor. Requires an authenticated student session.

**`POST /bookings`**

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `counselor_id` | string | ✅ | ID of the counselor to book |
| `start_time` | string | ✅ | ISO 8601 session start time |
| `end_time` | string | ✅ | ISO 8601 session end time |
| `notes` | string | ❌ | Reason for the session or additional context |
| `type` | string | ✅ | Either `"online"` or `"physical"` |

**Example**

```json
{
  "counselor_id": "7076f52b-89db-4922-9d12-bf8350c27d83",
  "start_time": "2026-03-10T09:00:00Z",
  "end_time": "2026-03-10T10:00:00Z",
  "notes": "Struggling with exam anxiety and general stress.",
  "type": "online"
}
```

**Response `200 OK`**

```json
{
  "booking_id": "e2f44163-4214-4fe1-a83b-14a83bd2c6e2"
}
```

---

### Update Booking Status

Accept or decline a booking request. This action is performed by the counselor.

**`PUT /bookings/:booking_id/status`**

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `status` | string | ✅ | Either `"accepted"` or `"declined"` |

**Example — Accept**

```json
{
  "status": "accepted"
}
```

**Example — Decline**

```json
{
  "status": "declined"
}
```

**Response `200 OK`**

```json
{
  "message": "Booking updated"
}
```

---

## Chatbot

Send a message to the mental health support chatbot. The bot detects crisis language and responds with appropriate resources.

**`POST /chatbot`**

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `message` | string | ✅ | The user's message |

**Example**

```json
{
  "message": "I've been feeling really overwhelmed with exams and can't sleep."
}
```

**Response `200 OK`**

```json
{
  "crisis_flagged": false,
  "reply": "I can sense how stressful this must be for you..."
}
```

**Response Fields**

| Field | Type | Description |
|---|---|---|
| `crisis_flagged` | boolean | `true` if the message contains crisis or self-harm indicators |
| `reply` | string | The chatbot's response message |

> **Crisis Detection:** When `crisis_flagged` is `true`, the reply will contain emergency resources and prompt the user to book a counselor session or contact emergency services. The frontend should visually distinguish crisis responses.

---

## Admin

All admin endpoints require an active admin session.

---

### Dashboard

Retrieve high-level platform statistics.

**`GET /admin/dashboard`**

**Response `200 OK`**

```json
{
  "users": 10,
  "campaigns": 4,
  "bookings": 2,
  "total_raised": 150000
}
```

| Field | Type | Description |
|---|---|---|
| `users` | integer | Total registered users |
| `campaigns` | integer | Total campaigns (all statuses) |
| `bookings` | integer | Total bookings |
| `total_raised` | integer | Total funds raised across all campaigns (UGX) |

---

### List Users

Retrieve a paginated list of users, filterable by role.

**`GET /admin/users`**

**Query Parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `role` | string | ❌ | Filter by `"student"`, `"counselor"`, or `"admin"` |
| `page` | integer | ❌ | Page number (default: `1`) |

**Example**

```
GET /admin/users?role=student&page=1
```

**Response `200 OK`**

```json
[
  {
    "id": "b3750f8a-7511-49ac-94a6-b8a9c13eabd7",
    "full_name": "Leila Hassan",
    "email": "leila.hassan@polytech.edu",
    "role": "student",
    "status": "active",
    "created_at": "2026-03-05T09:47:57.37729Z"
  }
]
```

---

### Update User Status

Suspend or reactivate a user account.

**`PUT /admin/users/:user_id/status`**

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `status` | string | ✅ | Either `"active"` or `"suspended"` |

**Example**

```json
{
  "status": "suspended"
}
```

**Response `200 OK`**

```json
{
  "message": "User updated"
}
```

---

### List Unapproved Campaigns

Retrieve all campaigns pending admin review.

**`GET /admin/campaigns`**

**Response `200 OK`**

```json
[
  {
    "id": "505a1165-9aff-4ed4-9685-508682b976dd",
    "student_id": "919ff2a2-6c7f-4117-9b36-40b44993f5af",
    "title": "Final Year Tuition for Sarah",
    "description": "Sarah is a brilliant Computer Science senior...",
    "target_amount": 1200000,
    "category": "education",
    "created_at": "2026-03-04T16:25:32.727707Z"
  }
]
```

---

### Approve / Reject Campaign

Update the approval status of a campaign.

**`PUT /admin/campaigns/:campaign_id`**

**Request Body**

| Field | Type | Required | Description |
|---|---|---|---|
| `status` | string | ✅ | Either `"approved"` or `"rejected"` |

**Example**

```json
{
  "status": "approved"
}
```

**Response `200 OK`**

```json
{
  "message": "Campaign status updated"
}
```

---

### Delete Campaign (Admin)

Permanently remove any campaign from the platform.

**`DELETE /admin/campaigns/:campaign_id`**

No request body required.

**Response `200 OK`**

```json
{
  "message": "Campaign removed"
}
```

---

### List All Bookings

Retrieve all counseling bookings across the platform.

**`GET /admin/bookings`**

**Response `200 OK`**

```json
[
  {
    "id": "e2f44163-4214-4fe1-a83b-14a83bd2c6e2",
    "student_id": "919ff2a2-6c7f-4117-9b36-40b44993f5af",
    "student_name": "",
    "counselor_id": "de5491d6-8c6d-4e4d-8d6e-c207af864c71",
    "counselor_name": "",
    "start_time": "2026-03-10T09:00:00Z",
    "end_time": "2026-03-10T10:00:00Z",
    "status": "accepted"
  }
]
```

**Booking Status Values**

| Status | Description |
|---|---|
| `accepted` | Counselor has confirmed the session |
| `declined` | Counselor has declined the request |
| `pending` | Awaiting counselor response |

---

### List All Contributions

Retrieve all donation contributions across the platform.

**`GET /admin/contributions`**

**Response `200 OK`**

```json
[
  {
    "id": "29138d40-968d-41f6-94df-1ec2eab9c986",
    "campaign_id": "4229914d-c0a3-4c32-8e1a-1ece98367955",
    "donor_name": "Jane Smith",
    "donor_email": "jane.smith@gmail.com",
    "amount": 150000,
    "status": "success",
    "created_at": "2026-03-04T16:46:36.892979Z"
  }
]
```

---

### Export Contributions

Download all contributions as a CSV file.

**`GET /admin/contributions/export`**

**Response `200 OK`**

Returns a `text/csv` file with the following columns:

```
Name, Email, Amount, Status
Jane Smith, jane.smith@gmail.com, 150000, success
```

> The frontend should trigger a file download when hitting this endpoint.