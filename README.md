# CBT Enterprise — Platform Ujian Online

Sistem backend + frontend CBT (Computer Based Test) enterprise, dibangun dengan Golang (Clean Architecture) + React.

---

## Stack Teknologi

| Layer        | Teknologi                              |
|--------------|----------------------------------------|
| Backend      | Go 1.21, Gin, GORM, gorilla/websocket  |
| Database     | PostgreSQL 16                          |
| Cache/Queue  | Redis 7 (LPUSH/BRPOP pattern)          |
| Frontend     | React 18, Vite, TypeScript, Tailwind   |
| AI Service   | Python 3.11, FastAPI, face_recognition, YOLO |
| Container    | Docker + Docker Compose                |

---

## Struktur Project

```
cbt-enterprise/
├── backend/                    # Golang API
│   ├── cmd/api/main.go         # Entry point, DI, router
│   ├── internal/
│   │   ├── domain/             # Entities + Repository interfaces
│   │   ├── usecase/            # Business logic
│   │   ├── repository/         # GORM implementations
│   │   ├── delivery/http/      # Handlers + Middleware
│   │   └── infrastructure/     # DB, Redis, Queue, WebSocket, AI client
│   └── pkg/                    # jwt, bcrypt, response
├── frontend/                   # React SPA
│   └── src/
│       ├── pages/              # auth/, exam/, admin/, teacher/
│       ├── components/exam/    # SoalCard, Timer, ProctorCamera, Overlays
│       ├── hooks/              # useAntiCheat, useProctoring, useWebSocket, useExamTimer
│       ├── store/              # Zustand: authStore, examStore
│       └── services/api.ts     # Axios API client
├── ai-service/                 # Python FastAPI
│   ├── app.py                  # Face detection, verification, YOLO
│   └── requirements.txt
└── docker-compose.yml
```

---

## Quick Start

### Development

```bash
# 1. Start infrastructure
docker compose up postgres redis -d

# 2. Backend
cd backend
cp .env.example .env
go run ./cmd/api

# 3. Frontend
cd frontend
npm install
npm run dev

# 4. AI Service
cd ai-service
pip install -r requirements.txt
python app.py
```

### Production (Docker)

```bash
docker compose up --build -d
```

Frontend tersedia di `http://localhost:80`  
Backend API di `http://localhost:8080`  
AI Service di `http://localhost:5001`

---

## API Reference

### Auth
| Method | Path                  | Body                            |
|--------|-----------------------|---------------------------------|
| POST   | /api/v1/auth/register | nama, email, password, role, …  |
| POST   | /api/v1/auth/login    | email, password                 |

### Exam (Peserta)
| Method | Path                     | Description               |
|--------|--------------------------|---------------------------|
| POST   | /api/v1/exam/start       | Mulai ujian, dapat soal   |
| POST   | /api/v1/exam/answer      | Autosave jawaban           |
| POST   | /api/v1/exam/finish      | Selesaikan ujian           |
| POST   | /api/v1/exam/violation   | Laporkan pelanggaran       |
| POST   | /api/v1/exam/proctor     | Kirim frame webcam         |
| GET    | /api/v1/exam/attempt/:id | Status attempt             |

### Ujian Management (Guru)
| Method | Path                     | Description        |
|--------|--------------------------|--------------------|
| POST   | /api/v1/ujian            | Buat ujian baru    |
| GET    | /api/v1/ujian            | List ujian         |
| PUT    | /api/v1/ujian/:id/status | Set status ujian   |
| POST   | /api/v1/ujian/:id/soal   | Tambah soal        |
| POST   | /api/v1/ujian/:id/peserta| Tambah peserta     |

### Admin
| Method | Path                            | Description           |
|--------|---------------------------------|-----------------------|
| GET    | /api/v1/admin/online            | Peserta online        |
| GET    | /api/v1/admin/ujian/:id/results | Hasil ujian           |
| POST   | /api/v1/admin/attempt/:id/unpause | Resume attempt      |

### WebSocket
```
ws://host/api/v1/ws?attempt_id=...&token=...
```

Events: `heartbeat`, `answer`, `tab_switch`, `cheating_detected`, `exam_paused`, `exam_resumed`, `face_alert`

---

## Fitur Anti-Cheat

| Fitur               | Implementasi                                  |
|---------------------|-----------------------------------------------|
| Tab switch          | `visibilitychange` event → laporkan ke backend |
| Fullscreen exit     | `fullscreenchange` event → laporkan            |
| Copy/paste disable  | Override clipboard API, `exam-mode` CSS class  |
| Klik kanan disable  | `contextmenu` event blocked                    |
| Auto-pause          | `cheating_score >= 5` → status = paused        |
| AI face detection   | Frame tiap 2.5 detik → Python service          |
| Face verification   | Embedding cosine distance > 0.6 = mismatch     |
| Multiple faces      | `face_count > 1` = violation                   |
| YOLO object         | Deteksi HP/buku/laptop di frame                |
| Cooldown buffer     | 3x deteksi berturut + 2s cooldown              |

---

## Arsitektur Clean Architecture

```
Handler (HTTP/WS)
    ↓
Usecase (business logic, framework-free)
    ↓
Repository Interface (domain)
    ↓
Repository Implementation (GORM)
    ↓
Database (PostgreSQL)

Usecase ←→ Redis (cache, queue)
Usecase ←→ WebSocket Hub (realtime)
Usecase ←→ AI Client (HTTP to Python)
```

---

## Environment Variables

```env
DB_DSN=postgres://user:pass@host/db?sslmode=disable
REDIS_ADDR=localhost:6379
JWT_SECRET=your-secret-here
PORT=8080
AI_SERVICE_URL=http://localhost:5001
CORS_ORIGIN=http://localhost:5173
```
