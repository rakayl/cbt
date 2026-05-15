import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, CheckCircle2, RefreshCcw, XCircle } from 'lucide-react';
import { Link, useParams } from 'react-router-dom';
import { api } from '../lib/api';
import { getApiErrorDetail, getApiErrorMessage } from '../lib/apiError';

export default function StudentResultDetailPage() {
  const { sessionId } = useParams();
  const result = useQuery({
    queryKey: ['student-result-detail', sessionId],
    enabled: Boolean(sessionId),
    queryFn: async () => (await api.get(`/exam-sessions/student/history/${sessionId}`)).data.data,
  });
  const session = result.data?.session || {};
  const summary = result.data?.summary || session.metadata || {};
  const questions = result.data?.questions || [];
  const passed = Boolean(summary.passed);

  const stats = useMemo(() => [
    ['Skor', `${formatNumber(summary.score)} / ${formatNumber(summary.max_score)}`],
    ['Persentase', `${formatNumber(summary.percentage)}%`],
    ['Benar', formatNumber(summary.correct_count)],
    ['Salah/Kosong', `${formatNumber(summary.wrong_count)} / ${formatNumber(summary.unanswered_count)}`],
  ], [summary]);

  return (
    <div className="space-y-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div>
          <Link to="/student/history" className="inline-flex items-center gap-2 text-sm font-bold text-[#009ef7]">
            <ArrowLeft size={17} />
            Kembali ke history
          </Link>
          <div className="mt-4 text-sm font-bold text-[#a1a5b7]">Detail Hasil Ujian</div>
          <h2 className="mt-1 text-2xl font-extrabold text-[#181c32]">{session.exam_name || 'Hasil Ujian'}</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">
            Review nilai, jawaban siswa, kunci jawaban, dan detail penilaian dari snapshot soal saat ujian berlangsung.
          </p>
        </div>
        <button className="btn btn-ghost justify-center" onClick={() => result.refetch()}>
          <RefreshCcw size={17} />
          Refresh
        </button>
      </section>

      {result.isLoading ? (
        <section className="panel p-10 text-center font-semibold text-[#7e8299]">Memuat detail hasil...</section>
      ) : null}

      {result.error ? (
        <section className="panel bg-[#fff5f8] p-5 text-sm font-bold text-[#f1416c]">
          <div>{getApiErrorMessage(result.error)}</div>
          {getApiErrorDetail(result.error) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(result.error)}</div> : null}
        </section>
      ) : null}

      {result.data ? (
        <>
          <section className="grid gap-4 xl:grid-cols-[1fr_340px]">
            <div className="panel p-5">
              <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
                <div>
                  <div className="text-xs font-bold uppercase text-[#a1a5b7]">Kode Sesi</div>
                  <div className="mt-1 font-extrabold text-[#181c32]">{session.code}</div>
                </div>
                <ResultBadge passed={passed} />
              </div>
              <div className="mt-5 grid gap-4 sm:grid-cols-3">
                <Info label="Status" value={session.status || '-'} />
                <Info label="Mulai" value={formatDate(session.started_at)} />
                <Info label="Submit" value={formatDate(session.submitted_at)} />
              </div>
            </div>
            <div className="panel p-5">
              <div className="grid grid-cols-2 gap-3">
                {stats.map(([label, value]) => (
                  <div key={label} className="rounded-lg bg-[#f5f8fa] p-4">
                    <div className="text-xs font-bold uppercase text-[#a1a5b7]">{label}</div>
                    <div className="mt-1 text-xl font-extrabold text-[#181c32]">{value}</div>
                  </div>
                ))}
              </div>
            </div>
          </section>

          <section className="space-y-4">
            {questions.map((question) => (
              <QuestionResult key={question.session_question_id} question={question} />
            ))}
            {questions.length === 0 ? (
              <div className="panel p-10 text-center font-semibold text-[#7e8299]">Belum ada detail soal untuk sesi ini.</div>
            ) : null}
          </section>
        </>
      ) : null}
    </div>
  );
}

