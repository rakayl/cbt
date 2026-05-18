import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CalendarClock, CheckCircle2, KeyRound, PlayCircle, RefreshCcw } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { api } from '../lib/api';
import { getApiErrorDetail, getApiErrorMessage } from '../lib/apiError';
import { getDeviceFingerprint, getDeviceName } from '../lib/deviceFingerprint';

export default function StudentExamsPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [joinCode, setJoinCode] = useState('');
  const [joinResult, setJoinResult] = useState(null);

  const exams = useQuery({
    queryKey: ['student-exams', search],
    queryFn: async () => (await api.get('/exams/student', { params: { page: 1, limit: 20, search } })).data.data,
  });

  const joinByCode = useMutation({
    mutationFn: async () => (await api.post('/exams/join-by-code', { code: joinCode.trim() })).data.data,
    onSuccess: (result) => {
      setJoinResult(result);
      queryClient.invalidateQueries({ queryKey: ['student-exams'] });
    },
  });

  const startExam = useMutation({
    mutationFn: async (exam) => {
      if (exam.session_id && ['started', 'reconnecting'].includes(exam.session_status)) {
        return { session_id: exam.session_id };
      }
      const token = exam.invitation_code || exam.token || joinCode.trim();
      return (
        await api.post('/exam-sessions/start', {
          exam_id: exam.exam_id,
          student_id: exam.student_id,
          token,
          device_fingerprint: getDeviceFingerprint(),
          device_name: getDeviceName(),
          user_agent: navigator.userAgent,
        })
      ).data.data;
    },
    onSuccess: (result) => navigate(`/exam/${result.session_id}`),
  });

  const currentError = exams.error || joinByCode.error || startExam.error;

  return (
    <div className="space-y-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <div className="text-sm font-bold text-[#a1a5b7]">Student Exam Center</div>
          <h2 className="mt-1 text-2xl font-extrabold text-[#181c32]">Ujian Saya</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">
            Lihat ujian yang sudah diundang oleh guru/admin, lalu mulai atau lanjutkan sesi ujian.
          </p>
        </div>
        <div className="grid grid-cols-2 gap-4 sm:w-[360px]">
          <Metric label="Tersedia" value={exams.data?.total || 0} icon={CalendarClock} />
          <Metric label="Diundang" value={exams.data?.items?.filter((item) => item.invited).length || 0} icon={CheckCircle2} />
        </div>
      </section>

      <section className="panel overflow-hidden">
        <div className="grid gap-4 border-b border-[#eff2f5] p-5 lg:grid-cols-[1fr_360px]">
          <label className="relative block">
            <input className="input h-11" placeholder="Cari ujian" value={search} onChange={(event) => setSearch(event.target.value)} />
          </label>
          <div className="flex gap-3">
            <input className="input h-11" placeholder="Masukkan kode undangan" value={joinCode} onChange={(event) => setJoinCode(event.target.value)} />
            <button className="btn btn-primary shrink-0" disabled={!joinCode.trim() || joinByCode.isPending} onClick={() => joinByCode.mutate()}>
              <KeyRound size={17} />
              Join
            </button>
          </div>
        </div>

        {joinResult ? (
          <div className="border-b border-[#eff2f5] bg-[#f1faff] p-5">
            <div className="text-sm font-extrabold text-[#009ef7]">Kode valid: {joinResult.name}</div>
            <button className="btn btn-primary mt-3" disabled={startExam.isPending} onClick={() => startExam.mutate({ ...joinResult, token: joinCode.trim() })}>
              <PlayCircle size={17} />
              Mulai Ujian Ini
            </button>
          </div>
        ) : null}

        {currentError ? (
          <div className="m-5 rounded-lg bg-[#fff5f8] px-4 py-3 text-sm font-bold text-[#f1416c]">
            <div>{getApiErrorMessage(currentError)}</div>
            {getApiErrorDetail(currentError) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(currentError)}</div> : null}
          </div>
        ) : null}

        <div className="grid gap-4 p-5 xl:grid-cols-2">
          {exams.isLoading ? <div className="rounded-lg bg-[#f5f8fa] p-4 text-sm font-semibold text-[#7e8299]">Memuat ujian...</div> : null}
          {exams.data?.items?.map((exam) => (
            <div key={exam.exam_id} className="rounded-lg border border-[#eff2f5] bg-white p-5">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <div className="font-extrabold text-[#181c32]">{exam.name}</div>
                  <div className="mt-1 text-sm font-semibold text-[#a1a5b7]">{exam.code}</div>
                </div>
                <div className="flex flex-col items-end gap-2">
                  <StatusBadge status={exam.session_status || exam.status} />
                  <AccessBadge status={accessStatusOf(exam)} />
                </div>
              </div>
              <div className="mt-4 grid gap-3 text-sm font-semibold text-[#7e8299] sm:grid-cols-4">
                <div>Durasi: {exam.duration_minutes} menit</div>
                <div>Soal: {exam.question_count || exam.metadata?.question_count || 40}</div>
                <div>Passing: {Number(exam.passing_grade || 0).toFixed(0)}</div>
                <div>Attempt: {exam.max_attempt}</div>
              </div>
              {exam.instruction ? <p className="mt-4 text-sm leading-6 text-[#7e8299]">{exam.instruction}</p> : null}
              {accessStatusOf(exam) === 'closed' && !['started', 'reconnecting'].includes(exam.session_status) ? (
                <div className="mt-5 rounded-lg bg-[#fff5f8] px-4 py-3 text-sm font-bold text-[#f1416c]">
                  Ujian sedang ditutup oleh guru/admin.
                </div>
              ) : null}
              <button className="btn btn-primary mt-5 w-full justify-center" disabled={startExam.isPending || exam.session_status === 'completed' || (accessStatusOf(exam) === 'closed' && !['started', 'reconnecting'].includes(exam.session_status))} onClick={() => startExam.mutate(exam)}>
                <PlayCircle size={17} />
                {exam.session_id && exam.session_status !== 'completed' ? 'Lanjutkan Ujian' : exam.session_status === 'completed' ? 'Sudah Selesai' : 'Mulai Ujian'}
              </button>
            </div>
          ))}
          {!exams.isLoading && exams.data?.items?.length === 0 ? (
            <div className="rounded-lg bg-[#f5f8fa] p-5 text-sm font-semibold text-[#7e8299]">Belum ada ujian tersedia.</div>
          ) : null}
        </div>

        <div className="border-t border-[#eff2f5] p-5">
          <button className="btn btn-ghost" onClick={() => exams.refetch()}>
            <RefreshCcw size={17} />
            Refresh
          </button>
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
      <div className="ml-3">
        <div className="text-xs font-bold uppercase text-[#a1a5b7]">{label}</div>
        <div className="text-xl font-extrabold text-[#181c32]">{value}</div>
      </div>
    </div>
  );
}

function StatusBadge({ status }) {
  const done = status === 'completed';
  return (
    <span className={done ? 'inline-flex rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold capitalize text-[#50cd89]' : 'inline-flex rounded-md bg-[#f1faff] px-2.5 py-1 text-xs font-extrabold capitalize text-[#009ef7]'}>
      {status || 'published'}
    </span>
  );
}

function AccessBadge({ status }) {
  const open = status === 'open';
  return (
    <span className={open ? 'inline-flex rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold text-[#50cd89]' : 'inline-flex rounded-md bg-[#fff5f8] px-2.5 py-1 text-xs font-extrabold text-[#f1416c]'}>
      {open ? 'Open' : 'Close'}
    </span>
  );
}

function accessStatusOf(exam) {
  return exam?.metadata?.access_status === 'closed' ? 'closed' : 'open';
}
