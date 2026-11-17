from app.main import app
from fastapi.testclient import TestClient

def test_health_route_exists():
    assert any(getattr(route, "path", None) == "/health" for route in app.routes)

def test_menu_route_exists():
    assert any(getattr(route, "path", None) == "/api/menu" for route in app.routes)
