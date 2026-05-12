# Agregar archivos a un personaje

Flujo completo: subir archivo → asociar al personaje con un rol.

## Paso 1: Subir archivo

```javascript
// Subir como persistente
const file = await uploadPersistent(myFile, 'images');
// file.id = "uuid-del-archivo"
```

## Paso 2: Asociar al personaje

```javascript
async function addFileToCharacter(characterId, fileId, role = 'reference') {
  const res = await fetch(`/api/v1/characters/${characterId}/files`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ file_id: fileId, role }),
  });
  return res.json();
}

// Roles disponibles: 'reference', 'portrait', 'asset', o cualquier string custom
await addFileToCharacter(charId, file.id, 'portrait');
// → { "message": "file added to character" }
```

## Listar archivos de un personaje

```javascript
const files = await fetch(`/api/v1/characters/${charId}/files`).then(r => r.json());
// [
//   { "file_id": "uuid", "role": "portrait", "created_at": "..." },
//   { "file_id": "uuid", "role": "reference", "created_at": "..." }
// ]
```

## Obtener metadata del archivo para mostrar preview

```javascript
const fileMeta = await fetch(`/api/v1/files/${fileId}`).then(r => r.json());
// { "id": "...", "filename": "...", "mime_type": "image/png", "category": "images", ... }

// Preview directa:
// <img src="/api/v1/files/{fileId}/serve" />
```

## Desvincular archivo del personaje

```javascript
await fetch(`/api/v1/characters/${charId}/files/${fileId}`, { method: 'DELETE' });
// → { "message": "file removed from character" }
```

> Esto NO elimina el archivo físico ni su registro en `files`. Solo remueve la relación.
