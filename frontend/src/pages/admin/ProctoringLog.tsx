import { useEffect, useState } from 'react'
import { clsx } from 'clsx'
import api from '@/services/api'
import { adminAPI } from '@/services/api'

interface Log {
  id: number
  attempt_id: string
  event_type: string
  confidence: number
  created_at: string
}

const EVENT_COLOR: Record<string, string> = {
  multiple_faces: 'bg-red-500',
  face_mismatch:  'bg-red-600',
  no_face:        'bg-purple-500',
  tab_switch:     'bg-amber-500',
  fullscreen_exit:'bg-amber-400',
  exam_paused:    'bg-red-700',
}

const EVENT_LABEL: Record<string, string> = {
  multiple_faces: 'Banyak Wajah',
  face_mismatch:  'Wajah Berbeda',
  no_face:        'Wajah Tidak Terdeteksi',
  tab_switch:     'Tab Switch',
  fullscreen_exit:'Keluar Fullscreen',
  exam_paused:    'Ujian Di-pause',
}

export default function ProctoringLog() {
  const [ujians, setUjians]   = useState<any[]>([])
  const [logs, setLogs]       = useState<Log[]>([])
  const [attempts, setAttempts] = useState<any[]>([])
  const [selUjian, setSelUjian] = useState('')
  const [filter, setFilter]   = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    api.get('/ujian').then(r => setUjians(r.data.data ?? []))
  }, [])

  const loadLogs = async (ujianID: string) => {
    setSelUjian(ujianID)
    setLoading(true)
    try {
      const r = await adminAPI.getResults(ujianID)
      setAttempts(r.data.data ?? [])
      // Collect proctoring logs for all attempts
      const allLogs: Log[] = []
      for (const a of (r.data.data ?? []).slice(0, 10)) {
        try {
          const lr = await api.get(`/admin/attempt/${a.id}/logs`)
          allLogs.push(...(lr.data.data ?? []))
        } catch { /* skip */ }
      }
      setLogs(allLogs)
    } finally { setLoading(false) }
  }

  const stats = {
    total:   logs.length,
    face:    logs.filter(l => l.event_type.includes('face')).length,
    tab:     logs.filter(l => l.event_type === 'tab_switch').length,
    paused:  attempts.filter((a: any) => a.status === 'paused').length,
  }

  const filtered = logs.filter(l => !filter || l.event_type === filter)

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="bg-white border-b border-slate-200 px-5 py-3 flex items-center gap-3">
        <div className="w-7 h-7 bg-blue-600 rounded-lg flex items-center justify-center">
          <svg className="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M15 10l4.553-2.069A1 1 0 0121 8.82v6.36a1 1 0 01-1.447.894L15 14M3 8a2 2 0 00-2 2v4a2 2 0 002 2h9a2 2 0 002-2v-4a2 2 0 00-2-2H3z"/>
          </svg>
        </div>
        <span className="font-semibold text-slate-800">Proctoring Log</span>
        <div className="ml-auto flex items-center gap-2">
          <select value={selUjian} onChange={e => loadLogs(e.target.value)} className="input w-52">
            <option value="">-- Pilih Ujian --</option>
            {ujians.map(u => <option key={u.id} value={u.id}>{u.judul}</option>)}
          </select>
          <select value={filter} onChange={e => setFilter(e.target.value)} className="input w-40">
            <option value="">Semua Event</option>
            {Object.keys(EVENT_LABEL).map(k => <option key={k} value={k}>{EVENT_LABEL[k]}</option>)}
          </select>
        </div>
      </header>

      <main className="max-w-5xl mx-auto p-5">
        {/* Stats */}
        <div className="grid grid-cols-4 gap-3 mb-5">
          {[
            { label: 'Total Violation', val: stats.total,  color: 'text-red-600' },
            { label: 'Face Alert',      val: stats.face,   color: 'text-amber-600' },
            { label: 'Tab Switch',      val: stats.tab,    color: 'text-blue-600' },
            { label: 'Attempt Paused',  val: stats.paused, color: 'text-red-700' },
          ].map(s => (
            <div key={s.label} className="card p-3 text-center">
              <p className={`text-2xl font-bold ${s.color}`}>{s.val}</p>
              <p className="text-xs text-slate-500 mt-0.5">{s.label}</p>
            </div>
          ))}
        </div>

        {loading ? (
          <div className="space-y-2">{[1,2,3].map(i=><div key={i} className="card p-4 animate-pulse"><div className="h-3 bg-slate-200 rounded w-2/3"/></div>)}</div>
        ) : !selUjian ? (
          <div className="card p-12 text-center"><p className="text-slate-400">Pilih ujian untuk melihat log proctoring</p></div>
        ) : filtered.length === 0 ? (
          <div className="card p-12 text-center"><p className="text-slate-400">Tidak ada log untuk filter ini</p></div>
        ) : (
          <div className="card overflow-hidden divide-y divide-slate-100">
            {filtered.map(log => (
              <div key={log.id} className="flex items-start gap-3 px-4 py-3">
                <div className={clsx('w-2 h-2 rounded-full mt-1.5 flex-shrink-0', EVENT_COLOR[log.event_type] ?? 'bg-slate-400')}/>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-sm font-medium text-slate-800">{EVENT_LABEL[log.event_type] ?? log.event_type}</span>
                    <span className="badge badge-gray font-mono text-xs">{log.attempt_id.slice(0,8)}…</span>
                    <span className="text-xs text-slate-400 ml-auto">{new Date(log.created_at).toLocaleTimeString('id-ID')}</span>
                  </div>
                  {log.confidence > 0 && (
                    <p className="text-xs text-slate-500">Confidence: {(log.confidence * 100).toFixed(1)}%</p>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}
