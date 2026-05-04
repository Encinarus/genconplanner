# API Manual Verification Guide

This guide provides step-by-step instructions for manually verifying the `api/v1` endpoints. Use this to establish a baseline before refactoring and to verify parity after changes.

## Prerequisites
- Server running at `http://localhost:8080`
- Access to a browser with DevTools
- `curl` installed on your machine

---

## 1. Public API Verification
These endpoints do not require a session cookie.

### Categories List
- **Command**: `curl -i http://localhost:8080/api/v1/category/2024`
- **Expectations**:
  - HTTP Status: `200 OK`
  - Body: A JSON array of objects.
  - Fields: Verify each object has `name`, `code`, `eventCount`, and `year`.
  - Casing: Ensure keys are camelCase (e.g., `eventCount`).

### Event Search
- **Command**: `curl -i "http://localhost:8080/api/v1/events?cat=BGM&year=2024"`
- **Expectations**:
  - HTTP Status: `200 OK`
  - Body: A JSON array of event summaries.
  - Game Metadata: Verify that the `gameSystem` object is populated with BGG data (e.g., `bggRating`, `numBggRatings`) if the game is found in the cache.

---

## 2. Authenticated API Verification
These endpoints require a valid `signinToken` session cookie.

### Session Setup
1. Log in to the application via the web UI.
2. Open Browser DevTools -> Application -> Cookies.
3. Copy the value of the `signinToken` cookie.

### User Profile
- **Command**: `curl -i -b "signinToken=[TOKEN]" http://localhost:8080/api/v1/user`
- **Expectations**:
  - HTTP Status: `200 OK`
  - Body: JSON object with `email` and `displayName`.

### Starred Events
- **Command**: `curl -i -b "signinToken=[TOKEN]" http://localhost:8080/api/v1/user/events/[USER_EMAIL]/2024`
- **Expectations**:
  - HTTP Status: `200 OK`
  - Body: JSON object with `starredClusters` and `starredEvents` arrays.

---

## 3. Security & Edge Case Verification

### Unauthorized Access
- **Command**: `curl -i http://localhost:8080/api/v1/user` (no cookie)
- **Baseline Expectation**: Document whether the current system returns `401 Unauthorized` or a `302 Redirect`.
- **Refactor Goal**: Ensure a consistent `401 Unauthorized` with body `{"error": "Unauthorized"}`.

### Invalid Token
- **Command**: `curl -i -b "signinToken=invalid-token-value" http://localhost:8080/api/v1/user`
- **Expectations**:
  - HTTP Status: `401 Unauthorized`.
  - Body: `{"error": "Unauthorized"}`.
