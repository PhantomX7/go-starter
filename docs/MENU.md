# Menu Module

## Overview

The menu module provides a hierarchical, drag-and-drop-reorderable navigation menu system. Each menu item has **content** (display text) and a **URL**, and can be nested up to **3 levels deep** (top → level 1 → level 2) via parent-child relationships. The frontend sends the full reordered tree state after a drag & drop operation.

### Key Features
- **Hierarchical Structure**: Parent-child relationships with tracked depth (max 3 levels: top → 1 → 2)
- **Drag & Drop Reorder**: Bulk update endpoint for reordering and re-parenting in a single request
- **Public Tree Endpoint**: Nested tree response for frontend rendering
- **Audit Logging**: All create, update, and delete operations are logged

---

## Database Schema

**Table:** `menus`

| Column         | Type                 | Nullable | Default | Description                     |
|----------------|----------------------|----------|---------|---------------------------------|
| `id`           | `bigserial`          | No       | auto    | Primary key                     |
| `content`      | `varchar(255)`       | No       |         | Display text of the menu item   |
| `url`          | `varchar(500)`       | No       |         | Link URL                        |
| `display_order`| `integer`            | No       | `0`     | Sort position within siblings   |
| `depth`        | `integer`            | No       | `0`     | Nesting level (0 = root)        |
| `is_active`    | `boolean`            | No       | `true`  | Whether the item is visible     |
| `parent_id`    | `bigint`             | Yes      | `NULL`  | Self-referencing FK to `menus`  |
| `created_at`   | `timestamptz`        | No       |         | Creation timestamp              |
| `updated_at`   | `timestamptz`        | No       |         | Last update timestamp           |
| `deleted_at`   | `timestamptz`        | Yes      | `NULL`  | Soft delete timestamp           |

**Indexes:** `deleted_at`, `parent_id`, `display_order`

---

## Permissions

| Permission      | Description          |
|-----------------|----------------------|
| `menu:create`   | Create menu items    |
| `menu:read`     | View menu items      |
| `menu:update`   | Update menu items    |
| `menu:delete`   | Delete menu items    |
| `menu:reorder`  | Reorder menu items   |

---

## Admin API Endpoints

All admin endpoints require authentication and the corresponding permission.

**Base path:** `/api/v1/admin/menu`

---

### 1. List Menus (Paginated)

```
GET /api/v1/admin/menu
```

**Permission:** `menu:read`

Each menu item includes nested children up to 3 levels deep (top → level 1 → level 2) and its parent.

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
      "content": "Home",
      "url": "/",
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

### 2. Get Menu Tree

```
GET /api/v1/admin/menu/tree
```

**Permission:** `menu:read`

Returns the full nested menu structure (root items with children preloaded up to 3 levels deep), sorted by `display_order`.

