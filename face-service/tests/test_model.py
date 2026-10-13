"""Tests against the real models. Run with `pytest -m model` (needs weights + network on first run).

Photos are looked up in tests/data: face_a1.jpg / face_a2.jpg (same person), face_b.jpg (someone else),
and optionally replay.jpg (a photo of a screen showing a face).
"""

import json
import pathlib

import pytest

from conftest import H

DATA = pathlib.Path(__file__).parent / "data"
pytestmark = pytest.mark.model


def photo(name):
    p = DATA / name
    if not p.exists():
        pytest.skip(f"{name} not available in tests/data")
    return p.read_bytes()


@pytest.fixture(scope="module")
def client():
    from fastapi.testclient import TestClient

    from app.main import app

    with TestClient(app) as c:
        yield c


def embed(client, data):
    r = client.post("/embed", headers=H, files={"image": ("p.jpg", data, "image/jpeg")})
    assert r.status_code == 200, r.text
    return r.json()["embedding"]


def verify(client, stored, data):
    return client.post("/verify", headers=H, files={"image": ("p.jpg", data, "image/jpeg")}, data={"embedding": json.dumps(stored)})


def test_same_person_verifies_and_other_person_does_not(client):
    stored = embed(client, photo("face_a1.jpg"))
    assert len(stored) == 512
    same = verify(client, stored, photo("face_a2.jpg")).json()
    other = verify(client, stored, photo("face_b.jpg")).json()
    assert same["verified"] and same["is_real"], same
    assert not other["verified"], other
    assert same["distance"] < other["distance"]


def test_replayed_photo_is_rejected_by_anti_spoofing(client):
    stored = embed(client, photo("face_a1.jpg"))
    body = verify(client, stored, photo("replay.jpg")).json()
    assert body["is_real"] is False and body["verified"] is False, body