function QuestionResult({ question }) {
  const selected = new Set(question.selected_option_ids || []);
  const correct = new Set(question.correct_option_ids || []);
  const manual = question.manual_required;
  const status = manual ? 'manual' : question.answered ? (question.is_correct ? 'correct' : 'wrong') : 'empty';
  const statusClass = {
    correct: 'bg-[#e8fff3] text-[#50cd89]',
    wrong: 'bg-[#fff5f8] text-[#f1416c]',
    empty: 'bg-[#fff8dd] text-[#a46a00]',
    manual: 'bg-[#f1faff] text-[#009ef7]',
  }[status];

  return (
    <article className="panel overflow-hidden">
      <div className="flex flex-col gap-3 border-b border-[#eff2f5] p-5 md:flex-row md:items-start md:justify-between">
        <div>
          <div className="text-xs font-bold uppercase text-[#a1a5b7]">Soal {question.position} · {question.answer_mode}</div>
          <h3 className="mt-2 whitespace-pre-wrap text-base font-extrabold leading-7 text-[#181c32]">{question.text || '-'}</h3>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <span className={`inline-flex rounded-md px-2.5 py-1 text-xs font-extrabold ${statusClass}`}>{statusLabel(status)}</span>
          <span className="inline-flex rounded-md bg-[#f5f8fa] px-2.5 py-1 text-xs font-extrabold text-[#5e6278]">
            {formatNumber(question.earned_score)} / {formatNumber(question.max_score)}
          </span>
        </div>
      </div>

      <div className="p-5">
        <ResultMediaGrid media={question.media || []} />
        {question.options?.length ? (
          <div className="mt-4 grid gap-3">
            {question.options.map((option) => {
              const isSelected = selected.has(option.id);
              const isCorrect = correct.has(option.id);
              return (
                <div key={option.id} className={optionClass(isSelected, isCorrect)}>
                  <div className={optionLabelClass(isSelected, isCorrect)}>{option.label}</div>
                  <div className="min-w-0 flex-1">
                    <div className="font-semibold text-[#181c32]">{option.text || '-'}</div>
                    <ResultMediaGrid media={option.media || []} compact />
                  </div>
                  <div className="flex shrink-0 items-center gap-2 text-xs font-extrabold">
                    {isSelected ? <span className="rounded-md bg-[#f1faff] px-2 py-1 text-[#009ef7]">Dipilih</span> : null}
                    {isCorrect ? <span className="rounded-md bg-[#e8fff3] px-2 py-1 text-[#50cd89]">Kunci</span> : null}
                  </div>
                </div>
              );
            })}
          </div>
        ) : (
          <div className="mt-4 rounded-lg border border-[#eff2f5] bg-[#f5f8fa] p-4">
            <div className="text-xs font-bold uppercase text-[#a1a5b7]">Jawaban</div>
            <div className="mt-2 whitespace-pre-wrap text-sm font-semibold text-[#181c32]">{question.answer_payload?.text || '-'}</div>
          </div>
        )}
        {question.feedback ? (
          <div className="mt-4 rounded-lg bg-[#f1faff] p-4 text-sm font-semibold text-[#3f4254]">
            Feedback: {question.feedback}
          </div>
        ) : null}
      </div>
    </article>
  );
}

function Info({ label, value }) {
  return (
    <div className="rounded-lg bg-[#f5f8fa] p-4">
      <div className="text-xs font-bold uppercase text-[#a1a5b7]">{label}</div>
      <div className="mt-1 text-sm font-extrabold text-[#181c32]">{value}</div>
    </div>
  );
}

function ResultBadge({ passed }) {
  return (
    <span className={passed ? 'inline-flex items-center gap-2 rounded-lg bg-[#e8fff3] px-3 py-2 text-sm font-extrabold text-[#50cd89]' : 'inline-flex items-center gap-2 rounded-lg bg-[#fff5f8] px-3 py-2 text-sm font-extrabold text-[#f1416c]'}>
      {passed ? <CheckCircle2 size={18} /> : <XCircle size={18} />}
      {passed ? 'Lulus' : 'Tidak Lulus'}
    </span>
  );
}

function statusLabel(status) {
  if (status === 'correct') return 'Benar';
  if (status === 'wrong') return 'Salah';
  if (status === 'manual') return 'Butuh Review';
  return 'Kosong';
}

function optionClass(selected, correct) {
  if (selected && correct) return 'flex gap-3 rounded-lg border border-[#50cd89] bg-[#e8fff3] p-4';
  if (selected && !correct) return 'flex gap-3 rounded-lg border border-[#f1416c] bg-[#fff5f8] p-4';
  if (correct) return 'flex gap-3 rounded-lg border border-[#50cd89] bg-white p-4';
  return 'flex gap-3 rounded-lg border border-[#eff2f5] bg-white p-4';
}

function optionLabelClass(selected, correct) {
  if (selected && correct) return 'grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-[#50cd89] font-extrabold text-white';
  if (selected && !correct) return 'grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-[#f1416c] font-extrabold text-white';
  if (correct) return 'grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-[#e8fff3] font-extrabold text-[#50cd89]';
  return 'grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-[#f5f8fa] font-extrabold text-[#5e6278]';
}

function ResultMediaGrid({ media = [], compact = false }) {
  if (!media.length) return null;
  return (
    <div className={compact ? 'mt-3 grid gap-2 sm:grid-cols-2' : 'mt-4 grid gap-3 md:grid-cols-2'}>
      {media.map((item) => (
        <ResultImage key={item.id} media={item} compact={compact} />
      ))}
    </div>
  );
}

function ResultImage({ media, compact }) {
  const [src, setSrc] = useState('');
  useEffect(() => {
    let active = true;
    let objectUrl = '';
    api.get(media.url, { responseType: 'blob' }).then((response) => {
      if (!active) return;
      objectUrl = URL.createObjectURL(response.data);
      setSrc(objectUrl);
    }).catch(() => {
      if (active) setSrc('');
    });
    return () => {
      active = false;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [media.url]);

  if (!src) {
    return <div className={compact ? 'grid h-24 place-items-center rounded-lg bg-[#f5f8fa] text-xs text-[#a1a5b7]' : 'grid h-48 place-items-center rounded-lg bg-[#f5f8fa] text-sm text-[#a1a5b7]'}>Memuat gambar...</div>;
  }
  return (
    <a href={src} target="_blank" rel="noreferrer" className="block overflow-hidden rounded-lg border border-[#eff2f5] bg-[#f5f8fa]">
      <img className={compact ? 'h-28 w-full object-contain' : 'h-64 w-full object-contain'} src={src} alt="Media hasil ujian" />
    </a>
  );
}

function formatNumber(value) {
  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(Number(value || 0));
}

function formatDate(value) {
  if (!value) return '-';
  return new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value));
}
