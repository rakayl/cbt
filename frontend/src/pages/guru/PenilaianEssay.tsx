import { useEffect, useState } from 'react'
import { clsx } from 'clsx'
import api, { ujianAPI, examAPI } from '@/services/api'

interface EssayItem {
  attempt_id: string
  peserta_nama: string
  soal_id: string
  pertanyaan: string
  jawaban: string
  poin_max: number
  nilai?: number
  status: 'pending' | 'dinilai'
}

export default function PenilaianEssay() {
  const [ujians, setUjians]   = useState<any[]>([])
  const [items, setItems]     = useState<EssayItem[]>([])
  const [selUjian, setSelUjian] = useState('')
  const [loading, setLoading] = useState(false)
  const [scores, setScores]   = useState<Record<string, {nilai:string; catatan:string}>>({})
  const [saved, setSaved]     = useState<Record<string, boolean>>({})

  useEffect(() => { ujianAPI.list().then(r => setUjians(r.data.data ?? [])) }, [])

  const load = async (id: string) => {
    setSelUjian(id); setLoading(true); setItems([])
    try {
      const r = await api.get(`/admin/ujian/${id}/essay-items`)
      setItems(r.data.data ?? [])
    } finally { setLoading(false) }
  }

  const save = async (item: EssayItem) => {
    const sc = scores[`${item.attempt_id}:${item.soal_id}`]
    if (!sc?.nilai) return
    try {
      await api.post('/admin/essay/grade', {
        attempt_id: item.attempt_id,
        soal_id:    item.soal_id,
        nilai:      parseFloat(sc.nilai),
        catatan:    sc.catatan ?? '',
      })
      setSaved(s => ({ ...s, [`${item.attempt_id}:${item.soal_id}`]: true }))
      setItems(prev => prev.map(i =>
        i.attempt_id === item.attempt_id && i.soal_id === item.soal_id
          ? { ...i, nilai: parseFloat(sc.nilai), status: 'dinilai' }
          : i
      ))
    } catch (e: any) { alert(e.response?.data?.error ?? 'Gagal menyimpan nilai') }
  }

  const key = (item: EssayItem) => `${item.attempt_id}:${item.soal_id}`
  const pending = items.filter(i => i.status === 'pending').length
  const initials = (nama: string) => nama.split(' ').map(w => w[0]).join('').slice(0,2).toUpperCase()
  const avatarColors = ['bg-purple-100 text-purple-700','bg-blue-100 text-blue-700','bg-emerald-100 text-emerald-700','bg-amber-100 text-amber-700']

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="bg-white border-b border-slate-200 px-5 py-3 flex items-center gap-3">
        <div className="w-7 h-7 bg-blue-600 rounded-lg flex items-center justify-center">
          <svg className="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01"/>
          </svg>
        </div>
        <span className="font-semibold text-slate-800">Penilaian Essay</span>
        {pending > 0 && <span className="badge badge-yellow">{pending} pending</span>}
        <div className="ml-auto">
          <select value={selUjian} onChange={e => load(e.target.value)} className="input w-52">
            <option value="">-- Pilih Ujian --</option>
            {ujians.map(u => <option key={u.id} value={u.id}>{u.judul}</option>)}
          </select>
        </div>
      </header>

      <main className="max-w-3xl mx-auto p-5 space-y-4">
        {!selUjian ? (
          <div className="card p-12 text-center"><p className="text-slate-400">Pilih ujian untuk mulai menilai essay</p></div>
        ) : loading ? (
          <div className="space-y-3">{[1,2].map(i=><div key={i} className="card p-5 animate-pulse"><div className="h-3 bg-slate-200 rounded w-2/3 mb-2"/><div className="h-16 bg-slate-100 rounded"/></div>)}</div>
        ) : items.length === 0 ? (
          <div className="card p-12 text-center"><p className="text-slate-400">Tidak ada soal essay pada ujian ini</p></div>
        ) : (
          items.map((item, idx) => (
            <div key={key(item)} className={clsx('card p-4', item.status === 'dinilai' && 'opacity-70')}>
              <div className="flex items-center gap-3 mb-3">
                <div className={clsx('w-8 h-8 rounded-full flex items-center justify-center text-xs font-semibold flex-shrink-0', avatarColors[idx % avatarColors.length])}>
                  {initials(item.peserta_nama)}
                </div>
                <div className="flex-1">
                  <p className="text-sm font-medium text-slate-800">{item.peserta_nama}</p>
                  <p className="text-xs text-slate-400 font-mono">{item.attempt_id.slice(0,12)}…</p>
                </div>
                <span className={clsx('badge', item.status === 'dinilai' ? 'badge-green' : 'badge-yellow')}>
                  {item.status === 'dinilai' ? `Dinilai: ${item.nilai}/${item.poin_max}` : 'Pending'}
                </span>
              </div>

              <div className="bg-slate-50 rounded-lg p-3 mb-2">
                <p className="text-xs font-medium text-slate-500 mb-1">Pertanyaan:</p>
                <p className="text-sm text-slate-700">{item.pertanyaan}</p>
              </div>

              <div className="bg-blue-50 border border-blue-100 rounded-lg p-3 mb-3">
                <p className="text-xs font-medium text-blue-700 mb-1">Jawaban Peserta:</p>
                <p className="text-sm text-blue-800 leading-relaxed">{item.jawaban || <em className="text-blue-400">Tidak dijawab</em>}</p>
              </div>

              {item.status !== 'dinilai' && (
                <div className="flex items-center gap-3">
                  <div className="flex items-center gap-2">
                    <label className="text-xs font-medium text-slate-600">Nilai (0–{item.poin_max}):</label>
                    <input type="number" min={0} max={item.poin_max}
                      value={scores[key(item)]?.nilai ?? ''}
                      onChange={e => setScores(s => ({ ...s, [key(item)]: { ...s[key(item)], nilai: e.target.value } }))}
                      className="input w-16 text-center"/>
                  </div>
                  <input type="text" placeholder="Catatan (opsional)"
                    value={scores[key(item)]?.catatan ?? ''}
                    onChange={e => setScores(s => ({ ...s, [key(item)]: { ...s[key(item)], catatan: e.target.value } }))}
                    className="input flex-1"/>
                  <button onClick={() => save(item)} className="btn-primary flex-shrink-0">
                    {saved[key(item)] ? '✓ Tersimpan' : 'Simpan Nilai'}
                  </button>
                </div>
              )}
            </div>
          ))
        )}
      </main>
    </div>
  )
}
