# CBT Enterprise — Panduan Instalasi Lengkap

## Ringkasan Isi Project

| Komponen | Teknologi | Jumlah File |
|---|---|---|
| Backend API | Go 1.21 + Gin + GORM | 22 file `.go` |
| Frontend SPA | React 18 + TypeScript + Tailwind | 27 file `.tsx/.ts` |
| AI Service | Python 3.11 + FastAPI | 1 file `app.py` |
| Database | PostgreSQL 16 | 1 migration SQL |
| Config | Docker Compose + Nginx | 3 file config |

---

## Struktur Project

```
cbt-enterprise/
├── backend/
│   ├── cmd/api/main.go               # Entry point + DI
│   ├── internal/
│   │   ├── domain/                   # Entities + Interfaces
│   │   ├── usecase/                  # Business logic
│   │   ├── repository/               # GORM implementations
│   │   ├── delivery/http/            # Handlers + Middleware
│   │   └── infrastructure/           # Redis, Queue, WS, AI, DB
│   ├── pkg/                          # jwt, bcrypt, response
│   ├── migrations/001_init.sql
│   ├── .env.example
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── pages/
│   │   │   ├── auth/LoginPage.tsx
│   │   │   ├── exam/                 # ExamLobby, ExamRoom, ExamResult
│   │   │   ├── admin/                # AdminDashboard, ProctoringLog, ProfileSetting
│   │   │   ├── teacher/TeacherDashboard.tsx
│   │   │   └── guru/                 # BankSoal, BuatSoal, ManajemenPeserta, PenilaianEssay
│   │   ├── components/exam/          # SoalCard, Timer, ProctorCamera, Overlays
│   │   ├── hooks/                    # useAntiCheat, useProctoring, useWebSocket, useExamTimer
│   │   ├── store/                    # authStore, examStore (Zustand)
│   │   ├── services/api.ts           # Axios client
│   │   └── types/index.ts
│   ├── package.json
│   ├── vite.config.ts
│   ├── tailwind.config.js
│   ├── Dockerfile
│   └── nginx.conf
├── ai-service/
│   ├── app.py                        # FastAPI + face_recognition + YOLO
│   ├── requirements.txt
│   └── Dockerfile
├── docker-compose.yml
└── README.md
```

---

## Prasyarat

Pastikan tools berikut sudah terinstal:

| Tool | Versi Minimum | Cek |
|---|---|---|
| Go | 1.21 | `go version` |
| Node.js | 20 | `node --version` |
| Python | 3.11 | `python --version` |
| Docker | 24 | `docker --version` |
| Docker Compose | 2.x | `docker compose version` |
| PostgreSQL | 16 (atau via Docker) | `psql --version` |
| Redis | 7 (atau via Docker) | `redis-cli --version` |

---

## Cara 1 — Docker (Paling Cepat, Semua Otomatis)

Cara ini menjalankan semua service (PostgreSQL, Redis, Backend, Frontend, AI) sekaligus.

### Langkah 1: Extract dan masuk ke folder

```bash
unzip cbt-enterprise-fullcode.zip
cd cbt-enterprise
```

### Langkah 2: Sesuaikan environment (opsional)

Edit `docker-compose.yml` jika ingin mengubah password atau secret:

```yaml
# Di bagian service "backend", ubah:
environment:
  JWT_SECRET: ganti-dengan-secret-panjang-acak-di-production
  DB_DSN: postgres://cbt_user:cbt_pass@postgres:5432/cbt_db?sslmode=disable
```

### Langkah 3: Build dan jalankan

```bash
docker compose up --build -d
```

Proses build pertama ±5–10 menit (download dependencies).

### Langkah 4: Cek status semua service

```bash
docker compose ps
```

Output yang diharapkan:

```
NAME            STATUS
cbt_postgres    Up (healthy)
cbt_redis       Up (healthy)
cbt_ai          Up
cbt_backend     Up
cbt_frontend    Up
```

### Langkah 5: Akses aplikasi

| Service | URL |
|---|---|
| Frontend (React) | http://localhost |
| Backend API | http://localhost:8080/api/v1 |
| AI Service | http://localhost:5001 |

### Langkah 6: Buat akun pertama

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "nama": "Super Admin",
    "email": "admin@cbt.id",
    "password": "password123",
    "role": "admin"
  }'
```

### Perintah berguna Docker

```bash
# Lihat log backend
docker compose logs -f backend

# Lihat log AI service
docker compose logs -f ai

# Stop semua
docker compose down

# Stop dan hapus data
docker compose down -v

# Restart satu service
docker compose restart backend
```

---

## Cara 2 — Manual (Development)

Gunakan cara ini untuk development aktif dengan hot-reload.

---

### A. PostgreSQL

#### Opsi A1 — Docker (Rekomendasi)

```bash
docker run -d \
  --name cbt_postgres \
  -e POSTGRES_USER=cbt_user \
  -e POSTGRES_PASSWORD=cbt_pass \
  -e POSTGRES_DB=cbt_db \
  -p 5432:5432 \
  postgres:16-alpine
