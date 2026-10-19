FROM python:3.11-slim

COPY --from=ghcr.io/astral-sh/uv:0.11 /uv /uvx /bin/

# System libraries required by OpenCV (used by DeepFace from phase 5).
RUN apt-get update \
	&& apt-get install -y --no-install-recommends libgl1 libglib2.0-0 \
	&& rm -rf /var/lib/apt/lists/*

ENV UV_COMPILE_BYTECODE=1 UV_LINK_MODE=copy PYTHONUNBUFFERED=1 \
	TF_CPP_MIN_LOG_LEVEL=2 TF_USE_LEGACY_KERAS=1 DEEPFACE_HOME=/root
WORKDIR /srv

# Dependencies first, in a cached layer.
RUN --mount=type=cache,target=/root/.cache/uv \
	--mount=type=bind,source=uv.lock,target=uv.lock \
	--mount=type=bind,source=pyproject.toml,target=pyproject.toml \
	uv sync --frozen --no-dev --no-install-project

COPY app ./app
COPY pyproject.toml uv.lock ./

EXPOSE 8001
CMD ["uv", "run", "--frozen", "--no-dev", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8001", "--workers", "4"]
