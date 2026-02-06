# COMP3011 | RetailDB Go Service

[![CI](https://github.com/Jacksery/COMP3011-Web-Services/actions/workflows/ci.yml/badge.svg)](https://github.com/Jacksery/COMP3011-Web-Services/actions/workflows/ci.yml)

Lightweight Go web service for the `retailDB.sqlite` dataset (COMP3011 coursework).

## Features

- SQLite-backed service that aggregates across `info`, `brands`, `finance`, `reviews`, and `traffic` tables from `retailDB.sqlite`.
- Admin auth endpoint to generate JWTs
- Endpoints (with OpenAPI spec and Swagger UI):
  - `POST /auth/login` -> {username, password} -> returns `token`
  - `GET /healthz` -> health check
  - `GET /products` -> list (query `limit` and `offset`)
  - `GET /products/:id` -> product aggregate
  - `POST /admin/products` -> create new product (requires Bearer token)
  - `PUT /admin/products/:id` -> update modified\_\* fields (requires Bearer token)
  - `DELETE /admin/products/:id` -> delete product (requires Bearer token)
  - `POST /query` -> AI-powered natural language database query (uses Gemini to generate read-only SQL)

## Requirements

- Go 1.24+
- (Optional) `GEMINI_API_KEY` environment variable for the `/query` endpoint
- `retailDB.sqlite` placed in the project root (or set `RETAILDB_PATH`)

## Quick start

1. Copy or set environment variables (see `.env.example`):

```
ADMIN_USER=admin
ADMIN_PASSWORD=password
JWT_SECRET=your-secret
RETAILDB_PATH=./retailDB.sqlite
PORT=8080
```

2. Run the service locally (dev):

```
go run .
```

3. Login and get a token:

```
POST /auth/login
{ "username": "admin", "password": "password" }
```

4. Use token with `Authorization: Bearer <token>` for `PUT /admin/products/:id`.

---

## Docker deployment (local or VPS) ⚙️

- Build and run with Docker:

```
docker-compose up -d --build
```

Notes:

- Ensure `./data` is mounted as a volume to `/app/data` and `RETAILDB_PATH` points to `/app/data/retailDB.sqlite` to persist data between container restarts. The included `init-db` helper will create `./data/retailDB.sqlite` and set safe permissions if it is missing.
- Set `JWT_SECRET`, `ADMIN_USER`, and `ADMIN_PASSWORD` via environment variables in production.

- Access the service at: `http://localhost:8080/`
  > Public API docs for this VPS: `http://188.245.149.135:8080/docs`

### DB file permissions & compose helper 🔧

If you encounter `readonly database` or `is a directory` errors when starting with `docker compose`, the repository includes a one-shot `init-db` service in `docker-compose.yml` that ensures `./retailDB.sqlite` exists and sets ownership to UID:GID `1000:1000` with permissions `660` before `app` starts.
**Important:** The host `./data` path must be a directory. If `./data` is a file, the init container will print an error and exit. Create the directory like this if needed:

```
rm -f ./data   # only if ./data is a stray file
mkdir -p ./data
```

- Find the runtime `appuser` UID:GID with:

```
docker run --rm --entrypoint id retaildb-service:latest appuser
```

- If the reported UID:GID differs from `1000:1000`, update the `chown` value in `docker-compose.yml` under the `init-db` service.

- Recreate the stack after edits:

```
docker compose down
docker compose up -d --build
```

- Alternative: mount a directory (safer) instead of a single file, for example:

```
# docker-compose.yml
volumes:
  - ./data:/app/data
# env
RETAILDB_PATH=/app/data/retailDB.sqlite
```

Ensure `./data/retailDB.sqlite` exists and is owned by the runtime UID:GID and writable.

If you'd like, I can (A) update `docker-compose.yml` to use `./data` and create the DB there automatically, or (B) add an entrypoint script to the Docker image that creates and chowns the DB at container startup. Which do you prefer?

---

## API docs

- OpenAPI spec available at: `/openapi.yaml`
- Interactive docs (Swagger UI) at: `/docs` (loads `/openapi.yaml`)

---

## Coursework relevance ✅

- **API endpoints & behavior** 🔧
  - Implemented: `GET /products`, `GET /products/:id`, `POST /auth/login`, `PUT /admin/products/:id`, `POST /admin/products`, `DELETE /admin/products/:id`, `GET /healthz`.
  - Pagination support via `limit`/`offset` on `GET /products` and sensible HTTP status codes (200, 201, 400, 401, 404, 409, 500).

- **Database integration** 🗄️
  - Aggregates data across `info`, `brands`, `finance`, `reviews`, and `traffic` using `database/sql` and SQLite driver.
  - Create/update/delete operations are executed in transactions to keep tables consistent.
  - Handles non-numeric values like `None` in numeric columns safely.

- **Authentication & security** 🔐
  - Admin login issues a JWT (HS256). Admin-protected endpoints require `Authorization: Bearer <token>`.
  - Server generates a dev JWT secret on startup if `JWT_SECRET` is not provided and hides it when `ENV=production`.
  - Middleware enforces algorithm checking (rejects unexpected `alg`).

- **AI-powered query endpoint** 🤖
  - `POST /query` accepts a natural-language question, uses the Google Gemini API to generate a read-only SQL SELECT, validates it against a keyword blocklist, and executes it.
  - Multiple safety layers: prompt instructions, keyword validation (`INSERT`/`UPDATE`/`DELETE`/`DROP`/etc. rejected), multi-statement detection, and `SELECT`/`WITH` prefix enforcement.
  - Returns both the generated SQL and the result rows for full transparency.

- **Documentation & API contract** 📄
  - OpenAPI (YAML) at `/openapi.yaml` and interactive Swagger UI at `/docs`.
  - Schemas and example requests/responses included to help testing and marking.

- **What to inspect for grading** 🔎
  - Code: `internal/models/` (DB logic), `internal/handlers/` (HTTP behavior), `internal/auth/` (JWT), `openapi.yaml` (spec).
  - Tests: `internal/*/*_test.go` — they demonstrate correctness and edge cases.

---

## Notes & next steps 💡

- Current implementation focuses on correctness and test coverage for core requirements. Optional enhancements to add:
  - Stronger request validation (field formats, ranges) and improved error messages
  - Additional endpoints (search/filtering, metrics)
  - CI (GitHub Actions) to run tests on push (see `.github/workflows/ci.yml`). The CI now also runs `golangci-lint` to enforce code quality.

  To run the linter locally:

```bash
# install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.0
# run in module
cd cwk1
golangci-lint run ./...
```

  - Swap to RS256 and key rotation for production-grade signing

## Acknowledgements

- Dataset from https://www.kaggle.com/datasets/angelobejaranociotti/retail-db
