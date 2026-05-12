# Subir archivos permanentes

Los archivos permanentes se almacenan en `UPLOAD_DIR/{category}/` con `storage = 'persistent'`.
No se purgan automáticamente — solo se eliminan por acción explícita del usuario.

## Frontend

```html
<form id="upload-persistent" enctype="multipart/form-data">
  <input type="file" name="file" accept="image/*,video/*,audio/*" />
  <select name="category">
    <option value="images">Imagen</option>
    <option value="videos">Video</option>
    <option value="audio">Audio</option>
  </select>
  <button type="submit">Upload</button>
</form>
```

```javascript
async function uploadPersistent(file, category) {
  const form = new FormData();
  form.append('file', file);
  form.append('category', category);
  form.append('storage', 'persistent');

  const res = await fetch('/api/v1/files/upload', { method: 'POST', body: form });
  return res.json();
}
```

## Listar archivos por categoría

```javascript
const images = await fetch('/api/v1/files?category=images&storage=persistent').then(r => r.json());
const videos = await fetch('/api/v1/files?category=videos&storage=persistent').then(r => r.json());
```

## Soft delete (va a trash, se puede restaurar)

```javascript
await fetch(`/api/v1/files/${id}`, { method: 'DELETE' });
// → { "message": "file moved to trash" }
```

## Restaurar desde trash

```javascript
await fetch(`/api/v1/files/${id}/restore`, { method: 'POST' });
// → { "message": "file restored" }
```

## Hard delete (permanente, no reversible)

```javascript
await fetch(`/api/v1/files/${id}/hard`, { method: 'DELETE' });
// → { "message": "file permanently deleted" }
```

## Ver trash

```javascript
const trash = await fetch('/api/v1/files/trash').then(r => r.json());
```
