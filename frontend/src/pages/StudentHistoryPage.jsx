import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Clock3, Eye, FileCheck2, RefreshCcw } from 'lucide-react';
import { Link } from 'react-router-dom';
import { api } from '../lib/api';

export default function StudentHistoryPage() {
  const [search, setSearch] = useState('');
  const history = useQuery({
    queryKey: ['student-exam-history', search],
    queryFn: async () => (await api.get('/exam-sessions/student/history', { params: { page: 1, limit: 20, search } })).data.data,
  });
  const completed = history.data?.items?.filter((item) => item.status === 'completed').length || 0;

  return (
    <div className="space-y-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <div className="text-sm font-bold text-[#a1a5b7]">Riwayat Siswa</div>
          <h2 className="mt-1 text-2xl font-extrabold text-[#181c32]">History Ujian</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">Riwayat sesi ujian, status pengerjaan, dan waktu submit siswa.</p>
        </div>
        <div className="grid grid-cols-2 gap-4 sm:w-[360px]">
          <Metric label="Total Sesi" value={history.data?.total || 0} icon={Clock3} />
          <Metric label="Selesai" value={completed} icon={FileCheck2} />
        </div>
      </section>

      <section className="panel overflow-hidden">
        <div className="flex flex-col gap-3 border-b border-[#eff2f5] p-5 sm:flex-row sm:items-center sm:justify-between">
          <label className="relative block sm:w-80">
            <input className="input h-11" placeholder="Cari history ujian" value={search} onChange={(event) => setSearch(event.target.value)} />
          </label>
          <button className="btn btn-ghost" onClick={() => history.refetch()}>
            <RefreshCcw size={17} />
            Refresh
          </button>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full min-w-[760px] text-left text-sm">
            <thead className="border-b border-[#eff2f5] text-xs font-extrabold uppercase text-[#a1a5b7]">
              <tr>
                <th className="px-5 py-4">Ujian</th>
                <th className="px-5 py-4">Status</th>
                <th className="px-5 py-4">Mulai</th>
                <th className="px-5 py-4">Berakhir</th>
                <th className="px-5 py-4">Submit</th>
                <th className="px-5 py-4">Skor</th>
                <th className="px-5 py-4 text-right">Aksi</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#eff2f5]">
              {history.isLoading ? (
                <tr><td className="px-5 py-10 text-center font-semibold text-[#7e8299]" colSpan={7}>Memuat history...</td></tr>
              ) : null}
              {history.data?.items?.map((item) => (
                <HistoryRow key={item.session_id} item={item} />
              ))}
              {!history.isLoading && history.data?.items?.length === 0 ? (
                <tr><td className="px-5 py-12 text-center font-semibold text-[#7e8299]" colSpan={7}>Belum ada history ujian.</td></tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}

function HistoryRow({ item }) {
  const metadata = item.metadata || {};
  const hasScore = item.score !== null && item.score !== undefined;
  const maxScore = Number(metadata.max_score || 0);
  const percentage = Number(metadata.percentage || 0);
  return (
    <tr className="hover:bg-[#f9fafb]">
      <td className="px-5 py-4">
        <div className="font-extrabold text-[#181c32]">{item.exam_name}</div>
        <div className="mt-1 text-xs font-semibold text-[#a1a5b7]">{item.code}</div>
      </td>
      <td className="px-5 py-4"><StatusBadge status={item.status} /></td>
      <td className="px-5 py-4 font-semibold text-[#7e8299]">{formatDate(item.started_at)}</td>
      <td className="px-5 py-4 font-semibold text-[#7e8299]">{formatDate(item.ends_at)}</td>
      <td className="px-5 py-4 font-semibold text-[#7e8299]">{formatDate(item.submitted_at)}</td>
      <td className="px-5 py-4">
        {hasScore ? (
          <div>
            <div className="font-extrabold text-[#181c32]">
              {formatNumber(item.score)}{maxScore ? ` / ${formatNumber(maxScore)}` : ''}
            </div>
            <div className="mt-1 flex items-center gap-2">
              <span className="text-xs font-bold text-[#7e8299]">{formatNumber(percentage)}%</span>
              <ResultBadge passed={Boolean(metadata.passed)} />
            </div>
          </div>
        ) : '-'}
      </td>
      <td className="px-5 py-4 text-right">
        <Link className="btn btn-ghost inline-flex h-10 justify-center" to={`/student/history/${item.session_id}`}>
          <Eye size={17} />
          Detail
        </Link>
      </td>
    </tr>
  );
}

function Metric({ label, value, icon: Icon }) {
  return (
    <div className="panel flex items-center p-4">
      <div className="grid h-11 w-11 place-items-center rounded-lg bg-[#f1faff] text-[#009ef7]"><Icon size={21} /></div>
      <div className="ml-3">
        <div className="text-xs font-bold uppercase text-[#a1a5b7]">{label}</div>
        <div className="text-xl font-extrabold text-[#181c32]">{value}</div>
      </div>
    </div>
  );
}

function StatusBadge({ status }) {
  const done = status === 'completed';
  return <span className={done ? 'inline-flex rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold capitalize text-[#50cd89]' : 'inline-flex rounded-md bg-[#f1faff] px-2.5 py-1 text-xs font-extrabold capitalize text-[#009ef7]'}>{status || '-'}</span>;
}

function ResultBadge({ passed }) {
  return (
    <span className={passed ? 'inline-flex rounded-md bg-[#e8fff3] px-2 py-0.5 text-xs font-extrabold text-[#50cd89]' : 'inline-flex rounded-md bg-[#fff5f8] px-2 py-0.5 text-xs font-extrabold text-[#f1416c]'}>
      {passed ? 'Lulus' : 'Tidak Lulus'}
    </span>
  );
}

function formatNumber(value) {
  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(Number(value || 0));
}

function formatDate(value) {
  if (!value) return '-';
  return new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value));
}
