import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '@/services/api'
import { useAuthStore } from '@/store/authStore'

interface Opsi { teks:string; is_benar:boolean }

export default function BuatSoal() {
  const navigate          = useNavigate()
  const { user }          = useAuthStore()
  const [saving, setSaving] = useState(false)
  const [msg, setMsg]     = useState('')
  const letters = 'ABCDE'

  const [form, setForm] = useState({
    pertanyaan: '', tipe: 'pilihan_ganda', poin: 2, pembahasan: ''
  })
  const [opsi, setOpsi] = useState<Opsi[]>([
    { teks: '', is_benar: true },
    { teks: '', is_benar: false },
    { teks: '', is_benar: false },
    { teks: '', is_benar: false },
  ])

  const setBenar = (idx: number) => setOpsi(o => o.map((op, i) => ({ ...op, is_benar: i === idx })))
  const setOpsiTeks = (idx: number, val: string) => setOpsi(o => o.map((op, i) => i === idx ? { ...op, teks: val } : op))
  const addOpsi = () => opsi.length < 5 && setOpsi(o => [...o, { teks: '', is_benar: false }])
  const delOpsi = (idx: number) => opsi.length > 2 && setOpsi(o => o.filter((_, i) => i !== idx))

  const save = async () => {
    if (!form.pertanyaan.trim()) { setMsg('Pertanyaan wajib diisi'); return }
    setSaving(true); setMsg('')
    try {
      await api.post('/soal', {
        ...form,
        guru_id: user?.id,
        opsi: form.tipe === 'pilihan_ganda' ? opsi.map((o, i) => ({ ...o, urutan: i+1 })) : [],
      })
      setMsg('Soal berhasil disimpan!')
      setTimeout(() => navigate('/teacher/bank-soal'), 1200)
    } catch (e: any) {
      setMsg(e.response?.data?.error ?? 'Gagal menyimpan soal')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="bg-white border-b border-slate-200 px-5 py-3 flex items-center gap-3">
        <button onClick={() => navigate(-1)} className="btn-secondary text-xs px-2.5 py-1">← Kembali</button>
        <span className="font-semibold text-slate-800">Buat Soal Baru</span>
      </header>
      <main className="max-w-4xl mx-auto p-5 grid grid-cols-2 gap-5 items-start">
        <div className="space-y-4">
          <div className="card p-4 space-y-3">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Informasi Soal</p>
            <div><label className="block text-xs font-medium text-slate-600 mb-1">Tipe Soal</label>
              <div className="flex gap-4">
                {['pilihan_ganda','essay'].map(t => (
                  <label key={t} className="flex items-center gap-2 text-sm cursor-pointer">
                    <input type="radio" name="tipe" value={t} checked={form.tipe===t} onChange={() => setForm(f=>({...f,tipe:t}))} className="accent-blue-600"/>
                    {t === 'pilihan_ganda' ? 'Pilihan Ganda' : 'Essay'}
                  </label>
                ))}
              </div>
            </div>
            <div><label className="block text-xs font-medium text-slate-600 mb-1">Poin</label>
              <input type="number" min={1} value={form.poin} onChange={e=>setForm(f=>({...f,poin:+e.target.value}))} className="input w-24"/>
            </div>
          </div>
          <div className="card p-4">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">Pertanyaan</p>
            <textarea value={form.pertanyaan} onChange={e=>setForm(f=>({...f,pertanyaan:e.target.value}))}
              rows={5} placeholder="Tulis pertanyaan di sini..." className="input resize-none"/>
          </div>
          <div className="card p-4">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">Pembahasan (opsional)</p>
            <textarea value={form.pembahasan} onChange={e=>setForm(f=>({...f,pembahasan:e.target.value}))}
              rows={3} placeholder="Tulis pembahasan / kunci jawaban..." className="input resize-none"/>
          </div>
        </div>

        <div className="space-y-4">
          {form.tipe === 'pilihan_ganda' && (
            <div className="card p-4">
              <div className="flex items-center justify-between mb-3">
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider">Opsi Jawaban</p>
                <button onClick={addOpsi} className="btn-secondary text-xs px-2 py-1">+ Opsi</button>
              </div>
              <div className="space-y-2">
                {opsi.map((o, i) => (
                  <div key={i} className="flex items-center gap-2">
                    <input type="radio" name="benar" checked={o.is_benar} onChange={() => setBenar(i)} className="accent-emerald-600 flex-shrink-0" title="Jawaban benar"/>
                    <span className={`w-6 h-6 rounded flex items-center justify-center text-xs font-bold flex-shrink-0 ${o.is_benar ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-500'}`}>{letters[i]}</span>
                    <input type="text" value={o.teks} onChange={e=>setOpsiTeks(i,e.target.value)} placeholder={`Opsi ${letters[i]}`} className="input flex-1"/>
                    {opsi.length > 2 && <button onClick={() => delOpsi(i)} className="text-red-400 hover:text-red-600 text-lg leading-none flex-shrink-0">×</button>}
                  </div>
                ))}
              </div>
              <p className="text-xs text-slate-400 mt-2">Klik radio untuk menandai jawaban benar</p>
            </div>
          )}

          {msg && <div className={`p-3 rounded-lg text-sm ${msg.includes('berhasil') ? 'bg-emerald-50 text-emerald-700 border border-emerald-200' : 'bg-red-50 text-red-700 border border-red-200'}`}>{msg}</div>}

          <div className="flex gap-2">
            <button onClick={save} disabled={saving} className="btn-primary">{saving ? 'Menyimpan...' : 'Simpan Soal'}</button>
            <button onClick={() => navigate(-1)} className="btn-secondary">Batal</button>
          </div>
        </div>
      </main>
    </div>
  )
}
