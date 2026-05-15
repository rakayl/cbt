import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { CheckCircle2, Circle, Download, Edit3, Plus, RefreshCcw, Save, Tag, Trash2, Upload, X } from 'lucide-react';
import { api } from '../lib/api';
import { getApiErrorDetail, getApiErrorMessage } from '../lib/apiError';
import { useAuthStore } from '../stores/authStore';

const initialOptions = [
  { label: 'A', text: '', is_correct: true, media: [], media_files: [] },
  { label: 'B', text: '', is_correct: false, media: [], media_files: [] },
  { label: 'C', text: '', is_correct: false, media: [], media_files: [] },
  { label: 'D', text: '', is_correct: false, media: [], media_files: [] },
];

function freshInitialOptions() {
  return initialOptions.map((option) => ({ ...option, media: [], media_files: [] }));
}

export default function QuestionsPage() {
  const queryClient = useQueryClient();
  const hasPermission = useAuthStore((state) => state.hasPermission);
  const canManageQuestionOwners = hasPermission('*') || hasPermission('users:read') || hasPermission('tenants:read');
  const [questionBankId, setQuestionBankId] = useState('');
  const [lecturerId, setLecturerId] = useState('');
  const [search, setSearch] = useState('');
  const [answerMode, setAnswerMode] = useState('single');
  const [questionText, setQuestionText] = useState('');
  const [questionMedia, setQuestionMedia] = useState([]);
  const [questionMediaFiles, setQuestionMediaFiles] = useState([]);
  const [explanation, setExplanation] = useState('');
  const [difficulty, setDifficulty] = useState('medium');
  const [options, setOptions] = useState(freshInitialOptions);
  const [importFile, setImportFile] = useState(null);
  const [importResult, setImportResult] = useState(null);
  const [deletingQuestionId, setDeletingQuestionId] = useState('');
  const [editingQuestionId, setEditingQuestionId] = useState('');
  const [editingQuestionCode, setEditingQuestionCode] = useState('');
  const [selectedTagIds, setSelectedTagIds] = useState([]);
  const [newTagName, setNewTagName] = useState('');
  const [tagLecturerId, setTagLecturerId] = useState('');
  const [questionUsage, setQuestionUsage] = useState(null);

  const banks = useQuery({
    queryKey: ['question-banks'],
    queryFn: async () => (await api.get('/question-banks/', { params: { page: 1, limit: 100 } })).data.data.items,
  });
  const questions = useQuery({
    queryKey: ['questions', search],
    queryFn: async () => (await api.get('/questions/', { params: { page: 1, limit: 20, search } })).data.data,
  });
  const lecturers = useQuery({
    queryKey: ['lecturers', 'question-owner'],
    enabled: canManageQuestionOwners,
    queryFn: async () => (await api.get('/lecturers/', { params: { page: 1, limit: 200 } })).data.data.items,
  });
  const tags = useQuery({
    queryKey: ['question-tags'],
    queryFn: async () => (await api.get('/question-tags/', { params: { page: 1, limit: 200 } })).data.data.items,
  });

  const selectedBank = useMemo(() => banks.data?.find((bank) => bank.id === questionBankId), [banks.data, questionBankId]);
  const lecturerNameById = useMemo(() => new Map((lecturers.data || []).map((lecturer) => [lecturer.id, lecturer.name])), [lecturers.data]);

  const saveQuestion = useMutation({
    mutationFn: async () => {
      const cleanQuestionText = questionText.trim();
      const payload = {
        code: editingQuestionCode || `Q-${Date.now().toString().slice(-8)}`,
        name: cleanQuestionText.slice(0, 80) || 'Soal bergambar',
        question_bank_id: questionBankId,
        lecturer_id: canManageQuestionOwners ? lecturerId : undefined,
        question_text: cleanQuestionText,
        answer_mode: answerMode,
        difficulty,
        score: 1,
        explanation,
        status: 'active',
        options: options.map((option) => ({
          id: option.id,
          label: option.label,
          text: option.text,
          is_correct: option.is_correct,
        })),
        tag_ids: selectedTagIds,
        metadata: {},
      };
      let saved;
      if (editingQuestionId) {
        saved = (await api.put(`/questions/${editingQuestionId}`, payload)).data.data;
      } else {
        saved = (await api.post('/questions/', payload)).data.data;
      }
      await uploadPendingMedia(saved);
      return (await api.get(`/questions/${saved.id}`)).data.data;
    },
    onSuccess: () => {
      resetQuestionForm();
      queryClient.invalidateQueries({ queryKey: ['questions'] });
    },
  });

  const createTag = useMutation({
    mutationFn: async () => {
      const cleanName = newTagName.trim();
      const code = `TAG-${Date.now().toString().slice(-6)}`;
      return (
        await api.post('/question-tags/', {
          code,
          name: cleanName,
          lecturer_id: canManageQuestionOwners ? tagLecturerId || lecturerId : undefined,
          description: '',
          status: 'active',
          metadata: {},
        })
      ).data.data;
    },
    onSuccess: (tag) => {
      setSelectedTagIds((current) => Array.from(new Set([...current, tag.id])));
      setNewTagName('');
      setTagLecturerId('');
      queryClient.invalidateQueries({ queryKey: ['question-tags'] });
    },
  });

  const importQuestions = useMutation({
    mutationFn: async () => {
      const form = new FormData();
      form.append('file', importFile);
      if (canManageQuestionOwners) {
        form.append('lecturer_id', lecturerId);
      }
      return (await api.post(`/question-banks/${questionBankId}/import`, form, { headers: { 'Content-Type': 'multipart/form-data' } })).data.data;
    },
    onSuccess: (result) => {
      setImportResult(result);
      setImportFile(null);
      queryClient.invalidateQueries({ queryKey: ['questions'] });
    },
  });

  const deleteQuestion = useMutation({
    mutationFn: async (id) => {
      setDeletingQuestionId(id);
      return (await api.delete(`/questions/${id}`)).data.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['questions'] });
    },
    onSettled: () => {
      setDeletingQuestionId('');
    },
  });
  const deleteMedia = useMutation({
    mutationFn: async ({ mediaId }) => (await api.delete(`/questions/media/${mediaId}`)).data.data,
    onSuccess: (_, variables) => {
      if (variables.optionIndex === undefined) {
        setQuestionMedia((current) => current.filter((item) => item.id !== variables.mediaId));
        return;
      }
      setOptions((current) => current.map((option, index) => (
        index === variables.optionIndex
          ? { ...option, media: (option.media || []).filter((item) => item.id !== variables.mediaId) }
          : option
      )));
    },
  });

  const loadQuestion = useMutation({
    mutationFn: async (id) => (await api.get(`/questions/${id}`)).data.data,
    onSuccess: (question) => {
      setEditingQuestionId(question.id);
      setEditingQuestionCode(question.code);
      setQuestionBankId(question.question_bank_id);
      setLecturerId(question.lecturer_id || question.metadata?.lecturer_id || '');
      setQuestionText(question.question_text || question.name || '');
      setExplanation(question.explanation || '');
      setAnswerMode(question.answer_mode || question.metadata?.answer_mode || 'single');
      setDifficulty(question.difficulty || 'medium');
      setSelectedTagIds(question.tag_ids || question.metadata?.tag_ids || []);
      setOptions(
        question.options?.length
          ? question.options.map((option, index) => ({
              id: option.id,
              label: option.label || String.fromCharCode(65 + index),
              text: option.text || '',
              is_correct: Boolean(option.is_correct),
              media: option.media || [],
              media_files: [],
            }))
          : freshInitialOptions(),
      );
      setQuestionMedia(question.media || []);
      setQuestionMediaFiles([]);
      setQuestionUsage(null);
      api.get(`/questions/${question.id}/usage`).then((response) => setQuestionUsage(response.data.data)).catch(() => setQuestionUsage(null));
      window.scrollTo({ top: 0, behavior: 'smooth' });
    },
  });

  async function uploadPendingMedia(savedQuestion) {
    for (const file of questionMediaFiles) {
      await uploadQuestionMedia(savedQuestion.id, file, 'question');
    }
    for (const option of options) {
      const files = option.media_files || [];
      if (!files.length) continue;
      const savedOption = savedQuestion.options?.find((item) => item.id === option.id) || savedQuestion.options?.find((item) => item.label === option.label);
      if (!savedOption) continue;
      for (const file of files) {
        await uploadQuestionMedia(savedQuestion.id, file, 'option', savedOption.id);
      }
    }
  }

  async function uploadQuestionMedia(questionId, file, usageType, optionId) {
    const form = new FormData();
    form.append('file', file);
    form.append('usage_type', usageType);
    if (optionId) {
      form.append('option_id', optionId);
    }
    await api.post(`/questions/${questionId}/media`, form, { headers: { 'Content-Type': 'multipart/form-data' } });
  }

  async function downloadTemplate() {
    const response = await api.get('/question-banks/template', { responseType: 'blob' });
    const url = window.URL.createObjectURL(new Blob([response.data], { type: 'text/csv;charset=utf-8' }));
    const link = document.createElement('a');
    link.href = url;
    link.download = 'template-bank-soal-pilihan-ganda.csv';
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.URL.revokeObjectURL(url);
  }

  function toggleCorrect(index) {
    setOptions((current) =>
      current.map((option, optionIndex) => ({
        ...option,
        is_correct: answerMode === 'single' ? optionIndex === index : optionIndex === index ? !option.is_correct : option.is_correct,
      })),
    );
  }

  function updateOption(index, value) {
    setOptions((current) => current.map((option, optionIndex) => (optionIndex === index ? { ...option, text: value } : option)));
  }

  function addQuestionMediaFiles(files) {
    setQuestionMediaFiles((current) => [...current, ...Array.from(files || [])]);
  }

  function removeQuestionMediaFile(index) {
    setQuestionMediaFiles((current) => current.filter((_, fileIndex) => fileIndex !== index));
  }

  function addOptionMediaFiles(index, files) {
    setOptions((current) => current.map((option, optionIndex) => (
      optionIndex === index
        ? { ...option, media_files: [...(option.media_files || []), ...Array.from(files || [])] }
        : option
    )));
  }

  function removeOptionMediaFile(optionIndex, fileIndex) {
    setOptions((current) => current.map((option, index) => (
      index === optionIndex
        ? { ...option, media_files: (option.media_files || []).filter((_, currentFileIndex) => currentFileIndex !== fileIndex) }
        : option
    )));
  }

  function addOption() {
    const label = String.fromCharCode(65 + options.length);
    setOptions((current) => [...current, { label, text: '', is_correct: false, media: [], media_files: [] }]);
  }

  function removeOption(index) {
    setOptions((current) => current.filter((_, optionIndex) => optionIndex !== index).map((option, optionIndex) => ({ ...option, label: String.fromCharCode(65 + optionIndex) })));
  }

  function resetQuestionForm() {
    setEditingQuestionId('');
    setEditingQuestionCode('');
    setQuestionText('');
    setQuestionMedia([]);
    setQuestionMediaFiles([]);
    setExplanation('');
    setAnswerMode('single');
    setDifficulty('medium');
    setOptions(freshInitialOptions());
    setSelectedTagIds([]);
    setQuestionUsage(null);
  }

  function toggleTag(tagId) {
    setSelectedTagIds((current) => (current.includes(tagId) ? current.filter((id) => id !== tagId) : [...current, tagId]));
  }

  async function handleDeleteQuestion(question) {
    try {
      const usage = (await api.get(`/questions/${question.id}/usage`)).data.data;
      const used = Number(usage.exam_session_count || 0) > 0 || Number(usage.answer_count || 0) > 0 || Number(usage.grading_count || 0) > 0;
      const message = used
        ? `Soal "${question.name}" pernah dipakai.\n\nSession ujian: ${usage.exam_session_count}\nExam: ${usage.exam_count}\nJawaban: ${usage.answer_count}\nGrading: ${usage.grading_count}\n\nSoal tidak akan hard delete. Sistem akan archive/soft delete agar riwayat ujian dan nilai tetap aman.\n\nLanjut archive soal ini?`
        : `Hapus soal "${question.name}"?\n\nSistem tetap memakai soft delete agar data audit aman.`;
      if (window.confirm(message)) {
        deleteQuestion.mutate(question.id);
      }
    } catch (error) {
      const ok = window.confirm(`Tidak bisa membaca usage soal sekarang.\n\nTetap lanjut soft delete soal "${question.name}"?`);
      if (ok) {
        deleteQuestion.mutate(question.id);
      }
    }
  }

  const canSave =
    questionBankId &&
    (!canManageQuestionOwners || lecturerId) &&
    (questionText.trim().length >= 3 || questionMedia.length > 0 || questionMediaFiles.length > 0) &&
    options.length >= 2 &&
    options.every((option) => option.text.trim() || option.media?.length || option.media_files?.length) &&
    options.some((option) => option.is_correct);
  const canImport = questionBankId && importFile && (!canManageQuestionOwners || lecturerId);

  return (
    <div className="space-y-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <div className="text-sm font-bold text-[#a1a5b7]">Create Soal</div>
          <h2 className="mt-1 text-2xl font-extrabold text-[#181c32]">Create dan Edit Soal</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">
            Simple CBT builder: satu tipe soal pilihan ganda dengan mode satu jawaban benar atau banyak jawaban benar.
          </p>
        </div>
        <div className="flex flex-col gap-3 sm:w-72">
          <div className="panel flex items-center p-4">
            <div className="grid h-11 w-11 place-items-center rounded-lg bg-[#f1faff] text-[#009ef7]">
              <CheckCircle2 size={21} />
            </div>
            <div className="ml-3">
              <div className="text-xs font-bold uppercase text-[#a1a5b7]">Total Soal</div>
              <div className="text-xl font-extrabold text-[#181c32]">{questions.data?.total || 0}</div>
            </div>
          </div>
          <Link className="btn btn-ghost justify-center" to="/question-banks">Kelola Bank Soal</Link>
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_390px]">
        <div className="space-y-6">
          <div className="panel overflow-hidden">
            <div className="border-b border-[#eff2f5] p-5">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h3 className="text-lg font-extrabold text-[#181c32]">{editingQuestionId ? 'Edit Soal' : 'Builder Soal'}</h3>
                  <p className="text-sm font-medium text-[#a1a5b7]">Pilih bank, tulis pertanyaan, tandai jawaban benar.</p>
                </div>
                {editingQuestionId ? (
                  <button className="btn btn-ghost justify-center" onClick={resetQuestionForm}>
                    <X size={17} />
                    Batal Edit
                  </button>
                ) : null}
              </div>
            </div>
            <div className="space-y-5 p-5">
              {editingQuestionId ? (
                <div className="grid gap-3 rounded-lg border border-[#eff2f5] bg-[#f9fafb] p-4 sm:grid-cols-4">
                  <InfoTile label="Versi Soal" value={`v${questionUsage?.version || questions.data?.items?.find((item) => item.id === editingQuestionId)?.metadata?.version || '-'}`} />
                  <InfoTile label="Dipakai Session" value={questionUsage?.exam_session_count ?? '-'} />
                  <InfoTile label="Jawaban Masuk" value={questionUsage?.answer_count ?? '-'} />
                  <InfoTile label="Aksi Aman" value={questionUsage?.recommended_action || 'archive'} />
                </div>
              ) : null}
              <div className={canManageQuestionOwners ? 'grid gap-4 md:grid-cols-[1fr_1fr_180px]' : 'grid gap-4 md:grid-cols-[1fr_220px]'}>
                <label className="block">
                  <span className="mb-2 block text-sm font-bold text-[#3f4254]">Bank Soal</span>
                  <select className="input" value={questionBankId} onChange={(event) => setQuestionBankId(event.target.value)}>
                    <option value="">Pilih bank soal</option>
                    {banks.data?.map((bank) => (
                      <option key={bank.id} value={bank.id}>{bank.name}</option>
                    ))}
                  </select>
                </label>
                {canManageQuestionOwners ? (
                  <label className="block">
                    <span className="mb-2 block text-sm font-bold text-[#3f4254]">Guru Pemilik</span>
                    <select className="input" value={lecturerId} onChange={(event) => setLecturerId(event.target.value)}>
                      <option value="">Pilih guru</option>
                      {lecturers.data?.map((lecturer) => (
                        <option key={lecturer.id} value={lecturer.id}>{lecturer.name}</option>
                      ))}
                    </select>
                  </label>
                ) : null}
                <label className="block">
                  <span className="mb-2 block text-sm font-bold text-[#3f4254]">Difficulty</span>
                  <select className="input" value={difficulty} onChange={(event) => setDifficulty(event.target.value)}>
                    <option value="easy">Easy</option>
                    <option value="medium">Medium</option>
                    <option value="hard">Hard</option>
                  </select>
                </label>
              </div>

              <div className="rounded-lg border border-[#eff2f5] bg-white p-4">
                <div className="flex items-center gap-2 text-sm font-extrabold text-[#181c32]">
                  <Tag size={17} />
                  Jenis/Kelompok Soal
                </div>
                <p className="mt-1 text-sm font-medium text-[#a1a5b7]">Tag ini dipakai saat publish ujian untuk menentukan komposisi jumlah soal.</p>
                <div className="mt-4">
                  <MultiTagSelect
                    tags={tags.data || []}
                    selectedIds={selectedTagIds}
                    onToggle={toggleTag}
                    showLecturer={canManageQuestionOwners}
                    loading={tags.isLoading}
                  />
                </div>
                <div className={canManageQuestionOwners ? 'mt-4 grid gap-3 lg:grid-cols-[1fr_220px_140px]' : 'mt-4 grid gap-3 sm:grid-cols-[1fr_140px]'}>
                  <input className="input bg-[#f9fafb]" placeholder="Buat tag baru, contoh: HOTS" value={newTagName} onChange={(event) => setNewTagName(event.target.value)} />
                  {canManageQuestionOwners ? (
                    <select className="input bg-[#f9fafb]" value={tagLecturerId || lecturerId} onChange={(event) => setTagLecturerId(event.target.value)}>
                      <option value="">Pilih guru pemilik tag</option>
                      {lecturers.data?.map((lecturer) => (
                        <option key={lecturer.id} value={lecturer.id}>{lecturer.name}</option>
                      ))}
                    </select>
                  ) : null}
                  <button className="btn btn-ghost justify-center" type="button" disabled={!newTagName.trim() || (canManageQuestionOwners && !(tagLecturerId || lecturerId)) || createTag.isPending} onClick={() => createTag.mutate()}>
                    <Plus size={17} />
                    Buat Tag
                  </button>
                </div>
                {createTag.error ? (
                  <div className="mt-3 rounded-lg bg-[#fff5f8] px-4 py-3 text-sm font-bold text-[#f1416c]">
                    <div>{getApiErrorMessage(createTag.error)}</div>
                    {getApiErrorDetail(createTag.error) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(createTag.error)}</div> : null}
                  </div>
                ) : null}
              </div>

              <div className="rounded-lg border border-[#eff2f5] bg-white p-4">
                <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                  <div>
                    <div className="text-sm font-extrabold text-[#181c32]">Import Soal</div>
                    <p className="mt-1 text-sm font-medium text-[#a1a5b7]">Gunakan template CSV: `single` untuk 1 jawaban, `multiple` untuk banyak jawaban.</p>
                  </div>
                  <button className="btn btn-ghost justify-center" onClick={downloadTemplate}>
                    <Download size={17} />
                    Download Template
                  </button>
                </div>
                <div className="mt-4 grid gap-3 lg:grid-cols-[1fr_170px]">
                  <input
                    className="input bg-[#f9fafb]"
                    type="file"
                    accept=".csv,.xlsx"
                    onChange={(event) => {
                      setImportFile(event.target.files?.[0] || null);
                      setImportResult(null);
                    }}
                  />
                  <button className="btn btn-primary justify-center" disabled={!canImport || importQuestions.isPending} onClick={() => importQuestions.mutate()}>
                    <Upload size={17} />
                    {importQuestions.isPending ? 'Import...' : 'Import'}
                  </button>
                </div>
                {importQuestions.error ? (
                  <div className="mt-3 rounded-lg bg-[#fff5f8] px-4 py-3 text-sm font-bold text-[#f1416c]">
                    <div>{getApiErrorMessage(importQuestions.error)}</div>
                    {getApiErrorDetail(importQuestions.error) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(importQuestions.error)}</div> : null}
                  </div>
                ) : null}
                {importResult ? (
                  <div className="mt-3 rounded-lg bg-[#e8fff3] px-4 py-3 text-sm font-bold text-[#50cd89]">
                    Import selesai: {importResult.imported} soal masuk, {importResult.skipped} baris dilewati.
                  </div>
                ) : null}
              </div>

              <label className="block">
                <span className="mb-2 block text-sm font-bold text-[#3f4254]">Pertanyaan</span>
                <textarea className="input min-h-32" placeholder="Tulis teks soal di sini" value={questionText} onChange={(event) => setQuestionText(event.target.value)} />
              </label>

              <div className="rounded-lg border border-[#eff2f5] bg-[#f9fafb] p-4">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <div className="text-sm font-extrabold text-[#181c32]">Gambar Soal</div>
                    <p className="mt-1 text-sm font-medium text-[#a1a5b7]">Bisa digunakan untuk diagram, grafik, peta, atau soal full gambar.</p>
                  </div>
                  <label className="btn btn-ghost cursor-pointer justify-center">
                    <Upload size={17} />
                    Upload Gambar
                    <input className="hidden" type="file" accept="image/png,image/jpeg,image/webp" multiple onChange={(event) => { addQuestionMediaFiles(event.target.files); event.target.value = ''; }} />
                  </label>
                </div>
                <MediaPreviewGrid
                  media={questionMedia}
                  files={questionMediaFiles}
                  onRemoveMedia={(mediaId) => deleteMedia.mutate({ mediaId })}
                  onRemoveFile={removeQuestionMediaFile}
                />
              </div>

              <div>
                <span className="mb-2 block text-sm font-bold text-[#3f4254]">Mode Jawaban</span>
                <div className="grid gap-3 sm:grid-cols-2">
                  <button className={answerMode === 'single' ? 'btn btn-primary justify-center' : 'btn btn-ghost justify-center'} onClick={() => { setAnswerMode('single'); setOptions((current) => current.map((option, index) => ({ ...option, is_correct: index === 0 }))); }}>
                    Jawaban hanya 1
                  </button>
                  <button className={answerMode === 'multiple' ? 'btn btn-primary justify-center' : 'btn btn-ghost justify-center'} onClick={() => setAnswerMode('multiple')}>
                    Bisa memilih banyak
                  </button>
                </div>
              </div>

              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-bold text-[#3f4254]">Opsi Jawaban</span>
                  <button className="btn btn-ghost" onClick={addOption}>
                    <Plus size={17} />
                    Tambah Opsi
                  </button>
                </div>
                {options.map((option, index) => (
                  <div key={option.label} className="grid gap-3 rounded-lg border border-[#eff2f5] bg-white p-3 md:grid-cols-[48px_1fr_120px_44px] md:items-center">
                    <div className="grid h-10 w-10 place-items-center rounded-lg bg-[#f1faff] font-extrabold text-[#009ef7]">{option.label}</div>
                    <input className="input" placeholder={`Teks opsi ${option.label}`} value={option.text} onChange={(event) => updateOption(index, event.target.value)} />
                    <button className={option.is_correct ? 'btn btn-primary justify-center' : 'btn btn-ghost justify-center'} onClick={() => toggleCorrect(index)}>
                      {option.is_correct ? <CheckCircle2 size={17} /> : <Circle size={17} />}
                      Benar
                    </button>
                    <button className="grid h-10 w-10 place-items-center rounded-lg bg-[#fff5f8] text-[#f1416c]" disabled={options.length <= 2} onClick={() => removeOption(index)}>
                      <Trash2 size={16} />
                    </button>
                    <div className="md:col-span-4">
                      <div className="flex flex-col gap-2 rounded-lg bg-[#f9fafb] p-3 sm:flex-row sm:items-center sm:justify-between">
                        <div className="text-sm font-bold text-[#3f4254]">Gambar opsi {option.label}</div>
                        <label className="btn btn-ghost cursor-pointer justify-center">
                          <Upload size={16} />
                          Upload
                          <input className="hidden" type="file" accept="image/png,image/jpeg,image/webp" multiple onChange={(event) => { addOptionMediaFiles(index, event.target.files); event.target.value = ''; }} />
                        </label>
                      </div>
                      <MediaPreviewGrid
                        media={option.media || []}
                        files={option.media_files || []}
                        onRemoveMedia={(mediaId) => deleteMedia.mutate({ mediaId, optionIndex: index })}
                        onRemoveFile={(fileIndex) => removeOptionMediaFile(index, fileIndex)}
                      />
                    </div>
                  </div>
                ))}
              </div>

              <label className="block">
                <span className="mb-2 block text-sm font-bold text-[#3f4254]">Pembahasan</span>
                <textarea className="input min-h-24" placeholder="Opsional: pembahasan jawaban" value={explanation} onChange={(event) => setExplanation(event.target.value)} />
              </label>

              {saveQuestion.error ? (
                <div className="rounded-lg bg-[#fff5f8] px-4 py-3 text-sm font-bold text-[#f1416c]">
                  <div>{getApiErrorMessage(saveQuestion.error)}</div>
                  {getApiErrorDetail(saveQuestion.error) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(saveQuestion.error)}</div> : null}
                </div>
              ) : null}

              <button className="btn btn-primary w-full justify-center" disabled={!canSave || saveQuestion.isPending} onClick={() => saveQuestion.mutate()}>
                <Save size={18} />
                {saveQuestion.isPending ? 'Menyimpan...' : editingQuestionId ? 'Update Soal' : 'Simpan Soal'}
              </button>
            </div>
          </div>
        </div>

        <aside className="panel h-fit overflow-hidden">
          <div className="border-b border-[#eff2f5] p-5">
            <h3 className="text-lg font-extrabold text-[#181c32]">Daftar Soal</h3>
            <p className="text-sm font-medium text-[#a1a5b7]">{selectedBank ? selectedBank.name : 'Semua bank soal'}</p>
          </div>
          <div className="p-5">
            <label className="block">
              <input className="input h-11" placeholder="Cari soal" value={search} onChange={(event) => setSearch(event.target.value)} />
            </label>
            <button className="btn btn-ghost mt-3 w-full justify-center" onClick={() => questions.refetch()}>
              <RefreshCcw size={17} />
              Refresh
            </button>
            <div className="mt-5 space-y-3">
              {questions.data?.items?.map((question) => (
                <div key={question.id} className="rounded-lg border border-[#eff2f5] p-4">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <div className="break-words font-extrabold text-[#181c32]">{question.name}</div>
                      <div className="mt-1 text-xs font-semibold text-[#a1a5b7]">{question.metadata?.answer_mode === 'multiple' ? 'Multiple answer' : 'Single answer'}</div>
                      <div className="mt-1 text-xs font-bold text-[#50cd89]">Version v{question.metadata?.version || 1}</div>
                      {question.metadata?.lecturer_id ? (
                        <div className="mt-1 text-xs font-semibold text-[#7e8299]">
                          Guru: {lecturerNameById.get(question.metadata.lecturer_id) || question.metadata.lecturer_id}
                        </div>
                      ) : null}
                      {question.metadata?.tags?.length ? (
                        <div className="mt-3 flex flex-wrap gap-1.5">
                          {question.metadata.tags.map((tag) => (
                            <span key={tag.id} className="rounded-md bg-[#f1faff] px-2 py-1 text-xs font-extrabold text-[#009ef7]">{tagLabel(tag, canManageQuestionOwners)}</span>
                          ))}
                        </div>
                      ) : null}
                    </div>
                    <div className="flex shrink-0 flex-col items-stretch gap-2 sm:flex-row">
                      <button
                        className="inline-flex h-9 items-center justify-center gap-2 rounded-lg bg-[#f1faff] px-3 text-sm font-extrabold text-[#009ef7] hover:bg-[#009ef7] hover:text-white disabled:opacity-60"
                        title="Edit soal"
                        disabled={loadQuestion.isPending}
                        onClick={() => loadQuestion.mutate(question.id)}
                      >
                        <Edit3 size={16} />
                        Edit
                      </button>
                      <button
                        className="inline-flex h-9 items-center justify-center gap-2 rounded-lg bg-[#fff5f8] px-3 text-sm font-extrabold text-[#f1416c] hover:bg-[#f1416c] hover:text-white disabled:opacity-60"
                        title="Archive / soft delete soal"
                        disabled={deletingQuestionId === question.id}
                        onClick={() => handleDeleteQuestion(question)}
                      >
                        <Trash2 size={16} />
                        Archive
                      </button>
                    </div>
                  </div>
                </div>
              ))}
              {deleteQuestion.error ? (
                <div className="rounded-lg bg-[#fff5f8] px-4 py-3 text-sm font-bold text-[#f1416c]">
                  <div>{getApiErrorMessage(deleteQuestion.error)}</div>
                  {getApiErrorDetail(deleteQuestion.error) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(deleteQuestion.error)}</div> : null}
                </div>
              ) : null}
              {!questions.isLoading && questions.data?.items?.length === 0 ? (
                <div className="rounded-lg bg-[#f5f8fa] p-4 text-sm font-semibold text-[#7e8299]">Belum ada soal.</div>
              ) : null}
            </div>
          </div>
        </aside>
      </section>
    </div>
  );
}

