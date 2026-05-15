import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { GraduationCap, History, RefreshCcw } from 'lucide-react';
import { api } from '../lib/api';

export default function StudentClassesPage() {
  const [search, setSearch] = useState('');
  const classes = useQuery({
    queryKey: ['student-classes', search],
    queryFn: async () => (await api.get('/enrollment/me', { params: { page: 1, limit: 30, search } })).data.data,
  });
  const active = classes.data?.items?.filter((item) => item.active).length || 0;

  return (
    <div className="space-y-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <div className="text-sm font-bold text-[#a1a5b7]">Akademik Siswa</div>
          <h2 className="mt-1 text-2xl font-extrabold text-[#181c32]">Kelas yang Diikuti</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">Kelas aktif dan history perpindahan kelas siswa dari data enrollment.</p>
        </div>
        <div className="grid grid-cols-2 gap-4 sm:w-[360px]">
          <Metric label="Total History" value={classes.data?.total || 0} icon={History} />
          <Metric label="Aktif" value={active} icon={GraduationCap} />
        </div>
      </section>

      <section className="panel overflow-hidden">
        <div className="flex flex-col gap-3 border-b border-[#eff2f5] p-5 sm:flex-row sm:items-center sm:justify-between">
          <label className="relative block sm:w-80">
            <input className="input h-11" placeholder="Cari kelas" value={search} onChange={(event) => setSearch(event.target.value)} />
          </label>
          <button className="btn btn-ghost" onClick={() => classes.refetch()}>
            <RefreshCcw size={17} />
            Refresh
          </button>
        </div>

        <div className="grid gap-4 p-5 lg:grid-cols-2">
          {classes.isLoading ? <div className="rounded-lg bg-[#f5f8fa] p-4 text-sm font-semibold text-[#7e8299]">Memuat kelas...</div> : null}
          {classes.data?.items?.map((item) => (
            <div key={item.id} className="rounded-lg border border-[#eff2f5] bg-white p-5">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <div className="font-extrabold text-[#181c32]">{item.class_room_name}</div>
                  <div className="mt-1 text-sm font-semibold text-[#a1a5b7]">{item.class_room_code}</div>
                </div>
                <StatusBadge active={item.active} status={item.status} />
              </div>
              <div className="mt-4 grid gap-3 text-sm font-semibold text-[#7e8299] sm:grid-cols-2">
                <div>Program: {item.study_program_name || '-'}</div>
                <div>Masuk: {formatDate(item.enrolled_at)}</div>
                <div>Keluar: {formatDate(item.exited_at)}</div>
                <div>Siswa: {item.student_name}</div>
              </div>
              {item.description ? <p className="mt-4 text-sm leading-6 text-[#7e8299]">{item.description}</p> : null}
            </div>
          ))}
          {!classes.isLoading && classes.data?.items?.length === 0 ? (
            <div className="rounded-lg bg-[#f5f8fa] p-5 text-sm font-semibold text-[#7e8299]">Belum ada enrollment kelas.</div>
          ) : null}
        </div>
      </section>
    </div>
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

function StatusBadge({ active, status }) {
  return <span className={active ? 'inline-flex rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold text-[#50cd89]' : 'inline-flex rounded-md bg-[#f5f8fa] px-2.5 py-1 text-xs font-extrabold text-[#7e8299]'}>{active ? 'Aktif' : status}</span>;
}

function formatDate(value) {
  if (!value) return '-';
  return new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium' }).format(new Date(value));
}
