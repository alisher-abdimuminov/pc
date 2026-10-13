"""A fake DeepFace so the HTTP layer can be tested without loading any model."""

import io

import cv2
import numpy as np


def png_bytes(color=127, size=64) -> bytes:
    ok, buf = cv2.imencode(".png", np.full((size, size, 3), color, dtype=np.uint8))
    assert ok
    return buf.tobytes()


class SpoofDetected(ValueError):
    pass


class FaceNotDetected(ValueError):
    pass


class FakeSpoofModel:
    def __init__(self, owner):
        self.owner = owner

    def analyze(self, img=None, facial_area=None):
        score = self.owner.spoof_score
        return (score >= 0.5, score)


class FakeDeepFace:
    """Counts model builds and returns scripted faces."""

    def __init__(self):
        self.build_calls = []
        self.represent_calls = 0
        self.faces = 1  # number of faces found by the "detector"
        self.embedding = [1.0, 0.0, 0.0, 0.0]
        self.spoof_score = 0.99
        self.spoof_model = FakeSpoofModel(self)
        self.delay = 0.0
        self.active = 0
        self.max_active = 0
        self._lock = None

    def build_model(self, model_name, task="facial_recognition"):
        self.build_calls.append((model_name, task))
        return self.spoof_model if task == "spoofing" else object()

    def represent(self, img_path, model_name, detector_backend, enforce_detection, anti_spoofing=False, align=True, **kw):
        import threading
        import time

        if self._lock is None:
            self._lock = threading.Lock()
        with self._lock:
            self.represent_calls += 1
            self.active += 1
            self.max_active = max(self.max_active, self.active)
        try:
            if self.delay:
                time.sleep(self.delay)
            if self.faces == 0:
                if enforce_detection:
                    raise FaceNotDetected("Face could not be detected in numpy array.")
                return [{"embedding": self.embedding}]
            if anti_spoofing:
                is_real, _ = self.spoof_model.analyze()
                if not is_real:
                    raise SpoofDetected("Spoof detected in the given image.")
            return [{"embedding": list(self.embedding)} for _ in range(self.faces)]
        finally:
            with self._lock:
                self.active -= 1
