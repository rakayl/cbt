import { useEffect, useState } from 'react'
import { clsx } from 'clsx'
import api from '@/services/api'

interface Peserta {
  id: string
  user: { id: string; nama: string; email: string }
  nis: string
  kelas: string
}

export default function ManajemenPeserta() {
  const [list, setList]       = useState<Peserta[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch]   = useState('')
  const [showModal, setShowModal] = useState(false)
  const [saving, setSaving]   = useState(false)
  const [msg, setMsg]         = useState('')
  const [form, setForm]       = useState({ nama:'', email:'', password:'', nis:'', kelas:'XII IPA 1' })

  const load = () => {
    setLoading(true)
    api.get('/peserta').then(r => setList(r.data.data ?? [])).finally(() => setLoading(false))
  }
  useEffect(load, [])

  const filtered = list.filter(p =>
    !search ||
    p.user.nama.toLowerCase().includes(search.toLowerCase()) ||
    p.nis.includes(search)
  )

  const save = async () => {
    setSaving(true); setMsg('')
    try {
      await api.post('/auth/register', { ...form, role: 'peserta' })
      setMsg('Peserta berhasil ditambahkan!')
      load()
      setTimeout(() => { setShowModal(false); setMsg('') }, 1200)
    } catch (e: any) {
      setMsg(e.response?.data?.error ?? 'Gagal menambah peserta')
    } finally { setSaving(false) }
  }

  const initials = (nama: string) => nama.split(' ').map(w => w[0]).join('').slice(0, 2).toUpperCase()
  const colors   = ['bg-purple-100 text-purple-700', 'bg-blue-100 text-blue-700',
                    'bg-emerald-100 text-emerald-700', 'bg-amber-100 text-amber-700',
                    'bg-rose-100 text-rose-700']

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="bg-white border-b border-slate-200 px-5 py-3 flex items-center gap-3">
        <div className="w-7 h-7 bg-blue-600 rounded-lg flex items-center justify-center">
          <svg className="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z"/>
          </svg>
        </div>
        <span className="font-semibold text-slate-800">Manajemen Peserta</span>
        <div className="ml-auto flex items-center gap-2">
          <input value={search} onChange={e => setSearch(e.target.value)} placeholder="Cari nama / NIS..." className="input w-52"/>
          <button onClick={() => setShowModal(true)} className="btn-primary">+ Tambah Peserta</button>
        </div>
      </header>

      <main className="max-w-4xl mx-auto p-5">
        {/* Stats */}
        <div className="grid grid-cols-4 gap-3 mb-5">
          {[
            { label: 'Total Peserta', val: list.length, color: 'text-slate-800' },
            { label: 'Kelas XII IPA', val: list.filter(p => p.kelas.includes('IPA')).length, color: 'text-blue-600' },
            { label: 'Kelas XII IPS', val: list.filter(p => p.kelas.includes('IPS')).length, color: 'text-emerald-600' },
          ].map(s => (
            <div key={s.label} className="card p-3 text-center">
              <p className={`text-2xl font-bold ${s.color}`}>{s.val}</p>
              <p className="text-xs text-slate-500 mt-0.5">{s.label}</p>
            </div>
          ))}
        </div>

        {loading ? (
          <div className="space-y-2">{[1,2,3,4].map(i => <div key={i} className="card p-4 animate-pulse"><div className="h-3 bg-slate-200 rounded w-1/2"/></div>)}</div>
        ) : (
          <div className="card overflow-hidden divide-y divide-slate-100">
            {filtered.map((p, idx) => (
              <div key={p.id} className="flex items-center gap-3 px-4 py-3 hover:bg-slate-50 transition-colors">
                <div className={clsx('w-8 h-8 rounded-full flex items-center justify-center text-xs font-semibold flex-shrink-0', colors[idx % colors.length])}>
                  {initials(p.user.nama)}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-slate-800">{p.user.nama}</p>
                  <p className="text-xs text-slate-400">NIS: {p.nis} · {p.kelas} · {p.user.email}</p>
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  <span className="badge badge-gray">{p.kelas}</span>
                  <button className="btn-secondary text-xs px-2.5 py-1">Detail</button>
                </div>
              </div>
            ))}
            {filtered.length === 0 && (
              <div className="p-10 text-center text-slate-400 text-sm">Tidak ada peserta ditemukan</div>
            )}
          </div>
        )}
      </main>

      {/* Add Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl w-full max-w-md shadow-2xl animate-slide-up overflow-hidden">
            <div className="px-5 py-4 border-b border-slate-100 flex items-center justify-between">
              <h3 className="font-semibold text-slate-800">Tambah Peserta Baru</h3>
              <button onClick={() => setShowModal(false)} className="text-slate-400 hover:text-slate-600 text-xl leading-none">×</button>
            </div>
            <div className="p-5 space-y-3">
              {msg && <div className={`p-3 rounded-lg text-sm ${msg.includes('berhasil') ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-700'}`}>{msg}</div>}
              {[
                { label: 'Nama Lengkap', key: 'nama', type: 'text', placeholder: 'Nama lengkap peserta' },
                { label: 'Email', key: 'email', type: 'email', placeholder: 'peserta@sekolah.id' },
                { label: 'Password', key: 'password', type: 'password', placeholder: '••••••••' },
                { label: 'NIS', key: 'nis', type: 'text', placeholder: '2024001' },
              ].map(f => (
                <div key={f.key}>
                  <label className="block text-xs font-medium text-slate-600 mb-1">{f.label}</label>
                  <input type={f.type} value={(form as any)[f.key]} placeholder={f.placeholder}
                    onChange={e => setForm(prev => ({...prev, [f.key]: e.target.value}))}
                    className="input"/>
                </div>
              ))}
              <div>
                <label className="block text-xs font-medium text-slate-600 mb-1">Kelas</label>
                <select value={form.kelas} onChange={e => setForm(f => ({...f, kelas: e.target.value}))} className="input">
                  {['XII IPA 1','XII IPA 2','XII IPS 1','XII IPS 2'].map(k => <option key={k}>{k}</option>)}
                </select>
              </div>
              <div className="flex gap-2 pt-2">
                <button onClick={save} disabled={saving} className="btn-primary flex-1 justify-center">
                  {saving ? 'Menyimpan...' : 'Simpan'}
                </button>
                <button onClick={() => setShowModal(false)} className="btn-secondary">Batal</button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
