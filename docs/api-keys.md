# 🔑 API Keys — guía de uso y permisos

Las **API keys** permiten que servidores, automatizaciones (n8n) y agentes de IA
usen Cloud-Report sin un login interactivo. Esta guía explica qué son, cómo
crearlas, **qué pueden y qué no pueden hacer**, y cómo usarlas de forma segura.

---

## 1. Qué es una API key

- Es un secreto con formato `cr_<prefix>_<secret>`.
- Se envía en el header **`x-api-key`** en cada request.
- **Hereda los permisos del usuario que la creó.** No es una credencial
  "reducida": si el dueño es admin, la key puede casi todo lo que puede el admin
  (con la excepción de la gestión de usuarios — ver §4).
- Solo se guarda su **hash SHA-256** en la base; el secreto en claro se muestra
  **una sola vez** al crearla.

### Crear / revocar
- En la UI: **Configuración → API Keys** (crear, ver, revocar).
- Por API (requiere **sesión JWT**, no se puede con otra API key):

```bash
# Crear
curl -k -X POST https://<host>:8443/api/apikeys/ \
  -H "Authorization: Bearer <JWT>" -H "Content-Type: application/json" \
  -d '{ "name": "n8n-prod", "scopes": [], "ttlDays": 90 }'
# → responde con { "key": "cr_xxx_yyy", "apiKey": {...}, "note": "..." }
#   guardá "key" ahora: no se vuelve a mostrar.

# Listar
curl -k https://<host>:8443/api/apikeys/ -H "Authorization: Bearer <JWT>"

# Revocar
curl -k -X DELETE https://<host>:8443/api/apikeys/<id> -H "Authorization: Bearer <JWT>"
```

> Validez: la key deja de funcionar si se **revoca** (`RevokedAt`) o **vence**
> (`ExpiresAt`). Cada uso actualiza `LastUsedAt`.

> ⚠️ **Sobre `scopes`:** hoy el campo se guarda pero **no se aplica** en la
> autorización. Una key con scopes igual puede todo lo de §3. La única
> restricción real activa es la de usuarios (§4). No confíes en `scopes` para
> limitar una key todavía.

---

## 2. Cómo autenticar

Agregá el header en cualquier endpoint que acepte auth:

```bash
curl -k https://<host>:8443/odata/templates -H "x-api-key: cr_xxx_yyy"
```

> También se acepta `Authorization: Bearer <JWT>` (sesión de la UI). Si mandás
> ambos, prevalece la API key.

---

## 3. Qué PUEDE hacer una API key

Prácticamente toda la API de contenido y automatización:

| Área | Endpoints |
|------|-----------|
| **Render** | `POST /api/report`, `POST /api/component` |
| **Templates / contenido (CRUD)** | `GET/POST/PATCH/PUT/DELETE /odata/templates` (y `assets`, `scripts`, `data`, `components`, `folders`, `tags`) |
| **Reportes** | `GET /reports/:shortid/status`, `GET /reports/:shortid/content`, `GET /odata/reports` |
| **Scheduling** | `POST /api/scheduling/runNow`, `GET /api/scheduling/nextRun/:cron`, CRUD `/odata/schedules` |
| **Assets** | `POST /api/assets/upload`, `GET /assets/:shortid/content` |
| **Import / Export** | `POST /api/export`, `POST /api/import` |
| **Version control** | `/api/version-control/*` |
| **Studio** | `/studio/hierarchyMove`, `/studio/text-search`, `/studio/validate-entity-name` |
| **KV / metadata** | `/api/kv/:key`, `/api/recipe`, `/api/engine`, `/api/schema/:set` |
| **Identidad propia** | `GET /api/current-user` (devuelve el usuario dueño de la key) |

Para crear plantillas con IA, ver **[templates-ai.md](./templates-ai.md)**.

---

## 4. Qué NO puede hacer una API key

> 🔒 **La gestión de usuarios está reservada a sesiones con login (JWT).**
> Una API key **no puede hacer nada relacionado con usuarios.** Todos estos
> devuelven **`403 Forbidden`** si se llaman con `x-api-key`:

| Acción | Endpoint | Resultado con API key |
|--------|----------|------------------------|
| Crear usuario | `POST /api/admin/users` | **403** |
| Editar / promover / degradar | `PATCH /api/admin/users/:shortid` | **403** |
| Borrar usuario | `DELETE /api/admin/users/:shortid` | **403** |
| Cambiar contraseña | `POST /api/users/:shortid/password` | **403** |
| Listar / leer usuarios | `GET /odata/users` | **403** |
| Crear/editar/borrar vía OData | `POST/PATCH/PUT/DELETE /odata/users` | **403** (read-only para todos) |

Además, **una API key no puede gestionar otras API keys**: los endpoints
`/api/apikeys/*` requieren sesión JWT.

> La entidad `users` es de **solo lectura** vía OData incluso para sesiones JWT;
> las altas/bajas/cambios se hacen por `/api/admin/users` (que valida que el
> actor sea admin) y desde la UI en *Configuración → Usuarios*.

---

## 5. Seguridad — buenas prácticas

- 🔐 **Tratá la key como una contraseña.** Una key filtrada da acceso a todo el
  contenido (y al render) de la cuenta. No la pegues en repos, logs ni chats.
- ⏳ **Poné vencimiento** (`expiresInDays`) en keys de automatización.
- ♻️ **Rotá** las keys periódicamente y **revocá** las que no uses.
- 🎯 **Una key por integración** (n8n, IA, script), con nombre claro, para poder
  revocar de forma quirúrgica y leer su `LastUsedAt`.
- 🧑‍💻 **No uses una key de admin** para automatizaciones si no hace falta:
  creá la key con un usuario de menos privilegios.

---

## 6. Errores frecuentes

| Código | Causa |
|--------|-------|
| `401 unauthorized` | Falta el header `x-api-key`, o la key es inválida / revocada / vencida. |
| `403` en endpoints de usuarios | Esperado: las API keys no gestionan usuarios (§4). Usá una sesión con login. |
| `403` al escribir `/odata/users` | Esperado: `users` es read-only vía OData. Usá `/api/admin/users`. |
| `500` al renderizar | Error del motor/worker. Revisá el perfil: header `Profile-Location` → `GET /api/profile/<id>/events`. |
