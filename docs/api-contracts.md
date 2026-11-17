# API contracts

All browser endpoints are exposed under `/api/...` on the same host by the ingress/reverse proxy.

## Menu API (FastAPI)
- `GET /api/menu` -> `{items:[{id,category_id,category,name,description,price,currency,image_url,available}]}`
- `GET /api/menu/{id}` -> one menu item; 404 if absent.
- `GET /health` -> service health.

## Orders API (Go)
- `POST /api/orders` body: `{ "customer_name":"Ayesha", "items":[{"menu_item_id":1,"quantity":2}] }`.
- Response 201: `{order:{id,order_number,customer_name,total_amount,currency,status,created_at,items:[...]}}`.
- `GET /api/orders` -> `{orders:[...]}` (latest first; kitchen board).
- `GET /api/orders/{id}` -> `{order:{...}}`.
- `GET /info` -> `{service,version,pod}`.
- `GET /health` -> health status.

Order status values: `new`, `cooking`, `ready`. Prices are always recalculated by the server from PostgreSQL; client-submitted prices are ignored.

## Stats API (Laravel)
- `GET /api/stats` -> `{orders_per_minute,top_dishes:[{dish,quantity}],updated_at}`.
- `GET /api/health` (also `/health` from Laravel bootstrap) -> health status.

## Queue contract
Redis list `dhaba:orders:queue` contains JSON objects `{ "order_id": 123 }`. The worker acknowledges by removing an item via `BRPOP`. The worker updates PostgreSQL status and refreshes `dhaba:worker:heartbeat` every 10 seconds.

## Redis statistics
- `dhaba:stats:orders:<YYYYMMDDHHmm>`: integer count for that minute.
- `dhaba:stats:dishes`: sorted set; member is dish name and score is ordered quantity.
- `dhaba:worker:heartbeat`: timestamp string.
