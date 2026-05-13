# Proveedores y modelos de IA

Cada **proveedor** (BytePlus, OpenAI, etc.) tiene múltiples **modelos** configurados con su propia API key, URL y endpoint.

## Crear proveedor

```javascript
async function createProvider(name) {
  const res = await fetch('/api/v1/providers', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  });
  return res.json();
}

// Ejemplo:
const provider = await createProvider('BytePlus');
```

## Crear modelo dentro de un proveedor

```javascript
async function createModel(providerId, name, apiKey, url, endpoint) {
  const res = await fetch('/api/v1/models', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      provider_id: providerId,
      name,
      api_key: apiKey,
      url,
      endpoint,
    }),
  });
  return res.json();
}

// Ejemplo:
const model = await createModel(
  provider.id,
  'seedance-2-0',
  'sk-xxx...',
  'https://api.byteplus.com',
  '/v1/generate'
);
```

## Respuesta estandarizada

Todas las respuestas usan el formato `{data, success, message}`:

```json
{
  "data": {
    "id": "uuid-del-modelo",
    "provider_id": "uuid-del-proveedor",
    "name": "seedance-2-0",
    "api_key": "sk-xxx...",
    "url": "https://api.byteplus.com",
    "endpoint": "/v1/generate",
    "active": true,
    "created_at": "2026-05-13T00:00:00Z",
    "updated_at": "2026-05-13T00:00:00Z",
    "deleted_at": null
  },
  "success": true,
  "message": "created"
}
```

## Listar proveedores (con sus modelos)

```javascript
const providers = await fetch('/api/v1/providers').then(r => r.json());
// providers.data → [{ provider: {...}, models: [...] }, ...]
```

## Obtener proveedor con modelos

```javascript
const detail = await fetch(`/api/v1/providers/${id}`).then(r => r.json());
// detail.data → { provider: {...}, models: [...] }
```

## Listar todos los modelos (con nombre del proveedor)

```javascript
const models = await fetch('/api/v1/models').then(r => r.json());
// models.data → [{ id, provider_id, name, api_key, url, endpoint, active, provider_name, ... }, ...]
```

## Actualizar

```javascript
// Actualizar proveedor
await fetch(`/api/v1/providers/${id}`, {
  method: 'PATCH',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ name: 'New Name', active: false }),
});

// Actualizar modelo
await fetch(`/api/v1/models/${id}`, {
  method: 'PATCH',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ api_key: 'sk-new-key', active: false }),
});
```

## Eliminar (soft delete)

```javascript
await fetch(`/api/v1/providers/${id}`, { method: 'DELETE' });
// → { data: null, success: true, message: "provider deleted" }

await fetch(`/api/v1/models/${id}`, { method: 'DELETE' });
// → { data: null, success: true, message: "model deleted" }
```

## Diagrama

```
┌──────────────┐       ┌──────────────┐
│  Proveedor   │──1:N──│    Modelo    │
│  (BytePlus)  │       │ (seedance)   │
│  (OpenAI)    │       │ (gpt-4o)     │
│  (Anthropic) │       │ (claude-4)   │
└──────────────┘       └──────────────┘
```
