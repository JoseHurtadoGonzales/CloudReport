<div align="center">

# ☁️ Cloud-Report

**Diseñá plantillas HTML y servílas como PDF, DOCX, XLSX o PPTX desde una sola API.**

Plataforma de generación de reportes: **Go API** + **Nuxt UI** + **workers Python**, detrás de un **reverse proxy Caddy con HTTPS**, con **SeaweedFS (S3)**, **PostgreSQL**, **Redis** y **n8n** para automatización.

</div>

---

## 📑 Tabla de contenidos

- [✨ Características](#-características)
- [🏗️ Arquitectura](#️-arquitectura)
- [🧩 Stack](#-stack)
- [🚀 Puesta en marcha (Docker)](#-puesta-en-marcha-docker)
  - [1. Requisitos](#1-requisitos)
  - [2. Clonar y configurar](#2-clonar-y-configurar)
  - [3. Levantar todo](#3-levantar-todo)
  - [4. Acceder](#4-acceder)
- [⚙️ Configuración (`.env`)](#️-configuración-env)
- [🔌 Puertos](#-puertos)
- [🔄 Flujo de render](#-flujo-de-render)
- [📡 Uso de la API](#-uso-de-la-api)
- [🤖 Automatización con n8n](#-automatización-con-n8n)
- [🗄️ SeaweedFS Admin Console](#️-seaweedfs-admin-console)
- [🗂️ Estructura del proyecto](#️-estructura-del-proyecto)
- [🛠️ Desarrollo](#️-desarrollo)
- [🧪 Tests](#-tests)
- [🧹 Retención de reportes](#-retención-de-reportes)
- [🌐 Internacionalización](#-internacionalización)
- [❓ Troubleshooting](#-troubleshooting)

---

## ✨ Características

- 🧾 **Plantillas Handlebars** → render a **PDF (Chromium / WeasyPrint)**, **DOCX**, **XLSX**, **PPTX**, **HTML**.
- ✏️ **Editor visual** con resaltado de sintaxis, preview en vivo, tabs de Contenido / Estilos / Helpers / Data / Scripts / Recipe / Página / PDF ops.
- 🖼️ **Assets** (CSS, imágenes, fuentes) inlineables con `{#asset nombre}` — sin servidor de archivos externo.
- 🧷 **Operaciones PDF**: concatenar (`append`/`prepend`), estampar headers/footers en cada página (`merge` + `renderForEveryPage`).
- ⏳ **Retención por plantilla**: cada template define cuántos días se guardan sus reportes; un sweeper los borra solos.
- 🔐 **Auth con JWT** (sesión deslizante "no me deslogués nunca") + **API keys** para servidores.
- 👥 **Gestión de usuarios** (admin): crear, promover/degradar, eliminar.
- 🌐 **i18n ES/EN** en toda la app, con toggle en vivo.
- 📖 **Docs de la API embebidas** en `/docs`.
- 🤖 **n8n** integrado para automatizar renders desde workflows.
- 🔒 **HTTPS** vía Caddy (reverse proxy de frontend + backend + UIs).

---

## 🏗️ Arquitectura

```mermaid
flowchart TD
    Browser([🌐 Navegador]) -->|HTTPS :8443| Caddy[🔒 Caddy<br/>reverse proxy]

    Caddy -->|/ ·| UI[🖥️ Nuxt UI<br/>:3030]
    Caddy -->|/api /odata /reports /assets| API[⚙️ Go API<br/>:5488]
    Caddy -->|:5678| N8N[🤖 n8n main]
    Caddy -->|:19080 / :19081 / :19082| SWUI[(SeaweedFS UIs<br/>filer · master · admin)]

    subgraph APP[Plataforma de reportes]
      API --> PG[(🐘 PostgreSQL)]
      API --> REDIS[(⚡ Redis · streams)]
      API --> S3[(🗄️ SeaweedFS S3<br/>blobs / reportes)]
      REDIS -->|streams| W1[🐍 chromium-pdf]
      REDIS -->|streams| W2[🐍 weasyprint]
      REDIS -->|streams| W3[🐍 docx]
      REDIS -->|streams| W4[🐍 pptx]
      REDIS -->|streams| W5[🐍 html-to-xlsx]
      W1 & W2 & W3 & W4 & W5 --> S3
    end

    subgraph AUTO[n8n · infraestructura propia y aislada]
      N8N --> NPG[(🐘 n8n-postgres)]
      N8N --> NRD[(⚡ n8n-redis · cola Bull)]
      NRD --> NW1[🔧 n8n-worker ×N]
      NW1 --> NPG
    end
```

El **navegador solo habla con Caddy** (HTTPS). El frontend usa rutas relativas (`/api`, `/odata`…) sobre el mismo origen, y Caddy las enruta al backend Go o al Nuxt según el path. Los workers de render corren en Python y reciben jobs por **streams de Redis**, devolviendo los binarios a **SeaweedFS**.

**n8n corre totalmente aislado**: su propio PostgreSQL y Redis (no comparte datos con la app), en **modo queue** con una flota de workers escalable. Un solo `docker compose up -d` levanta las dos mitades —la plataforma de reportes y la automatización— juntas.

---

## 🧩 Stack

| Capa | Tecnología |
|------|------------|
| **Frontend** | Nuxt 3 + Nuxt UI 4 (Tailwind v4), Pinia, Shiki, iconos Lucide |
| **Backend** | Go (Fiber), JWT, pgx |
| **Workers** | Python — Chromium (Playwright), WeasyPrint, docxtpl, python-pptx, html-to-xlsx |
| **Base de datos** | PostgreSQL 16 |
| **Cola** | Redis 7 (streams) |
| **Almacenamiento** | SeaweedFS (API S3) |
| **Proxy / TLS** | Caddy 2 |
| **Automatización** | n8n (queue mode) + Postgres y Redis propios |

---

## 🚀 Puesta en marcha (Docker)

### 1. Requisitos

- Docker + Docker Compose v2
- ~4 GB RAM libres (Chromium necesita headroom)

### 2. Clonar y configurar

```bash
git clone https://github.com/JoseHurtadoGonzales/CloudReport.git
cd CloudReport

# Copiá la plantilla de entorno y editala
cp .env.example .env
nano .env
```

Como **mínimo** cambiá en `.env`:

```bash
JWT_SECRET=$(openssl rand -hex 32)        # secreto random
INITIAL_ADMIN_USERNAME=tu-usuario          # admin inicial
INITIAL_ADMIN_PASSWORD=una-clave-fuerte
N8N_HOST=10.71.1.125                        # IP/dominio del server
N8N_WEBHOOK_URL=https://10.71.1.125:5678/
```

> Detrás de Caddy, **dejá `NUXT_PUBLIC_API_BASE` vacío** (el navegador usa rutas relativas).

### 3. Levantar todo

```bash
docker compose up -d --build
```

Esto construye y arranca: Postgres, Redis, SeaweedFS (master/volume/filer/s3 + admin console), API Go, UI Nuxt, los 5 workers, Caddy, y n8n con su propio Postgres/Redis + workers. En el **primer arranque** se crea automáticamente el usuario admin definido en `.env`.

### 4. Acceder

| Servicio | URL | Credenciales |
|----------|-----|--------------|
| **App** | `https://<host>:8443` | tu admin de `.env` |
| **n8n** | `https://<host>:5678` | se configura al primer ingreso |
| **SeaweedFS Admin Console** ⭐ | `https://<host>:19082` | `SEAWEEDFS_ADMIN_USER` / `_PASSWORD` (login propio) |
| **SeaweedFS filer UI** (debug) | `https://<host>:19080` | `SEAWEEDFS_UI_USER` / basic-auth |
| **SeaweedFS master UI** (debug) | `https://<host>:19081` | idem |

> La primera vez el navegador pedirá aceptar el certificado (es auto-firmado por Caddy en LAN sin dominio — normal).

---

## ⚙️ Configuración (`.env`)

| Variable | Default | Descripción |
|----------|---------|-------------|
| `JWT_SECRET` | `change-me-in-prod` | Secreto para firmar JWT. **Cambialo.** |
| `SESSION_TTL_HOURS` | `720` | Vida del token (720h = 30 días). El frontend lo renueva solo. |
| `ALLOW_REGISTRATION` | `true` | Habilita registro público. `false` para instalaciones cerradas. |
| `INITIAL_ADMIN_USERNAME/PASSWORD/EMAIL` | `admin / admin123` | Admin creado en el primer boot con DB vacía. |
| `NUXT_PUBLIC_API_BASE` | *(vacío)* | Vacío = rutas relativas detrás de Caddy. URL absoluta solo si NO usás proxy. |
| `CADDY_HTTPS_PORT` | `8443` | Puerto HTTPS público. En server dedicado poné `443`. |
| `N8N_HOST_PORT` | `5678` | Puerto de n8n. |
| `SEAWEEDFS_FILER_UI_PORT` / `MASTER_UI_PORT` / `ADMIN_PORT` | `19080` / `19081` / `19082` | UIs de SeaweedFS (filer, master, admin console). |
| `SEAWEEDFS_UI_USER` / `PASSWORD_HASH` | `admin` / *(default)* | Basic-auth de las UIs filer/master. |
| `SEAWEEDFS_ADMIN_USER` / `_PASSWORD` | `admin` / `cloudreport` | Login propio del Admin Console (sesión, no basic-auth). |
| `N8N_HOST` / `N8N_WEBHOOK_URL` | `10.71.1.125` | Host/URL pública de n8n (no `localhost` — los webhooks deben resolver desde afuera). |
| `N8N_VERSION` | `latest` | Tag de la imagen de n8n. |
| `N8N_WORKERS` | `2` | Réplicas de `n8n-worker` que consumen la cola. |
| `N8N_ENCRYPTION_KEY` | *(requerido)* | Cifra credenciales de n8n. **Constante.** `openssl rand -hex 32`. |
| `N8N_DB_NAME/USER/PASSWORD` | `n8n` | Postgres dedicado de n8n. |
| `N8N_REDIS_PASSWORD` | *(requerido)* | Password del Redis dedicado de n8n. |
| `N8N_SECURE_COOKIE` | `false` | `true` solo con cert de confianza end-to-end. |

> 🔑 Para una clave propia de las UIs de SeaweedFS (escapando los `$` para Compose):
> ```bash
> docker run --rm caddy:2-alpine caddy hash-password --plaintext 'tu-clave' | sed 's/\$/\$\$/g'
> ```

---

## 🔌 Puertos

| Puerto host | Servicio |
|-------------|----------|
| `8443` | HTTPS — frontend + API (vía Caddy) |
| `5678` | n8n (vía Caddy) |
| `19080` / `19081` / `19082` | SeaweedFS filer / master UI / **Admin Console** |
| `15432` | PostgreSQL (debug) |
| `16379` | Redis (debug) |
| `8333` / `8888` / `9333` / `8080` | SeaweedFS s3 / filer / master / volume (debug) |

> Los puertos por defecto son **alternos** para poder convivir con otro reverse proxy ya usando 80/443 en el mismo host. En un server dedicado, poné `CADDY_HTTPS_PORT=443`.

---

## 🔄 Flujo de render

```mermaid
sequenceDiagram
    participant C as Cliente / n8n
    participant API as Go API
    participant R as Redis
    participant W as Worker (Python)
    participant S3 as SeaweedFS
    participant DB as PostgreSQL

    C->>API: POST /api/report { template, data }
    API->>API: resuelve plantilla + assets + helpers
    API->>R: encola job de render (stream)
    R->>W: entrega job
    W->>W: HTML -> PDF/DOCX/XLSX/PPTX
    W->>S3: guarda binario
    W->>API: devuelve key del blob
    API->>API: aplica operaciones PDF (merge/stamp...)
    API->>S3: guarda reporte final
    API->>DB: registra report (con expires_at)
    API-->>C: archivo binario (Content-Type segun recipe)
```

---

## 📡 Uso de la API

**Renderizar una plantilla guardada:**

```bash
curl -X POST https://<host>:8443/api/report \
  -H "Authorization: Bearer <API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{ "template": { "shortid": "<TPL>" }, "data": { "name": "Mundo" } }' \
  --output reporte.pdf
```

**Plantilla inline (sin guardarla):**

```bash
curl -X POST https://<host>:8443/api/report \
  -H "Authorization: Bearer <API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{
    "template": { "content": "<h1>Hola {{name}}</h1>", "engine": "handlebars", "recipe": "chrome-pdf" },
    "data": { "name": "Ana" }
  }' --output out.pdf
```

> Generá tu API key en **Configuración → API Keys**. La doc completa está embebida en la app en **`/docs`**.

---

## 🤖 Automatización con n8n

n8n viene integrado para orquestar workflows (por ejemplo: disparar un render de Cloud-Report cuando llega un webhook, en un cron, o tras un evento). Corre con **su propia base de datos y su propio Redis** —separados de los de la app— en **modo queue**:

- **`n8n`** — instancia principal: UI, editor y manejo de webhooks. Es la **única** que corre migraciones de la DB.
- **`n8n-worker`** — flota de workers que ejecutan los workflows tomados de la cola Bull en Redis. Escalás con `N8N_WORKERS` (default 2).
- **`n8n-postgres`** / **`n8n-redis`** — datastores dedicados con volúmenes propios.

```mermaid
flowchart LR
    Trigger([Webhook / Cron]) --> Main[n8n main]
    Main -->|encola| Q[(n8n-redis · Bull)]
    Q --> W1[worker 1]
    Q --> W2[worker 2]
    W1 & W2 -->|HTTP POST /api/report| API[Cloud-Report API]
    W1 & W2 --> DB[(n8n-postgres)]
```

**Acceso:** `https://<host>:5678` (vía Caddy). La primera vez creás el usuario owner de n8n.

**Claves importantes:**
- `N8N_ENCRYPTION_KEY` — cifra las credenciales guardadas. **Debe ser constante**; si la cambiás, n8n no puede leer las credenciales existentes. Generala una vez con `openssl rand -hex 32`.
- Todas las instancias (main + workers) comparten esa misma key y la misma DB/Redis — por eso van en el ancla `x-n8n-common` del compose.
- Los workers arrancan **después** de que el main esté `healthy` (migraciones hechas), evitando el típico choque de migraciones concurrentes en el primer boot.

**Escalar workers:**
```bash
# en .env
N8N_WORKERS=4
docker compose up -d n8n-worker
```

---

## 🗄️ SeaweedFS Admin Console

Cloud-Report incluye la **UI oficial moderna de SeaweedFS** (`weed admin`) con dashboard, navegador de buckets S3, gestión de usuarios IAM con sus claves, filer browser, métricas del cluster y mantenimiento programado. Es lo que usás en el día a día para inspeccionar lo que se guardó.

**Acceso:** `https://<host>:19082` — login con `SEAWEEDFS_ADMIN_USER` / `SEAWEEDFS_ADMIN_PASSWORD` (form propio, sesión).

Las dos UIs viejas (`:19080` filer, `:19081` master) siguen disponibles bajo basic-auth de Caddy para debug crudo, pero **la Admin Console es la recomendada**.

```mermaid
flowchart LR
    Admin[weed admin :23646] -->|gRPC| Master[(master :9333)]
    Admin -->|HTTP| Filer[(filer :8888)]
    Browser([🌐]) -->|HTTPS :19082| Caddy[🔒 Caddy] --> Admin
```

Internamente, el binario `weed admin` se conecta al master y descubre los filers; expone su propia UI HTTP en `:23646`. Caddy la publica con TLS interno en el puerto `:19082` del host. La auth no la maneja Caddy (sería redundante) sino la propia app con sesión.

Para regenerar la clave del admin: editás `SEAWEEDFS_ADMIN_PASSWORD` en `.env` y `docker compose up -d seaweedfs-admin` (re-aplica el flag `-adminPassword` en el restart).

---

## 🗂️ Estructura del proyecto

```
cloud-report/
├── api/            # Backend Go (Fiber) — render, auth, OData, scheduler
│   ├── cmd/server/ # entrypoint
│   └── internal/   # auth, blob, config, odata, render, store...
├── ui/             # Frontend Nuxt 3 (pages, components, composables, stores)
├── workers/        # Workers Python (chromium, weasyprint, docx, pptx, html-to-xlsx)
├── caddy/          # Caddyfile (reverse proxy + TLS)
├── infra/          # config de SeaweedFS (s3.json)
├── docs/           # documentación adicional
├── docker-compose.yml
└── .env.example
```

---

## 🛠️ Desarrollo

```bash
# API (Go)
cd api && go run ./cmd/server

# UI (Nuxt) — hot reload en :3030
cd ui && npm install && npm run dev

# Levantar solo las dependencias (DB, Redis, SeaweedFS)
docker compose up -d postgres redis seaweedfs-s3
```

---

## 🧪 Tests

```bash
cd api
go test ./...        # ~189 casos: auth/JWT, config, OData, render, pdf-utils
go vet ./...
```

---

## 🧹 Retención de reportes

Cada plantilla tiene un campo **"Retención de reportes"** (pestaña *Página* del editor):

- Cada render se guarda en SeaweedFS + tabla `reports` con un `expires_at`.
- Un **sweeper** corre cada hora y borra (blob + fila) los reportes vencidos.
- Valores rápidos: 7 / 30 / 90 / 365 días, o **0 = no borrar nunca**.

```mermaid
flowchart LR
    R[Render] --> Store[Guarda reporte<br/>expires_at = now + N dias]
    Store --> Sweep{Sweeper<br/>cada 1h}
    Sweep -->|expires_at < now| Del[🗑️ Borra blob + fila]
    Sweep -->|vigente| Keep[✅ Conserva]
```

---

## 🌐 Internacionalización

Toda la UI está en **Español** e **Inglés**. El toggle **ES / EN** está arriba a la derecha; la preferencia se guarda en cookie por un año. Para agregar strings: editás `ui/composables/useI18n.ts` (claves en ambos locales) y usás `t('mi.clave')` en el template.

---

## ❓ Troubleshooting

| Problema | Causa / solución |
|----------|------------------|
| Los iconos no cargan / cargan al recargar | El bundle de iconos es local (`@iconify-json/lucide`). Hacé **hard refresh** (Ctrl+Shift+R) tras un deploy. |
| `address already in use` en :8443/:80 | Otro proceso usa el puerto. Cambiá `CADDY_HTTPS_PORT` en `.env`. |
| El navegador desconfía del certificado | Es auto-firmado (TLS internal en LAN). Aceptalo, o configurá un dominio real en el `Caddyfile` para Let's Encrypt. |
| 401 en todas las llamadas | Token vencido o `JWT_SECRET` cambiado (invalida sesiones). Volvé a loguearte. |
| El admin inicial no se crea | Solo se crea con **DB vacía** en el primer boot. Si ya hay usuarios, creá desde *Configuración → Usuarios*. |

---

<div align="center">
<sub>Cloud-Report — generá reportes lindos, rápido. 🚀</sub>
</div>
