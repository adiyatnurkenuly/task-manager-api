
# Task Manager API

A RESTful API for managing tasks, built with Go and PostgreSQL.

## Tech Stack

- Go
- PostgreSQL
- REST API
- JWT Authentication
- bcrypt
- Docker & Docker Compose
- GitHub Actions

## Features

- User registration and login
- JWT-based authentication
- Secure password hashing
- Create, read, update, and delete tasks
- User-specific task management
- Request validation
- JSON error responses
- Automated tests

## Architecture

The project separates HTTP handlers from database operations.

- `cmd/api` - application entry point
- `internal/auth` - authentication and JWT middleware
- `internal/tasks/handler.go` - HTTP request handlers
- `internal/tasks/repository.go` - PostgreSQL data access
- `internal/httpx` - JSON responses and validation
- `migrations` - database schema

Task handlers use a repository interface,
allowing them to be tested without a real database.

## API Endpoints

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | /health | Health check | No |
| POST | /auth/register | Register user | No |
| POST | /auth/login | Login user | No |
| GET | /tasks | List tasks | Yes |
| GET | /tasks/{id} | Get task | Yes |
| POST | /tasks | Create task | Yes |
| PUT | /tasks/{id} | Update task | Yes |
| DELETE | /tasks/{id} | Delete task | Yes |

## Task Filtering and Pagination

GET /tasks supports the following query parameters:

| Parameter | Values | Default |
|-----------|--------|---------|
| status | todo, in_progress, done | All |
| sort | newest, oldest | newest |
| page | 1–100000 | 1 |
| limit | 1–100 | 10 |

Example:

GET /tasks?status=done&sort=newest&page=1&limit=5

The response includes tasks, page, limit, total,
and total_pages.

## Getting Started

### Requirements

- Docker
- Docker Compose

### Run with Docker

1. Clone the repository:

    git clone https://github.com/adiyatnurkenuly/task-manager-api.git

2. Navigate to the project:

    cd task-manager-api

3. Create the environment file:

    Copy-Item .env.example .env

4. Set secure values for POSTGRES_PASSWORD and JWT_SECRET in .env.

5. Start the application:

    docker compose up --build -d

6. Check the API:

    GET http://localhost:8080/health

The initial database schema is created automatically when the PostgreSQL volume is first initialized.

## Authentication

Register and log in using:

- POST /auth/register
- POST /auth/login

Use the returned JWT in protected requests:

    Authorization: Bearer <token>

## Running Tests

Run all tests:

    go test -v ./...

Run static analysis:

    go vet ./...

## CI/CD

GitHub Actions automatically runs formatting checks,
tests, static analysis, and builds on pushes and
pull requests to the main branch.

## Development Notes

- Environment secrets are not committed to Git.
- PostgreSQL data is stored in a Docker volume.
- The API runs locally on port 8080.
- HTTPS and login rate limiting are required before public deployment.

## Author

Adiyat Nurkenuly
