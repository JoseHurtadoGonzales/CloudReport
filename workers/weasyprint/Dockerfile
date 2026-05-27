# Build context is workers/ (see docker-compose.yml)
FROM python:3.12-slim

# WeasyPrint runtime deps (Cairo, Pango, GDK-PixBuf, etc.)
RUN apt-get update && apt-get install -y --no-install-recommends \
        libpango-1.0-0 libpangoft2-1.0-0 libcairo2 libgdk-pixbuf-2.0-0 \
        libffi-dev shared-mime-info fonts-dejavu fonts-liberation \
        fonts-noto-color-emoji \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY weasyprint/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy worker code + shared protocol
COPY weasyprint/app /app/app
COPY shared /app/shared

ENV PYTHONUNBUFFERED=1
CMD ["python", "/app/app/main.py"]
