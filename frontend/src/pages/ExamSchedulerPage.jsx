import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CalendarClock, CheckCircle2, Copy, Lock, PlayCircle, Plus, RefreshCcw, Share2, Tag, Trash2, Trophy, UserPlus, X } from 'lucide-react';
import { api } from '../lib/api';
import { getApiErrorDetail, getApiErrorMessage } from '../lib/apiError';

const emptyQuestionPool = { question_tag_id: '', question_count: 10 };

const initialForm = {
  name: '',
  scheduled_at: '',
  duration_minutes: 120,
  passing_grade: 60,
  question_count: 40,
  question_pools: [{ ...emptyQuestionPool }],
  max_attempt: 1,
  random_question: true,
  random_option: true,
  timer_mode: 'recovery_pause',
  max_recovery_pause_minutes: 60,
  max_reconnect_attempts: 3,
  device_change_requires_approval: true,
  fullscreen_required: true,
  webcam_required: true,
  snapshot_interval_seconds: 60,
  critical_score_threshold: 90,
  result_visibility: 'immediate',
  result_release_at: '',
  instruction: '',
};

export default function ExamSchedulerPage() {
  const queryClient = useQueryClient();
  const [form, setForm] = useState(initialForm);
  const [search, setSearch] = useState('');
  const [publishedExam, setPublishedExam] = useState(null);
  const [copiedCode, setCopiedCode] = useState('');
  const [inviteExam, setInviteExam] = useState(null);
  const [selectedStudentIds, setSelectedStudentIds] = useState([]);

  const tags = useQuery({
    queryKey: ['question-tags'],
    queryFn: async () => (await api.get('/question-tags/', { params: { page: 1, limit: 200 } })).data.data.items,
  });

  const exams = useQuery({
    queryKey: ['exams', search],
    queryFn: async () => {
      const { data } = await api.get('/exams/', { params: { page: 1, limit: 20, search } });
      return data.data;
    },
  });
  const students = useQuery({
    queryKey: ['students', 'invite-select'],
    queryFn: async () => (await api.get('/students/', { params: { page: 1, limit: 300 } })).data.data.items,
  });
  const invites = useQuery({
    queryKey: ['exam-invites', inviteExam?.id],
    enabled: Boolean(inviteExam?.id),
    queryFn: async () => (await api.get(`/exams/${inviteExam.id}/invites`)).data.data,
  });

  const totalPublished = useMemo(
    () => exams.data?.items?.filter((exam) => exam.status === 'published').length || 0,
    [exams.data],
  );
  const poolQuestionCount = useMemo(() => questionCountFromPools(form.question_pools), [form.question_pools]);
  const usingPools = hasQuestionPools(form.question_pools);
  const invitedStudentIds = useMemo(
    () => new Set((invites.data || []).map((invite) => invite.student_id)),
    [invites.data],
  );

  const createAndPublish = useMutation({
    mutationFn: async () => {
      validateForm(form);
      const code = `EXM-${Date.now().toString().slice(-8)}`;
      const payload = toExamPayload({ ...form, code, status: 'draft' });
      const created = (await api.post('/exams/', payload)).data.data;
      const published = (await api.post(`/exams/${created.id}/publish`, toPublishPayload(form))).data.data;
      return { ...created, ...published, name: created.name };
    },
    onSuccess: (exam) => {
      setPublishedExam(exam);
      setForm({ ...initialForm, question_pools: [{ ...emptyQuestionPool }] });
      queryClient.invalidateQueries({ queryKey: ['exams'] });
    },
  });

  const publishExisting = useMutation({
    mutationFn: async (exam) => {
      const published = (await api.post(`/exams/${exam.id}/publish`, toPublishPayloadFromExam(exam))).data.data;
      return { ...exam, ...published };
    },
    onSuccess: (exam) => {
      setPublishedExam(exam);
      queryClient.invalidateQueries({ queryKey: ['exams'] });
    },
  });

  const shareCode = useMutation({
    mutationFn: async (exam) => (await api.post(`/exams/${exam.id}/share-code`)).data.data,
    onSuccess: (result) => {
      setPublishedExam({ id: result.exam_id, exam_token: result.code, status: 'published' });
      copyCode(result.code);
    },
  });

  const deleteDraft = useMutation({
    mutationFn: async (exam) => (await api.delete(`/exams/${exam.id}`)).data.data,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['exams'] });
    },
  });
  const createRevision = useMutation({
    mutationFn: async (exam) => (await api.post(`/exams/${exam.id}/revisions`)).data.data,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['exams'] });
    },
  });
  const inviteStudents = useMutation({
    mutationFn: async () => (await api.post(`/exams/${inviteExam.id}/invite-students`, { student_ids: selectedStudentIds })).data.data,
    onSuccess: () => {
      setSelectedStudentIds([]);
      queryClient.invalidateQueries({ queryKey: ['exam-invites', inviteExam?.id] });
      queryClient.invalidateQueries({ queryKey: ['student-exams'] });
    },
  });
  const updateAccess = useMutation({
    mutationFn: async ({ exam, accessStatus }) => (await api.put(`/exams/${exam.id}/access-status`, { access_status: accessStatus })).data.data,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['exams'] });
      queryClient.invalidateQueries({ queryKey: ['student-exams'] });
    },
  });

  function updateField(name, value) {
    setForm((current) => ({ ...current, [name]: value }));
  }

  function updateQuestionPool(index, key, value) {
    setForm((current) => {
      const questionPools = current.question_pools.map((pool, poolIndex) => (poolIndex === index ? { ...pool, [key]: value } : pool));
      const nextCount = hasQuestionPools(questionPools) ? questionCountFromPools(questionPools) : current.question_count;
      return { ...current, question_pools: questionPools, question_count: nextCount };
    });
  }

  function addQuestionPool() {
    setForm((current) => ({ ...current, question_pools: [...current.question_pools, { ...emptyQuestionPool }] }));
  }

  function removeQuestionPool(index) {
    setForm((current) => {
      const questionPools = current.question_pools.filter((_, poolIndex) => poolIndex !== index);
      const normalized = questionPools.length ? questionPools : [{ ...emptyQuestionPool }];
      return { ...current, question_pools: normalized, question_count: hasQuestionPools(normalized) ? questionCountFromPools(normalized) : current.question_count };
    });
  }

  async function copyCode(code) {
    if (!code) return;
    await navigator.clipboard?.writeText(code);
    setCopiedCode(code);
    window.setTimeout(() => setCopiedCode(''), 1800);
  }

  function submit(event) {
    event.preventDefault();
    createAndPublish.mutate();
  }

  const currentError =
    createAndPublish.error ||
    publishExisting.error ||
    shareCode.error ||
    deleteDraft.error ||
    createRevision.error ||
    inviteStudents.error ||
    updateAccess.error;

  function handleDeleteDraft(exam) {
    const ok = window.confirm(`Hapus draft ujian "${exam.name}"?`);
    if (ok) {
      deleteDraft.mutate(exam);
    }
  }

  function handleCreateRevision(exam) {
    const ok = window.confirm(`Buat draft revisi dari ujian published "${exam.name}"?\n\nExam published akan tetap terkunci untuk menjaga invite, session, dan hasil siswa yang sudah berjalan.`);
    if (ok) {
      createRevision.mutate(exam);
    }
  }

  function openInvitePanel(exam) {
    setInviteExam(exam);
    setSelectedStudentIds([]);
  }

  function toggleStudent(studentId) {
    if (invitedStudentIds.has(studentId)) return;
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
          <div className="text-sm font-bold text-[#a1a5b7]">Exam Management</div>
          <h2 className="mt-1 text-2xl font-extrabold text-[#181c32]">Scheduler & Publish Ujian</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">
            Buat jadwal ujian, publish token, lalu bagikan kode unik ke siswa atau invite siswa dari modul exam.
          </p>
        </div>
        <div className="grid grid-cols-2 gap-4 sm:w-[360px]">
          <Metric label="Total Ujian" value={exams.data?.total || 0} icon={CalendarClock} />
          <Metric label="Published" value={totalPublished} icon={CheckCircle2} />
        </div>
      </section>

      {publishedExam ? (
        <section className="panel flex flex-col gap-4 p-5 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <div className="text-sm font-extrabold text-[#50cd89]">Exam berhasil dipublish</div>
            <div className="mt-1 text-xl font-extrabold text-[#181c32]">{publishedExam.name || 'Published exam'}</div>
            <div className="mt-2 text-sm font-semibold text-[#7e8299]">Kode ujian: {publishedExam.exam_token}</div>
          </div>
          <button className="btn btn-primary justify-center" type="button" onClick={() => copyCode(publishedExam.exam_token)}>
            <Copy size={17} />
            {copiedCode === publishedExam.exam_token ? 'Copied' : 'Copy Code'}
          </button>
        </section>
      ) : null}

      {inviteExam ? (
        <section className="panel overflow-hidden">
          <div className="flex flex-col gap-3 border-b border-[#eff2f5] p-5 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <div className="text-sm font-extrabold text-[#009ef7]">Invite Siswa</div>
              <h3 className="mt-1 text-xl font-extrabold text-[#181c32]">{inviteExam.name}</h3>
              <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">
                Setelah siswa diundang, ujian baru muncul di halaman Ujian Saya milik siswa. Kode yang dipakai siswa adalah kode undangan personal.
              </p>
            </div>
            <button className="btn btn-ghost justify-center" type="button" onClick={() => setInviteExam(null)}>
              <X size={17} />
              Tutup
            </button>
          </div>
          <div className="grid gap-5 p-5 xl:grid-cols-[minmax(0,1fr)_360px]">
            <div>
              <div className="mb-3 text-sm font-extrabold text-[#181c32]">Pilih siswa</div>
              <div className="grid max-h-80 gap-3 overflow-y-auto pr-1 md:grid-cols-2">
                {students.isLoading ? (
                  <div className="rounded-lg bg-[#f5f8fa] p-4 text-sm font-semibold text-[#7e8299]">Memuat siswa...</div>
                ) : null}
                {students.data?.map((student) => {
                  const alreadyInvited = invitedStudentIds.has(student.id);
                  const checked = selectedStudentIds.includes(student.id) || alreadyInvited;
                  return (
                    <label
                      key={student.id}
                      className={alreadyInvited ? 'flex cursor-not-allowed items-start gap-3 rounded-lg border border-[#eff2f5] bg-[#f5f8fa] p-3 opacity-75' : 'flex cursor-pointer items-start gap-3 rounded-lg border border-[#eff2f5] bg-white p-3 hover:border-[#009ef7]'}
                    >
                      <input
                        className="mt-1"
                        type="checkbox"
                        checked={checked}
                        disabled={alreadyInvited}
                        onChange={() => toggleStudent(student.id)}
                      />
                      <span className="min-w-0">
                        <span className="block truncate text-sm font-extrabold text-[#181c32]">{student.name}</span>
                        <span className="mt-0.5 block truncate text-xs font-semibold text-[#a1a5b7]">{student.code}</span>
                        {alreadyInvited ? <span className="mt-1 inline-flex rounded-md bg-[#e8fff3] px-2 py-0.5 text-xs font-extrabold text-[#50cd89]">Sudah diundang</span> : null}
                      </span>
                    </label>
                  );
                })}
                {!students.isLoading && students.data?.length === 0 ? (
                  <div className="rounded-lg bg-[#f5f8fa] p-4 text-sm font-semibold text-[#7e8299]">Belum ada siswa.</div>
                ) : null}
              </div>
              <button className="btn btn-primary mt-4 w-full justify-center sm:w-auto" type="button" disabled={!selectedStudentIds.length || inviteStudents.isPending} onClick={() => inviteStudents.mutate()}>
                <UserPlus size={17} />
                {inviteStudents.isPending ? 'Mengundang...' : `Invite ${selectedStudentIds.length} Siswa`}
              </button>
            </div>
            <div className="rounded-lg border border-[#eff2f5] bg-[#f9fafb] p-4">
              <div className="text-sm font-extrabold text-[#181c32]">Siswa Terundang</div>
              <div className="mt-3 max-h-80 space-y-2 overflow-y-auto">
                {invites.isLoading ? <div className="text-sm font-semibold text-[#7e8299]">Memuat invite...</div> : null}
                {invites.data?.map((invite) => (
                  <div key={invite.id} className="rounded-lg bg-white p-3">
                    <div className="font-extrabold text-[#181c32]">{invite.student_name}</div>
                    <div className="mt-1 text-xs font-semibold text-[#a1a5b7]">{invite.student_code}</div>
                    <button className="mt-2 inline-flex rounded-md bg-[#f1faff] px-2 py-1 font-mono text-xs font-extrabold text-[#009ef7]" type="button" onClick={() => copyCode(invite.invitation_code)}>
                      {copiedCode === invite.invitation_code ? 'Copied' : invite.invitation_code}
                    </button>
                  </div>
                ))}
                {!invites.isLoading && invites.data?.length === 0 ? (
                  <div className="text-sm font-semibold text-[#7e8299]">Belum ada siswa terundang.</div>
                ) : null}
              </div>
            </div>
          </div>
        </section>
      ) : null}

      <section className="grid min-w-0 gap-6 2xl:grid-cols-[480px_minmax(0,1fr)]">
        <form className="panel h-fit min-w-0 overflow-visible" onSubmit={submit}>
          <div className="border-b border-[#eff2f5] p-5">
            <div className="text-sm font-bold text-[#a1a5b7]">Tambah Ujian</div>
            <h3 className="mt-1 text-xl font-extrabold text-[#181c32]">Publish Exam Baru</h3>
          </div>
          <div className="space-y-4 p-5">
            <Field label="Nama Ujian">
              <input
                className="input"
                placeholder="Contoh: UTS Matematika Kelas X"
                value={form.name}
                onChange={(event) => updateField('name', event.target.value)}
              />
            </Field>
            <Field label="Jadwal Mulai">
              <input
                className="input"
                type="datetime-local"
                value={form.scheduled_at}
                onChange={(event) => updateField('scheduled_at', event.target.value)}
              />
            </Field>
            <div className="grid gap-4 sm:grid-cols-2 2xl:grid-cols-1">
              <Field label="Durasi Menit">
                <input
                  className="input"
                  min="1"
                  max="1440"
                  type="number"
                  value={form.duration_minutes}
                  onChange={(event) => updateField('duration_minutes', Number(event.target.value))}
                />
              </Field>
              <Field label="Passing Grade">
                <input
                  className="input"
                  min="0"
                  max="100"
                  type="number"
                  value={form.passing_grade}
                  onChange={(event) => updateField('passing_grade', Number(event.target.value))}
                />
              </Field>
            </div>
            <div className="grid gap-4 sm:grid-cols-2 2xl:grid-cols-1">
              <Field label="Jumlah Soal">
                <input
                  className="input"
                  min="1"
                  max="500"
                  type="number"
                  disabled={usingPools}
                  value={usingPools ? poolQuestionCount : form.question_count}
                  onChange={(event) => updateField('question_count', Number(event.target.value))}
                />
              </Field>
              <Field label="Max Attempt">
                <input
                  className="input"
                  min="1"
                  max="20"
                  type="number"
                  value={form.max_attempt}
                  onChange={(event) => updateField('max_attempt', Number(event.target.value))}
                />
              </Field>
            </div>
            <div className="grid gap-3 sm:grid-cols-2 2xl:grid-cols-1">
              <label className="flex items-center gap-3 rounded-lg border border-[#eff2f5] bg-[#f9fafb] p-3 text-sm font-bold text-[#3f4254]">
                <input
                  type="checkbox"
                  checked={form.random_question}
                  onChange={(event) => updateField('random_question', event.target.checked)}
                />
                Random soal
              </label>
              <label className="flex items-center gap-3 rounded-lg border border-[#eff2f5] bg-[#f9fafb] p-3 text-sm font-bold text-[#3f4254]">
                <input
                  type="checkbox"
                  checked={form.random_option}
                  onChange={(event) => updateField('random_option', event.target.checked)}
                />
                Random opsi
              </label>
            </div>
            <div className="rounded-lg border border-[#eff2f5] bg-[#f9fafb] p-4">
              <div className="text-sm font-extrabold text-[#181c32]">Recovery & Anti Cheat Policy</div>
              <div className="mt-4 grid gap-3 sm:grid-cols-2 2xl:grid-cols-1">
                <Field label="Mode Timer">
                  <select className="input bg-white" value={form.timer_mode} onChange={(event) => updateField('timer_mode', event.target.value)}>
                    <option value="recovery_pause">Recovery pause</option>
                    <option value="strict">Strict berjalan terus</option>
                  </select>
                </Field>
                <Field label="Max Pause Menit">
                  <input className="input bg-white" min="1" max="240" type="number" value={form.max_recovery_pause_minutes} onChange={(event) => updateField('max_recovery_pause_minutes', Number(event.target.value))} />
                </Field>
                <Field label="Max Reconnect">
                  <input className="input bg-white" min="1" max="20" type="number" value={form.max_reconnect_attempts} onChange={(event) => updateField('max_reconnect_attempts', Number(event.target.value))} />
                </Field>
                <Field label="Snapshot Detik">
                  <input className="input bg-white" min="30" max="300" type="number" value={form.snapshot_interval_seconds} onChange={(event) => updateField('snapshot_interval_seconds', Number(event.target.value))} />
                </Field>
              </div>
              <div className="mt-3 grid gap-3 sm:grid-cols-2 2xl:grid-cols-1">
                <label className="flex items-center gap-3 rounded-lg border border-[#eff2f5] bg-white p-3 text-sm font-bold text-[#3f4254]">
                  <input type="checkbox" checked={form.device_change_requires_approval} onChange={(event) => updateField('device_change_requires_approval', event.target.checked)} />
                  Device berubah perlu review
                </label>
                <label className="flex items-center gap-3 rounded-lg border border-[#eff2f5] bg-white p-3 text-sm font-bold text-[#3f4254]">
                  <input type="checkbox" checked={form.fullscreen_required} onChange={(event) => updateField('fullscreen_required', event.target.checked)} />
                  Wajib fullscreen
                </label>
                <label className="flex items-center gap-3 rounded-lg border border-[#eff2f5] bg-white p-3 text-sm font-bold text-[#3f4254]">
                  <input type="checkbox" checked={form.webcam_required} onChange={(event) => updateField('webcam_required', event.target.checked)} />
                  Wajib webcam
                </label>
                <Field label="Critical Score">
                  <input className="input bg-white" min="30" max="200" type="number" value={form.critical_score_threshold} onChange={(event) => updateField('critical_score_threshold', Number(event.target.value))} />
                </Field>
              </div>
            </div>
            <div className="rounded-lg border border-[#eff2f5] bg-[#f9fafb] p-4">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div className="min-w-0">
                  <div className="flex items-center gap-2 text-sm font-extrabold text-[#181c32]">
                    <Tag size={17} />
                    Komposisi Jenis Soal
                  </div>
                  <p className="mt-1 text-sm font-medium text-[#a1a5b7]">Pilih tag dan kuota. Total soal otomatis mengikuti total komposisi.</p>
                </div>
                <button className="btn btn-ghost w-full justify-center sm:w-auto sm:shrink-0" type="button" onClick={addQuestionPool}>
                  <Plus size={17} />
                  Baris
                </button>
              </div>
              <div className="mt-4 space-y-3">
                {form.question_pools.map((pool, index) => (
                  <div key={`${pool.question_tag_id}-${index}`} className="grid min-w-0 gap-3 sm:grid-cols-[minmax(0,1fr)_110px_42px] 2xl:grid-cols-1">
                    <TagSelect2
                      tags={tags.data || []}
                      value={pool.question_tag_id}
                      onChange={(value) => updateQuestionPool(index, 'question_tag_id', value)}
                    />
                    <input
                      className="input bg-white"
                      min="1"
                      max="500"
                      type="number"
                      value={pool.question_count}
                      onChange={(event) => updateQuestionPool(index, 'question_count', Number(event.target.value))}
                    />
                    <button className="grid h-11 w-full place-items-center rounded-lg bg-[#fff5f8] text-[#f1416c] disabled:opacity-60 sm:w-11 2xl:w-full" type="button" disabled={form.question_pools.length <= 1} onClick={() => removeQuestionPool(index)}>
                      <Trash2 size={16} />
                    </button>
                  </div>
                ))}
              </div>
              <div className="mt-3 rounded-lg bg-white px-3 py-2 text-sm font-extrabold text-[#3f4254]">
                Total komposisi: {poolQuestionCount || 0} soal
              </div>
            </div>
            <Field label="Instruksi">
              <textarea
                className="input min-h-28"
                placeholder="Instruksi ujian untuk siswa"
                value={form.instruction}
                onChange={(event) => updateField('instruction', event.target.value)}
              />
            </Field>
            <div className="rounded-lg border border-[#eff2f5] bg-[#f9fafb] p-4">
              <div className="text-sm font-extrabold text-[#181c32]">Pengaturan Hasil Siswa</div>
              <div className="mt-4 grid gap-3 sm:grid-cols-2 2xl:grid-cols-1">
                <Field label="Visibilitas Nilai">
                  <select className="input bg-white" value={form.result_visibility} onChange={(event) => updateField('result_visibility', event.target.value)}>
                    <option value="immediate">Nilai langsung tampil</option>
                    <option value="hidden">Sembunyikan dari siswa</option>
                    <option value="manual_release">Tampil setelah dirilis guru/admin</option>
                    <option value="after_date">Tampil setelah tanggal tertentu</option>
                  </select>
                </Field>
                <Field label="Tanggal Rilis">
                  <input className="input bg-white" type="datetime-local" disabled={form.result_visibility !== 'after_date'} value={form.result_release_at} onChange={(event) => updateField('result_release_at', event.target.value)} />
                </Field>
              </div>
              <p className="mt-3 text-sm font-semibold text-[#7e8299]">Isi soal, opsi, jawaban siswa, dan kunci tetap tidak tampil di akun siswa untuk menjaga keamanan bank soal.</p>
            </div>
            {currentError ? (
              <div className="rounded-lg bg-[#fff5f8] px-4 py-3 text-sm font-bold text-[#f1416c]">
                <div>{getApiErrorMessage(currentError)}</div>
                {getApiErrorDetail(currentError) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(currentError)}</div> : null}
              </div>
            ) : null}
            <button className="btn btn-primary w-full justify-center" disabled={createAndPublish.isPending}>
              <PlayCircle size={18} />
              {createAndPublish.isPending ? 'Publishing...' : 'Publish Exam'}
            </button>
          </div>
        </form>

        <div className="panel min-w-0 overflow-hidden">
          <div className="flex flex-col gap-4 border-b border-[#eff2f5] p-5 md:flex-row md:items-center md:justify-between">
            <div>
              <h3 className="text-lg font-extrabold text-[#181c32]">Daftar Ujian</h3>
              <p className="text-sm font-medium text-[#a1a5b7]">Draft bisa dipublish, lalu published exam dipakai untuk invite siswa.</p>
            </div>
            <div className="flex flex-col gap-3 sm:flex-row">
              <label className="sm:w-72">
                <input className="input h-11" placeholder="Cari ujian" value={search} onChange={(event) => setSearch(event.target.value)} />
              </label>
              <button className="btn btn-ghost justify-center" type="button" onClick={() => exams.refetch()}>
                <RefreshCcw size={17} />
                Refresh
              </button>
            </div>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full min-w-[860px] text-left text-sm">
              <thead className="border-b border-[#eff2f5] text-xs font-extrabold uppercase text-[#a1a5b7]">
                <tr>
                  <th className="px-5 py-4">Kode</th>
                  <th className="px-5 py-4">Nama</th>
                  <th className="px-5 py-4">Status</th>
                  <th className="px-5 py-4">Akses</th>
                  <th className="px-5 py-4">Durasi</th>
                  <th className="px-5 py-4">Soal</th>
                  <th className="px-5 py-4">Token</th>
                  <th className="px-5 py-4 text-right">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#eff2f5]">
                {exams.isLoading ? (
                  <tr>
                    <td className="px-5 py-10 text-center font-semibold text-[#7e8299]" colSpan={8}>Memuat jadwal ujian...</td>
                  </tr>
                ) : null}
                {exams.data?.items?.map((exam) => {
                  const token = exam.metadata?.exam_token || '';
                  return (
                    <tr key={exam.id} className="hover:bg-[#f9fafb]">
                      <td className="px-5 py-4 font-extrabold text-[#181c32]">{exam.code}</td>
                      <td className="px-5 py-4">
                        <div className="font-bold text-[#181c32]">{exam.name}</div>
                        <div className="mt-1 text-xs font-semibold text-[#a1a5b7]">{exam.metadata?.scheduled_at || exam.description || '-'}</div>
                      </td>
                      <td className="px-5 py-4">
                        <StatusBadge status={exam.status} />
                      </td>
                      <td className="px-5 py-4">
                        <AccessBadge status={accessStatusOf(exam)} />
                      </td>
                    <td className="px-5 py-4 font-semibold text-[#7e8299]">{exam.metadata?.duration_minutes || 120} menit</td>
                    <td className="px-5 py-4 font-semibold text-[#7e8299]">
                      <div>{exam.metadata?.question_count || 40} soal</div>
                      {exam.metadata?.question_pools?.length ? (
                        <div className="mt-1 text-xs font-semibold text-[#a1a5b7]">{exam.metadata.question_pools.length} jenis soal</div>
                      ) : null}
                    </td>
                      <td className="px-5 py-4 font-mono text-xs font-bold text-[#181c32]">{token || '-'}</td>
                      <td className="px-5 py-4">
                        <div className="flex justify-end gap-2">
                          {exam.status !== 'published' ? (
                            <button
                              className="btn btn-primary"
                              type="button"
                              disabled={publishExisting.isPending || deleteDraft.isPending}
                              onClick={() => publishExisting.mutate(exam)}
                            >
                              <PlayCircle size={16} />
                              Publish
                            </button>
                          ) : null}
                          {exam.status === 'published' ? (
                            <button
                              className="btn btn-primary"
                              type="button"
                              onClick={() => openInvitePanel(exam)}
                            >
                              <UserPlus size={16} />
                              Invite
                            </button>
                          ) : null}
                          {exam.status === 'published' ? (
                            <button
                              className="btn btn-ghost"
                              type="button"
                              disabled={updateAccess.isPending}
                              onClick={() => updateAccess.mutate({ exam, accessStatus: accessStatusOf(exam) === 'open' ? 'closed' : 'open' })}
                            >
                              {accessStatusOf(exam) === 'open' ? <Lock size={16} /> : <CheckCircle2 size={16} />}
                              {accessStatusOf(exam) === 'open' ? 'Close' : 'Open'}
                            </button>
                          ) : null}
                          <Link className="btn btn-ghost" to={`/exams/${exam.id}`}>
                            Detail
                          </Link>
                          {exam.status === 'published' ? (
                            <button
                              className="btn btn-ghost"
                              type="button"
                              disabled={createRevision.isPending}
                              onClick={() => handleCreateRevision(exam)}
                            >
                              <RefreshCcw size={16} />
                              Revisi
                            </button>
                          ) : null}
                          {exam.status === 'draft' ? (
                            <button
                              className="grid h-10 w-10 place-items-center rounded-lg bg-[#fff5f8] text-[#f1416c] hover:bg-[#f1416c] hover:text-white disabled:opacity-60"
                              title="Delete draft"
                              type="button"
                              disabled={deleteDraft.isPending || publishExisting.isPending}
                              onClick={() => handleDeleteDraft(exam)}
                            >
                              <Trash2 size={16} />
                            </button>
                          ) : null}
                          <Link className="btn btn-ghost" to={`/exam-rankings?exam_id=${exam.id}`}>
                            <Trophy size={16} />
                            Ranking
                          </Link>
                          <button
                            className="btn btn-ghost"
                            type="button"
                            disabled={shareCode.isPending}
                            onClick={() => shareCode.mutate(exam)}
                          >
                            <Share2 size={16} />
                            Share Code
                          </button>
                          {token ? (
                            <button className="grid h-10 w-10 place-items-center rounded-lg bg-[#f5f8fa] text-[#7e8299]" title="Copy token" type="button" onClick={() => copyCode(token)}>
                              <Copy size={16} />
                            </button>
                          ) : null}
                        </div>
                      </td>
                    </tr>
                  );
                })}
                {!exams.isLoading && exams.data?.items?.length === 0 ? (
                  <tr>
                    <td className="px-5 py-12 text-center font-semibold text-[#7e8299]" colSpan={8}>
                      Belum ada exam.
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
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

function StatusBadge({ status }) {
  const published = status === 'published';
  return (
    <span className={published ? 'inline-flex rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold capitalize text-[#50cd89]' : 'inline-flex rounded-md bg-[#fff8dd] px-2.5 py-1 text-xs font-extrabold capitalize text-[#f1bc00]'}>
      {status || 'draft'}
    </span>
  );
}

function AccessBadge({ status }) {
  const open = status === 'open';
  return (
    <span className={open ? 'inline-flex rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold capitalize text-[#50cd89]' : 'inline-flex rounded-md bg-[#fff5f8] px-2.5 py-1 text-xs font-extrabold capitalize text-[#f1416c]'}>
      {open ? 'Open' : 'Close'}
    </span>
  );
}

function accessStatusOf(exam) {
  return exam?.metadata?.access_status === 'closed' ? 'closed' : 'open';
}

function TagSelect2({ tags, value, onChange }) {
  const [open, setOpen] = useState(false);
  const [keyword, setKeyword] = useState('');
  const selected = tags.find((tag) => tag.id === value);
  const filtered = useMemo(() => {
    const clean = keyword.trim().toLowerCase();
    if (!clean) return tags;
    return tags.filter((tag) => tagLabel(tag).toLowerCase().includes(clean));
  }, [keyword, tags]);

  return (
    <div className="relative min-w-0">
      <button className="input flex h-11 items-center justify-between bg-white text-left" type="button" onClick={() => setOpen((current) => !current)}>
        <span className={selected ? 'truncate font-bold text-[#3f4254]' : 'truncate text-[#a1a5b7]'}>
          {selected ? tagLabel(selected) : 'Pilih jenis/tag soal'}
        </span>
        <Tag size={16} className="shrink-0 text-[#a1a5b7]" />
      </button>
      {open ? (
        <div className="absolute z-30 mt-2 w-full min-w-[240px] max-w-[calc(100vw-3rem)] rounded-lg border border-[#e4e6ef] bg-white p-2 shadow-lg">
          <input
            className="input h-10 bg-[#f9fafb]"
            placeholder="Cari tag atau guru"
            value={keyword}
            onChange={(event) => setKeyword(event.target.value)}
          />
          <div className="mt-2 max-h-52 overflow-y-auto">
            {filtered.map((tag) => (
              <button
                key={tag.id}
                className={tag.id === value ? 'mb-1 flex w-full items-center justify-between rounded-lg bg-[#f1faff] px-3 py-2 text-left text-sm font-extrabold text-[#009ef7]' : 'mb-1 block w-full rounded-lg px-3 py-2 text-left text-sm font-bold text-[#3f4254] hover:bg-[#f5f8fa]'}
                type="button"
                onClick={() => {
                  onChange(tag.id);
                  setOpen(false);
                  setKeyword('');
                }}
              >
                <span className="min-w-0">
                  <span className="block truncate">{tag.name}</span>
                  {tag.metadata?.lecturer_name ? <span className="mt-0.5 block truncate text-xs font-semibold text-[#a1a5b7]">{tag.metadata.lecturer_name}</span> : null}
                </span>
                {tag.id === value ? <CheckCircle2 className="shrink-0" size={16} /> : null}
              </button>
            ))}
            {filtered.length === 0 ? <div className="rounded-lg bg-[#f5f8fa] px-3 py-2 text-sm font-semibold text-[#7e8299]">Tag tidak ditemukan.</div> : null}
          </div>
        </div>
      ) : null}
    </div>
  );
}

function tagLabel(tag) {
  const lecturerName = tag.metadata?.lecturer_name || tag.metadata?.owner_name || '';
  return lecturerName ? `${tag.name} - ${lecturerName}` : tag.name;
}

function validateForm(values) {
  if (!values.name.trim()) {
    throw new Error('Nama ujian wajib diisi');
  }
  if (Number(values.duration_minutes) < 1) {
    throw new Error('Durasi ujian minimal 1 menit');
  }
  const pools = normalizeQuestionPools(values.question_pools);
  const count = pools.length ? questionCountFromPools(pools) : Number(values.question_count);
  if (count < 1 || count > 500) {
    throw new Error('Jumlah soal harus 1 sampai 500');
  }
  if (pools.length && pools.some((pool) => !pool.question_tag_id || Number(pool.question_count) < 1)) {
    throw new Error('Komposisi jenis soal wajib memilih tag dan jumlah soal minimal 1');
  }
}

function toExamPayload(values) {
  return {
    code: values.code,
    name: values.name,
    description: values.instruction || '',
    status: values.status,
    duration_minutes: Number(values.duration_minutes) || 120,
    passing_grade: Number(values.passing_grade) || 60,
    question_count: questionCountForPayload(values),
    max_attempt: Number(values.max_attempt) || 1,
    random_question: values.random_question,
    random_option: values.random_option,
    result_visibility: values.result_visibility || 'immediate',
    result_release_at: values.result_release_at || '',
    instruction: values.instruction || '',
    metadata: {
      scheduled_at: values.scheduled_at || '',
      recovery_policy: recoveryPolicyPayload(values),
      anti_cheat_policy: antiCheatPolicyPayload(values),
    },
  };
}

function toPublishPayload(values) {
  const questionPools = normalizeQuestionPools(values.question_pools);
  return {
    duration_minutes: Number(values.duration_minutes) || 120,
    passing_grade: Number(values.passing_grade) || 60,
    question_count: questionPools.length ? questionCountFromPools(questionPools) : Number(values.question_count) || 40,
    question_pools: questionPools,
    max_attempt: Number(values.max_attempt) || 1,
    random_question: values.random_question,
    random_option: values.random_option,
    result_visibility: values.result_visibility || 'immediate',
    result_release_at: values.result_release_at || '',
    metadata: {
      recovery_policy: recoveryPolicyPayload(values),
      anti_cheat_policy: antiCheatPolicyPayload(values),
    },
    instruction: values.instruction || '',
  };
}

function toPublishPayloadFromExam(exam) {
  const questionPools = normalizeQuestionPools(exam.metadata?.question_pools || []);
  return {
    duration_minutes: Number(exam.metadata?.duration_minutes) || 120,
    passing_grade: Number(exam.metadata?.passing_grade) || 60,
    question_count: questionPools.length ? questionCountFromPools(questionPools) : Number(exam.metadata?.question_count) || 40,
    question_pools: questionPools,
    max_attempt: Number(exam.metadata?.max_attempt) || 1,
    random_question: exam.metadata?.random_question ?? true,
    random_option: exam.metadata?.random_option ?? true,
    result_visibility: exam.metadata?.result_policy?.visibility || 'immediate',
    result_release_at: exam.metadata?.result_policy?.release_at || '',
    metadata: {
      recovery_policy: exam.metadata?.recovery_policy || recoveryPolicyPayload(initialForm),
      anti_cheat_policy: exam.metadata?.anti_cheat_policy || antiCheatPolicyPayload(initialForm),
    },
    instruction: exam.metadata?.instruction || exam.description || '',
  };
}

function recoveryPolicyPayload(values) {
  return {
    timer_mode: values.timer_mode || 'recovery_pause',
    max_pause_seconds: Math.max(60, Number(values.max_recovery_pause_minutes || 60) * 60),
    max_reconnect_attempts: Math.max(1, Number(values.max_reconnect_attempts || 3)),
    device_change_requires_approval: values.device_change_requires_approval !== false,
    auto_submit_when_recovery_exceeded: false,
  };
}

function antiCheatPolicyPayload(values) {
  return {
    fullscreen_required: values.fullscreen_required !== false,
    webcam_required: values.webcam_required !== false,
    block_copy_paste: true,
    block_right_click: true,
    snapshot_interval_seconds: Math.max(30, Number(values.snapshot_interval_seconds || 60)),
    critical_score_threshold: Math.max(30, Number(values.critical_score_threshold || 90)),
  };
}

function normalizeQuestionPools(pools = []) {
  const grouped = new Map();
  for (const pool of pools) {
    const tagId = pool.question_tag_id || pool.questionTagId || '';
    const count = Number(pool.question_count || pool.questionCount || 0);
    if (!tagId || count < 1) continue;
    grouped.set(tagId, (grouped.get(tagId) || 0) + count);
  }
  return Array.from(grouped, ([question_tag_id, question_count]) => ({ question_tag_id, question_count }));
}

function hasQuestionPools(pools = []) {
  return normalizeQuestionPools(pools).length > 0;
}

function questionCountFromPools(pools = []) {
  return normalizeQuestionPools(pools).reduce((total, pool) => total + Number(pool.question_count || 0), 0);
}

function questionCountForPayload(values) {
  const pools = normalizeQuestionPools(values.question_pools);
  return pools.length ? questionCountFromPools(pools) : Number(values.question_count) || 40;
}
