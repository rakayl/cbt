// ─── AUTH ─────────────────────────────────────────────────────────────────────

export type Role = 'admin' | 'guru' | 'peserta'

export interface User {
  id: string
  nama: string
  email: string
  role: Role
}

// ─── UJIAN ───────────────────────────────────────────────────────────────────

export type StatusUjian = 'draft' | 'aktif' | 'selesai'
export type TipeSoal = 'pilihan_ganda' | 'essay'
export type StatusAttempt = 'ongoing' | 'paused' | 'selesai'

export interface SoalOpsi {
  id: string
  soal_id: string
  teks: string
  is_benar?: boolean
  urutan: number
}

export interface Soal {
  id: string
  pertanyaan: string
  tipe: TipeSoal
  poin: number
  opsi: SoalOpsi[]
}

export interface AttemptSoal {
  id: string
  attempt_id: string
  soal_id: string
  soal: Soal
  urutan: number
  opsi_order: string // JSON array of opsi IDs
}

export interface Ujian {
  id: string
  judul: string
  deskripsi: string
  durasi_menit: number
  status: StatusUjian
  acak_soal: boolean
  acak_opsi: boolean
  max_tab_switch: number
}

export interface Attempt {
  id: string
  ujian_id: string
  peserta_id: string
  status: StatusAttempt
  mulai_at: string
  sisa_detik: number
  cheating_score: number
}

export interface Jawaban {
  soal_id: string
  opsi_id?: string
  teks_jawaban?: string
}

// ─── WEBSOCKET EVENTS ─────────────────────────────────────────────────────────

export type WSEventType =
  | 'heartbeat'
  | 'answer'
  | 'tab_switch'
  | 'fullscreen_exit'
  | 'cheating_detected'
  | 'exam_paused'
  | 'exam_finished'
  | 'exam_resumed'
  | 'face_alert'

export interface WSMessage {
  event: WSEventType
  attempt_id?: string
  peserta_id?: string
  payload?: Record<string, unknown>
  reason?: string
}

// ─── ADMIN ───────────────────────────────────────────────────────────────────

export interface MonitoringAttempt {
  attempt_id: string
  peserta_nama: string
  ujian_judul: string
  status: StatusAttempt
  cheating_score: number
  is_online: boolean
}
