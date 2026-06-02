# 🤖 Crear plantillas por IA / automatización (API key)

Guía para que un agente de IA (o cualquier automatización, ej. n8n) cree y
renderice plantillas de Cloud-Report usando una **API key**. Está pensada para
pegarse como contexto en el prompt de la IA.

> **Auth:** todas las llamadas usan el header `x-api-key: cr_<prefix>_<secret>`.
> Una API key tiene **los mismos permisos que el usuario que la creó**, salvo la
> gestión de usuarios (bloqueada). Generala en la UI: *Configuración → API Keys*.
> Detalle completo de permisos en **[api-keys.md](./api-keys.md)**.

> **Base URL:** detrás de Caddy es `https://<host>:8443`. El certificado en LAN
> es autofirmado, por eso los ejemplos usan `curl -k`.

---

## 1. Introspección: pedir el schema

Antes de crear, la IA puede leer el schema con tipos, enums y un ejemplo listo:

```bash
curl -k https://<host>:8443/api/schema/templates
```

Devuelve un JSON-Schema (draft-07) con los campos reales en **camelCase**, los
valores válidos de `recipe`/`engine`, descripciones y un `examples`.

---

## 2. Campos de una plantilla

| Campo | Tipo | Valores / notas |
|-------|------|-----------------|
| `name` | string | **Requerido.** Nombre visible. |
| `engine` | enum | `handlebars` (interpola `{{vars}}`) · `none` |
| `recipe` | enum | `chrome-pdf`, `weasyprint`, `docx`, `pptx`, `xlsx`, `html-to-xlsx`, `html`, `text`, `static-pdf` |
| `content` | string | Cuerpo HTML (recetas web). Para `docx`/`pptx` el contenido viene del asset-plantilla. |
| `helpers` | string | JS de helpers Handlebars. Corre en sandbox. |
| `css` | string | Estilos. |
| `pageSize` | enum | `A4`, `A3`, `A5`, `Letter`, `Legal`, `Tabloid` (recetas PDF). |
| `pageOrientation` | enum | `portrait` · `landscape`. |
| `pageMargin` | string | ej `"1cm"`, `"20px"`. |
| `chrome` | object | Opciones de chrome-pdf (ver abajo). |
| `weasyprint` | object | Opciones de weasyprint. |
| `docx` / `pptx` | object | `{ "templateAsset": { "shortid": "<asset>" } }`. |
| `xlsx` | object | Opciones de xlsx. |
| `pdfOperations` | array | merge/append/prepend (ver abajo). |
| `dataShortid` | string | shortid de un `data` (JSON de ejemplo). |
| `reportRetentionDays` | int | Días de retención del reporte. `0` = nunca. Default 30. |
| `folder` | string\|null | Carpeta contenedora. |
| `isPublic` | bool | Si `true`, render sin auth. |
| `readPermissions` / `editPermissions` | string[] | IDs de usuarios/grupos. |

### Opciones `chrome` más usadas
```jsonc
{
  "printBackground": true,
  "landscape": false,
  "marginTop": "1cm", "marginBottom": "1cm",
  "displayHeaderFooter": true,
  "headerTemplate": "<div style='font-size:8px'>Encabezado</div>",
  "footerTemplate": "<div style='font-size:8px'>Pág <span class='pageNumber'></span></div>"
}
```

### `pdfOperations` (estampar / concatenar)
```jsonc
[
  { "type": "merge",   "templateShortid": "<tplHeader>", "renderForEveryPage": true },
  { "type": "append",  "templateShortid": "<tplAnexo>" }
]
```

---

## 3. Flujo recomendado para la IA

1. (Opcional) `GET /api/schema/templates` para conocer los campos.
2. `POST /odata/templates` con el body → guarda el template y devuelve su `shortid`.
3. `POST /api/report` con `{ "template": { "shortid": "<shortid>" }, "data": {...} }` → recibe el binario.
4. Iterar: `PATCH /odata/templates/<shortid>` para corregir, volver a renderizar.

---

## 4. Ejemplos

