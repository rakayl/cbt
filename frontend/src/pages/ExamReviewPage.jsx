import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CheckCircle2, ClipboardCheck, RefreshCcw, Save, XCircle } from 'lucide-react';
import { api } from '../lib/api';
import { getApiErrorDetail, getApiErrorMessage } from '../lib/apiError';

export default function ExamReviewPage() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [selectedSessionId, setSelectedSessionId] = useState('');
  const [manualScores, setManualScores] = useState({});

  const sessions = useQuery({
    queryKey: ['grading-review-sessions', search],
    queryFn: async () => (await api.get('/grading/review/sessions', { params: { page: 1, limit: 30, search } })).data.data,
  });

  const detail = useQuery({
    queryKey: ['grading-review-detail', selectedSessionId],
    enabled: Boolean(selectedSessionId),
    queryFn: async () => (await api.get(`/grading/review/sessions/${selectedSessionId}`)).data.data,
  });

  const manualScore = useMutation({
    mutationFn: async ({ gradingId, payload }) => (await api.put(`/grading/${gradingId}/manual-score`, payload)).data.data,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['grading-review-detail', selectedSessionId] });
      queryClient.invalidateQueries({ queryKey: ['grading-review-sessions'] });
    },
  });
  const releaseResult = useMutation({
    mutationFn: async () => (await api.put(`/grading/review/sessions/${selectedSessionId}/release`)).data.data,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['grading-review-detail', selectedSessionId] });
      queryClient.invalidateQueries({ queryKey: ['grading-review-sessions'] });
    },
  });

  const selected = detail.data?.session;
  const items = detail.data?.items || [];
  const stats = useMemo(() => ({
    total: sessions.data?.total || 0,
    manual: sessions.data?.items?.reduce((sum, item) => sum + Number(item.manual_required || 0), 0) || 0,
  }), [sessions.data]);

  function submitManual(item) {
    const draft = manualScores[item.grading_id] || {};
    const score = Number(draft.earned_score ?? item.earned_score ?? 0);
    const feedback = draft.feedback ?? item.feedback ?? '';
    manualScore.mutate({
      gradingId: item.grading_id,
      payload: { earned_score: score, feedback, status: 'reviewed' },
    });
  }

  return (
    <div className="space-y-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <div className="text-sm font-bold text-[#a1a5b7]">Grading Center</div>
          <h2 className="mt-1 text-2xl font-extrabold text-[#181c32]">Review Hasil Ujian</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">
            Guru/admin dapat melihat soal, jawaban siswa, kunci, skor per soal, dan memberi nilai manual tanpa membuka data ini ke siswa.
          </p>
        </div>
        <div className="grid grid-cols-2 gap-4 sm:w-[360px]">
          <Metric label="Sesi Selesai" value={stats.total} />
          <Metric label="Butuh Review" value={stats.manual} />
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-[380px_minmax(0,1fr)]">
        <aside className="panel h-fit overflow-hidden">
          <div className="border-b border-[#eff2f5] p-4">
            <label className="block">
              <input className="input h-11" placeholder="Cari ujian atau siswa" value={search} onChange={(event) => setSearch(event.target.value)} />
            </label>
          </div>
          <div className="max-h-[680px] overflow-y-auto p-3">
            {sessions.isLoading ? <div className="rounded-lg bg-[#f5f8fa] p-4 text-sm font-semibold text-[#7e8299]">Memuat sesi...</div> : null}
            {sessions.data?.items?.map((session) => (
              <button
                key={session.session_id}
                className={selectedSessionId === session.session_id ? 'mb-2 w-full rounded-lg border border-[#009ef7] bg-[#f1faff] p-4 text-left' : 'mb-2 w-full rounded-lg border border-[#eff2f5] bg-white p-4 text-left hover:border-[#009ef7]'}
                onClick={() => setSelectedSessionId(session.session_id)}
              >
                <div className="font-extrabold text-[#181c32]">{session.exam_name}</div>
                <div className="mt-1 text-sm font-semibold text-[#7e8299]">{session.student_name} · {session.student_code}</div>
                <div className="mt-3 flex items-center justify-between text-xs font-extrabold">
                  <span className="rounded-md bg-[#f5f8fa] px-2 py-1 text-[#5e6278]">{formatNumber(session.percentage)}%</span>
                  <ResultBadge passed={session.passed} />
                </div>
              </button>
            ))}
            {!sessions.isLoading && sessions.data?.items?.length === 0 ? <div className="rounded-lg bg-[#f5f8fa] p-4 text-sm font-semibold text-[#7e8299]">Belum ada sesi selesai.</div> : null}
          </div>
        </aside>

        <section className="space-y-4">
          {!selectedSessionId ? (
            <div className="panel p-10 text-center font-semibold text-[#7e8299]">Pilih sesi ujian untuk review detail.</div>
          ) : null}
          {detail.isLoading ? <div className="panel p-10 text-center font-semibold text-[#7e8299]">Memuat detail review...</div> : null}
          {detail.error || manualScore.error || releaseResult.error ? (
            <div className="panel bg-[#fff5f8] p-5 text-sm font-bold text-[#f1416c]">
              <div>{getApiErrorMessage(detail.error || manualScore.error || releaseResult.error)}</div>
              {getApiErrorDetail(detail.error || manualScore.error || releaseResult.error) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(detail.error || manualScore.error || releaseResult.error)}</div> : null}
            </div>
          ) : null}
          {selected ? (
            <div className="panel flex flex-col gap-4 p-5 lg:flex-row lg:items-center lg:justify-between">
              <div>
                <div className="text-sm font-bold text-[#a1a5b7]">{selected.student_name} · {selected.student_code}</div>
                <h3 className="mt-1 text-xl font-extrabold text-[#181c32]">{selected.exam_name}</h3>
              </div>
              <div className="flex flex-wrap gap-2">
                <span className="rounded-lg bg-[#f5f8fa] px-3 py-2 text-sm font-extrabold text-[#181c32]">{formatNumber(selected.score)} / {formatNumber(selected.max_score)}</span>
                <span className="rounded-lg bg-[#f5f8fa] px-3 py-2 text-sm font-extrabold text-[#181c32]">{formatNumber(selected.percentage)}%</span>
                {selected.metadata?.result_policy?.visibility === 'manual_release' && !selected.metadata?.result_policy?.released ? (
                  <button className="btn btn-primary h-10" disabled={releaseResult.isPending} onClick={() => releaseResult.mutate()}>
                    <CheckCircle2 size={17} /> Rilis Hasil
                  </button>
                ) : null}
                <button className="btn btn-ghost h-10" onClick={() => detail.refetch()}><RefreshCcw size={17} /> Refresh</button>
              </div>
            </div>
          ) : null}
          {items.map((item) => (
            <article key={item.session_question_id} className="panel overflow-hidden">
              <div className="flex flex-col gap-3 border-b border-[#eff2f5] p-5 lg:flex-row lg:items-start lg:justify-between">
                <div>
                  <div className="text-xs font-bold uppercase text-[#a1a5b7]">Soal {item.position}{item.question_tag_name ? ` -- ${item.question_tag_name}` : ''} · {item.answer_mode}</div>
                  <h4 className="mt-2 whitespace-pre-wrap text-base font-extrabold leading-7 text-[#181c32]">{item.text || '-'}</h4>
                </div>
                <div className="flex shrink-0 gap-2">
                  <span className="rounded-md bg-[#f5f8fa] px-2.5 py-1 text-xs font-extrabold text-[#5e6278]">{formatNumber(item.earned_score)} / {formatNumber(item.max_score)}</span>
                  <StatusBadge item={item} />
                </div>
              </div>
              <div className="space-y-4 p-5">
                <MediaGrid media={item.media || []} />
                {item.options?.length ? (
                  <div className="grid gap-3">
                    {item.options.map((option) => (
                      <div key={option.id} className={optionClass(option)}>
                        <div className="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-[#f5f8fa] font-extrabold text-[#5e6278]">{option.label}</div>
                        <div className="min-w-0 flex-1">
                          <div className="font-semibold text-[#181c32]">{option.text || '-'}</div>
                          <MediaGrid media={option.media || []} compact />
                        </div>
                        <div className="flex shrink-0 flex-wrap gap-2 text-xs font-extrabold">
                          {option.selected ? <span className="rounded-md bg-[#f1faff] px-2 py-1 text-[#009ef7]">Dipilih</span> : null}
                          {option.correct ? <span className="rounded-md bg-[#e8fff3] px-2 py-1 text-[#50cd89]">Kunci</span> : null}
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="rounded-lg bg-[#f5f8fa] p-4">
                    <div className="text-xs font-bold uppercase text-[#a1a5b7]">Jawaban</div>
                    <div className="mt-2 whitespace-pre-wrap text-sm font-semibold text-[#181c32]">{item.answer_payload?.text || '-'}</div>
                  </div>
                )}
                {item.manual_required ? (
                  <div className="grid gap-3 rounded-lg border border-[#eff2f5] bg-[#f9fafb] p-4 md:grid-cols-[160px_minmax(0,1fr)_auto]">
                    <input className="input bg-white" type="number" min="0" max={item.max_score} step="0.1" value={manualScores[item.grading_id]?.earned_score ?? item.earned_score ?? 0} onChange={(event) => setManualScores((current) => ({ ...current, [item.grading_id]: { ...(current[item.grading_id] || {}), earned_score: event.target.value } }))} />
                    <input className="input bg-white" placeholder="Feedback untuk internal/guru" value={manualScores[item.grading_id]?.feedback ?? item.feedback ?? ''} onChange={(event) => setManualScores((current) => ({ ...current, [item.grading_id]: { ...(current[item.grading_id] || {}), feedback: event.target.value } }))} />
                    <button className="btn btn-primary justify-center" disabled={manualScore.isPending || !item.grading_id} onClick={() => submitManual(item)}>
                      <Save size={17} /> Simpan
                    </button>
                  </div>
                ) : null}
              </div>
            </article>
          ))}
        </section>
      </section>
    </div>
  );
}

function Metric({ label, value }) {
  return <div className="panel p-4"><div className="text-xs font-bold uppercase text-[#a1a5b7]">{label}</div><div className="mt-1 text-xl font-extrabold text-[#181c32]">{value}</div></div>;
}

function ResultBadge({ passed }) {
  return <span className={passed ? 'inline-flex rounded-md bg-[#e8fff3] px-2 py-1 text-[#50cd89]' : 'inline-flex rounded-md bg-[#fff5f8] px-2 py-1 text-[#f1416c]'}>{passed ? 'Lulus' : 'Tidak Lulus'}</span>;
}

function StatusBadge({ item }) {
  if (item.manual_required) return <span className="inline-flex items-center gap-1 rounded-md bg-[#f1faff] px-2.5 py-1 text-xs font-extrabold text-[#009ef7]"><ClipboardCheck size={13} /> Manual</span>;
  return item.is_correct ? <span className="inline-flex items-center gap-1 rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold text-[#50cd89]"><CheckCircle2 size={13} /> Benar</span> : <span className="inline-flex items-center gap-1 rounded-md bg-[#fff5f8] px-2.5 py-1 text-xs font-extrabold text-[#f1416c]"><XCircle size={13} /> Salah</span>;
}

function optionClass(option) {
  if (option.selected && option.correct) return 'flex gap-3 rounded-lg border border-[#50cd89] bg-[#e8fff3] p-4';
  if (option.selected && !option.correct) return 'flex gap-3 rounded-lg border border-[#f1416c] bg-[#fff5f8] p-4';
  if (option.correct) return 'flex gap-3 rounded-lg border border-[#50cd89] bg-white p-4';
  return 'flex gap-3 rounded-lg border border-[#eff2f5] bg-white p-4';
}

function MediaGrid({ media = [], compact = false }) {
  if (!media.length) return null;
  return <div className={compact ? 'mt-3 grid gap-2 sm:grid-cols-2' : 'grid gap-3 md:grid-cols-2'}>{media.map((item) => <SecureImage key={item.id} media={item} compact={compact} />)}</div>;
}

function SecureImage({ media, compact }) {
  const [src, setSrc] = useState('');
  useEffect(() => {
    let active = true;
    let objectUrl = '';
    api.get(media.url, { responseType: 'blob' }).then((response) => {
      if (!active) return;
      objectUrl = URL.createObjectURL(response.data);
      setSrc(objectUrl);
    }).catch(() => active && setSrc(''));
    return () => {
      active = false;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [media.url]);
  if (!src) return <div className={compact ? 'grid h-24 place-items-center rounded-lg bg-[#f5f8fa] text-xs text-[#a1a5b7]' : 'grid h-48 place-items-center rounded-lg bg-[#f5f8fa] text-sm text-[#a1a5b7]'}>Memuat gambar...</div>;
  return <a href={src} target="_blank" rel="noreferrer" className="block overflow-hidden rounded-lg border border-[#eff2f5] bg-[#f5f8fa]"><img className={compact ? 'h-28 w-full object-contain' : 'h-64 w-full object-contain'} src={src} alt="Media review" /></a>;
}

function formatNumber(value) {
  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(Number(value || 0));
}
