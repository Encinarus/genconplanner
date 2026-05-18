# Refactoring for API Testing

This plan outlines the refactorings needed to make the `api/v1` endpoints easily testable and to improve the overall architecture of the API layer.

## Overview

The current API implementation has tight coupling with external dependencies like Firebase and the GameCache. Testing currently relies on manual stubbing in each test, which is brittle and boilerplate-heavy.

## Goals
- Decouple API handlers from concrete implementations of Firebase and GameCache.
- Reduce boilerplate in API tests.
- Improve authentication handling via middleware.
- Standardize API response and error formats.

## Proposed Refactorings

### 1. Introduce Service Interfaces
Define interfaces for external dependencies to allow easy mocking/stubbing.

- **`AuthService`**: Wraps Firebase authentication logic.
  ```go
  type AuthService interface {
      VerifyToken(ctx context.Context, token string) (string, error)
  }
  ```
- **`GameService`**: Wraps the game cache lookup logic.
  ```go
  type GameService interface {
      FindGame(name string) *background.Game
  }
  ```
- **`EventRepository`**: (Already exists) Ensure it covers all DB needs.

### 2. Struct-based Handler Pattern
Create a `Server` struct to hold dependencies. All API handlers will be migrated from standalone functions to methods on this struct.

```go
type Server struct {
    Repo  EventRepository
    Auth  AuthService
    Games GameService
}
```

### 3. Authentication Middleware & Context
The middleware will be a method on the `Server` struct to access `AuthService`.

#### **Logic Flow**
1. **Extract**: Retrieve the `signinToken` from the cookie.
2. **Verify**: Call `s.Auth.VerifyToken(c.Request.Context(), token)`.
3. **Set**: If successful, `c.Set(userEmailKey, email)`.
4. **Abort**: If the cookie is missing or verification fails, immediately `c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})`.

#### **Context Helpers**
To avoid magic strings and type assertions in handlers:
- **`const userEmailKey = "userEmail"`**: Defined in `server.go`.
- **`func GetUserEmail(c *gin.Context) string`**: A package-level or `Server` method to safely extract the email.

#### **Optional Auth**
For routes that can benefit from a user context but don't require it (e.g., public search that highlights starred events), a separate `s.OptionalAuth()` middleware will be created that populates the context but never calls `Abort`.

### 4. Route Reorganization & Grouping
Update `BuildAPIRoutes` to initialize the `Server` and organize routes into public and authorized groups.

```go
func (s *Server) RegisterRoutes(group *gin.RouterGroup) {
    v1 := group.Group("/v1")
    {
        v1.GET("/category/:year", s.ListCategories)
        
        // Auth protected group
        auth := v1.Group("/")
        auth.Use(s.AuthMiddleware())
        {
            auth.GET("/user", s.GetUser)
            auth.GET("/user/events/:email/:year", s.LoadUserEvents)
        }
    }
}
```

### 5. Standardized Responses
- Use `c.JSON(code, data)` exclusively (replace `json.NewEncoder`).
- Define a standard error response: `type ErrorResponse struct { Error string `json:"error"` }`.

### 6. Testing Infrastructure
Enhance testing by adding:
- `stubs_test.go`: Shared stubs for `AuthService` and `GameService`.
- `setupTestServer()`: Returns a pre-configured Gin router with stubs injected.
- Standardized assertions for JSON response bodies.

## File-Specific Changes

| File | Changes |
| :--- | :--- |
| `internal/api/interfaces.go` | **[NEW]** Define `AuthService`, `GameService`, and `ErrorResponse`. |
| `internal/api/server.go` | **[NEW]** Define `Server` struct, `RegisterRoutes`, and middleware logic. |
| `internal/api/users.go` | Move handlers to `Server` methods; use `GetUserEmail` helper. |
| `internal/api/events.go` | Move handlers to `Server` methods; use `GameService` interface. |
| `internal/api/categories.go`| Move handlers to `Server` methods. |
| `internal/api/api_routes.go`| Update to use the new `Server` initialization. |
| `internal/api/stubs_test.go` | **[NEW]** Implement reusable stubs for testing. |
| `internal/api/api_test.go` | Refactor to use `setupTestServer` and standard assertions. |

## Verification Plan

### 1. Automated Regression Testing
Run existing tests before and after refactoring to ensure no loss of functionality:
```bash
go test -v ./internal/api/...
```
The following existing tests must pass after migration to the `Server` struct:
- `TestCategoryValidation`
- `TestEventLookup`
- `TestEventSearch`

### 2. Manual Verification
Follow the [Manual Verification Guide](file:///Users/alek/projects/genconplanner/ai/manual_testing.md) to establish a baseline before refactoring and to verify parity after changes. This guide covers:
- Public endpoints (Categories, Search).
- Authenticated endpoints (User Profile, Stars).
- Security behavior (Unauthorized/Invalid tokens).

## Next Steps
1. **Execute Baseline Verification**: Follow the [Manual Verification Guide](file:///Users/alek/projects/genconplanner/ai/manual_testing.md) and document results.
2. **Setup Test Infrastructure**: Create `interfaces.go`, `server.go`, and `stubs_test.go`.
3. **Implement AuthMiddleware**: Refactor auth logic into middleware and verify against the guide.
4. **Endpoint Migration**: Move handlers to `Server` methods one by one, starting with `CategoryList`.
5. **Final Parity Check**: Re-run the manual verification guide to ensure no regressions.
