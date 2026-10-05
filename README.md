# applypilot-backend

Go API for ApplyPilot, a job application copilot. It handles auth, the profile vault, resumes, the job tracker, and the Claude-powered AI features used by the dashboard and the Chrome extension.

Related repos:

- [`applypilot-migrations`](../applypilot-migrations): Postgres schema (Soda).
- [`applypilot-web`](../applypilot-web): Next.js dashboard and the Chrome extension.

## Layout

```
cmd/api/main.go              entrypoint
internal/
  server/server.go           wires apps together, builds the router
  apps/                      one package per feature
    auth/                    signup, login, /me
    profile/                 profile vault (JSONB)
    resume/                  resume upload/download/default
    job/                     job tracker + stats
    ai/                      Claude features + generation log/quota
      handler.go             HTTP: decode → validate → service → respond
      request.go             request DTOs + Validate()
      resource.go            response DTOs (the JSON the clients see)
      service.go             business logic, returns apperr errors
      repository.go          SQL + persisted model
      prompts.go             (ai only) prompts and tool schemas
  platform/                  shared infrastructure
    config/  database/  apperr/  httpx/  middleware/  security/  claude/
```

**Conventions**

- **Layering:** the handler talks only to the service, and the service only to its repository. The repository returns package-private errors, and the service maps them to `apperr` kinds.
- **Cross-app dependencies:** these go through small interfaces defined by the consumer. For example, `ai.JobStore` is satisfied by `job.Service`, and `auth.ProfileSeeder` by `profile.Service`. `internal/server` does the wiring.
- **Response shapes:** `resource.go` defines the API contract. The web app and the extension depend on these JSON shapes.

## Run

```bash
# 1. Schema (in applypilot-migrations)
make up

# 2. API
cp .env.example .env   # set JWT_SECRET and ANTHROPIC_API_KEY
make run
```

Or run everything with Docker: `make docker` starts Postgres, runs the Soda migrations, then starts the API. It needs `applypilot-migrations` cloned next to this repo.

The API refuses to start if the schema is missing.

## Adding an app

1. Create `internal/apps/<name>/` with `repository.go`, `service.go`, `request.go`, `resource.go` and `handler.go`.
2. Add a `Routes(mux, authed)` method on the handler.
3. Construct it and register its routes in `internal/server/server.go`.
4. Add the tables it needs as a new migration in `applypilot-migrations`.

## API

Auth: `POST /api/auth/signup`, `POST /api/auth/login` → `{token, expires_at, user}`. Every other route needs `Authorization: Bearer <token>`. Errors look like `{"error": "..."}`.

| App | Routes |
|---|---|
| auth | `GET /api/me` |
| profile | `GET/PUT /api/profile` |
| resume | `GET/POST /api/resumes`, `GET /api/resumes/{id\|default}/file`, `POST /api/resumes/{id}/default`, `DELETE /api/resumes/{id}` |
| job | `GET/POST /api/jobs` (POST upserts by URL), `GET /api/jobs/lookup?url=`, `GET/PATCH/DELETE /api/jobs/{id}`, `GET /api/stats` |
| ai | `POST /api/ai/{parse-resume,match,cover-letter,tailor-resume,answer,map-fields,extract-job}`, `GET /api/ai/usage`, `GET /api/ai/generations?job_id=`, `GET /api/ai/generations/{id}/resume.pdf` |

## Claude usage

- **Models:** `CLAUDE_MODEL_SMART` handles resumes, fit scores, cover letters, tailoring and answers. `CLAUDE_MODEL_FAST` handles form-field mapping and job extraction.
- **Structured output:** these calls force a `tool_choice`, so the JSON always matches the schema.
- **Prompt caching:** the user's profile is sent as its own cached system block.
- **Usage logging:** every call is logged in `ai_generations` with its token counts, and `AI_DAILY_LIMIT` caps calls per user.
- **Truthful output:** the prompts forbid inventing experience, and select/radio answers are checked against the form's real options.
- **Resume PDF:** `tailor-resume` returns a `generation_id`; fetching `GET /api/ai/generations/{id}/resume.pdf` renders that result as a downloadable PDF (`internal/platform/pdfgen`), merging it with the profile's original dates, locations and education (the AI call itself doesn't touch those).
