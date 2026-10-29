# Meetups Backend (Go)

Reescritura en **Go** del backend original de la plataforma **Meetups** (anteriormente en Node.js), manteniendo compatibilidad 100% con el frontend en React (`meetups-frontend`): mismos endpoints, formato de respuestas JSON, autenticación JWT (`HS256`) y servido de imágenes estáticas.

Construido sobre la librería estándar de Go (`net/http`, `database/sql`) con dependencias mínimas (`github.com/go-sql-driver/mysql` y `golang.org/x/crypto/bcrypt`).

## Estructura del proyecto

```text
.
├── main.go                  # Punto de entrada del servidor HTTP
├── cmd/
│   └── dbsetup/main.go      # CLI para inicializar y poblar la base de datos MySQL
├── internal/
│   ├── api/                 # Router, middlewares (CORS, JWT), handlers de usuarios, meetups e inscripciones
│   └── db/                  # Conexión a MySQL y carga de variables de entorno (.env)
├── public/                  # Imágenes por defecto (avatar y meetup)
├── uploads/                 # Imágenes subidas por usuarios
└── api.http                 # Colección de peticiones HTTP de ejemplo
```

## Requisitos

- **Go** 1.22+
- **MySQL** en ejecución

## Configuración

Copia `.env.example` a `.env` y ajusta las variables:

```env
PORT=
DATABASE_HOST=
DATABASE_PORT=
DATABASE_USER=
DATABASE_PASSWORD=
DATABASE_NAME=
JWT_SECRET=
UPLOADS_DIR=
```

## Base de datos (creación y datos de prueba)

El comando `cmd/dbsetup` recrea las tablas (`users`, `meetups`, `inscriptions`) e inserta datos de prueba (contraseña por defecto de los usuarios de seed: `12345`):

```bash
# Crear tablas y poblar con datos de prueba
go run ./cmd/dbsetup all

# O por separado:
go run ./cmd/dbsetup init      # Solo recrear base de datos y tablas
go run ./cmd/dbsetup populate  # Solo insertar datos de prueba
go run ./cmd/dbsetup ping      # Verificar conexión a MySQL
```

## Ejecución

```bash
# Modo desarrollo
go run .

# Compilar y ejecutar binario
go build -o meetups-server .
./meetups-server

# Ejecutar tests
go test ./...
```

## Endpoints

| Método | Ruta | Auth | Descripción |
| :--- | :--- | :---: | :--- |
| `POST` | `/users` | No | Registro de usuario (`multipart/form-data` o JSON) |
| `POST` | `/login` | No | Inicio de sesión (devuelve token JWT) |
| `GET` | `/users` | Sí | Obtener perfil del usuario autenticado |
| `DELETE` | `/users/{id}` | No | Eliminar usuario por ID |
| `GET` | `/meetups` | No | Listar meetups futuras ordenadas por fecha |
| `GET` | `/meetup/{id}` | No | Detalle de una meetup e inscritos |
| `POST` | `/meetups` | Sí | Crear meetup (`multipart/form-data` o JSON) |
| `DELETE` | `/meetup/{id}` | No | Eliminar meetup por ID |
| `POST` | `/inscription/{id}` | Sí | Inscribirse a una meetup |
| `DELETE` | `/inscription/{id}` | Sí | Cancelar inscripción a una meetup |
