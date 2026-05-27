FROM python:3.12-slim

# LibreOffice for the optional `exportPdf` flag.
RUN apt-get update && apt-get install -y --no-install-recommends \
        libreoffice-core libreoffice-calc libreoffice-common \
        fonts-dejavu fonts-liberation \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY html-to-xlsx/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY html-to-xlsx/app /app/app
COPY shared /app/shared

ENV PYTHONUNBUFFERED=1
CMD ["python", "/app/app/main.py"]
