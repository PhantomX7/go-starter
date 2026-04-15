# Change Admin Password (Root Only)

## Overview

This endpoint allows a **root** user to change the password of any **admin** user. It does not require the admin's current password. After a successful password change, all of the target admin's refresh tokens are revoked, forcing them to re-login.

---

## Endpoint

```
POST /api/v1/admin/user/:id/change-password
```

### Authorization

- **Role Required:** `root`
- **Middleware:** `RequireAuth()` + `RequireRole("root")`

---

## Request

### Headers

```
Authorization: Bearer <access_token>
Content-Type: application/json
```

### URL Parameters

| Parameter | Type   | Required | Description                        |
|-----------|--------|----------|------------------------------------|
| `id`      | `uint` | Yes      | The ID of the target admin user    |

### Body

| Field          | Type     | Required | Validation | Description              |
|----------------|----------|----------|------------|--------------------------|
| `new_password` | `string` | Yes      | min=8      | The new password to set  |

### Example

```json
{
  "new_password": "newSecurePassword456"
}
```

---

## Response

### Success (200 OK)

```json
{
  "status": true,
  "message": "Password changed successfully",
  "data": null
}
```

### Error Responses

| Status | Condition                                          | Message                                  |
|--------|----------------------------------------------------|------------------------------------------|
| 400    | `new_password` missing or less than 8 characters   | Validation error                         |
| 400    | Target user is not an admin-type role              | `can only change password of admin users`|
| 401    | Missing or invalid access token                    | `unauthorized`                           |
| 403    | Caller is not a root user                          | `insufficient permissions`               |
| 403    | Target user is a root user                         | `cannot change root user password`       |
| 404    | Target user ID not found                           | `user not found`                         |

---

## Flow

1. Root user sends request with the target admin's user ID and a new password
2. `RequireAuth()` middleware validates the access token
3. `RequireRole("root")` middleware ensures the caller has the `root` role
4. Controller parses the user ID from the URL and validates the request body
5. Service looks up the target user by ID
6. Service verifies the target user has an admin-type role (not `user` or `reseller`)
7. Service rejects the request if the target is a `root` user
8. New password is hashed using bcrypt (cost factor: 12)
9. Password is updated in the database
10. All of the target user's refresh tokens are revoked
11. An audit log entry is created with action `change_password`
12. Success response is returned

---

## Security Considerations

- **Root-only access:** Only users with the `root` role can use this endpoint. Admin users cannot change other admins' passwords.
- **Root protection:** A root user cannot change another root user's password through this endpoint. Root users can only change their own password via `POST /api/v1/auth/change-password`.
- **Token revocation:** All refresh tokens for the target user are revoked immediately, forcing them to re-authenticate with the new password.
- **Audit logging:** Every password change is logged with the acting user's identity for traceability.
- **bcrypt hashing:** Passwords are hashed with bcrypt at cost factor 12 before storage.

---

## Code References

| Layer      | File                                                    |
|------------|---------------------------------------------------------|
| DTO        | `internal/dto/user.go` — `ChangeAdminPasswordRequest`  |
| Controller | `internal/modules/user/controller/controller.go`        |
| Service    | `internal/modules/user/service/service.go`              |
| Route      | `internal/routes/routes.go`                             |

---

## Example Usage (cURL)

```bash
curl -X POST http://localhost:8080/api/v1/admin/user/5/change-password \
  -H "Authorization: Bearer <root_access_token>" \
  -H "Content-Type: application/json" \
  -d '{"new_password": "newSecurePassword456"}'
```
