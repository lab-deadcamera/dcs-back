# Subir archivos temporales

Los archivos temporales se almacenan en `UPLOAD_DIR/temp/` con `storage = 'temp'` en la tabla `files`.
Un cron los purga automáticamente a la medianoche si tienen más de 1 mes de antigüedad.

## Frontend

```html
<form id="upload-temp" enctype="multipart/form-data">
  <input type="file" name="file" />
  <input type="hidden" name="category" value="temp" />
  <input type="hidden" name="storage" value="temp" />
  <button type="submit">Upload temp</button>
</form>
```

```javascript
async function uploadTemp(file) {
  const form = new FormData();
  form.append('file', file);
  form.append('category', 'temp');
  form.append('storage', 'temp');

  const res = await fetch('/api/v1/files/upload', { method: 'POST', body: form });
  return res.json();
}
```

## Respuesta

```json
{
  "id": "uuid",
  "filename": "preview.png",
  "url": "http://localhost:9099/api/v1/files/uuid",
  "size": 12345,
  "mime_type": "image/png",
  "format": "png",
  "category": "temp"
}
```

## Preview

```html
<img src="/api/v1/files/{id}/serve" />
```

## Eliminación manual (soft delete → trash)

```javascript
await fetch(`/api/v1/files/${id}`, { method: 'DELETE' });
// → { "message": "file moved to trash" }
```

## Recuperar desde trash (solo para temp)

```javascript
await fetch(`/api/v1/files/${id}/recover-temp`, { method: 'POST' });
// → { "message": "temp file recovered" }
```
