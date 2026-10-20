import asyncio
import json
import logging
import math
import os
import secrets
from contextlib import asynccontextmanager

from fastapi import (
    Depends,
    FastAPI,
    File,
    Form,
    Header,
    HTTPException,
    Request,
    UploadFile,
)
from fastapi.concurrency import run_in_threadpool
from fastapi.responses import JSONResponse

from app.engine import Engine, FaceError, InvalidEmbedding, InvalidImage

log = logging.getLogger("face-service")
MAX_IMAGE_BYTES = 10 * 1024 * 1024


def make_engine() -> Engine:
    threshold = os.getenv("FACE_THRESHOLD", "").strip()
    return Engine(
        model=os.getenv("FACE_MODEL", "Facenet512"),
        detector=os.getenv("FACE_DETECTOR", "retinaface"),
        metric=os.getenv("FACE_METRIC", "cosine"),
        threshold=float(threshold) if threshold else None,
        antispoofing=os.getenv("FACE_ANTISPOOFING", "true").strip().lower() not in ("0", "false", "no", "off"),
    )


def require_token(x_internal_token: str | None = Header(default=None)) -> None:
    expected = os.getenv("FACE_SERVICE_TOKEN", "")
    if (
        not expected
        or not x_internal_token
        or not secrets.compare_digest(x_internal_token, expected)
    ):
        raise HTTPException(status_code=401, detail={"code": "unauthorized"})


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Load the model exactly once, before the first request is served."""
    engine = getattr(app.state, "engine", None) or make_engine()
    app.state.engine = engine
    concurrency = max(1, int(os.getenv("FACE_CONCURRENCY", "2")))
    app.state.semaphore = asyncio.Semaphore(concurrency)
    # requests running or waiting for a slot; /health reports "busy" once this reaches the limit
    app.state.inflight = 0
    app.state.busy_limit = max(
        1, int(os.getenv("FACE_BUSY_THRESHOLD", str(concurrency * 2)))
    )
    await run_in_threadpool(engine.load)
    log.info(
        "model %s / detector %s ready (threshold %.4f)",
        engine.model_name,
        engine.detector,
        engine.threshold,
    )
    yield


app = FastAPI(
    title="face-service", dependencies=[Depends(require_token)], lifespan=lifespan
)


@app.exception_handler(FaceError)
async def face_error_handler(_: Request, exc: FaceError):
    return JSONResponse(
        status_code=422, content={"code": exc.code, "message": str(exc)}
    )


@app.exception_handler(HTTPException)
async def http_error_handler(_: Request, exc: HTTPException):
    detail = (
        exc.detail
        if isinstance(exc.detail, dict)
        else {"code": "error", "message": str(exc.detail)}
    )
    return JSONResponse(status_code=exc.status_code, content=detail)


@asynccontextmanager
async def inference_slot(app: FastAPI):
    """Holds a concurrency slot and counts the request as in flight while it waits and runs."""
    app.state.inflight += 1
    try:
        async with app.state.semaphore:
            yield
    finally:
        app.state.inflight -= 1


@app.get("/health")
async def health(request: Request):
    engine: Engine = request.app.state.engine
    if not engine.loaded:
        raise HTTPException(status_code=503, detail={"code": "loading"})
    state = request.app.state
    return {
        "status": "ok",
        "model": engine.model_name,
        "detector": engine.detector,
        "busy": state.inflight >= state.busy_limit,
        "inflight": state.inflight,
    }


async def read_image(image: UploadFile) -> bytes:
    data = await image.read(MAX_IMAGE_BYTES + 1)
    if len(data) > MAX_IMAGE_BYTES:
        raise InvalidImage("image too large")
    return data


@app.post("/embed")
async def embed(
    request: Request, image: UploadFile = File(...), anti_spoofing: bool = Form(False)
):
    data = await read_image(image)
    engine: Engine = request.app.state.engine
    async with inference_slot(request.app):
        res = await run_in_threadpool(engine.embed, data, anti_spoofing)
    body = {
        "embedding": res.embedding,
        "model": engine.model_name,
        "face_count": res.face_count,
    }
    if anti_spoofing:
        body.update(is_real=res.is_real, antispoof_score=res.antispoof_score)
    return body


@app.post("/verify")
async def verify(
    request: Request, image: UploadFile = File(...), embedding: str = Form(...)
):
    try:
        stored = [float(x) for x in json.loads(embedding)]
    except (ValueError, TypeError) as exc:
        raise InvalidEmbedding("embedding must be a JSON array of numbers") from exc
    if not stored or not all(math.isfinite(x) for x in stored):
        raise InvalidEmbedding("embedding must be a non-empty array of finite numbers")
    data = await read_image(image)
    engine: Engine = request.app.state.engine
    async with inference_slot(request.app):
        res = await run_in_threadpool(engine.verify, data, stored)
    return {
        "face_found": res.face_found,
        "is_real": res.is_real,
        "antispoof_score": res.antispoof_score,
        "distance": res.distance,
        "threshold": res.threshold,
        "verified": res.verified,
    }
