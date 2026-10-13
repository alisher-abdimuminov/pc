import os

import pytest

os.environ.setdefault("FACE_SERVICE_TOKEN", "test-token")

TOKEN = "test-token"
H = {"X-Internal-Token": TOKEN}


@pytest.fixture(autouse=True)
def _token(monkeypatch):
    monkeypatch.setenv("FACE_SERVICE_TOKEN", TOKEN)


@pytest.fixture
def fake():
    from fakes import FakeDeepFace

    return FakeDeepFace()


@pytest.fixture
def fake_client(fake, monkeypatch):
    """A TestClient running the real lifespan against a fake DeepFace."""
    from fastapi.testclient import TestClient

    import app.engine as engine_module
    from app.engine import Engine
    from app.main import app

    monkeypatch.setattr(engine_module, "DeepFace", fake)
    # skip DeepFace's own threshold table; the engine is given an explicit threshold
    app.state.engine = Engine(model="Facenet512", detector="retinaface", metric="cosine", threshold=0.3)
    with TestClient(app) as client:
        client.fake = fake
        yield client
    del app.state.engine
