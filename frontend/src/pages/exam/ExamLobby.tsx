import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ujianAPI } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { format } from 'date-fns'
import { id as idLocale } from 'date-fns/locale'

interface Ujian {
  id: string
  judul: string
  deskripsi: string
  durasi_menit: number
  status: string
  tanggal_mulai: string | null
  tanggal_selesai: string | null
}

export default function ExamLobby() {
  const [ujians, setUjians]   = useState<Ujian[]>([])
  const [loading, setLoading] = useState(true)
  const { user, logout }      = useAuthStore()
  const navigate              = useNavigate()

  useEffect(() => {
    ujianAPI.list()
      .then(r => setUjians(r.data.data ?? []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const startExam = (ujianID: string) => {
    navigate(`/exam/room?ujian_id=${ujianID}`)
  }

  return (
    <div className="min-h-screen bg-slate-50">
      {/* Header */}
      <header className="bg-white border-b border-slate-200 px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-blue-600 flex items-center justify-center">
            <svg className="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2"/>
            </svg>
          </div>
          <span className="font-bold text-slate-800">CBT Enterprise</span>
        </div>
        <div className="flex items-center gap-3">
          <div className="text-right">
            <p className="text-sm font-medium text-slate-800">{user?.nama}</p>
            <p className="text-xs text-slate-400 capitalize">{user?.role}</p>
          </div>
          <button onClick={logout} className="btn-secondary text-xs px-3 py-1.5">
            Keluar
          </button>
        </div>
      </header>

      <main className="max-w-3xl mx-auto px-4 py-8">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-slate-900">Ujian Tersedia</h1>
          <p className="text-slate-500 text-sm mt-1">Pilih ujian yang ingin Anda kerjakan</p>
        </div>

        {loading ? (
          <div className="space-y-3">
            {[1,2,3].map(i => (
              <div key={i} className="card p-5 animate-pulse">
                <div className="h-4 bg-slate-200 rounded w-1/2 mb-3"/>
                <div className="h-3 bg-slate-100 rounded w-3/4"/>
              </div>
            ))}
          </div>
        ) : ujians.length === 0 ? (
          <div className="card p-12 text-center">
            <svg className="w-12 h-12 text-slate-300 mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2"/>
            </svg>
            <p className="text-slate-500 font-medium">Belum ada ujian tersedia</p>
          </div>
        ) : (
          <div className="space-y-3">
            {ujians.map(ujian => (
              <div key={ujian.id} className="card p-5 hover:shadow-md transition-all duration-200">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <h2 className="font-semibold text-slate-800 truncate">{ujian.judul}</h2>
                      <span className={`badge ${
                        ujian.status === 'aktif' ? 'badge-green' :
                        ujian.status === 'selesai' ? 'badge-gray' : 'badge-yellow'
                      }`}>
                        {ujian.status === 'aktif' ? 'Aktif' : ujian.status === 'selesai' ? 'Selesai' : 'Draft'}
                      </span>
                    </div>
                    {ujian.deskripsi && (
                      <p className="text-sm text-slate-500 mb-3 line-clamp-2">{ujian.deskripsi}</p>
                    )}
                    <div className="flex items-center gap-4 text-xs text-slate-400">
                      <span className="flex items-center gap-1">
                        <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                          <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
                        </svg>
                        {ujian.durasi_menit} menit
                      </span>
                      {ujian.tanggal_mulai && (
                        <span className="flex items-center gap-1">
                          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                            <path strokeLinecap="round" strokeLinejoin="round" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/>
                          </svg>
                          {format(new Date(ujian.tanggal_mulai), 'dd MMM yyyy HH:mm', { locale: idLocale })}
                        </span>
                      )}
                    </div>
                  </div>
                  <button
                    onClick={() => startExam(ujian.id)}
                    disabled={ujian.status !== 'aktif'}
                    className="btn-primary flex-shrink-0"
                  >
                    Mulai
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7"/>
                    </svg>
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}
