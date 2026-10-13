"""Load test for the face-service, the CPU bottleneck of an attendance attempt.

Many students confirm at the start of a step, so this fires N concurrent /verify calls (a real selfie against
an enrolled embedding, anti-spoofing on) at increasing concurrency and reports latency and throughput. Run
inside the compose network: `make loadtest`. Standard library only.
"""
import json
import os
import statistics
import sys
import time
import urllib.error
import urllib.request
import uuid
from concurrent.futures import ThreadPoolExecutor

URL = os.environ.get("FACE_SERVICE_URL", "http://face-service:8001").rstrip("/")
TOKEN = os.environ["FACE_SERVICE_TOKEN"]
LEVELS = [int(x) for x in os.environ.get("LOAD_LEVELS", "1,2,4,8,16,32").split(",")]
PER_LEVEL = int(os.environ.get("LOAD_REQUESTS", "0"))  # 0 = 2 x concurrency
TIMEOUT = float(os.environ.get("FACE_TIMEOUT", "15"))  # what the Go app waits (FACE_TIMEOUT)
DATA = os.environ.get("LOAD_DATA", "/data")


def multipart(fields, files):
    b = uuid.uuid4().hex
    out = []
    for k, v in fields.items():
        out.append(f'--{b}\r\nContent-Disposition: form-data; name="{k}"\r\n\r\n{v}\r\n'.encode())
    for k, (name, data) in files.items():
        out.append(f'--{b}\r\nContent-Disposition: form-data; name="{k}"; filename="{name}"\r\nContent-Type: image/jpeg\r\n\r\n'.encode() + data + b"\r\n")
    out.append(f"--{b}--\r\n".encode())
    return b"".join(out), f"multipart/form-data; boundary={b}"


def call(path, fields, files, timeout):
    body, ctype = multipart(fields, files)
    req = urllib.request.Request(URL + path, data=body, headers={"Content-Type": ctype, "X-Internal-Token": TOKEN})
    with urllib.request.urlopen(req, timeout=timeout) as r:
        return json.load(r)


def read(name):
    with open(os.path.join(DATA, name), "rb") as f:
        return f.read()


def main():
    enrolled = call("/embed", {"anti_spoofing": "false"}, {"image": ("a.jpg", read("face_a2.jpg"))}, 120)
    selfie = read("face_a1.jpg")
    fields = {"embedding": json.dumps(enrolled["embedding"])}
    call("/verify", fields, {"image": ("s.jpg", selfie)}, 120)  # warm-up

    print(f"target {URL}   client timeout {TIMEOUT:g}s (the Go app's FACE_TIMEOUT)\n")
    print(f"{'conc':>4} {'reqs':>5} {'ok':>4} {'fail':>4} {'req/s':>6} {'p50':>6} {'p95':>6} {'max':>6}   (seconds)")
    worst = 0
    for c in LEVELS:
        n = PER_LEVEL or 2 * c
        lat, fail = [], 0

        def one(_):
            t = time.monotonic()
            try:
                r = call("/verify", fields, {"image": ("s.jpg", selfie)}, TIMEOUT)
                ok = r.get("face_found") is True
            except (urllib.error.URLError, TimeoutError, OSError, ValueError):
                ok = False
            return time.monotonic() - t, ok

        t0 = time.monotonic()
        with ThreadPoolExecutor(c) as ex:
            for dt, ok in ex.map(one, range(n)):
                lat.append(dt)
                fail += not ok
        wall = time.monotonic() - t0
        lat.sort()
        p95 = lat[min(len(lat) - 1, int(len(lat) * 0.95))]
        worst = max(worst, fail)
        print(f"{c:>4} {n:>5} {n - fail:>4} {fail:>4} {n / wall:>6.2f} {statistics.median(lat):>6.2f} {p95:>6.2f} {lat[-1]:>6.2f}")
    if worst:
        print("\nFAILURES: requests timed out or errored; lower the concurrency this server can absorb or add CPU.")
        sys.exit(1)


if __name__ == "__main__":
    main()
