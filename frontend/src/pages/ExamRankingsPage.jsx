import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Award, Download, Medal, RefreshCcw, Trophy, Users } from 'lucide-react';
import { api } from '../lib/api';
import { getApiErrorDetail, getApiErrorMessage } from '../lib/apiError';

export default function ExamRankingsPage() {
  const [searchParams] = useSearchParams();
  const [examId, setExamId] = useState('');
  const [classId, setClassId] = useState('');
  const [search, setSearch] = useState('');

  const exams = useQuery({
    queryKey: ['ranking-exams'],
    queryFn: async () => (await api.get('/exams/', { params: { page: 1, limit: 200 } })).data.data.items,
  });

  const classes = useQuery({
    queryKey: ['ranking-classes'],
    queryFn: async () => (await api.get('/class-rooms/', { params: { page: 1, limit: 300 } })).data.data.items,
  });

  useEffect(() => {
    const requestedExamId = searchParams.get('exam_id');
    if (requestedExamId && requestedExamId !== examId) {
      setExamId(requestedExamId);
      return;
    }
    if (!examId && exams.data?.length) {
      const firstPublished = exams.data.find((exam) => exam.status === 'published') || exams.data[0];
      setExamId(firstPublished.id);
    }
  }, [examId, exams.data, searchParams]);

  const rankings = useQuery({
    queryKey: ['exam-rankings', examId, classId, search],
    enabled: Boolean(examId),
    queryFn: async () => (
      await api.get(`/exams/${examId}/rankings`, {
        params: {
          page: 1,
          limit: 200,
          class_id: classId || undefined,
          search: search || undefined,
        },
      })
    ).data.data,
  });

  const items = rankings.data?.items || [];
  const topThree = useMemo(() => items.slice(0, 3), [items]);

  function exportCsv() {
    if (!rankings.data) return;
    const header = ['Rank', 'Siswa', 'Kode', 'Kelas', 'Nilai', 'Max', 'Persen', 'Benar', 'Salah', 'Kosong', 'Durasi', 'Status', 'Submit'];
    const rows = items.map((item) => [
      item.rank,
      item.student_name,
      item.student_code,
      item.class_name || '-',
      item.score,
      item.max_score,
      item.percentage,
      item.correct_count,
      item.wrong_count,
      item.unanswered_count,
      formatDuration(item.duration_seconds),
      item.passed ? 'Lulus' : 'Tidak Lulus',
      formatDateTime(item.submitted_at),
    ]);
    const csv = [header, ...rows].map((row) => row.map(csvCell).join(',')).join('\n');
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `ranking-${rankings.data.exam_code || rankings.data.exam_id}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div className="space-y-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <div className="text-sm font-bold text-[#a1a5b7]">Exam Ranking Center</div>
          <h2 className="mt-1 text-2xl font-extrabold text-[#181c32]">Peringkat Hasil Ujian</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">
            Ranking dihitung dari nilai final session yang sudah completed. Guru hanya melihat ujian miliknya, admin bisa melihat semua ujian.
          </p>
        </div>
        <button className="btn btn-primary justify-center" type="button" disabled={!items.length} onClick={exportCsv}>
          <Download size={17} />
          Export CSV
        </button>
      </section>

      <section className="panel p-5">
        <div className="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(220px,.7fr)_minmax(220px,.7fr)_auto]">
          <Field label="Pilih Ujian">
            <select className="input bg-white" value={examId} onChange={(event) => setExamId(event.target.value)}>
              {exams.isLoading ? <option value="">Memuat ujian...</option> : null}
              {exams.data?.map((exam) => (
                <option key={exam.id} value={exam.id}>
                  {exam.name} - {exam.status}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Filter Kelas">
            <select className="input bg-white" value={classId} onChange={(event) => setClassId(event.target.value)}>
              <option value="">Semua kelas</option>
              {classes.data?.map((item) => (
                <option key={item.id} value={item.id}>{item.name}</option>
              ))}
            </select>
          </Field>
          <Field label="Cari Siswa">
            <input className="input bg-white" placeholder="Nama, kode, NIS" value={search} onChange={(event) => setSearch(event.target.value)} />
          </Field>
          <div className="flex items-end">
            <button className="btn btn-ghost h-11 w-full justify-center" type="button" onClick={() => rankings.refetch()} disabled={!examId || rankings.isFetching}>
              <RefreshCcw size={17} />
              Refresh
            </button>
          </div>
        </div>
      </section>

      {rankings.error ? (
        <div className="rounded-lg bg-[#fff5f8] px-4 py-3 text-sm font-bold text-[#f1416c]">
          <div>{getApiErrorMessage(rankings.error)}</div>
          {getApiErrorDetail(rankings.error) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(rankings.error)}</div> : null}
        </div>
      ) : null}

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Metric label="Peserta" value={rankings.data?.participant_count || 0} icon={Users} />
        <Metric label="Selesai" value={rankings.data?.completed_count || 0} icon={Trophy} />
        <Metric label="Rata-rata" value={`${formatNumber(rankings.data?.average_percentage || 0)}%`} icon={Award} />
        <Metric label="Belum Selesai" value={rankings.data?.pending_count || 0} icon={RefreshCcw} />
      </section>

      {topThree.length ? (
        <section className="grid gap-4 lg:grid-cols-3">
          {topThree.map((item, index) => (
            <div key={item.session_id} className="panel p-5">
              <div className="flex items-center justify-between">
                <div className={index === 0 ? 'grid h-12 w-12 place-items-center rounded-lg bg-[#fff8dd] text-[#f1bc00]' : 'grid h-12 w-12 place-items-center rounded-lg bg-[#f1faff] text-[#009ef7]'}>
                  <Medal size={24} />
                </div>
                <div className="text-3xl font-extrabold text-[#181c32]">#{item.rank}</div>
              </div>
              <div className="mt-4 text-lg font-extrabold text-[#181c32]">{item.student_name}</div>
              <div className="mt-1 text-sm font-semibold text-[#7e8299]">{item.class_name || item.student_code}</div>
              <div className="mt-4 flex items-end justify-between">
                <div>
                  <div className="text-xs font-bold uppercase text-[#a1a5b7]">Nilai</div>
                  <div className="text-2xl font-extrabold text-[#009ef7]">{formatNumber(item.percentage)}%</div>
                </div>
                <StatusBadge passed={item.passed} />
              </div>
            </div>
          ))}
        </section>
      ) : null}

      <section className="panel overflow-hidden">
        <div className="border-b border-[#eff2f5] p-5">
          <h3 className="text-lg font-extrabold text-[#181c32]">{rankings.data?.exam_name || 'Peringkat Ujian'}</h3>
          <p className="mt-1 text-sm font-semibold text-[#a1a5b7]">
            Total data ranking: {rankings.data?.total || 0}. Tie-break: persentase tertinggi, durasi tercepat, submit lebih awal.
          </p>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[1120px] text-left text-sm">
            <thead className="border-b border-[#eff2f5] text-xs font-extrabold uppercase text-[#a1a5b7]">
              <tr>
                <th className="px-5 py-4">Rank</th>
                <th className="px-5 py-4">Siswa</th>
                <th className="px-5 py-4">Kelas</th>
                <th className="px-5 py-4">Nilai</th>
                <th className="px-5 py-4">Persen</th>
                <th className="px-5 py-4">Benar</th>
                <th className="px-5 py-4">Salah</th>
                <th className="px-5 py-4">Kosong</th>
                <th className="px-5 py-4">Durasi</th>
                <th className="px-5 py-4">Status</th>
                <th className="px-5 py-4">Submit</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#eff2f5]">
              {rankings.isLoading || rankings.isFetching ? (
                <tr>
                  <td className="px-5 py-10 text-center font-semibold text-[#7e8299]" colSpan={11}>Memuat ranking...</td>
                </tr>
              ) : null}
              {!rankings.isFetching && items.map((item) => (
                <tr key={item.session_id} className="hover:bg-[#f9fafb]">
                  <td className="px-5 py-4">
                    <span className={item.rank <= 3 ? 'inline-flex h-8 min-w-8 items-center justify-center rounded-lg bg-[#fff8dd] px-2 font-extrabold text-[#f1bc00]' : 'inline-flex h-8 min-w-8 items-center justify-center rounded-lg bg-[#f5f8fa] px-2 font-extrabold text-[#3f4254]'}>
                      #{item.rank}
                    </span>
                  </td>
                  <td className="px-5 py-4">
                    <div className="font-extrabold text-[#181c32]">{item.student_name}</div>
                    <div className="mt-1 text-xs font-semibold text-[#a1a5b7]">{item.student_code}</div>
                  </td>
                  <td className="px-5 py-4 font-semibold text-[#7e8299]">{item.class_name || '-'}</td>
                  <td className="px-5 py-4 font-extrabold text-[#181c32]">{formatNumber(item.score)} / {formatNumber(item.max_score)}</td>
                  <td className="px-5 py-4">
                    <div className="w-28 rounded-full bg-[#f5f8fa]">
                      <div className="h-2 rounded-full bg-[#009ef7]" style={{ width: `${Math.min(100, Math.max(0, item.percentage || 0))}%` }} />
                    </div>
                    <div className="mt-1 text-xs font-extrabold text-[#009ef7]">{formatNumber(item.percentage)}%</div>
                  </td>
                  <td className="px-5 py-4 font-semibold text-[#50cd89]">{item.correct_count}</td>
                  <td className="px-5 py-4 font-semibold text-[#f1416c]">{item.wrong_count}</td>
                  <td className="px-5 py-4 font-semibold text-[#7e8299]">{item.unanswered_count}</td>
                  <td className="px-5 py-4 font-semibold text-[#3f4254]">{formatDuration(item.duration_seconds)}</td>
                  <td className="px-5 py-4"><StatusBadge passed={item.passed} /></td>
                  <td className="px-5 py-4 font-semibold text-[#7e8299]">{formatDateTime(item.submitted_at)}</td>
                </tr>
              ))}
              {!rankings.isFetching && items.length === 0 ? (
                <tr>
                  <td className="px-5 py-12 text-center font-semibold text-[#7e8299]" colSpan={11}>
                    Belum ada siswa yang menyelesaikan ujian ini.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}

function Field({ label, children }) {
  return (
    <label className="block min-w-0">
      <span className="mb-2 block text-sm font-bold text-[#3f4254]">{label}</span>
      {children}
    </label>
  );
}

function Metric({ label, value, icon: Icon }) {
  return (
    <div className="panel flex items-center p-4">
      <div className="grid h-11 w-11 place-items-center rounded-lg bg-[#f1faff] text-[#009ef7]">
        <Icon size={21} />
      </div>
      <div className="ml-3">
        <div className="text-xs font-bold uppercase text-[#a1a5b7]">{label}</div>
        <div className="text-xl font-extrabold text-[#181c32]">{value}</div>
      </div>
    </div>
  );
}

function StatusBadge({ passed }) {
  return (
    <span className={passed ? 'inline-flex rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold text-[#50cd89]' : 'inline-flex rounded-md bg-[#fff5f8] px-2.5 py-1 text-xs font-extrabold text-[#f1416c]'}>
      {passed ? 'Lulus' : 'Tidak Lulus'}
    </span>
  );
}

function formatNumber(value) {
  const num = Number(value || 0);
  return Number.isInteger(num) ? String(num) : num.toFixed(2).replace(/\.?0+$/, '');
}

function formatDuration(seconds) {
  const total = Math.max(0, Number(seconds || 0));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = Math.floor(total % 60);
  if (h > 0) return `${h}j ${m}m`;
  if (m > 0) return `${m}m ${s}d`;
  return `${s}d`;
}

function formatDateTime(value) {
  if (!value) return '-';
  return new Date(value).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' });
}

function csvCell(value) {
  const text = String(value ?? '');
  return `"${text.replace(/"/g, '""')}"`;
}
