FROM python:3.12-slim

# LibreOffice + fonts for the optional `exportPdf` flag.
RUN apt-get update && apt-get install -y --no-install-recommends \
        libreoffice-core libreoffice-writer libreoffice-common \
        fonts-dejavu fonts-liberation fonts-noto-color-emoji \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY docx/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY docx/app /app/app
COPY shared /app/shared

ENV PYTHONUNBUFFERED=1
CMD ["python", "/app/app/main.py"]
