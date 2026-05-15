# Use Cases — File & Character Management

## Índice

| # | Caso de uso | Archivo |
|---|-------------|---------|
| 1 | Subir archivos temporales | [`upload-temp.md`](./upload-temp.md) |
| 2 | Subir archivos permanentes | [`upload-persistent.md`](./upload-persistent.md) |
| 3 | Crear personajes | [`create-characters.md`](./create-characters.md) |
| 4 | Agregar archivos a un personaje | [`add-files-to-character.md`](./add-files-to-character.md) |
| 5 | Proveedores y modelos de IA | [`providers-and-models.md`](./providers-and-models.md) |
| 6 | Generación Studio (unified payload) | [`studio-generation.md`](./studio-generation.md) |

## Diagrama de flujo

```
                    ┌──────────────────────────────────────┐
                    │              FRONTEND                 │
                    └──────┬──────────────┬──────────┬──────┘
                           │              │          │
                    POST /files/upload     │   POST /characters
                           │              │          │
                           ▼              ▼          ▼
                    ┌────────────┐  ┌──────────┐  ┌──────────────┐
                    │  Archivo   │  │ Personaje│  │  Personaje   │
                    │ (temp o    │  │  creado  │  │  con archivo │
                    │  persist.) │  │          │  │  asociado    │
                    └─────┬──────┘  └──────────┘  └──────┬───────┘
                          │                              │
                          ▼                              ▼
                    ┌────────────┐               ┌──────────────┐
                    │   files    │               │    files +   │
                    │   tabla    │               │ character_files
                    └────────────┘               └──────────────┘
```

## Comparativa de categorías

| Categoría | storage | ¿Purga automática? | ¿Va a trash? | ¿Recuperable? |
|-----------|---------|--------------------|---------------|---------------|
| images    | persistent | No | Sí (soft delete) | restore |
| videos    | persistent | No | Sí (soft delete) | restore |
| audio     | persistent | No | Sí (soft delete) | restore |
| temp      | temp     | Sí (cron medianoche, >1 mes) | Sí (soft delete manual) | recover-temp |
