# Footer Menu Module

## Overview

The footer menu module provides a hierarchical, drag-and-drop-reorderable navigation menu system for the website footer. Each menu item has **content** (display text) and a **URL**, and can be nested up to **2 levels deep** (top → level 1) via parent-child relationships. The frontend sends the full reordered tree state after a drag & drop operation.

### Key Features
- **Hierarchical Structure**: Parent-child relationships with tracked depth (max 2 levels: top → 1)
- **Drag & Drop Reorder**: Bulk update endpoint for reordering and re-parenting in a single request
- **Public Tree Endpoint**: Nested tree response for frontend rendering
- **Audit Logging**: All create, update, and delete operations are logged

---

## Database Schema

**Table:** `footer_menus`

| Column         | Type                 | Nullable | Default | Description                            |
|----------------|----------------------|----------|---------|----------------------------------------|
| `id`           | `bigserial`          | No       | auto    | Primary key                            |
| `content`      | `varchar(255)`       | No       |         | Display text of the menu item          |
| `url`          | `varchar(500)`       | No       |         | Link URL                               |
| `display_order`| `integer`            | No       | `0`     | Sort position within siblings          |
| `depth`        | `integer`            | No       | `0`     | Nesting level (0 = root)               |
| `is_active`    | `boolean`            | No       | `true`  | Whether the item is visible            |
| `parent_id`    | `bigint`             | Yes      | `NULL`  | Self-referencing FK to `footer_menus`  |
| `created_at`   | `timestamptz`        | No       |         | Creation timestamp                     |
| `updated_at`   | `timestamptz`        | No       |         | Last update timestamp                  |
| `deleted_at`   | `timestamptz`        | Yes      | `NULL`  | Soft delete timestamp                  |

**Indexes:** `deleted_at`, `parent_id`, `display_order`

---

## Permissions

| Permission              | Description               |
|-------------------------|---------------------------|
| `footer_menu:create`    | Create footer menu items  |
| `footer_menu:read`      | View footer menu items    |
| `footer_menu:update`    | Update footer menu items  |
| `footer_menu:delete`    | Delete footer menu items  |
| `footer_menu:reorder`   | Reorder footer menu items |

---

## Admin API Endpoints

All admin endpoints require authentication and the corresponding permission.

**Base path:** `/api/v1/admin/footer-menu`

---

### 1. List Footer Menus (Paginated)

```
GET /api/v1/admin/footer-menu
```

**Permission:** `footer_menu:read`

Each footer menu item includes nested children up to 2 levels deep (top → level 1) and its parent.

**Query Parameters:**

| Parameter       | Type     | Description                          |
|-----------------|----------|--------------------------------------|
| `limit`         | `int`    | Items per page (default: 20, max: 200) |
| `offset`        | `int`    | Offset for pagination                |
| `sort`          | `string` | Sort field (id, display_order, content, created_at) |
| `content`       | `string` | Filter by content text               |
| `is_active`     | `bool`   | Filter by active status              |
| `parent_id`     | `uint`   | Filter by parent ID                  |
| `created_at`    | `date`   | Filter by creation date              |

**Response:**
```json
{
  "status": true,
  "data": [
    {
      "id": 1,
      "content": "Company",
      "url": "/company",
      "display_order": 0,
      "depth": 0,
      "is_active": true,
      "parent_id": null,
      "created_at": "2026-04-01T12:00:00Z",
      "children": [],
      "parent": null
    }
  ],
  "meta": {
    "limit": 20,
    "offset": 0,
    "total": 1
  }
}
```

---

### 2. Get Footer Menu Tree

```
GET /api/v1/admin/footer-menu/tree
```

**Permission:** `footer_menu:read`

Returns the full nested footer menu structure (root items with children preloaded up to 2 levels deep), sorted by `display_order`.

**Response:**
```json
{
  "status": true,
  "message": "Footer menu tree fetched successfully",
  "data": [
    {
      "id": 1,
      "content": "Company",
      "url": "/company",
      "display_order": 0,
      "depth": 0,
      "is_active": true,
      "parent_id": null,
      "created_at": "2026-04-01T12:00:00Z",
      "children": [
        {
          "id": 3,
          "content": "About Us",
          "url": "/company/about",
          "display_order": 0,
          "depth": 1,
          "is_active": true,
          "parent_id": 1,
          "created_at": "2026-04-01T12:00:00Z",
          "children": []
        },
        {
          "id": 4,
          "content": "Careers",
          "url": "/company/careers",
          "display_order": 1,
          "depth": 1,
          "is_active": true,
          "parent_id": 1,
          "created_at": "2026-04-01T12:00:00Z",
          "children": []
        }
      ]
    },
    {
      "id": 2,
      "content": "Support",
      "url": "/support",
      "display_order": 1,
      "depth": 0,
      "is_active": true,
      "parent_id": null,
      "created_at": "2026-04-01T12:00:00Z",
      "children": []
    }
  ]
}
```

