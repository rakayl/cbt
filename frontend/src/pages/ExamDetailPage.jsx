import { useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, CheckCircle2, Copy, Lock, RefreshCcw, UserPlus, Users } from 'lucide-react';
import { api } from '../lib/api';
import { getApiErrorDetail, getApiErrorMessage } from '../lib/apiError';

export default function ExamDetailPage() {
  const { examId } = useParams();
  const queryClient = useQueryClient();
  const [searchInvited, setSearchInvited] = useState('');
  const [searchUninvited, setSearchUninvited] = useState('');
  const [selectedStudentIds, setSelectedStudentIds] = useState([]);
  const [copiedCode, setCopiedCode] = useState('');

  const exam = useQuery({
    queryKey: ['exam-detail', examId],
    enabled: Boolean(examId),
    queryFn: async () => (await api.get(`/exams/${examId}`)).data.data,
  });

  const roster = useQuery({
    queryKey: ['exam-invite-roster', examId],
    enabled: Boolean(examId),
    queryFn: async () => (await api.get(`/exams/${examId}/invite-roster`)).data.data,
  });

  const inviteStudents = useMutation({
    mutationFn: async () => (await api.post(`/exams/${examId}/invite-students`, { student_ids: selectedStudentIds })).data.data,
    onSuccess: () => {
      setSelectedStudentIds([]);
      queryClient.invalidateQueries({ queryKey: ['exam-invite-roster', examId] });
      queryClient.invalidateQueries({ queryKey: ['exam-detail', examId] });
      queryClient.invalidateQueries({ queryKey: ['exams'] });
    },
  });

  const updateAccess = useMutation({
    mutationFn: async (accessStatus) => (await api.put(`/exams/${examId}/access-status`, { access_status: accessStatus })).data.data,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['exam-invite-roster', examId] });
      queryClient.invalidateQueries({ queryKey: ['exam-detail', examId] });
      queryClient.invalidateQueries({ queryKey: ['exams'] });
    },
  });

  const accessStatus = accessStatusOf(exam.data || roster.data);
  const invited = useMemo(
    () => filterStudents(roster.data?.invited || [], searchInvited, true),
    [roster.data, searchInvited],
  );
  const uninvited = useMemo(
    () => filterStudents(roster.data?.uninvited || [], searchUninvited, false),
    [roster.data, searchUninvited],
  );
  const currentError = exam.error || roster.error || inviteStudents.error || updateAccess.error;

  async function copyCode(code) {
    if (!code) return;
    await navigator.clipboard?.writeText(code);
    setCopiedCode(code);
    window.setTimeout(() => setCopiedCode(''), 1600);
  }

  function toggleStudent(studentId) {
    setSelectedStudentIds((current) => (
      current.includes(studentId)
        ? current.filter((id) => id !== studentId)
        : [...current, studentId]
    ));
  }

  return (
    <div className="space-y-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <Link className="mb-3 inline-flex items-center gap-2 text-sm font-extrabold text-[#009ef7]" to="/exams">
            <ArrowLeft size={17} />
            Kembali ke Jadwal Ujian
          </Link>
          <div className="text-sm font-bold text-[#a1a5b7]">Detail Jadwal Ujian</div>
          <h2 className="mt-1 text-2xl font-extrabold text-[#181c32]">{exam.data?.name || 'Memuat detail ujian...'}</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">
            Kelola status open/close dan undangan siswa. Exam close tidak bisa dimulai siswa, tetapi sesi yang sudah berjalan tetap bisa dilanjutkan.
          </p>
        </div>
        <div className="flex flex-col gap-3 sm:flex-row">
          <button className={accessStatus === 'open' ? 'btn btn-primary justify-center' : 'btn btn-ghost justify-center'} type="button" disabled={updateAccess.isPending || exam.data?.status !== 'published'} onClick={() => updateAccess.mutate('open')}>
            <CheckCircle2 size={17} />
            Open
          </button>
          <button className={accessStatus === 'closed' ? 'btn justify-center bg-[#fff5f8] text-[#f1416c]' : 'btn btn-ghost justify-center'} type="button" disabled={updateAccess.isPending || exam.data?.status !== 'published'} onClick={() => updateAccess.mutate('closed')}>
            <Lock size={17} />
            Close
          </button>
        </div>
      </section>

      {currentError ? (
        <div className="rounded-lg bg-[#fff5f8] px-4 py-3 text-sm font-bold text-[#f1416c]">
          <div>{getApiErrorMessage(currentError)}</div>
          {getApiErrorDetail(currentError) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(currentError)}</div> : null}
        </div>
      ) : null}

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Metric label="Status Publish" value={exam.data?.status || '-'} icon={CheckCircle2} />
        <Metric label="Akses Exam" value={accessStatus} icon={accessStatus === 'open' ? CheckCircle2 : Lock} />
        <Metric label="Sudah Diinvite" value={roster.data?.invited_count || 0} icon={Users} />
        <Metric label="Belum Diinvite" value={roster.data?.uninvited_count || 0} icon={UserPlus} />
      </section>

      <section className="panel p-5">
        <div className="grid gap-4 lg:grid-cols-4">
          <Info label="Kode" value={exam.data?.code || '-'} />
          <Info label="Token" value={exam.data?.metadata?.exam_token || '-'} mono />
          <Info label="Durasi" value={`${exam.data?.metadata?.duration_minutes || 120} menit`} />
          <Info label="Jumlah Soal" value={`${exam.data?.metadata?.question_count || 40} soal`} />
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-2">
        <div className="panel overflow-hidden">
          <div className="border-b border-[#eff2f5] p-5">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h3 className="text-lg font-extrabold text-[#181c32]">Siswa Sudah Diinvite</h3>
                <p className="mt-1 text-sm font-semibold text-[#a1a5b7]">Kode undangan personal tersedia untuk setiap siswa.</p>
              </div>
              <button className="btn btn-ghost justify-center" type="button" onClick={() => roster.refetch()}>
                <RefreshCcw size={17} />
                Refresh
              </button>
            </div>
            <input className="input mt-4 h-11" placeholder="Cari siswa terundang" value={searchInvited} onChange={(event) => setSearchInvited(event.target.value)} />
          </div>
          <div className="max-h-[560px] overflow-y-auto p-5">
            {roster.isLoading ? <div className="rounded-lg bg-[#f5f8fa] p-4 text-sm font-semibold text-[#7e8299]">Memuat siswa terundang...</div> : null}
            <div className="space-y-3">
              {invited.map((item) => (
                <div key={item.id} className="rounded-lg border border-[#eff2f5] bg-white p-4">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <div className="truncate font-extrabold text-[#181c32]">{item.student_name}</div>
                      <div className="mt-1 text-xs font-semibold text-[#a1a5b7]">{item.student_code}</div>
                    </div>
                    <StatusBadge status={item.status} />
                  </div>
                  <button className="mt-3 inline-flex rounded-md bg-[#f1faff] px-2.5 py-1 font-mono text-xs font-extrabold text-[#009ef7]" type="button" onClick={() => copyCode(item.invitation_code)}>
                    <Copy size={13} className="mr-1" />
                    {copiedCode === item.invitation_code ? 'Copied' : item.invitation_code}
                  </button>
                </div>
              ))}
            </div>
            {!roster.isLoading && invited.length === 0 ? (
              <div className="rounded-lg bg-[#f5f8fa] p-5 text-sm font-semibold text-[#7e8299]">Belum ada siswa terundang.</div>
            ) : null}
          </div>
        </div>

        <div className="panel overflow-hidden">
          <div className="border-b border-[#eff2f5] p-5">
            <h3 className="text-lg font-extrabold text-[#181c32]">Siswa Belum Diinvite</h3>
            <p className="mt-1 text-sm font-semibold text-[#a1a5b7]">Pilih siswa, lalu invite agar exam muncul di halaman Ujian Saya siswa.</p>
            <input className="input mt-4 h-11" placeholder="Cari siswa belum diinvite" value={searchUninvited} onChange={(event) => setSearchUninvited(event.target.value)} />
          </div>
          <div className="max-h-[560px] overflow-y-auto p-5">
            {roster.isLoading ? <div className="rounded-lg bg-[#f5f8fa] p-4 text-sm font-semibold text-[#7e8299]">Memuat siswa...</div> : null}
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-1 2xl:grid-cols-2">
              {uninvited.map((item) => {
                const checked = selectedStudentIds.includes(item.id);
                return (
                  <label key={item.id} className={checked ? 'flex cursor-pointer items-start gap-3 rounded-lg border border-[#009ef7] bg-[#f1faff] p-3' : 'flex cursor-pointer items-start gap-3 rounded-lg border border-[#eff2f5] bg-white p-3 hover:border-[#009ef7]'}>
                    <input className="mt-1" type="checkbox" checked={checked} onChange={() => toggleStudent(item.id)} />
                    <span className="min-w-0">
                      <span className="block truncate text-sm font-extrabold text-[#181c32]">{item.name}</span>
                      <span className="mt-0.5 block truncate text-xs font-semibold text-[#a1a5b7]">{item.code}</span>
                    </span>
                  </label>
                );
              })}
            </div>
            {!roster.isLoading && uninvited.length === 0 ? (
              <div className="rounded-lg bg-[#f5f8fa] p-5 text-sm font-semibold text-[#7e8299]">Semua siswa sudah diinvite atau data tidak ditemukan.</div>
            ) : null}
          </div>
          <div className="border-t border-[#eff2f5] p-5">
            <button className="btn btn-primary w-full justify-center" type="button" disabled={!selectedStudentIds.length || inviteStudents.isPending || exam.data?.status !== 'published'} onClick={() => inviteStudents.mutate()}>
              <UserPlus size={17} />
              {inviteStudents.isPending ? 'Mengundang...' : `Invite ${selectedStudentIds.length} Siswa`}
            </button>
          </div>
        </div>
      </section>
    </div>
  );
}

