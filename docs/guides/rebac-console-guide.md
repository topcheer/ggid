# ReBAC Console UI — User Guide

> Feature: F-48 ReBAC (Relationship-Based Access Control) Console
> Location: **Security > ReBAC** (`/security/rebac`)

## What It Does

The ReBAC Console provides a visual interface for managing relationship-based access control (ReBAC). It lets administrators define fine-grained authorization relationships between users, groups, documents, folders, and projects — then query and test those relationships in real time.

ReBAC uses **Google Zanzibar-style** relationship tuples: `(namespace, object, relation, subject)`. For example, `document:report-q4 can_view user:alice` means Alice can view the Q4 report.

## How to Access

1. Log in to the GGID Admin Console.
2. Navigate to **Security** in the sidebar.
3. Click **ReBAC**.

Alternatively, go to `/security/rebac` directly.

## Tabs and Sections

### 1. Playground

The authorization playground lets you test access checks in real time:

- **Namespace**: The resource type (e.g., `document`, `folder`, `project`).
- **Object**: The specific resource identifier (e.g., `report-q4`).
- **Relation**: The permission to check (e.g., `can_view`, `can_edit`, `can_delete`).
- **Subject**: The user or group to check (e.g., `user:alice`, `group:engineering#member`).

**Workflow — Test a user's access:**
1. Go to the Playground tab.
2. Enter namespace `document`, object `report-q4`, relation `can_view`.
3. Enter subject `user:alice`.
4. Click **Check**.
5. The result shows **Allowed** (green) or **Denied** (red) with a reason.

### 2. Tuples

Manage relationship tuples directly:

- **Create**: Add a new tuple (namespace, object, relation, subject).
- **Delete**: Remove an existing tuple.
- **Filter**: Filter by namespace or relation.

Each tuple row shows:
- Namespace and object
- Relation type
- Subject
- Creation timestamp

**Workflow — Grant a user edit access:**
1. Go to the Tuples tab.
2. Click **Add Tuple**.
3. Enter namespace `document`, object `report-q4`, relation `editor`, subject `user:bob`.
4. Submit. Bob can now edit report-q4.

**Workflow — Revoke access:**
1. Find the tuple in the list.
2. Click the trash icon.
3. Confirm deletion.

### 3. Schema

View and edit the ReBAC namespace schema. The schema defines:

- **Namespaces**: Resource types (document, folder, group, etc.).
- **Relations**: How subjects relate to objects (owner, editor, viewer).
- **Permissions**: Computed permissions derived from relations (e.g., `can_view = viewer or editor or owner`).

The page shows a sample schema and lets you submit a new schema definition.

**Example schema:**
```
namespace document {
  relation owner: user
  relation editor: user | group#member
  relation viewer: user | group#member

  permission can_view = viewer or editor or owner
  permission can_edit = editor or owner
  permission can_delete = owner
}
```

### 4. Explorer

The relationship explorer lets you discover all objects or subjects connected through a specific relation:

- **Mode**: `objects` (find all objects a subject can access) or `subjects` (find all subjects with access to an object).
- **Namespace + Relation**: The relationship type to explore.
- **Entity**: The starting subject or object.

**Workflow — Find all documents Alice can view:**
1. Go to the Explorer tab.
2. Set mode to `objects`.
3. Set namespace `document`, relation `can_view`.
4. Enter entity `user:alice`.
5. Click **Expand**. Results list all documents Alice can view.

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/auth/rebac/check` | POST | Check if subject has relation on object |
| `/api/v1/auth/rebac/tuples` | GET | List all tuples |
| `/api/v1/auth/rebac/tuples` | POST | Create a tuple |
| `/api/v1/auth/rebac/tuples/:id` | DELETE | Delete a tuple |
| `/api/v1/auth/rebac/schema` | GET/PUT | Get or update schema |
| `/api/v1/auth/rebac/expand` | POST | Expand relationships |

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Check returns "Denied" unexpectedly | Tuple not created or schema missing permission | Verify tuple exists in Tuples tab; check schema defines the permission |
| Cannot create tuple | Permission denied | Ensure your role has `rebac:write` scope |
| Schema update fails | Syntax error in schema | Validate Zanzibar schema syntax before submitting |
| Explorer returns empty | No relationships exist for this entity | Create tuples first using the Tuples tab |

## Best Practices

- **Start with a clear schema**: Define namespaces and permissions before adding tuples.
- **Use group memberships**: Grant access to groups (`group:engineering#member`) rather than individual users.
- **Test in Playground first**: Always verify access changes using the Playground before relying on them.
- **Audit tuples regularly**: Use the Tuples filter to find and remove stale relationships.
