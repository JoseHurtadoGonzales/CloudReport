FROM python:3.12-slim

# LibreOffice (Impress) for the optional `exportPdf` flag.
RUN apt-get update && apt-get install -y --no-install-recommends \
        libreoffice-core libreoffice-impress libreoffice-common \
        fonts-dejavu fonts-liberation \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY pptx/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY pptx/app /app/app
COPY shared /app/shared

ENV PYTHONUNBUFFERED=1
CMD ["python", "/app/app/main.py"]