function Metric({ label, value, icon: Icon }) {
  return (
    <div className="panel flex items-center p-4">
      <div className="grid h-11 w-11 place-items-center rounded-lg bg-[#f1faff] text-[#009ef7]">
        <Icon size={21} />
      </div>
      <div className="ml-3 min-w-0">
        <div className="text-xs font-bold uppercase text-[#a1a5b7]">{label}</div>
        <div className="truncate text-xl font-extrabold capitalize text-[#181c32]">{value}</div>
      </div>
    </div>
  );
}

function Info({ label, value, mono }) {
  return (
    <div className="rounded-lg bg-[#f5f8fa] p-4">
      <div className="text-xs font-bold uppercase text-[#a1a5b7]">{label}</div>
      <div className={mono ? 'mt-1 truncate font-mono text-sm font-extrabold text-[#181c32]' : 'mt-1 truncate text-sm font-extrabold text-[#181c32]'}>{value}</div>
    </div>
  );
}

function StatusBadge({ status }) {
  const accepted = status === 'accepted';
  return (
    <span className={accepted ? 'inline-flex rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold capitalize text-[#50cd89]' : 'inline-flex rounded-md bg-[#f1faff] px-2.5 py-1 text-xs font-extrabold capitalize text-[#009ef7]'}>
      {accepted ? 'Accepted' : status || 'Invited'}
    </span>
  );
}

function filterStudents(items, keyword, invited) {
  const clean = keyword.trim().toLowerCase();
  if (!clean) return items;
  return items.filter((item) => {
    const name = invited ? item.student_name : item.name;
    const code = invited ? item.student_code : item.code;
    return `${name} ${code}`.toLowerCase().includes(clean);
  });
}

function accessStatusOf(source) {
  return source?.metadata?.access_status || source?.access_status || 'open';
}
