# Cloud-Report

A jsreport-compatible reporting backend written in Go, with Python microservices
for the heavy formats (PDF via WeasyPrint, DOCX via docxtpl, PPTX via
python-pptx, HTML→XLSX via pandas). Storage is SeaweedFS (S3 API), the queue
is Redis Streams, and metadata lives in Postgres.

## Architecture

```
Nuxt UI (future)
    │
    ▼
┌─────────────────────┐
│  Go API (Fiber)     │  CRUD + OData + render pipeline + Handlebars + XLSX (excelize) + PDF utils (pdfcpu)
└─┬─────────────┬─────┘
  │             │
  │             ▼
  │     ┌──────────────┐
  │     │    Redis     │  request/reply streams: renders:<recipe>
  │     └──┬───────────┘
  │        │
  │   ┌────┼──────────────┬──────────────┬─────────────────┐
  │   ▼    ▼              ▼              ▼                 ▼
  │  weasyprint        docx (docxtpl)  pptx (python-pptx) html-to-xlsx (pandas)
  │
  ▼
PostgreSQL + SeaweedFS (S3)
```

## Stack

| Layer              | Tech                                      |
|--------------------|-------------------------------------------|
| HTTP server        | Go 1.22 + Fiber v2                        |
| DB                 | PostgreSQL 16                             |
| Object storage     | **SeaweedFS** (S3-compatible API)         |
| Queue / RPC        | Redis 7 (Streams)                         |
| Auth               | JWT (HS256) + API keys (`x-api-key`)      |
| Templating         | Handlebars (raymond)                      |
| In-process recipes | `excelize` (XLSX), `pdfcpu` (PDF utils)   |
| Worker recipes     | WeasyPrint, docxtpl, python-pptx, pandas  |

## Quick start

```bash
git clone <repo>
cd cloud-report
cp .env.example .env

docker compose up --build
```

The API listens on `http://localhost:5488`.

## Authentication flow

1. **Register the first user** (becomes admin automatically):
   ```bash
   curl -X POST http://localhost:5488/api/auth/register \
        -H 'content-type: application/json' \
        -d '{"username":"admin","password":"secret123","email":"a@b.c"}'
   ```
   Response includes a JWT token (12h TTL).

2. **Login**:
   ```bash
   curl -X POST http://localhost:5488/api/auth/login \
        -H 'content-type: application/json' \
        -d '{"username":"admin","password":"secret123"}'
   ```

3. **Create an API key** (uses the JWT from step 2):
   ```bash
   curl -X POST http://localhost:5488/api/apikeys \
        -H "Authorization: Bearer $JWT" \
        -H 'content-type: application/json' \
        -d '{"name":"prod","ttlDays":90}'
   # → returns { "key": "cr_ab12cd34_<40 hex chars>" } – save this!
   ```

4. **Use the API key**:
   ```bash
   curl -X POST http://localhost:5488/api/report \
        -H "x-api-key: cr_ab12cd34_..." \
        -H 'content-type: application/json' \
        -d '{
              "template": {
                "content": "<h1>Hi {{name}}</h1>",
                "engine":  "handlebars",
                "recipe":  "weasyprint"
              },
              "data": {"name": "World"}
            }' \
        --output report.pdf
   ```

The API key is hashed (SHA-256) before storage; the raw value is shown
exactly once. Keys can be scoped (`render`, `read`, `write`), given an
expiry, or revoked.

## Endpoints (highlights)

