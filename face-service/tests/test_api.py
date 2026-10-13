import json
from concurrent.futures import ThreadPoolExecutor

from conftest import H
from fakes import png_bytes


def post_embed(client, anti_spoofing=False, data=None):
    return client.post(
        "/embed", headers=H, files={"image": ("a.png", data if data is not None else png_bytes(), "image/png")},
        data={"anti_spoofing": str(anti_spoofing).lower()},
    )


def post_verify(client, stored, data=None):
    return client.post(
        "/verify", headers=H, files={"image": ("a.png", data if data is not None else png_bytes(), "image/png")},
        data={"embedding": json.dumps(stored)},
    )


# ---- model lifecycle ----

def test_model_is_built_exactly_once_across_requests(fake_client):
    fake = fake_client.fake
    assert fake.build_calls.count(("Facenet512", "facial_recognition")) == 1  # at startup
    for _ in range(5):
        assert post_embed(fake_client).status_code == 200
        assert post_verify(fake_client, [1, 0, 0, 0]).status_code == 200
        assert fake_client.get("/health", headers=H).status_code == 200
    recognition = [c for c in fake.build_calls if c[1] == "facial_recognition"]
    assert recognition == [("Facenet512", "facial_recognition")], "the model must not be rebuilt per request"
    assert fake.represent_calls == 1 + 10  # one warm-up + the requests above


def test_health_is_unavailable_until_loaded(fake, monkeypatch):
    from fastapi.testclient import TestClient

    import app.engine as engine_module
    from app.engine import Engine
    from app.main import app

    monkeypatch.setattr(engine_module, "DeepFace", fake)
    app.state.engine = Engine("Facenet512", "retinaface", threshold=0.3)  # not loaded: no lifespan
    client = TestClient(app)  # no context manager -> lifespan not run
    assert client.get("/health", headers=H).status_code == 503
    del app.state.engine


# ---- auth ----

def test_every_endpoint_requires_the_token(fake_client):
    files = {"image": ("a.png", png_bytes(), "image/png")}
    for headers in ({}, {"X-Internal-Token": "wrong"}):
        assert fake_client.post("/embed", files=files, headers=headers).status_code == 401
        assert fake_client.post("/verify", files=files, data={"embedding": "[1]"}, headers=headers).status_code == 401
    assert fake_client.get("/health").status_code == 401


# ---- /embed ----

def test_embed_returns_vector(fake_client):
    r = post_embed(fake_client)
    assert r.status_code == 200
    assert r.json() == {"embedding": [1.0, 0.0, 0.0, 0.0], "model": "Facenet512", "face_count": 1}


def test_embed_without_anti_spoofing_ignores_liveness(fake_client):
    fake_client.fake.spoof_score = 0.01  # would be a spoof
    assert post_embed(fake_client, anti_spoofing=False).status_code == 200


def test_embed_no_face(fake_client):
    fake_client.fake.faces = 0
    r = post_embed(fake_client)
    assert r.status_code == 422 and r.json()["code"] == "no_face"


def test_embed_multiple_faces(fake_client):
    fake_client.fake.faces = 2
    r = post_embed(fake_client)
    assert r.status_code == 422 and r.json()["code"] == "multiple_faces"


def test_invalid_image(fake_client):
    r = post_embed(fake_client, data=b"definitely not an image")
    assert r.status_code == 422 and r.json()["code"] == "invalid_image"
    r = post_embed(fake_client, data=b"")
    assert r.status_code == 422 and r.json()["code"] == "invalid_image"


# ---- /verify ----

def test_verify_match(fake_client):
    r = post_verify(fake_client, [1, 0, 0, 0])
    body = r.json()
    assert r.status_code == 200
    assert body["face_found"] and body["is_real"] and body["verified"]
    assert body["distance"] < 1e-9 and body["threshold"] == 0.3 and body["antispoof_score"] == 0.99


def test_verify_mismatch(fake_client):
    r = post_verify(fake_client, [0, 1, 0, 0])  # orthogonal vectors: cosine distance 1
    body = r.json()
    assert r.status_code == 200 and not body["verified"] and body["is_real"]
    assert abs(body["distance"] - 1.0) < 1e-9


def test_verify_spoof_is_rejected_without_a_distance(fake_client):
    fake_client.fake.spoof_score = 0.02
    body = post_verify(fake_client, [1, 0, 0, 0]).json()
    assert body["face_found"] and body["is_real"] is False and body["verified"] is False
    assert body["distance"] is None and body["antispoof_score"] == 0.02


def test_verify_always_runs_anti_spoofing(fake_client):
    # an image DeepFace would call a spoof must never verify, regardless of how close the embedding is
    fake_client.fake.spoof_score = 0.1
    assert post_verify(fake_client, [1, 0, 0, 0]).json()["verified"] is False


def test_verify_no_face_and_multiple_faces(fake_client):
    fake_client.fake.faces = 0
    r = post_verify(fake_client, [1, 0, 0, 0])
    assert r.status_code == 422 and r.json()["code"] == "no_face"
    fake_client.fake.faces = 3
    r = post_verify(fake_client, [1, 0, 0, 0])
    assert r.status_code == 422 and r.json()["code"] == "multiple_faces"


def test_verify_rejects_bad_embeddings(fake_client):
    for bad in ["not json", "[]", '["a"]', "[1,2]"]:  # last: wrong dimension
        r = fake_client.post("/verify", headers=H, files={"image": ("a.png", png_bytes(), "image/png")}, data={"embedding": bad})
        assert r.status_code == 422, bad
        assert r.json()["code"] == "invalid_embedding", bad


# ---- concurrency ----

def test_concurrency_is_limited_by_the_semaphore(fake, monkeypatch):
    from fastapi.testclient import TestClient

    import app.engine as engine_module
    from app.engine import Engine
    from app.main import app

    monkeypatch.setenv("FACE_CONCURRENCY", "2")
    monkeypatch.setattr(engine_module, "DeepFace", fake)
    app.state.engine = Engine("Facenet512", "retinaface", threshold=0.3)
    with TestClient(app) as client:
        fake.delay = 0.15
        fake.max_active = 0
        with ThreadPoolExecutor(8) as pool:
            codes = list(pool.map(lambda _: post_embed(client).status_code, range(8)))
    del app.state.engine
    assert codes == [200] * 8
    assert 1 <= fake.max_active <= 2, f"{fake.max_active} inferences ran at once"