---

### 3. Get Footer Menu by ID

```
GET /api/v1/admin/footer-menu/:id
```

**Permission:** `footer_menu:read`

**Response:**
```json
{
  "status": true,
  "message": "Footer menu found successfully",
  "data": {
    "id": 1,
    "content": "Company",
    "url": "/company",
    "display_order": 0,
    "depth": 0,
    "is_active": true,
    "parent_id": null,
    "created_at": "2026-04-01T12:00:00Z",
    "children": [...],
    "parent": null
  }
}
```

---

### 4. Create Footer Menu

```
POST /api/v1/admin/footer-menu
```

**Permission:** `footer_menu:create`

**Request Body:**

| Field          | Type    | Required | Validation                              | Description              |
|----------------|---------|----------|-----------------------------------------|--------------------------|
| `content`      | `string`| Yes      |                                         | Display text             |
| `url`          | `string`| Yes      |                                         | Link URL                 |
| `display_order`| `int`   | Yes      |                                         | Sort position            |
| `is_active`    | `bool`  | No       | Defaults to `true`                      | Visibility flag          |
| `parent_id`    | `uint`  | No       | Must reference an existing footer menu  | Parent footer menu ID    |

**Example:**
```json
{
  "content": "About Us",
  "url": "/company/about",
  "display_order": 0,
  "is_active": true,
  "parent_id": 1
}
```

**Response (201):**
```json
{
  "status": true,
  "message": "Footer menu created successfully",
  "data": {
    "id": 3,
    "content": "About Us",
    "url": "/company/about",
    "display_order": 0,
    "depth": 1,
    "is_active": true,
    "parent_id": 1,
    "created_at": "2026-04-01T12:00:00Z",
    "children": []
  }
}
```

**Notes:**
- The `depth` field is calculated automatically based on the parent's depth.
- If `parent_id` is `null`, the item is a root footer menu (depth 0).
- Maximum depth is 1 (2 levels total). Attempting to create a deeper item returns a 400 error.

---

### 5. Update Footer Menu

```
PATCH /api/v1/admin/footer-menu/:id
```

**Permission:** `footer_menu:update`

**Request Body:** (all fields optional)

| Field          | Type    | Validation                              | Description              |
|----------------|---------|-------- --------------------------------|--------------------------|
| `content`      | `string`|                                         | Display text             |
| `url`          | `string`|                                         | Link URL                 |
| `display_order`| `int`   |                                         | Sort position            |
| `is_active`    | `bool`  |                                         | Visibility flag          |
| `parent_id`    | `uint`  | Must reference an existing footer menu  | Parent footer menu ID    |

**Example:**
```json
{
  "content": "About Our Company",
  "url": "/company/about-us"
}
```

**Response (200):**
```json
{
  "status": true,
  "message": "Footer menu updated successfully",
  "data": { ... }
}
```

**Validation Rules:**
- A footer menu cannot be its own parent
- A footer menu with children cannot be moved under another parent (to prevent orphaned subtrees)
- Maximum depth is 1 (2 levels total)

---

### 6. Delete Footer Menu

```
DELETE /api/v1/admin/footer-menu/:id
```

**Permission:** `footer_menu:delete`

**Response (200):**
```json
{
  "status": true,
  "message": "Footer menu deleted successfully",
  "data": null
}
```

**Validation Rules:**
- Cannot delete a footer menu that has children. Remove or re-parent children first.

---

### 7. Reorder Footer Menus (Drag & Drop)

```
PUT /api/v1/admin/footer-menu/reorder
```

**Permission:** `footer_menu:reorder`

This endpoint is designed for **drag & drop** interfaces. After the user finishes dragging, the frontend sends the entire flattened tree state with updated positions, parents, and depths.

**Request Body:**

| Field   | Type    | Required | Description                            |
|---------|---------|----------|----------------------------------------|
| `items` | `array` | Yes      | Array of reorder items                 |

Each item in the array:

| Field          | Type   | Required | Description                          |
|----------------|--------|----------|--------------------------------------|
| `id`           | `uint` | Yes      | Footer menu item ID                  |
| `parent_id`    | `uint` | No       | New parent ID (`null` for root)      |
| `display_order`| `int`  | Yes      | New sort position within siblings    |
| `depth`        | `int`  | Yes      | New nesting level (max 1)            |

**Example:**

