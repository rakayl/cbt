import { useEffect, useState, useCallback } from 'react'
import { adminAPI, ujianAPI } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useWebSocket } from '@/hooks/useWebSocket'
import { clsx } from 'clsx'

interface OnlineData {
  online_count: number
  attempt_ids: string[]
}

interface Alert {
  id: string
  attempt_id: string
  event: string
  detail: string
  ts: number
}

export default function AdminDashboard() {
  const { user, logout }          = useAuthStore()
  const [online, setOnline]       = useState<OnlineData>({ online_count: 0, attempt_ids: [] })
  const [alerts, setAlerts]       = useState<Alert[]>([])
  const [selectedUjian, setSelectedUjian] = useState('')
  const [results, setResults]     = useState<any[]>([])
  const [ujians, setUjians]       = useState<any[]>([])
  const [activeTab, setActiveTab] = useState<'monitor'|'results'>('monitor')

  const refreshOnline = useCallback(() => {
    adminAPI.getOnline().then(r => setOnline(r.data.data)).catch(() => {})
  }, [])

  useEffect(() => {
    refreshOnline()
    ujianAPI.list().then(r => setUjians(r.data.data ?? [])).catch(() => {})
    const iv = setInterval(refreshOnline, 10000)
    return () => clearInterval(iv)
  }, [refreshOnline])

  // WebSocket for realtime alerts
  useWebSocket({
    role: 'admin',
    enabled: true,
    onMessage: (msg) => {
      if (['cheating_detected', 'face_alert', 'exam_paused'].includes(msg.event)) {
        setAlerts(prev => [{
          id: Math.random().toString(36).slice(2),
          attempt_id: msg.attempt_id ?? '',
          event: msg.event,
          detail: JSON.stringify(msg.payload ?? {}),
          ts: Date.now(),
        }, ...prev].slice(0, 50))
      }
    },
  })

  const loadResults = (ujianID: string) => {
    setSelectedUjian(ujianID)
    adminAPI.getResults(ujianID).then(r => setResults(r.data.data ?? [])).catch(() => {})
  }

  const unpause = (attemptID: string) => {
    adminAPI.unpause(attemptID).then(() => {
      setAlerts(prev => prev.filter(a => a.attempt_id !== attemptID))
    }).catch(() => {})
  }

  const eventLabel: Record<string, string> = {
    cheating_detected: 'Kecurangan',
    face_alert: 'Wajah Alert',
    exam_paused: 'Ujian Dihentikan',
  }

  const eventColor: Record<string, string> = {
    cheating_detected: 'border-red-400 bg-red-50',
    face_alert: 'border-amber-400 bg-amber-50',
    exam_paused: 'border-red-500 bg-red-100',
  }

  return (
    <div className="min-h-screen bg-slate-50">
      {/* Sidebar + content layout */}
      <div className="flex h-screen overflow-hidden">

        {/* Sidebar */}
        <aside className="w-56 bg-slate-900 flex flex-col flex-shrink-0">
          <div className="px-4 py-5 border-b border-slate-800">
            <div className="flex items-center gap-2.5">
              <div className="w-7 h-7 rounded-lg bg-blue-600 flex items-center justify-center flex-shrink-0">
                <svg className="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2"/>
                </svg>
              </div>
              <span className="text-sm font-bold text-white">CBT Admin</span>
            </div>
          </div>

          <nav className="flex-1 px-3 py-4 space-y-1">
            {[
              { key: 'monitor', label: 'Live Monitor', icon: 'M15 12a3 3 0 11-6 0 3 3 0 016 0z M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z' },
              { key: 'results', label: 'Hasil Ujian',  icon: 'M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z' },
            ].map(item => (
              <button
                key={item.key}
                onClick={() => setActiveTab(item.key as any)}
                className={clsx(
                  'w-full flex items-center gap-2.5 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                  activeTab === item.key
                    ? 'bg-blue-600 text-white'
                    : 'text-slate-400 hover:text-white hover:bg-slate-800'
                )}
              >
                <svg className="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d={item.icon}/>
                </svg>
                {item.label}
              </button>
            ))}
          </nav>

          <div className="px-3 py-4 border-t border-slate-800">
            <div className="flex items-center gap-2 mb-3">
              <div className="w-7 h-7 rounded-full bg-slate-700 flex items-center justify-center text-xs font-bold text-slate-300">
                {user?.nama?.[0]?.toUpperCase()}
              </div>
              <div className="min-w-0">
                <p className="text-xs font-medium text-slate-300 truncate">{user?.nama}</p>
                <p className="text-xs text-slate-500 capitalize">{user?.role}</p>
              </div>
            </div>
            <button onClick={logout} className="w-full text-xs text-slate-500 hover:text-slate-300 transition-colors text-left px-1">
              Keluar →
            </button>
          </div>
        </aside>

        {/* Main */}
        <main className="flex-1 overflow-y-auto">
          {activeTab === 'monitor' && (
            <div className="p-6">
              <div className="mb-6">
                <h1 className="text-xl font-bold text-slate-900">Live Monitor</h1>
                <p className="text-sm text-slate-500 mt-0.5">Pemantauan ujian secara realtime</p>
              </div>

              {/* Stats */}
              <div className="grid grid-cols-3 gap-4 mb-6">
                <div className="card p-4">
                  <p className="text-xs text-slate-500 mb-1">Peserta Online</p>
                  <p className="text-3xl font-bold text-slate-800">{online.online_count}</p>
                  <div className="flex items-center gap-1 mt-1">
                    <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"/>
                    <span className="text-xs text-emerald-600">Live</span>
                  </div>
                </div>
                <div className="card p-4">
                  <p className="text-xs text-slate-500 mb-1">Alert Aktif</p>
                  <p className="text-3xl font-bold text-red-600">{alerts.length}</p>
                  <p className="text-xs text-slate-400 mt-1">Perlu perhatian</p>
                </div>
                <div className="card p-4">
                  <p className="text-xs text-slate-500 mb-1">Ujian Aktif</p>
                  <p className="text-3xl font-bold text-blue-600">
                    {ujians.filter(u => u.status === 'aktif').length}
                  </p>
                  <p className="text-xs text-slate-400 mt-1">Sedang berlangsung</p>
                </div>
              </div>

              {/* Alert feed */}
              <div className="card">
                <div className="px-5 py-4 border-b border-slate-100 flex items-center justify-between">
                  <h2 className="font-semibold text-slate-800 text-sm">Alert Realtime</h2>
                  {alerts.length > 0 && (
                    <button onClick={() => setAlerts([])} className="text-xs text-slate-400 hover:text-slate-600">
                      Hapus semua
                    </button>
                  )}
                </div>

                {alerts.length === 0 ? (
                  <div className="px-5 py-10 text-center">
                    <svg className="w-8 h-8 text-slate-300 mx-auto mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
                    </svg>
                    <p className="text-sm text-slate-400">Tidak ada alert. Semua kondusif.</p>
                  </div>
                ) : (
                  <div className="divide-y divide-slate-100 max-h-96 overflow-y-auto">
                    {alerts.map(alert => (
                      <div key={alert.id} className={clsx('px-5 py-3.5 border-l-4 flex items-start justify-between gap-4', eventColor[alert.event] ?? 'border-slate-300 bg-white')}>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-0.5">
                            <span className="text-xs font-semibold text-slate-700">{eventLabel[alert.event] ?? alert.event}</span>
                            <span className="text-xs text-slate-400 font-mono">{alert.attempt_id.slice(0, 8)}…</span>
                          </div>
                          <p className="text-xs text-slate-500 truncate">{alert.detail}</p>
                          <p className="text-xs text-slate-400 mt-0.5">{new Date(alert.ts).toLocaleTimeString('id-ID')}</p>
                        </div>
                        {alert.event === 'exam_paused' && (
                          <button
                            onClick={() => unpause(alert.attempt_id)}
                            className="btn-secondary text-xs px-2.5 py-1 flex-shrink-0"
                          >
                            Resume
                          </button>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Online attempts */}
              {online.attempt_ids?.length > 0 && (
                <div className="card mt-4">
                  <div className="px-5 py-4 border-b border-slate-100">
                    <h2 className="font-semibold text-slate-800 text-sm">Attempt Aktif</h2>
                  </div>
                  <div className="p-4 flex flex-wrap gap-2">
                    {online.attempt_ids.map(id => (
                      <span key={id} className="badge badge-blue font-mono text-xs">{id.slice(0, 12)}…</span>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {activeTab === 'results' && (
            <div className="p-6">
              <div className="mb-6">
                <h1 className="text-xl font-bold text-slate-900">Hasil Ujian</h1>
                <p className="text-sm text-slate-500 mt-0.5">Rekap nilai dan status peserta</p>
              </div>

              {/* Select ujian */}
              <div className="card p-4 mb-4">
                <label className="text-sm font-medium text-slate-700 mb-2 block">Pilih Ujian</label>
                <select
                  value={selectedUjian}
                  onChange={e => loadResults(e.target.value)}
                  className="input"
                >
                  <option value="">-- Pilih ujian --</option>
                  {ujians.map(u => (
                    <option key={u.id} value={u.id}>{u.judul}</option>
                  ))}
                </select>
              </div>

              {results.length > 0 && (
                <div className="card overflow-hidden">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="bg-slate-50 border-b border-slate-200">
                        <th className="px-4 py-3 text-left text-xs font-semibold text-slate-500 uppercase tracking-wider">Peserta</th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-slate-500 uppercase tracking-wider">Status</th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-slate-500 uppercase tracking-wider">Cheating Score</th>
                        <th className="px-4 py-3 text-left text-xs font-semibold text-slate-500 uppercase tracking-wider">Mulai</th>
                        <th className="px-4 py-3 text-right text-xs font-semibold text-slate-500 uppercase tracking-wider">Aksi</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {results.map(r => (
                        <tr key={r.id} className="hover:bg-slate-50 transition-colors">
                          <td className="px-4 py-3 font-mono text-xs text-slate-600">{r.peserta_id?.slice(0, 12)}…</td>
                          <td className="px-4 py-3">
                            <span className={`badge ${
                              r.status === 'ongoing' ? 'badge-green' :
                              r.status === 'paused'  ? 'badge-red' :
                              r.status === 'selesai' ? 'badge-gray' : 'badge-yellow'
                            }`}>
                              {r.status}
                            </span>
                          </td>
                          <td className="px-4 py-3">
                            <span className={clsx('font-semibold', r.cheating_score > 0 ? 'text-red-600' : 'text-slate-600')}>
                              {r.cheating_score}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-slate-500 text-xs">
                            {new Date(r.mulai_at).toLocaleString('id-ID')}
                          </td>
                          <td className="px-4 py-3 text-right">
                            {r.status === 'paused' && (
                              <button onClick={() => unpause(r.id)} className="btn-secondary text-xs px-2.5 py-1">
                                Resume
                              </button>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}
        </main>
      </div>
    </div>
  )
}
