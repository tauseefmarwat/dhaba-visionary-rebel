import json, os
from decimal import Decimal
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
import psycopg
from psycopg.rows import dict_row
from redis import Redis

app = FastAPI(title="Dhaba Orders Menu API", version=os.getenv("APP_VERSION", "dev"))
app.add_middleware(CORSMiddleware, allow_origins=os.getenv("CORS_ORIGINS", "*").split(","), allow_methods=["GET"], allow_headers=["*"])
REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379/0")

def db_conn():
    return psycopg.connect(host=os.getenv("DB_HOST", "localhost"), port=os.getenv("DB_PORT", "5432"), dbname=os.getenv("DB_DATABASE", "dhaba_orders"), user=os.getenv("DB_USERNAME", "dhaba"), password=os.getenv("DB_PASSWORD", "dhaba_dev_password"), row_factory=dict_row)

def serialize(row):
    row["price"] = float(row["price"])
    return row

@app.get("/health")
def health():
    try:
        with db_conn() as conn: conn.execute("SELECT 1")
        Redis.from_url(REDIS_URL).ping()
        return {"status":"ok", "service":"menu-api"}
    except Exception as exc:
        raise HTTPException(status_code=503, detail="dependency unavailable") from exc

@app.get("/api/menu")
def get_menu():
    redis = Redis.from_url(REDIS_URL, decode_responses=True)
    try:
        cached = redis.get("dhaba:menu:v1")
        if cached: return {"items": json.loads(cached)}
    except Exception: pass
    with db_conn() as conn:
        rows = conn.execute("""SELECT m.id,m.category_id,c.name AS category,m.name,m.description,m.price,m.currency,m.image_url,m.available
          FROM menu_items m JOIN categories c ON c.id=m.category_id WHERE m.available=TRUE ORDER BY c.name,m.name""").fetchall()
    items = [serialize(dict(r)) for r in rows]
    try: redis.setex("dhaba:menu:v1", 60, json.dumps(items))
    except Exception: pass
    return {"items":items}

@app.get("/api/menu/{item_id}")
def get_menu_item(item_id: int):
    with db_conn() as conn:
        row = conn.execute("""SELECT m.id,m.category_id,c.name AS category,m.name,m.description,m.price,m.currency,m.image_url,m.available
          FROM menu_items m JOIN categories c ON c.id=m.category_id WHERE m.id=%s AND m.available=TRUE""", (item_id,)).fetchone()
    if not row: raise HTTPException(status_code=404, detail="Menu item not found")
    return {"item":serialize(dict(row))}
