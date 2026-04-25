import { useState } from 'react'
import { useAuthStore } from '@/store/authStore'
import api from '@/services/api'

export default function ProfileSetting() {
  const { user }     = useAuthStore()
  const [msg, setMsg] = useState('')
  const [form, setForm] = useState({ nama: user?.nama ?? '', email: user?.email ?? '', password: '' })
  const [settings, setSettings] = useState({
    ai_proctoring: true, yolo_detection: false,
    auto_pause: true, realtime_notif: true,
    cheating_threshold: 5,
  })
  const [saving, setSaving] = useState(false)

  const saveProfile = async () => {
    setSaving(true)
    try {
      await api.put('/profile', form)
      setMsg('Profil berhasil disimpan!')
    } catch { setMsg('Gagal menyimpan') }
    finally { setSaving(false); setTimeout(() => setMsg(''), 2000) }
  }

  const tog = (key: keyof typeof settings) => {
    setSettings(s => ({ ...s, [key]: !s[key as keyof typeof settings] }))
  }

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="bg-white border-b border-slate-200 px-5 py-3 flex items-center gap-3">
        <div className="w-7 h-7 bg-blue-600 rounded-lg flex items-center justify-center">
          <svg className="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
          </svg>
        </div>
        <span className="font-semibold text-slate-800">Profil &amp; Pengaturan</span>
      </header>

      <main className="max-w-3xl mx-auto p-5 grid grid-cols-2 gap-5 items-start">
        {/* Left: Profile */}
        <div className="space-y-4">
          <div className="card p-5">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-4">Profil Akun</p>
            <div className="flex items-center gap-3 mb-4">
              <div className="w-12 h-12 rounded-full bg-purple-100 text-purple-700 flex items-center justify-center text-lg font-bold">
                {user?.nama?.split(' ').map(w=>w[0]).join('').slice(0,2).toUpperCase()}
              </div>
              <div>
                <p className="font-medium text-slate-800">{user?.nama}</p>
                <p className="text-xs text-slate-400">{user?.email}</p>
                <span className="badge badge-blue capitalize mt-1">{user?.role}</span>
              </div>
            </div>
            {msg && <div className={`mb-3 p-2 rounded text-xs ${msg.includes('berhasil') ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-700'}`}>{msg}</div>}
            <div className="space-y-3">
              {[{ k:'nama', label:'Nama Lengkap', type:'text' }, { k:'email', label:'Email', type:'email' }, { k:'password', label:'Password Baru', type:'password' }].map(f => (
                <div key={f.k}>
                  <label className="block text-xs font-medium text-slate-600 mb-1">{f.label}</label>
                  <input type={f.type} value={(form as any)[f.k]} placeholder={f.k==='password' ? 'Kosongkan jika tidak diubah' : ''}
                    onChange={e => setForm(prev => ({...prev, [f.k]: e.target.value}))} className="input"/>
                </div>
              ))}
              <button onClick={saveProfile} disabled={saving} className="btn-primary">{saving ? 'Menyimpan...' : 'Simpan Perubahan'}</button>
            </div>
          </div>

          {/* System Status */}
          <div className="card p-5">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Status Sistem</p>
            <div className="space-y-2">
              {[{ label:'API Server', ok:true },{ label:'Redis Cache', ok:true },{ label:'AI Service', ok:true },{ label:'Scoring Worker', ok:true }].map(s => (
                <div key={s.label} className="flex items-center justify-between text-sm">
                  <span className="text-slate-600">{s.label}</span>
                  <span className={`badge ${s.ok ? 'badge-green' : 'badge-red'}`}>{s.ok ? 'Online' : 'Offline'}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Right: Settings */}
        <div className="space-y-4">
          <div className="card p-5">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Pengaturan Sistem</p>
            <div className="divide-y divide-slate-100">
              {[
                { key:'ai_proctoring',   label:'AI Proctoring Aktif',         desc:'Deteksi wajah dan verifikasi identitas' },
                { key:'yolo_detection',  label:'YOLO Object Detection',        desc:'Deteksi benda (HP, buku) di frame' },
                { key:'auto_pause',      label:'Auto-pause on Violation',      desc:'Hentikan ujian otomatis saat kecurangan' },
                { key:'realtime_notif',  label:'Notifikasi Realtime',          desc:'Push alert via WebSocket ke admin' },
              ].map(item => (
                <div key={item.key} className="flex items-center justify-between py-3">
                  <div>
                    <p className="text-sm font-medium text-slate-700">{item.label}</p>
                    <p className="text-xs text-slate-400">{item.desc}</p>
                  </div>
                  <button
                    onClick={() => tog(item.key as any)}
                    className={`relative w-10 h-5 rounded-full transition-colors flex-shrink-0 ${(settings as any)[item.key] ? 'bg-blue-600' : 'bg-slate-300'}`}
                  >
                    <span className={`absolute top-0.5 w-4 h-4 bg-white rounded-full shadow transition-all ${(settings as any)[item.key] ? 'left-5' : 'left-0.5'}`}/>
                  </button>
                </div>
              ))}
            </div>
          </div>

          <div className="card p-5">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Threshold Cheating Score</p>
            <p className="text-xs text-slate-500 mb-3">Ujian di-pause otomatis jika cheating score mencapai nilai ini</p>
            <div className="flex items-center gap-3">
              <input type="range" min={1} max={10} value={settings.cheating_threshold}
                onChange={e => setSettings(s => ({...s, cheating_threshold: +e.target.value}))}
                className="flex-1 accent-blue-600"/>
              <span className="text-lg font-bold text-slate-800 w-8 text-center">{settings.cheating_threshold}</span>
            </div>
            <div className="flex justify-between text-xs text-slate-400 mt-1"><span>Ketat (1)</span><span>Longgar (10)</span></div>
          </div>

          <div className="card p-5">
            <p className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Kapasitas Sistem</p>
            {[
              { label:'Redis Cache', used:42, total:512, unit:'MB' },
              { label:'PostgreSQL',  used:1200, total:10240, unit:'MB' },
              { label:'Queue Jobs',  used:0, total:1000, unit:'' },
            ].map(s => (
              <div key={s.label} className="mb-3">
                <div className="flex justify-between text-xs mb-1">
                  <span className="text-slate-500">{s.label}</span>
                  <span className="text-slate-700 font-medium">{s.used}{s.unit} / {s.total}{s.unit}</span>
                </div>
                <div className="h-1.5 bg-slate-100 rounded-full overflow-hidden">
                  <div className="h-full bg-blue-500 rounded-full transition-all" style={{ width: `${Math.min(s.used/s.total*100, 100)}%` }}/>
                </div>
              </div>
            ))}
          </div>
        </div>
      </main>
    </div>
  )
}