function InfoTile({ label, value }) {
  return (
    <div className="rounded-lg bg-white px-3 py-2">
      <div className="text-[11px] font-extrabold uppercase text-[#a1a5b7]">{label}</div>
      <div className="mt-1 truncate text-sm font-extrabold text-[#181c32]">{value}</div>
    </div>
  );
}

function MultiTagSelect({ tags, selectedIds, onToggle, showLecturer, loading }) {
  const [keyword, setKeyword] = useState('');
  const filteredTags = useMemo(() => {
    const clean = keyword.trim().toLowerCase();
    if (!clean) return tags;
    return tags.filter((tag) => tagLabel(tag, true).toLowerCase().includes(clean));
  }, [keyword, tags]);

  return (
    <div className="rounded-lg border border-[#e4e6ef] bg-white">
      <div className="border-b border-[#eff2f5] p-3">
        <input
          className="input h-10 bg-[#f9fafb]"
          placeholder="Cari tag atau nama guru"
          value={keyword}
          onChange={(event) => setKeyword(event.target.value)}
        />
      </div>
      <div className="max-h-56 overflow-y-auto p-2">
        {filteredTags.map((tag) => {
          const active = selectedIds.includes(tag.id);
          return (
            <button
              key={tag.id}
              className={active ? 'mb-2 flex w-full items-center justify-between rounded-lg bg-[#f1faff] px-3 py-2 text-left text-sm font-extrabold text-[#009ef7]' : 'mb-2 flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm font-bold text-[#3f4254] hover:bg-[#f5f8fa]'}
              type="button"
              onClick={() => onToggle(tag.id)}
            >
              <span className="min-w-0">
                <span className="block truncate">{tag.name}</span>
                {showLecturer ? <span className="mt-0.5 block truncate text-xs font-semibold text-[#a1a5b7]">{tag.metadata?.lecturer_name || 'Tanpa guru'}</span> : null}
              </span>
              {active ? <CheckCircle2 className="shrink-0" size={17} /> : <Circle className="shrink-0 text-[#a1a5b7]" size={17} />}
            </button>
          );
        })}
        {!loading && filteredTags.length === 0 ? (
          <div className="rounded-lg bg-[#f5f8fa] px-3 py-2 text-sm font-semibold text-[#7e8299]">Tag tidak ditemukan.</div>
        ) : null}
      </div>
    </div>
  );
}

