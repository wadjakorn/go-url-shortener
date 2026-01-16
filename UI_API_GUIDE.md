# API Integration Guide for UI Team

This document details the API endpoints for the URL Shortener service. The API is RESTful and returns JSON.

**Base URL**: `https://<your-vercel-app>.vercel.app` (or `http://localhost:8080` for local dev)

## Authentication (Google OAuth)
All `/api/v1/*` endpoints are protected and require a valid session.

### 1. Login
*   **Redirect User To**: `/auth/google/login`
*   **Flow**:
    1.  User clicks "Login with Google".
    2.  Frontend redirects browser to `/auth/google/login`.
    3.  Backend handles OAuth dance with Google.
    4.  Backend sets an `HTTPOnly` cookie named `auth_token`.
    5.  Backend redirects user back to the `FRONTEND_URL` (configured in .env, default: `/dashboard`).

### 2. Making Requests
The browser automatically handles the `auth_token` cookie.
*   **Same-Origin**: No special configuration needed.
*   **Cross-Origin (e.g. localhost:3000 -> localhost:8080)**: Ensure requests include credentials.
    *   `fetch`: `credentials: 'include'`
    *   `axios`: `withCredentials: true`

### 3. Handling Errors
*   **401 Unauthorized**: Session expired or invalid. Redirect user to `/auth/google/login`.

## Common Types

**Link Object**
```typescript
interface Link {
  id: number;
  original_url: string;
  short_code: string;
  title: string;
  tags: string[]; // e.g. ["marketing", "social"]
  clicks: number; // Global click count
  created_at: string; // ISO8601
  updated_at: string; // ISO8601
}
```

## 1. Dashboard & Ranking
*Best for: Home Page / Analytics Overview*

### Get Dashboard Stats
Returns top 10 links by popularity and total system-wide clicks.

*   **Endpoint**: `GET /api/v1/dashboard`
*   **Query Params**:
    *   `limit` (optional): Number of top links (default 10)
    *   `tag` (optional): Filter by tag
    *   `search` (optional): Filter by title
    *   `domain` (optional): Filter by original URL domain
*   **Response**:
    ```json
    {
      "total_system_clicks": 1250,
      "top_links": [ { ...Link Object... } ]
    }
    ```

## 2. Link Management
*Best for: "My Links" Page / List View*

### List All Links
Returns a paginated list of all active links.

*   **Endpoint**: `GET /api/v1/links`
*   **Query Params**:
    *   `page`: Page number (default 1)
    *   `limit`: Items per page (default 10)
    *   `search`: Search by title or URL
    *   `tag`: Filter by tag
*   **Response**:
    ```json
    {
      "data": [ { ...Link Object... } ],
      "total": 50,
      "page": 1,
      "limit": 10
    }
    ```

### Create Short Link
*Best for: "New Link" Modal/Page*

*   **Endpoint**: `POST /api/v1/links`
*   **Body**:
    ```json
    {
      "original_url": "https://example.com/very-long-url",
      "title": "Campaign Q1",
      "tags": ["promo", "q1"],
      "custom_code": "summer-sale" // Optional
    }
    ```
*   **Response**: `201 Created` with full `Link` object.

### Update Link
*   **Endpoint**: `PUT /api/v1/links/{id}`
*   **Body** (partial updates allowed):
    ```json
    {
      "title": "New Title",
      "tags": ["updated"]
    }
    ```

### Delete Link (Soft)
*   **Endpoint**: `DELETE /api/v1/links/{id}`
*   **Response**: `204 No Content`

## 3. detailed Analytics
*Best for: Link Details Page*

### Get Link Stats
Returns detailed breakdown of clicks for a single link.

*   **Endpoint**: `GET /api/v1/links/{id}/stats`
*   **Response**:
    ```json
    {
      "total_clicks": 150,
      "referrers": {
        "google.com": 100,
        "twitter.com": 40,
        "Direct": 10
      },
      "daily_clicks": [
        { "date": "2023-10-25", "count": 12 },
        { "date": "2023-10-24", "count": 5 }
      ]
    }
    ```
