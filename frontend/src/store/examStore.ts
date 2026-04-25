import { create } from 'zustand'

export interface SoalOpsi { id:string; teks:string; is_benar?:boolean; urutan:number }
export interface Soal { id:string; pertanyaan:string; tipe:'pilihan_ganda'|'essay'; opsi:SoalOpsi[]; poin:number }
export interface AttemptSoal { id:string; attempt_id:string; soal_id:string; soal:Soal; urutan:number; opsi_order:string }

export type ExamStatus = 'idle'|'ongoing'|'paused'|'finished'|'loading'

interface ExamState {
  attemptID: string | null
  ujianID:   string | null
  soalList:  AttemptSoal[]
  jawabans:  Record<string, string>      // soal_id → opsi_id or teks
  currentIdx: number
  sisaDetik:  number
  status:     ExamStatus
  cheatingScore: number
  pauseReason:   string

  setAttempt:  (id: string, ujianID: string) => void
  setSoalList: (list: AttemptSoal[]) => void
  setJawaban:  (soalID: string, value: string) => void
  setTimer:    (s: number) => void
  setStatus:   (s: ExamStatus) => void
  setCurrentIdx:(i: number) => void
  setPause:    (reason: string) => void
  incrementCheating: () => void
  reset:       () => void
}

export const useExamStore = create<ExamState>((set) => ({
  attemptID: null, ujianID: null, soalList: [], jawabans: {},
  currentIdx: 0, sisaDetik: 0, status: 'idle', cheatingScore: 0, pauseReason: '',

  setAttempt:    (id, ujianID) => set({ attemptID: id, ujianID }),
  setSoalList:   (list) => set({ soalList: list }),
  setJawaban:    (soalID, value) => set(s => ({ jawabans: { ...s.jawabans, [soalID]: value } })),
  setTimer:      (s) => set({ sisaDetik: s }),
  setStatus:     (status) => set({ status }),
  setCurrentIdx: (i) => set({ currentIdx: i }),
  setPause:      (reason) => set({ status: 'paused', pauseReason: reason }),
  incrementCheating: () => set(s => ({ cheatingScore: s.cheatingScore + 1 })),
  reset:         () => set({ attemptID: null, ujianID: null, soalList: [], jawabans: {}, currentIdx: 0, sisaDetik: 0, status: 'idle', cheatingScore: 0 }),
}))
