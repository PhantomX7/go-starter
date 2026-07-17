# Handling "password change required" (frontend)

Seeded **admin** / **root** accounts start with a password chosen by the seeder,
not by the user. Until that account sets its own password, the backend blocks it
from every `/api/v1/admin/*` endpoint. The frontend must detect this state and
walk the user through a password change.

## What the backend returns

When a gated request is blocked, the API responds with:

- **HTTP status:** `403 Forbidden`
- **Body:**

```json
{
  "status": false,
  "message": "password change required"
}
```

> If your HTTP client logs this as `[500]`, that's the client mislabeling it —
> the server sends **403**. Check the real `response.status`.

### Who is affected

| Condition | Gated? |
|---|---|
| Role `admin` or `root`, password never changed (`password_changed_at` is null) | **Yes** — 403 on `/api/v1/admin/*` |
| Role `admin`/`root`, password already changed | No |
| Role `user` (self-registered) | No — never seeded with a default |

### Where it triggers

The gate lives on the `/api/v1/admin` group only. Login still succeeds and
returns tokens; the block appears on the **first** `/admin/*` call.

| Endpoint | Gated by "password change required"? |
|---|---|
| `POST /api/v1/auth/login` | No |
| `POST /api/v1/auth/refresh` | No |
| `GET  /api/v1/auth/me` | No |
| `POST /api/v1/auth/change-password` | **No** (escape hatch) |
| `POST /api/v1/auth/logout` | No |
| `* /api/v1/admin/**` (e.g. `GET /api/v1/admin/user`, `GET /api/v1/admin/log`) | **Yes** |

## Detecting it

### Preferred: the `must_change_password` flag

The `login`, `refresh`, `register` (`AuthResponse`) and `/auth/me` (`MeResponse`)
payloads include a boolean hint:

```json
{
  "access_token": "...",
  "refresh_token": "...",
  "token_type": "Bearer",
  "must_change_password": true
}
```

Check it right after login and route straight to the change-password screen —
no failed request, no message parsing:

```ts
const { data } = await api.post("/api/v1/auth/login", creds);
if (data.data.must_change_password) {
  router.push("/change-password?reason=forced");
}
```

> The flag is a **hint** for UX. The server still enforces the rule with the 403
> below on every `/admin/*` request, so a client that ignores the flag is not a
> security hole — it just gets a worse error experience.

### Fallback / safety net: the 403 response

Also handle the 403 (for any gated request that slips through, e.g. a deep link
straight into `/admin`). `403` is overloaded: a permission denial also returns
`403`, but with a different message (`"insufficient permissions"`).
**Distinguish by the `message` field**, not by status alone.

```ts
const PASSWORD_CHANGE_REQUIRED = "password change required";

function isPasswordChangeRequired(res: { status: number }, body: { message?: string }) {
  return res.status === 403 && body?.message === PASSWORD_CHANGE_REQUIRED;
}
```

### Example: axios interceptor

```ts
api.interceptors.response.use(
  (r) => r,
  (error) => {
    const res = error.response;
    if (res && res.status === 403 && res.data?.message === "password change required") {
      // Remember where the user was headed, then send them to the change screen.
      router.push("/change-password?reason=forced");
    }
    return Promise.reject(error);
  }
);
```

## Resolving it

Call the change-password endpoint (it is **not** gated, so a blocked account can
always reach it):

```
POST /api/v1/auth/change-password
Authorization: Bearer <access_token>
Content-Type: application/json
```

**Request body**

```json
{
  "old_password": "<current default password>",
  "new_password": "<new password, 8–72 chars>",
  "except_token": "<the current session's refresh token>"
}
```

| Field | Rules | Notes |
|---|---|---|
| `old_password` | required, ≤ 72 chars | the password the account currently has |
| `new_password` | required, 8–72 chars | must differ from `old_password` |
| `except_token` | required | **the refresh token you want to keep** |

> **Why `except_token` matters:** changing the password revokes *every* refresh
> session for the user **except** the one you pass here. Pass the current
> session's refresh token so the user stays logged in on this device; every other
> device is signed out. Omitting it (or passing a wrong value) logs the current
> session out too.

**On success (`200`)**

```json
{ "status": true, "message": "..." }
```

`password_changed_at` is now set, the gate is cleared, and the account can use
`/admin/*` normally. Retry the original request (or reload the page the user was
trying to reach).

## End-to-end flow

```
1. POST /auth/login                      -> 200, { access_token, refresh_token }
2. GET  /admin/... (e.g. /admin/user)    -> 403 { message: "password change required" }
3. UI redirects to the change-password screen
4. POST /auth/change-password
        { old_password, new_password, except_token: <refresh_token from step 1> }
                                         -> 200
5. Retry GET /admin/...                  -> 200
```

## Quick checklist

- [ ] Treat `403` + `message === "password change required"` as the trigger (not any 403).
- [ ] Route the user to a password-change screen instead of showing a generic error.
- [ ] Send the current **refresh token** as `except_token` so the user isn't logged out.
- [ ] `new_password` is 8–72 characters and differs from the old one.
- [ ] After `200`, retry the blocked request.
