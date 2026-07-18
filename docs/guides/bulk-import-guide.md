# Bulk Import Guide (KB-060)

## Overview

GGID's async bulk import pipeline processes CSV/JSON user imports in background workers, with dry-run validation, error reporting, and progress tracking.

## Import Flow

```
Upload CSV/JSON → Create import_job (status=pending) → 
  Worker picks up → Parse rows → Validate → Dry-run? → 
    Yes: return preview, no changes
    No:  Create users in batches → Update job progress → 
  Job complete (status=completed/failed)
```

## CSV Format

```csv
email,first_name,last_name,department,role
alice@company.com,Alice,Chen,Engineering,engineer
bob@company.com,Bob,Smith,Sales,rep
```

### Required Fields
| Column | Required | Description |
|--------|----------|-------------|
| email | Yes | Unique user email |
| first_name | Yes | Given name |
| last_name | Yes | Family name |
| department | No | Department name |
| role | No | Role assignment |

## API Usage

### Create Import Job
```http
POST /api/v1/identity/import
Content-Type: multipart/form-data

file: users.csv
dry_run: true
```

Response:
```json
{
  "job_id": "job-abc123",
  "status": "pending",
  "total_rows": 247,
  "valid_rows": 241,
  "invalid_rows": 6,
  "preview": [...]
}
```

### Check Job Status
```http
GET /api/v1/identity/import/jobs/{job_id}
```

```json
{
  "job_id": "job-abc123",
  "status": "completed",
  "progress": 100,
  "imported": 241,
  "failed": 6,
  "errors": [
    {"row": 42, "email": "invalid", "error": "invalid email format"},
    {"row": 89, "error": "duplicate email"}
  ]
}
```

### List All Jobs
```http
GET /api/v1/identity/import/jobs?status=completed&page=1
```

## Dry-Run Mode

Set `dry_run=true` to validate without creating users. Returns:
- Row-by-row validation results
- Error count and details
- No database changes

## Error Troubleshooting

| Error | Cause | Fix |
|-------|-------|-----|
| `invalid email format` | Missing @ or bad domain | Fix CSV data |
| `duplicate email` | Email already exists | Skip or use update mode |
| `missing required field` | Empty first_name/last_name | Fill in CSV |
| `role not found` | Invalid role name | Check role exists first |
| `job timeout` | File >10,000 rows | Split into batches |

## Best Practices

- **Always dry-run first** — validate before committing
- **Batch size**: 500 rows per batch for optimal performance
- **Monitor progress** — poll job status every 5s for large imports
- **Error tolerance**: job continues on row errors, fails only on system errors
