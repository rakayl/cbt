import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { RefreshCcw, ShieldAlert } from 'lucide-react';
import { api } from '../lib/api';

const eventOptions = ['', 'tab_switch', 'fullscreen_exit', 'copy_paste', 'right_click', 'no_face', 'multiple_face', 'audio_noise', 'webcam_unavailable', 'webcam_snapshot'];
const severityOptions = ['', 'low', 'medium', 'high', 'critical'];

export default function MonitoringPage() {
  const [search, setSearch] = useState('');
  const [eventType, setEventType] = useState('');
  const [severity, setSeverity] = useState('');
  const [sessionId, setSessionId] = useState('');
  const events = useQuery({
    queryKey: ['proctoring-events', search, eventType, severity, sessionId],
    queryFn: async () => (
      await api.get('/proctoring/events', {
        params: { page: 1, limit: 50, search, event_type: eventType, severity, session_id: sessionId },
      })
    ).data.data,
  });
  const timeline = useQuery({
    queryKey: ['proctoring-timeline', sessionId],
    enabled: sessionId.trim().length >= 32,
    queryFn: async () => (await api.get(`/proctoring/sessions/${sessionId.trim()}/timeline`)).data.data,
  });

  return (
    <div className="space-y-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <div className="text-sm font-bold text-[#a1a5b7]">Proctoring Review</div>
          <h2 className="mt-1 text-2xl font-extrabold text-[#181c32]">Monitoring Anti Cheat</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">Event browser, face detection, noise, dan snapshot webcam dari sesi ujian siswa.</p>
        </div>
        <div className="panel flex items-center p-4 sm:w-72">
          <div className="grid h-11 w-11 place-items-center rounded-lg bg-[#fff5f8] text-[#f1416c]">
            <ShieldAlert size={21} />
          </div>
          <div className="ml-3">
            <div className="text-xs font-bold uppercase text-[#a1a5b7]">Total Event</div>
            <div className="text-xl font-extrabold text-[#181c32]">{events.data?.total || 0}</div>
          </div>
        </div>
      </section>

      {sessionId.trim() ? (
        <section className="panel overflow-hidden">
          <div className="border-b border-[#eff2f5] p-5">
            <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <div>
                <h3 className="text-lg font-extrabold text-[#181c32]">Timeline Session</h3>
                <p className="mt-1 text-sm font-semibold text-[#a1a5b7]">
                  {timeline.data ? `${timeline.data.student_name || 'Siswa'} - ${timeline.data.exam_name || 'Exam'}` : 'Masukkan session ID lengkap untuk melihat urutan kejadian.'}
                </p>
              </div>
              <div className="grid gap-2 sm:grid-cols-3">
                <TimelineMetric label="Score" value={timeline.data?.suspicious_score?.toFixed?.(1) || '0.0'} />
                <TimelineMetric label="Status" value={timeline.data?.status || '-'} />
                <TimelineMetric label="Event" value={timeline.data?.items?.length || 0} />
              </div>
            </div>
          </div>
          <div className="grid gap-4 p-5 lg:grid-cols-[260px_1fr]">
            <div className="rounded-lg bg-[#f9fafb] p-4">
              <div className="text-xs font-extrabold uppercase text-[#a1a5b7]">Ringkasan Event</div>
              <div className="mt-3 space-y-2">
                {Object.entries(timeline.data?.summary || {}).map(([key, value]) => (
                  <div key={key} className="flex items-center justify-between rounded-md bg-white px-3 py-2 text-sm">
                    <span className="font-bold text-[#3f4254]">{key}</span>
                    <span className="font-extrabold text-[#181c32]">{value}</span>
                  </div>
                ))}
                {timeline.isLoading ? <div className="text-sm font-semibold text-[#7e8299]">Memuat timeline...</div> : null}
                {!timeline.isLoading && !Object.keys(timeline.data?.summary || {}).length ? <div className="text-sm font-semibold text-[#7e8299]">Belum ada ringkasan.</div> : null}
              </div>
            </div>
            <div className="space-y-3">
              {timeline.data?.items?.map((item) => (
                <div key={`${item.source}-${item.id}`} className="rounded-lg border border-[#eff2f5] bg-white p-4">
                  <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                    <div>
                      <div className="font-extrabold text-[#181c32]">{item.event_type}</div>
                      <div className="mt-1 text-xs font-bold uppercase text-[#a1a5b7]">{item.source} - {formatDate(item.created_at)}</div>
                    </div>
                    <span className="inline-flex w-fit rounded-md bg-[#fff5f8] px-2.5 py-1 text-xs font-extrabold uppercase text-[#f1416c]">{item.severity}</span>
                  </div>
                  {item.metadata?.object_key ? (
                    <div className="mt-3 rounded-md bg-[#f1faff] px-3 py-2 text-xs font-bold text-[#009ef7]">Snapshot tersimpan: {item.metadata.object_key}</div>
                  ) : null}
                </div>
              ))}
              {!timeline.isLoading && timeline.data && timeline.data.items?.length === 0 ? (
                <div className="rounded-lg bg-[#f5f8fa] p-4 text-sm font-semibold text-[#7e8299]">Belum ada timeline untuk session ini.</div>
              ) : null}
            </div>
          </div>
        </section>
      ) : null}

      <section className="panel overflow-hidden">
        <div className="grid gap-3 border-b border-[#eff2f5] p-5 lg:grid-cols-[1fr_180px_150px_1fr_120px]">
          <label className="block">
            <input className="input h-11" placeholder="Cari event" value={search} onChange={(event) => setSearch(event.target.value)} />
          </label>
          <select className="input h-11" value={eventType} onChange={(event) => setEventType(event.target.value)}>
            {eventOptions.map((item) => <option key={item || 'all'} value={item}>{item || 'Semua event'}</option>)}
          </select>
          <select className="input h-11" value={severity} onChange={(event) => setSeverity(event.target.value)}>
            {severityOptions.map((item) => <option key={item || 'all'} value={item}>{item || 'Semua severity'}</option>)}
          </select>
          <input className="input h-11" placeholder="Filter session ID" value={sessionId} onChange={(event) => setSessionId(event.target.value)} />
          <button className="btn btn-ghost justify-center" onClick={() => events.refetch()}>
            <RefreshCcw size={17} />
            Refresh
          </button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[900px] text-left text-sm">
            <thead className="border-b border-[#eff2f5] text-xs font-extrabold uppercase text-[#a1a5b7]">
              <tr>
                <th className="px-5 py-4">Event</th>
                <th className="px-5 py-4">Severity</th>
                <th className="px-5 py-4">Score</th>
                <th className="px-5 py-4">Session</th>
                <th className="px-5 py-4">Snapshot</th>
                <th className="px-5 py-4">Waktu</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#eff2f5]">
              {events.isLoading ? (
                <tr><td className="px-5 py-10 text-center font-semibold text-[#7e8299]" colSpan={6}>Memuat event...</td></tr>
              ) : null}
              {events.data?.items?.map((item) => {
                const detail = item.metadata?.detail || {};
                const score = item.metadata?.score || 0;
                const severityValue = item.metadata?.severity || '-';
                return (
                  <tr key={item.id} className="hover:bg-[#f9fafb]">
                    <td className="px-5 py-4 font-extrabold text-[#181c32]">{item.metadata?.event_type || item.name}</td>
                    <td className="px-5 py-4">
                      <span className="inline-flex rounded-md bg-[#fff5f8] px-2.5 py-1 text-xs font-extrabold uppercase text-[#f1416c]">{severityValue}</span>
                    </td>
                    <td className="px-5 py-4 font-mono font-bold">{Number(score).toFixed(1)}</td>
                    <td className="px-5 py-4 font-mono text-xs text-[#7e8299]">{item.metadata?.exam_session_id || '-'}</td>
                    <td className="px-5 py-4 font-medium text-[#7e8299]">{detail.object_key ? 'Tersimpan' : '-'}</td>
                    <td className="px-5 py-4 font-medium text-[#7e8299]">{formatDate(item.created_at)}</td>
                  </tr>
                );
              })}
              {!events.isLoading && events.data?.items?.length === 0 ? (
                <tr><td className="px-5 py-12 text-center font-semibold text-[#7e8299]" colSpan={6}>Belum ada event proctoring.</td></tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}

function TimelineMetric({ label, value }) {
  return (
    <div className="rounded-lg bg-[#f9fafb] px-4 py-2">
      <div className="text-[11px] font-extrabold uppercase text-[#a1a5b7]">{label}</div>
      <div className="text-sm font-extrabold text-[#181c32]">{value}</div>
    </div>
  );
}

function formatDate(value) {
  if (!value) return '-';
  return new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value));
}
