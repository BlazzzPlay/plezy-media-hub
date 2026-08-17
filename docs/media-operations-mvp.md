# Media Operations MVP

## Objetivo

Extender Plezy Media Hub con una pestaña **Gestionar** para inventariar,
identificar, aprobar y copiar contenido local de forma segura, mientras Plezy
continúa siendo el cliente de Plex y Jellyfin.

## Límites

- Plezy no ejecuta operaciones de archivos ni almacena claves de proveedores.
- El servicio `media-orchestrator` se ejecuta en el NUC y más adelante en el
  NAS; es el único proceso con acceso a las rutas multimedia.
- La aplicación solamente consume una API autenticada y muestra progreso.
- Ninguna operación modifica una fuente sin una aprobación explícita.

## Arquitectura

```text
Plezy Media Hub (Flutter, GPL-3.0)
  ├─ Plex / Jellyfin / Emby existentes
  └─ Gestionar ── HTTPS/Tailscale ──> media-orchestrator (Go)
                                        ├─ PostgreSQL
                                        ├─ FileBot
                                        ├─ TMDB / TVDB / OMDb
                                        └─ rutas locales del NUC o NAS
```

El relay propio de Plezy no forma parte de este flujo: el orquestador es un
servicio independiente, con su propio contrato API y ciclo de despliegue.

## Dominio PostgreSQL

- `library_roots`: raíces configuradas para escaneo y destino.
- `media_files`: archivo físico, SHA-256, MediaInfo, estado y ruta.
- `media_items`: contenido lógico aprobado (película o serie).
- `seasons` / `episodes`: estructura de serie y guía esperada.
- `identity_candidates`: coincidencias de proveedores por archivo o serie.
- `operations`: copy/move/verify con checkpoint, rollback y auditoría.
- `operation_events`: historial inmutable y visible en la app.

## Agrupación de UI

- Película: una ficha por archivo lógico con sus alternativas.
- Serie: serie → temporada → episodios disponibles/faltantes → versiones.
- Las alternativas de calidad de un episodio se muestran en el mismo episodio.

## Vertical slice MVP

1. Registrar una raíz de prueba.
2. Escanear y persistir archivos con SHA-256.
3. Identificar película y serie con FileBot/TVDB/TMDB.
4. Mostrar candidatos agrupados en Gestionar.
5. Aprobar una identidad.
6. Crear y ejecutar una copia verificada sin sobrescribir destino.
7. Mostrar estado, eventos y resultado en Plezy.

## Seguridad

- API exclusiva de red privada/Tailscale o HTTPS mediante proxy.
- Token por dispositivo, no clave de administración embebida en la APK.
- Secretos de TMDB/TVDB/OMDb sólo en el orquestador.
- Rutas autorizadas por allow-list; destino y fuente jamás son arbitrarios.