```

#### Opsi A2 — PostgreSQL Native (Ubuntu/Debian)

```bash
sudo apt update
sudo apt install -y postgresql postgresql-contrib

# Buat user dan database
sudo -u postgres psql << 'SQL'
CREATE USER cbt_user WITH PASSWORD 'cbt_pass';
CREATE DATABASE cbt_db OWNER cbt_user;
GRANT ALL PRIVILEGES ON DATABASE cbt_db TO cbt_user;
SQL
```

#### Opsi A3 — PostgreSQL Native (macOS)

```bash
brew install postgresql@16
brew services start postgresql@16

createuser -s cbt_user
psql postgres -c "ALTER USER cbt_user WITH PASSWORD 'cbt_pass';"
createdb -O cbt_user cbt_db
```

#### Opsi A4 — PostgreSQL Native (Windows)

1. Download installer dari https://www.postgresql.org/download/windows/
2. Install dengan password `cbt_pass` untuk user `postgres`
3. Buka pgAdmin atau psql, lalu jalankan:

```sql
CREATE USER cbt_user WITH PASSWORD 'cbt_pass';
CREATE DATABASE cbt_db OWNER cbt_user;
```

---

### B. Redis

#### Opsi B1 — Docker (Rekomendasi)

```bash
docker run -d \
  --name cbt_redis \
  -p 6379:6379 \
  redis:7-alpine
```

#### Opsi B2 — Redis Native (Ubuntu/Debian)

```bash
sudo apt install -y redis-server
sudo systemctl start redis-server
sudo systemctl enable redis-server
redis-cli ping   # harus jawab: PONG
```

#### Opsi B3 — Redis Native (macOS)

```bash
brew install redis
brew services start redis
redis-cli ping
```

#### Opsi B4 — Redis Native (Windows)

Download dari https://github.com/microsoftarchive/redis/releases atau gunakan WSL2 dengan cara Ubuntu di atas.

---

### C. Backend (Golang)

```bash
cd cbt-enterprise/backend

# 1. Copy dan edit environment
cp .env.example .env
```

Edit file `.env`:

```env
DB_DSN=postgres://cbt_user:cbt_pass@localhost:5432/cbt_db?sslmode=disable
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
JWT_SECRET=rahasia-panjang-acak-minimal-32-karakter
PORT=8080
GIN_MODE=debug
CORS_ORIGIN=http://localhost:5173
AI_SERVICE_URL=http://localhost:5001
```

```bash
# 2. Download dependencies
go mod download

# 3. Jalankan server (auto-migrate database)
go run ./cmd/api
```

Output yang diharapkan:

```
[GIN-debug] Listening and serving HTTP on :8080
🚀 CBT Server listening on :8080
Database migrated successfully
```

#### Build binary untuk production

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o cbt-v2 ./cmd/api
./cbt-v2
```

---

### D. AI Service (Python)

```bash
cd cbt-enterprise/ai-service

# 1. Buat virtual environment
python -m venv venv

# Aktivasi (Linux/macOS)
source venv/bin/activate

# Aktivasi (Windows)
venv\Scripts\activate

# 2. Install dependencies
pip install -r requirements.txt
```

> **Catatan:** `face_recognition` memerlukan `cmake` dan `dlib`.
>
> **Ubuntu/Debian:**
> ```bash
> sudo apt install -y cmake build-essential libopenblas-dev liblapack-dev
> ```
>
> **macOS:**
> ```bash
> brew install cmake
> ```
>
> **Windows:**
> Install Visual Studio Build Tools dan cmake dari https://cmake.org/download/

```bash
# 3. Jalankan AI service
python app.py
```

Output yang diharapkan:

```
INFO:     Started server process
INFO:     Uvicorn running on http://0.0.0.0:5001
```

> **Mode tanpa face_recognition** (untuk dev awal): Service tetap berjalan, tapi deteksi wajah mengembalikan data mock. Fitur anti-cheat berbasis tab/fullscreen tetap berfungsi.

---

### E. Frontend (React)

```bash
cd cbt-enterprise/frontend

# 1. Install dependencies
npm install

# 2. Jalankan dev server
npm run dev
```

Output yang diharapkan:

```
  VITE v5.x.x  ready in 500ms

  ➜  Local:   http://localhost:5173/
  ➜  Network: http://192.168.x.x:5173/
```

Buka http://localhost:5173 di browser.

---

## Cara 3 — Seeder Data Awal

Setelah backend berjalan, buat akun untuk setiap role:

```bash
BASE=http://localhost:8080/api/v1

# Admin
curl -s -X POST $BASE/auth/register -H "Content-Type: application/json" \
  -d '{"nama":"Super Admin","email":"admin@cbt.id","password":"admin123","role":"admin"}' | python -m json.tool

# Guru
curl -s -X POST $BASE/auth/register -H "Content-Type: application/json" \
  -d '{"nama":"Bu Sari Dewi","email":"guru@cbt.id","password":"guru123","role":"guru","nip":"198501012010","mapel":"Matematika"}' | python -m json.tool

# Peserta
curl -s -X POST $BASE/auth/register -H "Content-Type: application/json" \
  -d '{"nama":"Andi Pratama","email":"peserta@cbt.id","password":"peserta123","role":"peserta","nis":"2024001","kelas":"XII IPA 1"}' | python -m json.tool
```

