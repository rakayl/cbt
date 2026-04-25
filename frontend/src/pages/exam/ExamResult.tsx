import { useEffect, useState } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { examAPI } from '@/services/api'
import { clsx } from 'clsx'

interface ResultData {
  attempt: {
    id: string
    status: string
    mulai_at: string
    selesai_at: string
    cheating_score: number
    tab_switch_count: number
  }
  jawabans: Array<{ soal_id: string; opsi_id?: string; teks_jawaban?: string }>
}

export default function ExamResult() {
  const [params]  = useSearchParams()
  const navigate  = useNavigate()
  const attemptID = params.get('attempt_id') ?? ''
  const [data, setData]     = useState<ResultData | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!attemptID) { navigate('/exam'); return }
    examAPI.getAttempt(attemptID)
      .then(r => setData(r.data.data))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [attemptID, navigate])

  if (loading) return (
    <div className="min-h-screen bg-slate-50 flex items-center justify-center">
      <svg className="w-8 h-8 text-blue-600 animate-spin" fill="none" viewBox="0 0 24 24">
        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"/>
        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
      </svg>
    </div>
  )

  const answeredCount = data?.jawabans?.length ?? 0
  const durationMs = data?.attempt
    ? new Date(data.attempt.selesai_at).getTime() - new Date(data.attempt.mulai_at).getTime()
    : 0
  const durationMin = Math.round(durationMs / 60000)

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-blue-50 flex items-center justify-center p-4">
      <div className="max-w-md w-full animate-slide-up">

        {/* Success icon */}
        <div className="text-center mb-6">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-emerald-100 mb-4">
            <svg className="w-8 h-8 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
            </svg>
          </div>
          <h1 className="text-2xl font-bold text-slate-900">Ujian Selesai!</h1>
          <p className="text-slate-500 text-sm mt-1">Jawaban Anda telah tersimpan dan sedang diproses</p>
        </div>

        {/* Stats card */}
        <div className="card p-6 mb-4">
          <h2 className="text-sm font-semibold text-slate-500 uppercase tracking-wider mb-4">Ringkasan</h2>
          <div className="grid grid-cols-2 gap-4">
            {[
              { label: 'Soal Dijawab', value: answeredCount, color: 'text-blue-600' },
              { label: 'Durasi', value: `${durationMin} mnt`, color: 'text-slate-800' },
              {
                label: 'Pelanggaran',
                value: data?.attempt?.cheating_score ?? 0,
                color: (data?.attempt?.cheating_score ?? 0) > 0 ? 'text-red-600' : 'text-emerald-600',
              },
              {
                label: 'Status',
                value: data?.attempt?.status === 'selesai' ? 'Selesai' : data?.attempt?.status ?? '-',
                color: 'text-slate-800',
              },
            ].map(stat => (
              <div key={stat.label} className="bg-slate-50 rounded-xl p-3">
                <p className="text-xs text-slate-500 mb-1">{stat.label}</p>
                <p className={clsx('text-xl font-bold', stat.color)}>{stat.value}</p>
              </div>
            ))}
          </div>
        </div>

        {/* Info box */}
        <div className="bg-blue-50 border border-blue-200 rounded-xl p-4 mb-5">
          <p className="text-sm text-blue-800 leading-relaxed">
            <strong>Penilaian sedang diproses.</strong> Hasil nilai akan tersedia setelah pengawas menyelesaikan
            penilaian (khususnya soal essay). Hubungi guru/pengawas untuk informasi lebih lanjut.
          </p>
        </div>

        <button
          onClick={() => navigate('/exam')}
          className="btn-primary w-full justify-center"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"/>
          </svg>
          Kembali ke Beranda
        </button>
      </div>
    </div>
  )
}
