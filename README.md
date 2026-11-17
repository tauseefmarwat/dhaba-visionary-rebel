# Dhaba Orders

A seven-service restaurant ordering demo for the Visionary Rebels ShipIt Sprint 3 project.

## Services
- `web`: React + Vite customer ordering, kitchen board and live statistics UI.
- `menu-api`: FastAPI menu service with Redis caching.
- `orders-api`: Go order service; PostgreSQL persistence and Redis queue publishing.
- `kitchen-worker`: Spring Boot worker; consumes Redis jobs and advances orders `new -> cooking -> ready`.
- `stats-api`: Laravel API for live statistics stored in Redis.
- `postgres`: PostgreSQL schema and seed data in `database/`.
- `redis`: queue, menu cache, statistics and worker heartbeat.

The application uses PKR as its demo currency. Prices and menu data are editable in `database/seed.sql`.

## Requirements
- Node.js 20+
- Python 3.11+
- Go 1.22+
- Java 21 and Maven 3.9+
- PHP 8.2+, Composer 2, Laravel 11/12-compatible dependencies
- PostgreSQL 16+
- Redis 7+

## Local setup
Start PostgreSQL and Redis locally (or use your team's approved development instances). Create a database and role (or use equivalent credentials from your trainer):

```sql
CREATE USER dhaba WITH PASSWORD 'dhaba_dev_password';
CREATE DATABASE dhaba_orders OWNER dhaba;
```

Then run the schema and seed commands:

```bash
psql "$DATABASE_URL" -f database/schema.sql
psql "$DATABASE_URL" -f database/seed.sql
```

Use these development environment variables (adjust credentials/hosts):

```bash
export DATABASE_URL='postgres://dhaba:dhaba_dev_password@localhost:5432/dhaba_orders?sslmode=disable'
export DB_HOST=localhost DB_PORT=5432 DB_DATABASE=dhaba_orders DB_USERNAME=dhaba DB_PASSWORD=dhaba_dev_password
export REDIS_URL=redis://localhost:6379/0
export REDIS_ADDR=localhost:6379
export REDIS_HOST=localhost REDIS_PORT=6379 REDIS_PASSWORD=
export APP_VERSION=local
```

### Start each service in a separate terminal

```bash
# menu-api
cd menu-api && python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt && uvicorn app.main:app --reload --port 8001

# orders-api
cd orders-api && go mod download && go run ./cmd/server

# kitchen-worker
cd kitchen-worker && mvn spring-boot:run

# stats-api
cd stats-api && composer install && cp .env.example .env
php artisan key:generate && php artisan serve --host 0.0.0.0 --port 8004

# web
cd web && npm install && npm run dev
```

Open `http://localhost:5173`. Vite proxies `/api/menu` to 8001, `/api/orders` and `/info` to 8002, `/api/stats` to 8004, and `/health` to the relevant service. The Vite proxy is for local development; Kubernetes ingress should route the same paths as documented in `docs/api-contracts.md`.

## Environment variables

| Service | Variables |
|---|---|
| menu-api | `DB_HOST`, `DB_PORT`, `DB_DATABASE`, `DB_USERNAME`, `DB_PASSWORD`, `REDIS_URL`, `APP_VERSION` |
| orders-api | `DATABASE_URL`, `REDIS_ADDR` (host:port), `REDIS_PASSWORD`, `APP_VERSION`, `POD_NAME`, `PORT` |
| kitchen-worker | `DB_HOST`, `DB_PORT`, `DB_DATABASE`, `DB_USERNAME`, `DB_PASSWORD`, `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `APP_VERSION`, `WORKER_ID` |
| stats-api | Laravel `DB_*`, `REDIS_*`, `APP_KEY`, `APP_DEBUG=false` |
| web | `VITE_*` variables are optional; browser paths are same-host `/api/...` |

## Order flow
1. React fetches menu from `GET /api/menu`.
2. Customer submits `{customer_name, items:[{menu_item_id, quantity}]}` to `POST /api/orders`.
3. Go validates current menu prices, stores order and items in PostgreSQL, updates Redis statistics and queues the order ID.
4. Java worker consumes the order ID, updates it to `cooking`, then `ready`.
5. React polls `GET /api/orders/{id}`; Laravel reads Redis statistics at `GET /api/stats`.

## Tests
- `cd menu-api && pytest`
- `cd orders-api && go test ./...`
- `cd kitchen-worker && mvn test`
- `cd stats-api && php artisan test` (after Composer install)
- `cd web && npm test`

The tests are starter-level unit tests. Run the integration workflow with PostgreSQL and Redis to validate cross-service behavior. This project has not been certified as production-ready; secrets, ingress, TLS, resource limits, image builds, Kubernetes storage and deployment controls belong to the team's DevOps implementation.

## Important operational notes
- Do not commit real passwords, kubeconfigs, cloud credentials or Slack webhooks.
- PostgreSQL is the source of truth; Redis is used for cache, queue and derived statistics.
- The worker moves queue messages into a Redis processing list before updating PostgreSQL, then acknowledges after completion. A worker crash leaves the message visible in the processing list rather than deleting it; implement a stale-processing reaper/dead-letter policy before treating this as production-grade.
- Configure database credentials and CORS through environment variables in each environment.
