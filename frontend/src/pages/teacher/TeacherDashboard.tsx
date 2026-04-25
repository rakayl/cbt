import { useEffect, useState } from 'react'
import { ujianAPI } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { clsx } from 'clsx'

interface Ujian {
  id: string; judul: string; deskripsi: string
  durasi_menit: number; status: string; created_at: string
}

type Tab = 'ujian' | 'create'

export default function TeacherDashboard() {
  const { user, logout }           = useAuthStore()
  const [tab, setTab]              = useState<Tab>('ujian')
  const [ujians, setUjians]        = useState<Ujian[]>([])
  const [loading, setLoading]      = useState(true)

  // Create form state
  const [form, setForm] = useState({
    judul: '', deskripsi: '', durasi_menit: 60,
    acak_soal: true, acak_opsi: true, max_tab_switch: 3,
  })
  const [saving, setSaving]  = useState(false)
  const [saveMsg, setSaveMsg] = useState('')

  const loadUjians = () => {
    setLoading(true)
    ujianAPI.list().then(r => setUjians(r.data.data ?? [])).finally(() => setLoading(false))
  }

  useEffect(() => { loadUjians() }, [])

  const createUjian = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true); setSaveMsg('')
    try {
      await ujianAPI.create(form as any)
      setSaveMsg('Ujian berhasil dibuat!')
      setTab('ujian')
      loadUjians()
      setForm({ judul: '', deskripsi: '', durasi_menit: 60, acak_soal: true, acak_opsi: true, max_tab_switch: 3 })
    } catch (err: any) {
      setSaveMsg(err.response?.data?.error ?? 'Gagal membuat ujian')
    } finally {
      setSaving(false)
    }
  }

  const setStatus = (id: string, status: string) => {
    ujianAPI.setStatus(id, status).then(loadUjians).catch(() => {})
  }

  return (
    <div className="min-h-screen bg-slate-50">
      {/* Header */}
      <header className="bg-white border-b border-slate-200 px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-blue-600 flex items-center justify-center">
            <svg className="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"/>
            </svg>
          </div>
          <span className="font-bold text-slate-800">CBT — Guru</span>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm text-slate-600">{user?.nama}</span>
          <button onClick={logout} className="btn-secondary text-xs px-3 py-1.5">Keluar</button>
        </div>
      </header>

      <main className="max-w-4xl mx-auto px-4 py-8">
        {/* Tabs */}
        <div className="flex gap-1 bg-slate-100 p-1 rounded-xl mb-6 w-fit">
          {(['ujian', 'create'] as Tab[]).map(t => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={clsx(
                'px-4 py-2 rounded-lg text-sm font-medium transition-all duration-150',
                tab === t ? 'bg-white text-slate-800 shadow-sm' : 'text-slate-500 hover:text-slate-700'
              )}
            >
              {t === 'ujian' ? 'Daftar Ujian' : '+ Buat Ujian'}
            </button>
          ))}
        </div>

        {tab === 'ujian' && (
          <div>
            <div className="flex items-center justify-between mb-4">
              <h1 className="text-lg font-bold text-slate-900">Ujian Saya</h1>
              <span className="text-sm text-slate-500">{ujians.length} ujian</span>
            </div>

            {loading ? (
              <div className="space-y-3">
                {[1,2].map(i => <div key={i} className="card p-5 animate-pulse"><div className="h-4 bg-slate-200 rounded w-1/2 mb-2"/><div className="h-3 bg-slate-100 rounded w-3/4"/></div>)}
              </div>
            ) : ujians.length === 0 ? (
              <div className="card p-10 text-center">
                <p className="text-slate-400">Belum ada ujian. Buat ujian pertama Anda.</p>
                <button onClick={() => setTab('create')} className="btn-primary mt-4">+ Buat Ujian</button>
              </div>
            ) : (
              <div className="space-y-3">
                {ujians.map(u => (
                  <div key={u.id} className="card p-5">
                    <div className="flex items-center justify-between gap-4">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <h2 className="font-semibold text-slate-800 truncate">{u.judul}</h2>
                          <span className={`badge ${
                            u.status === 'aktif'   ? 'badge-green' :
                            u.status === 'selesai' ? 'badge-gray' : 'badge-yellow'
                          }`}>{u.status}</span>
                        </div>
                        <p className="text-xs text-slate-400 font-mono">{u.id}</p>
                      </div>
                      <div className="flex items-center gap-2 flex-shrink-0">
                        {u.status === 'draft' && (
                          <button onClick={() => setStatus(u.id, 'aktif')} className="btn-primary text-xs px-3 py-1.5">
                            Aktifkan
                          </button>
                        )}
                        {u.status === 'aktif' && (
                          <button onClick={() => setStatus(u.id, 'selesai')} className="btn-secondary text-xs px-3 py-1.5">
                            Selesaikan
                          </button>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {tab === 'create' && (
          <div>
            <h1 className="text-lg font-bold text-slate-900 mb-5">Buat Ujian Baru</h1>
            <div className="card p-6">
              {saveMsg && (
                <div className={clsx(
                  'mb-4 px-4 py-3 rounded-lg text-sm',
                  saveMsg.includes('berhasil') ? 'bg-emerald-50 text-emerald-700 border border-emerald-200' : 'bg-red-50 text-red-700 border border-red-200'
                )}>{saveMsg}</div>
              )}
              <form onSubmit={createUjian} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1.5">Judul Ujian *</label>
                  <input
                    value={form.judul} onChange={e => setForm(f => ({...f, judul: e.target.value}))}
                    placeholder="Ujian Tengah Semester Matematika" required
                    className="input"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1.5">Deskripsi</label>
                  <textarea
                    value={form.deskripsi} onChange={e => setForm(f => ({...f, deskripsi: e.target.value}))}
                    rows={3} placeholder="Deskripsi singkat ujian..."
                    className="input resize-none"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1.5">Durasi (menit) *</label>
                  <input
                    type="number" min={1} value={form.durasi_menit}
                    onChange={e => setForm(f => ({...f, durasi_menit: Number(e.target.value)}))}
                    className="input"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1.5">Maks. Tab Switch</label>
                  <input
                    type="number" min={1} max={10} value={form.max_tab_switch}
                    onChange={e => setForm(f => ({...f, max_tab_switch: Number(e.target.value)}))}
                    className="input"
                  />
                </div>
                <div className="flex gap-6 pt-1">
                  {[['acak_soal','Acak Soal'],['acak_opsi','Acak Opsi']].map(([key, label]) => (
                    <label key={key} className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={(form as any)[key]}
                        onChange={e => setForm(f => ({...f, [key]: e.target.checked}))}
                        className="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-400"
                      />
                      <span className="text-sm text-slate-700">{label}</span>
                    </label>
                  ))}
                </div>
                <div className="pt-2">
                  <button type="submit" disabled={saving} className="btn-primary">
                    {saving ? 'Menyimpan...' : 'Buat Ujian'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}