**Response:**
```json
{
  "status": true,
  "message": "Menu tree fetched successfully",
  "data": [
    {
      "id": 1,
      "content": "Products",
      "url": "/products",
      "display_order": 0,
      "depth": 0,
      "is_active": true,
      "parent_id": null,
      "created_at": "2026-04-01T12:00:00Z",
      "children": [
        {
          "id": 3,
          "content": "Laptops",
          "url": "/products/laptops",
          "display_order": 0,
          "depth": 1,
          "is_active": true,
          "parent_id": 1,
          "created_at": "2026-04-01T12:00:00Z",
          "children": [
            {
              "id": 5,
              "content": "Gaming Laptops",
              "url": "/products/laptops/gaming",
              "display_order": 0,
              "depth": 2,
              "is_active": true,
              "parent_id": 3,
              "created_at": "2026-04-01T12:00:00Z",
              "children": []
            }
          ]
        },
        {
          "id": 4,
          "content": "Desktops",
          "url": "/products/desktops",
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
      "content": "About",
      "url": "/about",
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

### 3. Get Menu by ID

```
GET /api/v1/admin/menu/:id
```

**Permission:** `menu:read`

**Response:**
```json
{
  "status": true,
  "message": "Menu found successfully",
  "data": {
    "id": 1,
    "content": "Products",
    "url": "/products",
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

### 4. Create Menu

```
POST /api/v1/admin/menu
```

**Permission:** `menu:create`

**Request Body:**

| Field          | Type    | Required | Validation                    | Description              |
|----------------|---------|----------|-------------------------------|--------------------------|
| `content`      | `string`| Yes      |                               | Display text             |
| `url`          | `string`| Yes      |                               | Link URL                 |
| `display_order`| `int`   | Yes      |                               | Sort position            |
| `is_active`    | `bool`  | No       | Defaults to `true`            | Visibility flag          |
| `parent_id`    | `uint`  | No       | Must reference an existing menu | Parent menu ID          |

**Example:**
```json
{
  "content": "Laptops",
  "url": "/products/laptops",
  "display_order": 0,
  "is_active": true,
  "parent_id": 1
}
```

**Response (201):**
```json
{
  "status": true,
  "message": "Menu created successfully",
  "data": {
    "id": 3,
    "content": "Laptops",
    "url": "/products/laptops",
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
- If `parent_id` is `null`, the item is a root menu (depth 0).

---

### 5. Update Menu

```
PATCH /api/v1/admin/menu/:id
```

**Permission:** `menu:update`

**Request Body:** (all fields optional)

| Field          | Type    | Validation                    | Description              |
|----------------|---------|-------------------------------|--------------------------|
| `content`      | `string`|                               | Display text             |
| `url`          | `string`|                               | Link URL                 |
| `display_order`| `int`   |                               | Sort position            |
| `is_active`    | `bool`  |                               | Visibility flag          |
| `parent_id`    | `uint`  | Must reference an existing menu | Parent menu ID          |

**Example:**
```json
{
  "content": "Gaming Laptops",
  "url": "/products/gaming-laptops"
}
```

**Response (200):**
```json
{
  "status": true,
  "message": "Menu updated successfully",
  "data": { ... }
}
```

**Validation Rules:**
- A menu cannot be its own parent
- A menu with children cannot be moved under another parent (to prevent orphaned subtrees)

---

### 6. Delete Menu

```
DELETE /api/v1/admin/menu/:id
```

**Permission:** `menu:delete`

**Response (200):**
```json
{
  "status": true,
  "message": "Menu deleted successfully",
  "data": null
}
```

**Validation Rules:**
- Cannot delete a menu that has children. Remove or re-parent children first.

---

### 7. Reorder Menus (Drag & Drop)

```
PUT /api/v1/admin/menu/reorder
```

**Permission:** `menu:reorder`

This endpoint is designed for **drag & drop** interfaces. After the user finishes dragging, the frontend sends the entire flattened tree state with updated positions, parents, and depths.

**Request Body:**

| Field   | Type    | Required | Description                            |
|---------|---------|----------|----------------------------------------|
| `items` | `array` | Yes      | Array of reorder items                 |

Each item in the array:

| Field          | Type   | Required | Description                          |
|----------------|--------|----------|--------------------------------------|
| `id`           | `uint` | Yes      | Menu item ID                         |
| `parent_id`    | `uint` | No       | New parent ID (`null` for root)      |
| `display_order`| `int`  | Yes      | New sort position within siblings    |
| `depth`        | `int`  | Yes      | New nesting level                    |

**Example:**

Before reorder:
```
Products (order: 0)
  ├── Laptops (order: 0)
  └── Desktops (order: 1)
About (order: 1)
```

User drags "About" above "Products" and moves "Desktops" to root level:

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
About (order: 0)
Products (order: 1)
  └── Laptops (order: 0)
Desktops (order: 2)
```

**Response (200):**
```json
{
  "status": true,
  "message": "Menu reordered successfully",
  "data": null
}
```

---

## Public API Endpoint

No authentication required.

### Get Menu Tree

```
GET /api/v1/public/menu/tree
```

Returns the same nested tree structure as the admin tree endpoint (up to 3 levels deep). Use this to render the navigation menu on the frontend.

**Response:** Same format as [Admin Get Menu Tree](#2-get-menu-tree).

---

## Error Responses

| Status | Condition                              | Message                                    |
|--------|----------------------------------------|--------------------------------------------|
| 400    | Validation error                       | Field-specific validation messages          |
| 400    | Invalid parent menu ID                 | `invalid parent menu`                      |
| 400    | Menu set as its own parent             | `menu cannot be its own parent`            |
| 400    | Moving menu with children under parent | `menu with children cannot be a sub-menu`  |
| 400    | Deleting menu with children            | `cannot delete menu with children`         |
| 401    | Missing or invalid access token        | `unauthorized`                             |
| 403    | Insufficient permissions               | `insufficient permissions`                 |
| 404    | Menu ID not found                      | `menu not found`                           |

---

## Frontend Integration (Drag & Drop)

### Recommended Flow

1. **Load tree:** `GET /api/v1/admin/menu/tree` to get the nested menu structure
2. **Render:** Display the tree in a drag & drop UI component
3. **On drop:** Flatten the tree into an array of `{ id, parent_id, display_order, depth }` items
4. **Save:** `PUT /api/v1/admin/menu/reorder` with the flattened items array
5. **Refresh:** Optionally re-fetch the tree to confirm the new state

### Calculating Fields on the Frontend

When the user drops an item:
- **`parent_id`**: Set to the ID of the new parent, or `null` if dropped at root level
- **`display_order`**: Assign sequential integers (0, 1, 2, ...) to siblings based on their visual position
- **`depth`**: Parent's depth + 1 (root items have depth 0)

---

## Code References

| Layer      | File                                                      |
|------------|-----------------------------------------------------------|
| Model      | `internal/models/menu.go`                                 |
| DTO        | `internal/dto/menu.go`                                    |
| Repository | `internal/modules/menu/repository/repository.go`          |
| Service    | `internal/modules/menu/service/service.go`                |
| Controller | `internal/modules/menu/controller/controller.go`          |
| Module     | `internal/modules/menu/module.go`                         |
| Routes     | `internal/routes/routes.go`                               |
| Permissions| `pkg/constants/permissions/permissions.go`                |
| Migration  | `database/migrations/20260401120000_create_menus_table.*` |

---

## Example Usage (cURL)

**Create a root menu:**
```bash
curl -X POST http://localhost:8080/api/v1/admin/menu \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"content": "Products", "url": "/products", "display_order": 0}'
```

**Create a child menu:**
```bash
curl -X POST http://localhost:8080/api/v1/admin/menu \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"content": "Laptops", "url": "/products/laptops", "display_order": 0, "parent_id": 1}'
```

**Fetch public tree:**
```bash
curl http://localhost:8080/api/v1/public/menu/tree
```

**Reorder after drag & drop:**
```bash
curl -X PUT http://localhost:8080/api/v1/admin/menu/reorder \
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