### A. Crear un template PDF (chrome-pdf)
```bash
curl -k -X POST https://<host>:8443/odata/templates \
  -H "x-api-key: cr_xxx_yyy" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Factura PDF",
    "engine": "handlebars",
    "recipe": "chrome-pdf",
    "content": "<h1>Factura {{number}}</h1><p>Cliente: {{client.name}}</p><p>Total: {{total}}</p>",
    "css": "h1{color:#0b5}",
    "pageSize": "A4",
    "pageOrientation": "portrait",
    "pageMargin": "1cm",
    "chrome": { "printBackground": true }
  }'
```
La respuesta incluye `"shortid": "..."`. Guardalo.

### B. Renderizar ese template con datos
```bash
curl -k -X POST https://<host>:8443/api/report \
  -H "x-api-key: cr_xxx_yyy" \
  -H "Content-Type: application/json" \
  -d '{
    "template": { "shortid": "<SHORTID>" },
    "data": { "number": "0001", "client": { "name": "ACME" }, "total": "$1.500" }
  }' --output factura.pdf
```

### C. Render directo sin guardar (template inline)
```bash
curl -k -X POST https://<host>:8443/api/report \
  -H "x-api-key: cr_xxx_yyy" \
  -H "Content-Type: application/json" \
  -d '{
    "template": {
      "content": "<h1>Hola {{name}}</h1>",
      "engine": "handlebars",
      "recipe": "chrome-pdf"
    },
    "data": { "name": "Ana" }
  }' --output out.pdf
```

### D. Excel desde tablas HTML (html-to-xlsx)
```bash
curl -k -X POST https://<host>:8443/odata/templates \
  -H "x-api-key: cr_xxx_yyy" -H "Content-Type: application/json" \
  -d '{
    "name": "Reporte ventas",
    "engine": "handlebars",
    "recipe": "html-to-xlsx",
    "content": "<table><tr><th>Producto</th><th>Cantidad</th></tr>{{#each rows}}<tr><td>{{name}}</td><td>{{qty}}</td></tr>{{/each}}</table>"
  }'
```

### E. DOCX desde un asset-plantilla
1. Subir el `.docx` como asset:
```bash
curl -k -X POST https://<host>:8443/api/assets/upload \
  -H "x-api-key: cr_xxx_yyy" -F "files=@plantilla.docx"
```
   → devuelve el `shortid` del asset.
2. Crear el template apuntando a ese asset:
```bash
curl -k -X POST https://<host>:8443/odata/templates \
  -H "x-api-key: cr_xxx_yyy" -H "Content-Type: application/json" \
  -d '{
    "name": "Contrato DOCX",
    "engine": "handlebars",
    "recipe": "docx",
    "docx": { "templateAsset": { "shortid": "<ASSET_SHORTID>" } }
  }'
```
   El `.docx` debe tener marcadores `{{variable}}` (sintaxis docxtpl).

### F. Editar un template existente
```bash
curl -k -X PATCH https://<host>:8443/odata/templates/<SHORTID> \
  -H "x-api-key: cr_xxx_yyy" -H "Content-Type: application/json" \
  -d '{ "css": "h1{color:#c00}" }'
```

### G. Listar / buscar templates
```bash
curl -k "https://<host>:8443/odata/templates?\$top=50&\$select=shortid,name,recipe&\$orderby=modificationDate%20desc" \
  -H "x-api-key: cr_xxx_yyy"
```

---

## 5. Entidades de apoyo (mismo CRUD OData con la key)

| Entidad | Endpoint | Para qué |
|---------|----------|----------|
| `data` | `/odata/data` | Guardar JSON de ejemplo y asociarlo con `dataShortid`. |
| `assets` | `/odata/assets` + `/api/assets/upload` | Imágenes, fuentes, CSS, plantillas docx/pptx. Referenciables con `{#asset nombre}`. |
| `scripts` | `/odata/scripts` | Scripts beforeRender/afterRender. |
| `components` | `/odata/components` | Fragmentos reutilizables. |
| `folders` | `/odata/folders` | Organización. |

---

## 6. Errores comunes

| Síntoma | Causa |
|---------|-------|
| `401 unauthorized` | Falta o es inválida la `x-api-key` (o el template inline sin auth). |
| `400` al crear | JSON mal formado o campo no insertable. Revisá los nombres camelCase. |
| `500` al renderizar | Error del motor/worker (HTML inválido, helper roto, asset faltante). Mirá el perfil: header `Profile-Location` → `GET /api/profile/<id>/events`. |
| docx/pptx sin contenido | Falta `templateAsset.shortid` o el asset no es un .docx/.pptx válido. |
