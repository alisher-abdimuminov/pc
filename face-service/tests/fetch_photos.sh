#!/bin/sh
# Downloads the public DeepFace sample photos used by the real-model tests and builds a simulated
# "photo of a phone screen" (replay attack) from one of them. Photos are not committed to the repo.
#   face_a1 / face_a2: the same person   face_b: a different person   replay.jpg: replay of face_a2
set -eu
cd "$(dirname "$0")"
mkdir -p data
base=https://raw.githubusercontent.com/serengil/deepface/master/tests/unit/dataset
curl -fsSL -o data/face_a1.jpg "$base/img11.jpg"
curl -fsSL -o data/face_a2.jpg "$base/img1.jpg"
curl -fsSL -o data/face_b.jpg  "$base/img3.jpg"
cd ..
uv run python tests/make_replay.py tests/data/face_a2.jpg tests/data/replay.jpg
