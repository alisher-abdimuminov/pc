
from conftest import H


def test_health_requires_token(fake_client):
    assert fake_client.get("/health").status_code == 401
    assert fake_client.get("/health", headers={"X-Internal-Token": "nope"}).status_code == 401


def test_health_ok_after_load(fake_client):
    r = fake_client.get("/health", headers=H)
    assert r.status_code == 200
    assert r.json() == {
        "status": "ok",
        "model": "Facenet512",
        "detector": "retinaface",
        "busy": False,
        "inflight": 0,
    }


def test_health_reports_busy_when_the_queue_is_full(fake_client):
    state = fake_client.app.state
    state.inflight = state.busy_limit - 1
    assert fake_client.get("/health", headers=H).json()["busy"] is False
    state.inflight = state.busy_limit
    body = fake_client.get("/health", headers=H).json()
    assert body["busy"] is True and body["inflight"] == state.busy_limit
