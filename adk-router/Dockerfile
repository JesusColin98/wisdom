# Wisdom ADK Router — Docker image.
# Runs the Python FastAPI + ADK routing service on Cloud Run.

FROM python:3.12-slim

WORKDIR /app

# Install system dependencies.
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies first (layer cache).
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application source.
COPY . .

# Cloud Run uses the PORT env variable.
ENV PORT=8081

# Use exec form for proper signal handling.
ENTRYPOINT ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8081", "--workers", "1"]
