"""Thin wrapper around DeepFace. The only module that talks to DeepFace.

The model and detector are loaded once (``Engine.load``) and reused for every request.
"""

from __future__ import annotations

import logging
import math
import threading
from dataclasses import dataclass

import cv2
import numpy as np

# DeepFace pulls in TensorFlow (slow to import). It is imported lazily by Engine.load();
# unit tests assign a fake to this module attribute instead.
DeepFace = None  # type: ignore[assignment]

log = logging.getLogger("face-service")


class FaceError(Exception):
    """A request-level problem that maps to HTTP 422 with a stable ``code``."""

    code = "face_error"

    def __init__(self, message: str = ""):
        super().__init__(message or self.code)


class NoFace(FaceError):
    code = "no_face"


class MultipleFaces(FaceError):
    code = "multiple_faces"


class InvalidImage(FaceError):
    code = "invalid_image"


class InvalidEmbedding(FaceError):
    code = "invalid_embedding"


@dataclass
class EmbedResult:
    embedding: list[float]
    face_count: int
    is_real: bool | None = None
    antispoof_score: float | None = None


@dataclass
class VerifyResult:
    face_found: bool
    is_real: bool
    antispoof_score: float | None
    distance: float | None
    threshold: float
    verified: bool


def decode_image(data: bytes) -> np.ndarray:
    """Decode jpeg/png/webp bytes into a BGR array."""
    if not data:
        raise InvalidImage("empty image")
    arr = np.frombuffer(data, dtype=np.uint8)
    img = cv2.imdecode(arr, cv2.IMREAD_COLOR)
    if img is None:
        raise InvalidImage("cannot decode image")
    return img


def cosine_distance(a: list[float], b: list[float]) -> float:
    va, vb = np.asarray(a, dtype=np.float64), np.asarray(b, dtype=np.float64)
    denom = float(np.linalg.norm(va) * np.linalg.norm(vb))
    if denom == 0 or not math.isfinite(denom):
        return 1.0
    return float(1.0 - np.dot(va, vb) / denom)


def euclidean_l2_distance(a: list[float], b: list[float]) -> float:
    va, vb = np.asarray(a, dtype=np.float64), np.asarray(b, dtype=np.float64)
    na, nb = np.linalg.norm(va), np.linalg.norm(vb)
    if na == 0 or nb == 0:
        return 2.0
    return float(np.linalg.norm(va / na - vb / nb))


DISTANCES = {"cosine": cosine_distance, "euclidean_l2": euclidean_l2_distance}


class Engine:
    def __init__(self, model: str, detector: str, metric: str = "cosine", threshold: float | None = None, antispoofing: bool = True):
        if metric not in DISTANCES:
            raise ValueError(f"unsupported FACE_METRIC {metric!r}; use one of {sorted(DISTANCES)}")
        self.model_name = model
        self.detector = detector
        self.metric = metric
        self.antispoofing = antispoofing  # FACE_ANTISPOOFING=false: /verify skips the liveness check
        self._threshold_override = threshold
        self.threshold: float = threshold if threshold is not None else 0.0
        self.loaded = False
        self._local = threading.local()  # last anti-spoofing score of the current worker thread

    # ---- lifecycle -------------------------------------------------------------------------

    def load(self) -> None:
        """Build the recognition model, the anti-spoofing model and the detector, then warm up."""
        global DeepFace
        if DeepFace is None:
            from deepface import DeepFace as _deepface

            DeepFace = _deepface
        DeepFace.build_model(self.model_name)
        self._hook_antispoof()
        if self._threshold_override is None:
            from deepface.modules import verification

            self.threshold = float(verification.find_threshold(self.model_name, self.metric))
        # One dummy pass so detector weights and TF graphs are initialised before real traffic.
        blank = np.full((224, 224, 3), 127, dtype=np.uint8)
        try:
            DeepFace.represent(
                img_path=blank,
                model_name=self.model_name,
                detector_backend=self.detector,
                enforce_detection=False,
                anti_spoofing=True,
            )
        except Exception as exc:  # a blank frame may be flagged as a spoof; the weights are loaded anyway
            log.info("warm-up finished with: %s", exc)
        self.loaded = True

    def _hook_antispoof(self) -> None:
        """Capture the anti-spoofing score, which DeepFace.represent does not return."""
        try:
            spoof = DeepFace.build_model("Fasnet", task="spoofing")
            original = spoof.analyze
            local = self._local

            def analyze(*args, **kwargs):
                result = original(*args, **kwargs)
                try:
                    local.score = float(result[1])
                except Exception:  # pragma: no cover
                    local.score = None
                return result

            spoof.analyze = analyze
        except Exception as exc:  # pragma: no cover - degrade to "no score", never to "no check"
            log.warning("could not hook the anti-spoofing score: %s", exc)

    # ---- inference -------------------------------------------------------------------------

    def _represent(self, img: np.ndarray, anti_spoofing: bool):
        self._local.score = None
        try:
            return DeepFace.represent(
                img_path=img,
                model_name=self.model_name,
                detector_backend=self.detector,
                enforce_detection=True,
                align=True,
                anti_spoofing=anti_spoofing,
            )
        except ValueError as exc:
            name = type(exc).__name__
            if name == "SpoofDetected":
                raise
            msg = str(exc).lower()
            if name == "FaceNotDetected" or "face could not be detected" in msg or "exception while extracting faces" in msg:
                raise NoFace(str(exc)) from exc
            raise

    def embed(self, data: bytes, anti_spoofing: bool = False) -> EmbedResult:
        img = decode_image(data)
        try:
            faces = self._represent(img, anti_spoofing)
        except ValueError as exc:
            if type(exc).__name__ == "SpoofDetected":
                return EmbedResult([], 1, is_real=False, antispoof_score=getattr(self._local, "score", None))
            raise
        if not faces:
            raise NoFace()
        if len(faces) > 1:
            raise MultipleFaces()
        emb = [float(x) for x in faces[0]["embedding"]]
        score = getattr(self._local, "score", None) if anti_spoofing else None
        return EmbedResult(emb, 1, is_real=True if anti_spoofing else None, antispoof_score=score)

    def verify(self, data: bytes, stored: list[float]) -> VerifyResult:
        """Liveness + identity check of one live capture against a stored embedding."""
        res = self.embed(data, anti_spoofing=self.antispoofing)
        if res.is_real is False:
            return VerifyResult(True, False, res.antispoof_score, None, self.threshold, False)
        if len(stored) != len(res.embedding):
            raise InvalidEmbedding("stored embedding has a different size")
        dist = DISTANCES[self.metric](stored, res.embedding)
        return VerifyResult(True, True, res.antispoof_score, dist, self.threshold, dist <= self.threshold)
