# Crear personajes

Cada personaje tiene `name`, `description`, `metadata` (JSONB para datos adicionales), y puede tener múltiples archivos asociados.

## Crear personaje

```javascript
async function createCharacter(name, description, metadata = {}) {
  const res = await fetch('/api/v1/characters', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name,
      description,
      metadata: JSON.stringify(metadata),
    }),
  });
  return res.json();
}

// Ejemplo:
const character = await createCharacter(
  'Neon Samurai',
  'Cyberpunk protagonist with katana',
  { age: 28, style: 'cyberpunk', gender: 'female' }
);
```

## Respuesta

```json
{
  "id": "uuid",
  "name": "Neon Samurai",
  "description": "Cyberpunk protagonist with katana",
  "metadata": "{\"age\":28,\"style\":\"cyberpunk\",\"gender\":\"female\"}",
  "created_at": "2026-05-12T00:00:00Z",
  "updated_at": "2026-05-12T00:00:00Z",
  "deleted_at": null
}
```

## Listar personajes

```javascript
const characters = await fetch('/api/v1/characters').then(r => r.json());
```

## Obtener personaje con sus archivos

```javascript
const charDetail = await fetch(`/api/v1/characters/${id}`).then(r => r.json());
// { character: {...}, files: [...] }
```

## Actualizar personaje

```javascript
await fetch(`/api/v1/characters/${id}`, {
  method: 'PATCH',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ name: 'New Name', description: 'Updated' }),
});
```

## Eliminar personaje (soft delete)

```javascript
await fetch(`/api/v1/characters/${id}`, { method: 'DELETE' });
// → { "message": "character deleted" }
```
