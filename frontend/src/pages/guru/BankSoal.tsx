import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { clsx } from 'clsx'
import api from '@/services/api'

interface Opsi { id:string; teks:string; is_benar:boolean; urutan:number }
interface Soal {
  id:string; pertanyaan:string; tipe:'pilihan_ganda'|'essay'
  poin:number; opsi:Opsi[]; mapel?:{nama:string}
}

export default function BankSoal() {
  const [soals, setSoals]     = useState<Soal[]>([])
  const [loading, setLoading] = useState(true)
  const [filter, setFilter]   = useState('')
  const [tipe, setTipe]       = useState('')
  const navigate              = useNavigate()
  const letters = 'ABCDE'

  const load = () => {
    setLoading(true)
    api.get('/soal').then(r => setSoals(r.data.data ?? [])).finally(() => setLoading(false))
  }
  useEffect(load, [])

  const filtered = soals.filter(s =>
    (!filter || s.pertanyaan.toLowerCase().includes(filter.toLowerCase())) &&
    (!tipe   || s.tipe === tipe)
  )

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="bg-white border-b border-slate-200 px-5 py-3 flex items-center gap-3">
        <div className="w-7 h-7 bg-blue-600 rounded-lg flex items-center justify-center">
          <svg className="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}><path strokeLinecap="round" strokeLinejoin="round" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/></svg>
        </div>
        <span className="font-semibold text-slate-800">Bank Soal</span>
        <div className="ml-auto flex items-center gap-2">
          <input value={filter} onChange={e=>setFilter(e.target.value)} placeholder="Cari soal..." className="input w-48"/>
          <select value={tipe} onChange={e=>setTipe(e.target.value)} className="input w-36">
            <option value="">Semua Tipe</option>
            <option value="pilihan_ganda">Pilihan Ganda</option>
            <option value="essay">Essay</option>
          </select>
          <button onClick={() => navigate('/teacher/soal/baru')} className="btn-primary">+ Soal Baru</button>
        </div>
      </header>
      <main className="max-w-4xl mx-auto p-5">
        {loading ? (
          <div className="space-y-3">{[1,2,3].map(i=><div key={i} className="card p-4 animate-pulse"><div className="h-3 bg-slate-200 rounded w-3/4 mb-2"/></div>)}</div>
        ) : (
          <div className="card overflow-hidden divide-y divide-slate-100">
            {filtered.map((s, idx) => (
              <div key={s.id} className="p-4 hover:bg-slate-50">
                <div className="flex items-start gap-3">
                  <span className="w-6 h-6 rounded bg-slate-100 flex items-center justify-center text-xs font-semibold text-slate-500 flex-shrink-0 mt-0.5">{idx+1}</span>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-slate-800 leading-relaxed mb-2">{s.pertanyaan}</p>
                    <div className="flex items-center gap-2 mb-2">
                      <span className={clsx('badge', s.tipe==='pilihan_ganda' ? 'badge-blue' : 'badge-yellow')}>{s.tipe==='pilihan_ganda' ? 'Pilihan Ganda' : 'Essay'}</span>
                      <span className="badge badge-gray">{s.poin} poin</span>
                    </div>
                    {s.tipe==='pilihan_ganda' && (
                      <div className="flex flex-wrap gap-1.5">
                        {s.opsi.map((o,i) => (
                          <span key={o.id} className={clsx('text-xs px-2 py-0.5 rounded border', o.is_benar ? 'bg-emerald-50 border-emerald-200 text-emerald-700 font-medium' : 'bg-slate-50 border-slate-200 text-slate-500')}>
                            {letters[i]}. {o.teks}{o.is_benar && ' ✓'}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                  <div className="flex gap-2 flex-shrink-0">
                    <button onClick={() => navigate('/teacher/soal/'+s.id)} className="btn-secondary text-xs px-2.5 py-1">Edit</button>
                    <button onClick={() => api.delete('/soal/'+s.id).then(load)} className="btn-danger text-xs px-2.5 py-1">Hapus</button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}