---

## Ringkasan URL & Akun Default

| Halaman | URL | Akun |
|---|---|---|
| Login | http://localhost:5173/login | (sesuai seeder) |
| Admin Dashboard | http://localhost:5173/admin | admin@cbt.id |
| Guru Dashboard | http://localhost:5173/teacher | guru@cbt.id |
| Peserta Exam | http://localhost:5173/exam | peserta@cbt.id |
| API Health | http://localhost:8080/api/v1/auth/login | — |
| AI Health | http://localhost:5001/health | — |

---

## Cara 4 — Troubleshooting

### Backend gagal connect ke PostgreSQL

```
FATAL: password authentication failed for user "cbt_user"
```

Solusi:
```bash
# Cek koneksi manual
psql "postgres://cbt_user:cbt_pass@localhost:5432/cbt_db"

# Atau reset password
sudo -u postgres psql -c "ALTER USER cbt_user WITH PASSWORD 'cbt_pass';"
```

### Redis connection refused

```
dial tcp localhost:6379: connect: connection refused
```

Solusi:
```bash
# Cek status Redis
redis-cli ping

# Start Redis
sudo systemctl start redis  # Linux
brew services start redis   # macOS
```

### Frontend CORS error

Pastikan `CORS_ORIGIN` di `.env` backend sesuai URL frontend:
```env
CORS_ORIGIN=http://localhost:5173
```

### face_recognition tidak terinstall

AI service tetap jalan dalam mode mock. Untuk install di Ubuntu:
```bash
sudo apt install -y python3-dev cmake build-essential
pip install face-recognition
```

### Port 8080 sudah digunakan

```bash
# Cari proses di port 8080
lsof -i :8080       # Linux/macOS
netstat -ano | findstr :8080  # Windows

# Ganti port di .env backend
PORT=9090
```

### Docker: port already allocated

```bash
docker compose down
docker ps -a | grep cbt | awk '{print $1}' | xargs docker rm -f
docker compose up -d
```

---

## Cara 5 — Production Deployment

### Environment Variables Production

```env
GIN_MODE=release
JWT_SECRET=<random 64 karakter>
DB_DSN=postgres://user:pass@db-host:5432/cbt_db?sslmode=require
REDIS_ADDR=redis-host:6379
REDIS_PASSWORD=<redis password>
CORS_ORIGIN=https://domain-anda.com
AI_SERVICE_URL=http://ai-service:5001
```

### Dengan Nginx Reverse Proxy

```nginx
server {
    listen 443 ssl;
    server_name domain-anda.com;

    ssl_certificate     /etc/ssl/certs/cert.pem;
    ssl_certificate_key /etc/ssl/private/key.pem;

    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 3600s;
    }

    location / {
        root /var/www/cbt-frontend/dist;
        try_files $uri /index.html;
    }
}
```

### Build frontend untuk production

```bash
cd frontend
npm run build
# Output di: frontend/dist/
```

---

## Alur Penggunaan Sistem

```
1. Admin/Guru login
2. Guru buat ujian di /teacher → buat soal di bank soal
3. Guru tambah soal ke ujian → tambah peserta
4. Guru aktifkan ujian (status: aktif)
5. Peserta login → lihat daftar ujian → klik Mulai
6. Sistem generate attempt + acak soal
7. Peserta kerjakan ujian (anti-cheat aktif, kamera proctoring berjalan)
8. Peserta klik Selesai → scoring worker berjalan di background
9. Admin monitor realtime di /admin (WebSocket)
10. Guru nilai essay di /teacher/essay
```

---

## API Quick Reference

```bash
TOKEN="eyJhbG..."  # Dari response login

# Login
curl -X POST $BASE/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"guru@cbt.id","password":"guru123"}'

# Buat ujian
curl -X POST $BASE/ujian \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"judul":"UTS Matematika","durasi_menit":60,"acak_soal":true,"acak_opsi":true}'

# Mulai ujian (peserta)
curl -X POST $BASE/exam/start \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"ujian_id":"uuid-ujian"}'

# Simpan jawaban
curl -X POST $BASE/exam/answer \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"attempt_id":"uuid","soal_id":"uuid","opsi_id":"uuid","sisa_detik":3540}'

# Selesaikan ujian
curl -X POST $BASE/exam/finish \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"attempt_id":"uuid"}'
```

---

## WebSocket Events

```
ws://localhost:8080/api/v1/ws?attempt_id=UUID&token=JWT

Event diterima peserta:
  exam_paused    → freeze UI, stop timer
  exam_resumed   → lanjutkan
  exam_finished  → redirect ke hasil

Event diterima admin:
  cheating_detected  → tampil alert
  face_alert         → detail AI detection
  exam_paused        → tombol Resume muncul
```
