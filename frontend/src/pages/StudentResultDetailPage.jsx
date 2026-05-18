import { useMemo } from 'react';
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
  const resultVisible = summary.result_visible !== false;

  const stats = useMemo(() => summary.result_visible === false ? [
    ['Status', 'Belum dirilis'],
    ['Skor', '-'],
    ['Persentase', '-'],
    ['Detail', 'Terkunci'],
  ] : [
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
            Review ringkasan nilai dan status penilaian. Isi soal, opsi jawaban, jawaban siswa, dan kunci tidak ditampilkan untuk mencegah kebocoran bank soal.
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

          {!resultVisible ? (
            <section className="panel p-8 text-center">
              <div className="text-lg font-extrabold text-[#181c32]">Hasil belum dipublish</div>
              <p className="mt-2 text-sm font-semibold text-[#7e8299]">{summary.message || 'Nilai akan tampil setelah dirilis oleh guru/admin.'}</p>
            </section>
          ) : null}

          {resultVisible ? <section className="space-y-4">
            {questions.map((question) => (
              <QuestionResult key={question.session_question_id} question={question} />
            ))}
            {questions.length === 0 ? (
              <div className="panel p-10 text-center font-semibold text-[#7e8299]">Belum ada detail soal untuk sesi ini.</div>
            ) : null}
          </section> : null}
        </>
      ) : null}
    </div>
  );
}

function QuestionResult({ question }) {
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
          <div className="text-xs font-bold uppercase text-[#a1a5b7]">
            Soal {question.position}
            {question.question_tag_name ? ` -- ${question.question_tag_name}` : ''}
          </div>
          <h3 className="mt-2 text-base font-extrabold leading-7 text-[#181c32]">Detail soal disembunyikan untuk keamanan bank soal.</h3>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <span className={`inline-flex rounded-md px-2.5 py-1 text-xs font-extrabold ${statusClass}`}>{statusLabel(status)}</span>
          <span className="inline-flex rounded-md bg-[#f5f8fa] px-2.5 py-1 text-xs font-extrabold text-[#5e6278]">
            {formatNumber(question.earned_score)} / {formatNumber(question.max_score)}
          </span>
        </div>
      </div>

      <div className="p-5">
        <div className="grid gap-3 sm:grid-cols-3">
          <div className="rounded-lg bg-[#f5f8fa] p-4">
            <div className="text-xs font-bold uppercase text-[#a1a5b7]">Status Jawaban</div>
            <div className="mt-1 text-sm font-extrabold text-[#181c32]">{question.answered ? 'Terjawab' : 'Kosong'}</div>
          </div>
          <div className="rounded-lg bg-[#f5f8fa] p-4">
            <div className="text-xs font-bold uppercase text-[#a1a5b7]">Jenis Soal</div>
            <div className="mt-1 text-sm font-extrabold text-[#181c32]">{question.question_tag_name || '-'}</div>
          </div>
          <div className="rounded-lg bg-[#f5f8fa] p-4">
            <div className="text-xs font-bold uppercase text-[#a1a5b7]">Mode</div>
            <div className="mt-1 text-sm font-extrabold capitalize text-[#181c32]">{question.answer_mode || '-'}</div>
          </div>
        </div>
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

function formatNumber(value) {
  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(Number(value || 0));
}

function formatDate(value) {
  if (!value) return '-';
  return new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value));
}