Before reorder:
```
Company (order: 0)
  ├── About Us (order: 0)
  └── Careers (order: 1)
Support (order: 1)
```

User drags "Support" above "Company" and moves "Careers" to root level:

```json
{
  "items": [
    { "id": 2, "parent_id": null, "display_order": 0, "depth": 0 },
    { "id": 1, "parent_id": null, "display_order": 1, "depth": 0 },
    { "id": 3, "parent_id": 1,    "display_order": 0, "depth": 1 },
    { "id": 4, "parent_id": null, "display_order": 2, "depth": 0 }
  ]
}
```

After reorder:
```
Support (order: 0)
Company (order: 1)
  └── About Us (order: 0)
Careers (order: 2)
```

**Response (200):**
```json
{
  "status": true,
  "message": "Footer menu reordered successfully",
  "data": null
}
```

**Note:** Depth values in reorder items must not exceed 1 (max 2 levels).

---

## Public API Endpoint

No authentication required.

### Get Footer Menu Tree

```
GET /api/v1/public/footer-menu/tree
```

Returns the same nested tree structure as the admin tree endpoint (up to 2 levels deep). Use this to render the footer navigation on the frontend.

**Response:** Same format as [Admin Get Footer Menu Tree](#2-get-footer-menu-tree).

---

## Error Responses

| Status | Condition                                     | Message                                           |
|--------|-----------------------------------------------|---------------------------------------------------|
| 400    | Validation error                              | Field-specific validation messages                |
| 400    | Invalid parent footer menu ID                 | `invalid parent footer menu`                      |
| 400    | Footer menu set as its own parent             | `footer menu cannot be its own parent`            |
| 400    | Moving footer menu with children under parent | `footer menu with children cannot be a sub-menu`  |
| 400    | Deleting footer menu with children            | `cannot delete footer menu with children`         |
| 400    | Exceeding max depth (2 levels)                | `footer menu supports only 2 levels deep`         |
| 401    | Missing or invalid access token               | `unauthorized`                                    |
| 403    | Insufficient permissions                      | `insufficient permissions`                        |
| 404    | Footer menu ID not found                      | `footer menu not found`                           |

---

## Frontend Integration (Drag & Drop)

### Recommended Flow

1. **Load tree:** `GET /api/v1/admin/footer-menu/tree` to get the nested footer menu structure
2. **Render:** Display the tree in a drag & drop UI component
3. **On drop:** Flatten the tree into an array of `{ id, parent_id, display_order, depth }` items
4. **Save:** `PUT /api/v1/admin/footer-menu/reorder` with the flattened items array
5. **Refresh:** Optionally re-fetch the tree to confirm the new state

### Calculating Fields on the Frontend

When the user drops an item:
- **`parent_id`**: Set to the ID of the new parent, or `null` if dropped at root level
- **`display_order`**: Assign sequential integers (0, 1, 2, ...) to siblings based on their visual position
- **`depth`**: Parent's depth + 1 (root items have depth 0). **Max depth is 1.**

---

## Code References

| Layer      | File                                                                  |
|------------|-----------------------------------------------------------------------|
| Model      | `internal/models/footer_menu.go`                                      |
| DTO        | `internal/dto/footer_menu.go`                                         |
| Repository | `internal/modules/footer_menu/repository/repository.go`               |
| Service    | `internal/modules/footer_menu/service/service.go`                     |
| Controller | `internal/modules/footer_menu/controller/controller.go`               |
| Module     | `internal/modules/footer_menu/module.go`                              |
| Routes     | `internal/routes/routes.go`                                           |
| Permissions| `pkg/constants/permissions/permissions.go`                            |
| Migration  | `database/migrations/20260406070000_create_footer_menus_table.*`      |

---

## Example Usage (cURL)

**Create a root footer menu:**
```bash
curl -X POST http://localhost:8080/api/v1/admin/footer-menu \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"content": "Company", "url": "/company", "display_order": 0}'
```

**Create a child footer menu:**
```bash
curl -X POST http://localhost:8080/api/v1/admin/footer-menu \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"content": "About Us", "url": "/company/about", "display_order": 0, "parent_id": 1}'
```

**Fetch public tree:**
```bash
curl http://localhost:8080/api/v1/public/footer-menu/tree
```

**Reorder after drag & drop:**
```bash
curl -X PUT http://localhost:8080/api/v1/admin/footer-menu/reorder \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {"id": 2, "parent_id": null, "display_order": 0, "depth": 0},
      {"id": 1, "parent_id": null, "display_order": 1, "depth": 0},
      {"id": 3, "parent_id": 1,    "display_order": 0, "depth": 1}
    ]
  }'
```
