
from conftest import H


def test_health_requires_token(fake_client):
    assert fake_client.get("/health").status_code == 401
    assert fake_client.get("/health", headers={"X-Internal-Token": "nope"}).status_code == 401


def test_health_ok_after_load(fake_client):
    r = fake_client.get("/health", headers=H)
    assert r.status_code == 200
    assert r.json() == {"status": "ok", "model": "Facenet512", "detector": "retinaface"}
