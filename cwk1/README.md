# COMP3011 | RetailDB Go Service

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

## Requirements

- Go 1.20+
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

2. Run the service:

```
go run .
```

3. Login and get a token:

```
POST /auth/login
{ "username": "admin", "password": "password" }
```

4. Use token with `Authorization: Bearer <token>` for `PUT /admin/products/:id`.

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
  - CI (GitHub Actions) to run tests on push
  - Swap to RS256 and key rotation for production-grade signing

## Acknowledgements

- Dataset from https://www.kaggle.com/datasets/angelobejaranociotti/retail-db