function MediaPreviewGrid({ media = [], files = [], onRemoveMedia, onRemoveFile }) {
  if (!media.length && !files.length) return null;
  return (
    <div className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {media.map((item) => (
        <div key={item.id} className="overflow-hidden rounded-lg border border-[#eff2f5] bg-white">
          <SecureImage media={item} />
          <div className="flex items-center justify-between gap-2 p-2">
            <span className="truncate text-xs font-semibold text-[#7e8299]">{item.original_filename || item.mime_type}</span>
            <button className="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-[#fff5f8] text-[#f1416c]" type="button" onClick={() => onRemoveMedia(item.id)}>
              <Trash2 size={14} />
            </button>
          </div>
        </div>
      ))}
      {files.map((file, index) => (
        <div key={`${file.name}-${index}`} className="overflow-hidden rounded-lg border border-dashed border-[#b5b5c3] bg-white">
          <LocalImage file={file} />
          <div className="flex items-center justify-between gap-2 p-2">
            <span className="truncate text-xs font-semibold text-[#7e8299]">{file.name}</span>
            <button className="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-[#fff5f8] text-[#f1416c]" type="button" onClick={() => onRemoveFile(index)}>
              <Trash2 size={14} />
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}

function SecureImage({ media }) {
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

  return src ? (
    <img className="h-40 w-full bg-[#f5f8fa] object-contain" src={src} alt={media.original_filename || 'Media soal'} />
  ) : (
    <div className="grid h-40 place-items-center bg-[#f5f8fa] text-xs font-semibold text-[#a1a5b7]">Memuat gambar...</div>
  );
}

function LocalImage({ file }) {
  const [src, setSrc] = useState('');
  useEffect(() => {
    const objectUrl = URL.createObjectURL(file);
    setSrc(objectUrl);
    return () => URL.revokeObjectURL(objectUrl);
  }, [file]);

  return <img className="h-40 w-full bg-[#f5f8fa] object-contain" src={src} alt={file.name} />;
}

function tagLabel(tag, showLecturer) {
  const lecturerName = tag.lecturer_name || tag.metadata?.lecturer_name || tag.metadata?.owner_name || '';
  return showLecturer && lecturerName ? `${tag.name} - ${lecturerName}` : tag.name;
}