| Method | Path                                  | Purpose                              |
|--------|---------------------------------------|--------------------------------------|
| GET    | `/api/ping`                           | Health check                         |
| GET    | `/api/version`                        | Server version                       |
| POST   | `/api/auth/register`                  | Create user (+ JWT)                  |
| POST   | `/api/auth/login`                     | Login → JWT cookie + body            |
| POST   | `/api/apikeys`                        | Create an API key                    |
| GET    | `/api/apikeys`                        | List the caller's keys               |
| DELETE | `/api/apikeys/:id`                    | Revoke a key                         |
| GET    | `/api/recipe`                         | Recipes available                    |
| GET    | `/api/engine`                         | Engines available                    |
| GET    | `/api/extensions`                     | Loaded modules                       |
| GET    | `/api/schema/:set`                    | JSON Schema for an entity            |
| POST   | `/api/report`                         | Render a template                    |
| GET    | `/api/profile/:id`                    | Profile info for a render            |
| GET    | `/api/profile/:id/events`             | Profile event log                    |
| GET    | `/reports/:shortid/content`           | Download a stored report             |
| GET    | `/reports/public/:shortid/content`    | Public report download (no auth)     |
| GET    | `/assets/:shortid/content`            | Asset binary                         |
| GET    | `/api/scheduling/nextRun/:cron`       | Compute next cron fire time          |
| POST   | `/api/scheduling/runNow`              | Trigger a schedule immediately       |
| POST   | `/api/version-control/commit`         | Snapshot all entities to ZIP         |
| GET    | `/api/version-control/history`        | List snapshots                       |
| POST   | `/api/version-control/revert`         | Restore a snapshot (upsert)          |
| POST   | `/api/export`                         | Download a ZIP of everything         |
| POST   | `/api/import`                         | Import a ZIP                         |
| POST   | `/api/assets/upload`                  | Multipart upload to SeaweedFS        |
| POST   | `/studio/hierarchyMove`               | Move entity to another folder        |
| POST   | `/studio/validate-entity-name`        | Check name uniqueness in folder      |
| GET    | `/studio/text-search?q=...`           | Full-text search across entities     |
| GET    | `/api/kv/:key`                        | Read a settings value                |
| PUT    | `/api/kv/:key`                        | Write a settings value               |
| GET    | `/api/health`                         | Detailed health (pg, redis, s3)      |
| GET    | `/ws`                                 | WebSocket stream of entity events    |
| GET/POST/PATCH/DELETE | `/odata/:entitySet`    | CRUD on entities (templates, etc.)   |

OData query options: `$filter`, `$select`, `$top`, `$skip`, `$orderby`,
`$count`, `$inlinecount`. Entity sets: `templates`, `assets`, `scripts`,
`data`, `folders`, `components`, `reports`, `schedules`, `profiles`,
`settings`, `versions`, `tags`.

## Recipes

| Recipe         | Where it runs       | Backed by             |
|----------------|---------------------|-----------------------|
| `html`         | Go (in-process)     | engine output         |
| `text`         | Go (in-process)     | engine output         |
| `xlsx`         | Go (in-process)     | `excelize`            |
| `static-pdf`   | Go (in-process)     | `pdfcpu` passthrough  |
| `weasyprint`   | Python worker       | WeasyPrint            |
| `chrome-pdf`   | Python worker       | WeasyPrint (alias)    |
| `docx`         | Python worker       | docxtpl               |
| `pptx`         | Python worker       | python-pptx + Jinja2  |
| `html-to-xlsx` | Python worker       | pandas + openpyxl     |

PDF post-processing (merge / watermark / encrypt) runs through `pdfcpu`
on top of any PDF-producing recipe via the template's `pdfOperations` array.

### Scripts, helpers, components

- **Helpers** — `template.helpers` is a JavaScript source string. Functions
  declared at top level become Handlebars helpers in the same render.
  Executed in a goja VM with a 5s timeout, no network, no fs.

  ```js
  function upper(s) { return String(s).toUpperCase(); }
  // → template can do {{upper name}}
  ```

- **Scripts** — `template.scripts` is a JSON array of `{shortid}` or
  `{content}` entries. Each entry can export `beforeRender(req, res)` and
  mutate `req.data` to feed the template. Same sandbox guarantees.

  ```js
  function beforeRender(req, res) {
    req.data.now = new Date().toISOString();
  }
  ```

- **Components** — `{{> componentName}}` in a template is replaced with the
  body of the component (looked up by `name` in the `components` table)
  before rendering, so the engine sees one flat template.

## Development

```bash
# Run only the data plane locally:
docker compose up postgres redis seaweedfs-master seaweedfs-volume seaweedfs-filer seaweedfs-s3

# Then run the Go API on the host:
cd api
go run ./cmd/server
```

Migrations are embedded in the binary (`internal/store/migrations.sql`) and
applied on startup. To wipe everything:

```bash
docker compose down -v
```

## Folder layout

```
.
├── api/                  # Go monolith
│   ├── cmd/server/       # main()
│   ├── internal/
│   │   ├── api/          # HTTP handlers
│   │   ├── auth/         # JWT + API key
│   │   ├── blob/         # SeaweedFS S3 client
│   │   ├── config/
│   │   ├── models/       # entity structs
│   │   ├── odata/        # parser + generic CRUD
│   │   ├── queue/        # Redis Streams
│   │   ├── render/       # orchestrator + recipes + engines
│   │   ├── scheduler/    # cron
│   │   ├── server/       # Fiber setup
│   │   └── store/        # Postgres
│   ├── migrations/
│   └── Dockerfile
├── workers/
│   ├── weasyprint/
│   ├── docx/
│   ├── pptx/
│   ├── html-to-xlsx/
│   └── shared/protocol.py
├── infra/seaweed/s3.json
├── docker-compose.yml
└── .env.example
```
