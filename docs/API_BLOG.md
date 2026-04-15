# Blog & Blog Category API Documentation

## Overview

This document outlines the API endpoints and data structures for the Blog and Blog Category modules. It provides details on how to manage blog posts and their categories, including hierarchical category structures and the assignment of categories to blogs.

## Data Models

### Blog Category

Represents a category for blog posts. Categories can be hierarchical.

| Field | Type | Description |
|---|---|---|
| `id` | `uint` | Unique identifier |
| `name` | `string` | Category name |
| `slug` | `string` | Unique slug for URLs |
| `is_active` | `bool` | Activation status |
| `parent_id` | `uint` | (Optional) ID of the parent category |
| `children` | `[]BlogCategoryResponse` | List of sub-categories |
| `created_at` | `time.Time` | Creation timestamp |

### Blog

Represents a blog post.

| Field | Type | Description |
|---|---|---|
| `id` | `uint` | Unique identifier |
| `title` | `string` | Blog title |
| `slug` | `string` | Unique slug for URLs |
| `content` | `string` | Blog content (HTML/Markdown) |
| `short_description` | `string` | Brief summary |
| `thumbnail_url` | `string` | URL of the thumbnail image |
| `is_active` | `bool` | Activation status |
| `published_at` | `time.Time` | (Optional) Publish date/time. When set and in the past, the blog is visible on public endpoints. `null` means draft. |
| `categories` | `[]BlogCategoryResponse` | List of associated categories |
| `created_at` | `time.Time` | Creation timestamp |

---

## API Endpoints

### Blog Category Endpoints

#### 1. List Blog Categories (Admin)
Retrieves a paginated list of blog categories.

- **URL**: `/api/v1/admin/blog-category`
- **Method**: `GET`
- **Auth**: Required (`blog_category:read`)
- **Query Params**:
    - `page`: Page number (default: 1)
    - `limit`: Items per page (default: 10)
    - `search`: Search term
    - `sort`: Sort field (e.g., `created_at desc`)

#### 2. Get Blog Category by ID (Admin)
Retrieves a specific blog category by ID.

- **URL**: `/api/v1/admin/blog-category/:id`
- **Method**: `GET`
- **Auth**: Required (`blog_category:read`)

#### 3. Create Blog Category (Admin)
Creates a new blog category.

- **URL**: `/api/v1/admin/blog-category`
- **Method**: `POST`
- **Auth**: Required (`blog_category:create`)
- **Request Body**:
  ```json
  {
    "name": "Technology",
    "slug": "technology",
    "is_active": true,
    "parent_id": 1 // Optional, for sub-categories
  }
  ```

#### 4. Update Blog Category (Admin)
Updates an existing blog category.

- **URL**: `/api/v1/admin/blog-category/:id`
- **Method**: `PATCH`
- **Auth**: Required (`blog_category:update`)
- **Request Body**:
  ```json
  {
    "name": "Tech News",
    "is_active": false
  }
  ```

#### 5. Delete Blog Category (Admin)
Deletes a blog category.

- **URL**: `/api/v1/admin/blog-category/:id`
- **Method**: `DELETE`
- **Auth**: Required (`blog_category:delete`)

#### 6. List Blog Categories (Public)
Retrieves a list of active blog categories.

- **URL**: `/api/v1/public/blog-category`
- **Method**: `GET`
- **Auth**: None

#### 7. Get Blog Category by Slug (Public)
Retrieves a specific blog category by slug.

- **URL**: `/api/v1/public/blog-category/:slug`
- **Method**: `GET`
- **Auth**: None

---

### Blog Endpoints

#### 1. List Blogs (Admin)
Retrieves a paginated list of blogs.

- **URL**: `/api/v1/admin/blog`
- **Method**: `GET`
- **Auth**: Required (`blog:read`)
- **Response**: Includes `categories` array for each blog.

#### 2. Create Blog (Admin)
Creates a new blog post with optional categories.

- **URL**: `/api/v1/admin/blog`
- **Method**: `POST`
- **Auth**: Required (`blog:create`)
- **Content-Type**: `multipart/form-data`
- **Form Fields**:
    - `title`: string (required)
    - `slug`: string (required)
    - `content`: string (required)
    - `short_description`: string (required)
    - `thumbnail`: file (required)
    - `is_active`: bool
    - `published_at`: string (optional) - ISO 8601 format (e.g., `2026-04-06T10:00:00Z`). Set to schedule publishing. Leave empty/null for draft.
    - `category_ids[]`: array of uint (optional) - e.g., `category_ids[]=1&category_ids[]=2`

#### 3. Update Blog (Admin)
Updates an existing blog post.

- **URL**: `/api/v1/admin/blog/:id`
- **Method**: `PATCH`
- **Auth**: Required (`blog:update`)
- **Content-Type**: `multipart/form-data`
- **Form Fields**:
    - `title`: string
    - `published_at`: string (optional) - ISO 8601 format (e.g., `2026-04-06T10:00:00Z`). Set to schedule publishing.
    - `category_ids[]`: array of uint - e.g., `category_ids[]=1&category_ids[]=3`. To clear categories, send empty array or don't send if functionality implies replacement. **Note**: Sending `category_ids` replaces all existing associations.

#### 4. Get Blog by ID (Admin)

Retrieves a specific blog by ID.

- **URL**: `/api/v1/admin/blog/:id`
- **Method**: `GET`
- **Auth**: Required (`blog:read`)
- **Response**: Returns the blog regardless of `published_at` status.

#### 5. Delete Blog (Admin)
Deletes a blog post.

- **URL**: `/api/v1/admin/blog/:id`
- **Method**: `DELETE`
- **Auth**: Required (`blog:delete`)

#### 6. List Blogs (Public)
Retrieves a paginated list of published blogs. Only blogs with `published_at` set and in the past are returned.

- **URL**: `/api/v1/public/blog`
- **Method**: `GET`
- **Auth**: None

#### 7. Get Featured Blogs (Public)

Retrieves active categories with their latest published blog post.

- **URL**: `/api/v1/public/blog/featured`
- **Method**: `GET`
- **Auth**: None
- **Response**: Only includes blogs where `published_at` is set and in the past.

#### 8. Get Blog by Slug (Public)
Retrieves a specific published blog by slug. Returns 404 if the blog is not yet published.

- **URL**: `/api/v1/public/blog/:slug`
- **Method**: `GET`
- **Auth**: None
- **Response**: Includes `categories` details.

---

## Integration Notes for Frontend

1.  **Category Selection**: When creating or updating a blog, use a multi-select component populated by the `GET /api/v1/admin/blog-category` endpoint.
2.  **Form Data**: Since Blog creation/update requires a file upload (`thumbnail`), you must use `FormData`.
    *   Append `category_ids` as `category_ids[]` for each selected ID to ensure correct binding in the backend.
    *   Example JS:
        ```javascript
        const formData = new FormData();
        formData.append('title', data.title);
        // ... other fields
        selectedCategoryIds.forEach(id => {
            formData.append('category_ids[]', id);
        });
        ```
3.  **Hierarchy**: The Blog Category list endpoint returns a flat list (or hierarchical depending on implementation, check if `children` are populated or if it uses `parent_id` for client-side tree building). *Currently, the response structure supports `children`, so it might be returned as a tree.*
