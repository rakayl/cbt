import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { Edit3, LibraryBig, Plus, RefreshCcw, Trash2, X } from 'lucide-react';
import { api } from '../lib/api';
import { getApiErrorDetail, getApiErrorMessage } from '../lib/apiError';
import { useAuthStore } from '../stores/authStore';

const emptyForm = {
  code: '',
  name: '',
  description: '',
  status: 'active',
  lecturer_id: '',
};

export default function QuestionBanksPage() {
  const queryClient = useQueryClient();
  const hasPermission = useAuthStore((state) => state.hasPermission);
  const canManageOwners = hasPermission('*') || hasPermission('users:read') || hasPermission('tenants:read');
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');
  const [form, setForm] = useState(emptyForm);
  const [editing, setEditing] = useState(null);

  const banks = useQuery({
    queryKey: ['question-banks', page, search],
    queryFn: async () => (await api.get('/question-banks/', { params: { page, limit: 10, search } })).data.data,
  });

  const lecturers = useQuery({
    queryKey: ['lecturers', 'question-bank-owner'],
    enabled: canManageOwners,
    queryFn: async () => (await api.get('/lecturers/', { params: { page: 1, limit: 300 } })).data.data.items,
  });

  const lecturerNameById = useMemo(() => new Map((lecturers.data || []).map((lecturer) => [lecturer.id, lecturer.name])), [lecturers.data]);
  const totalPages = useMemo(() => Math.max(1, Math.ceil((banks.data?.total || 0) / (banks.data?.limit || 10))), [banks.data]);

  const saveBank = useMutation({
    mutationFn: async () => {
      const payload = {
        code: form.code.trim() || `BANK-${Date.now().toString().slice(-6)}`,
        name: form.name.trim(),
        description: form.description.trim(),
        status: form.status,
        lecturer_id: canManageOwners ? form.lecturer_id : undefined,
        metadata: {},
      };
      if (editing) {
        return (await api.put(`/question-banks/${editing.id}`, payload)).data.data;
      }
      return (await api.post('/question-banks/', payload)).data.data;
    },
    onSuccess: () => {
      setForm(emptyForm);
      setEditing(null);
      queryClient.invalidateQueries({ queryKey: ['question-banks'] });
    },
  });

  const deleteBank = useMutation({
    mutationFn: async (bank) => (await api.delete(`/question-banks/${bank.id}`)).data.data,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['question-banks'] }),
  });

  function update(key, value) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  function startEdit(bank) {
    setEditing(bank);
    setForm({
      code: bank.code || '',
      name: bank.name || '',
      description: bank.description || '',
      status: bank.status || 'active',
      lecturer_id: bank.metadata?.lecturer_id || '',
    });
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  function cancelEdit() {
    setEditing(null);
    setForm(emptyForm);
  }

  function confirmDelete(bank) {
    if (window.confirm(`Hapus bank soal "${bank.name}"? Soal di dalam bank tetap mengikuti kebijakan archive/soft delete.`)) {
      deleteBank.mutate(bank);
    }
  }

  const canSave = form.name.trim().length >= 2 && (!canManageOwners || form.lecturer_id);

  return (
    <div className="space-y-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <div className="text-sm font-bold text-[#a1a5b7]">Question Bank</div>
          <h2 className="mt-1 text-2xl font-extrabold text-[#181c32]">Bank Soal</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">
            Kelola wadah bank soal. Guru hanya melihat dan membuat bank miliknya sendiri, admin dapat memilih guru pemilik.
          </p>
        </div>
        <div className="flex flex-col gap-3 sm:w-72">
          <div className="panel flex items-center p-4">
            <div className="grid h-11 w-11 place-items-center rounded-lg bg-[#f1faff] text-[#009ef7]">
              <LibraryBig size={21} />
            </div>
            <div className="ml-3">
              <div className="text-xs font-bold uppercase text-[#a1a5b7]">Total Bank</div>
              <div className="text-xl font-extrabold text-[#181c32]">{banks.data?.total || 0}</div>
            </div>
          </div>
          <Link className="btn btn-primary justify-center" to="/questions">Create Soal</Link>
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_390px]">
        <div className="panel overflow-hidden">
          <div className="flex flex-col gap-4 border-b border-[#eff2f5] p-5 md:flex-row md:items-center md:justify-between">
            <div>
              <h3 className="text-lg font-extrabold text-[#181c32]">Daftar Bank Soal</h3>
              <p className="text-sm font-medium text-[#a1a5b7]">Bank yang tampil sudah mengikuti owner dan RBAC.</p>
            </div>
            <div className="flex flex-col gap-3 sm:flex-row">
              <label className="sm:w-72">
                <input
                  className="input h-11"
                  placeholder="Cari bank soal"
                  value={search}
                  onChange={(event) => {
                    setPage(1);
                    setSearch(event.target.value);
                  }}
                />
              </label>
              <button className="btn btn-ghost justify-center" onClick={() => banks.refetch()}>
                <RefreshCcw size={17} />
                Refresh
              </button>
            </div>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full min-w-[760px] text-left text-sm">
              <thead className="border-b border-[#eff2f5] text-xs font-extrabold uppercase text-[#a1a5b7]">
                <tr>
                  <th className="px-5 py-4">Kode</th>
                  <th className="px-5 py-4">Nama</th>
                  <th className="px-5 py-4">Guru</th>
                  <th className="px-5 py-4">Status</th>
                  <th className="px-5 py-4 text-right">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#eff2f5]">
                {banks.isLoading ? (
                  <tr><td className="px-5 py-10 text-center font-semibold text-[#7e8299]" colSpan={5}>Memuat data...</td></tr>
                ) : null}
                {banks.data?.items?.map((bank) => (
                  <tr key={bank.id} className="hover:bg-[#f9fafb]">
                    <td className="px-5 py-4 font-extrabold text-[#181c32]">{bank.code}</td>
                    <td className="px-5 py-4">
                      <div className="font-bold text-[#181c32]">{bank.name}</div>
                      {bank.description ? <div className="mt-1 text-xs font-semibold text-[#a1a5b7]">{bank.description}</div> : null}
                    </td>
                    <td className="px-5 py-4 font-semibold text-[#7e8299]">
                      {bank.metadata?.lecturer_name || lecturerNameById.get(bank.metadata?.lecturer_id) || bank.metadata?.lecturer_id || '-'}
                    </td>
                    <td className="px-5 py-4">
                      <span className="inline-flex rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold capitalize text-[#50cd89]">{bank.status}</span>
                    </td>
                    <td className="px-5 py-4">
                      <div className="flex justify-end gap-2">
                        <button className="grid h-9 w-9 place-items-center rounded-lg bg-[#f5f8fa] text-[#7e8299] hover:text-[#009ef7]" title="Edit" onClick={() => startEdit(bank)}>
                          <Edit3 size={16} />
                        </button>
                        <button className="grid h-9 w-9 place-items-center rounded-lg bg-[#fff5f8] text-[#f1416c]" title="Delete" disabled={deleteBank.isPending} onClick={() => confirmDelete(bank)}>
                          <Trash2 size={16} />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
                {!banks.isLoading && banks.data?.items?.length === 0 ? (
                  <tr><td className="px-5 py-12 text-center font-semibold text-[#7e8299]" colSpan={5}>Belum ada bank soal.</td></tr>
                ) : null}
              </tbody>
            </table>
          </div>

          <div className="flex flex-col gap-3 border-t border-[#eff2f5] p-5 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-sm font-semibold text-[#7e8299]">Halaman {page} dari {totalPages}</p>
            <div className="flex gap-2">
              <button className="btn btn-ghost" disabled={page <= 1} onClick={() => setPage((value) => Math.max(1, value - 1))}>Previous</button>
              <button className="btn btn-ghost" disabled={page >= totalPages} onClick={() => setPage((value) => value + 1)}>Next</button>
            </div>
          </div>
        </div>

        <aside className="panel h-fit overflow-hidden">
          <div className="flex items-start justify-between border-b border-[#eff2f5] p-5">
            <div>
              <div className="text-sm font-bold text-[#a1a5b7]">{editing ? 'Edit Bank' : 'Tambah Bank'}</div>
              <h3 className="mt-1 text-xl font-extrabold text-[#181c32]">{editing ? editing.name : 'Bank Soal'}</h3>
            </div>
            {editing ? (
              <button className="grid h-9 w-9 place-items-center rounded-lg bg-[#f5f8fa] text-[#7e8299]" type="button" onClick={cancelEdit}>
                <X size={18} />
              </button>
            ) : null}
          </div>
          <div className="space-y-4 p-5">
            <Field label="Kode">
              <input className="input" placeholder="BANK-MTK" value={form.code} onChange={(event) => update('code', event.target.value)} />
            </Field>
            <Field label="Nama">
              <input className="input" placeholder="Matematika - Limit Fungsi" value={form.name} onChange={(event) => update('name', event.target.value)} />
            </Field>
            {canManageOwners ? (
              <Field label="Guru Pemilik">
                <select className="input" value={form.lecturer_id} onChange={(event) => update('lecturer_id', event.target.value)}>
                  <option value="">Pilih guru pemilik</option>
                  {lecturers.data?.map((lecturer) => (
                    <option key={lecturer.id} value={lecturer.id}>{lecturer.name}</option>
                  ))}
                </select>
              </Field>
            ) : null}
            <Field label="Status">
              <select className="input" value={form.status} onChange={(event) => update('status', event.target.value)}>
                <option value="active">Active</option>
                <option value="inactive">Inactive</option>
                <option value="draft">Draft</option>
              </select>
            </Field>
            <Field label="Keterangan">
              <textarea className="input min-h-24" placeholder="Catatan bank soal" value={form.description} onChange={(event) => update('description', event.target.value)} />
            </Field>
            {saveBank.error || deleteBank.error ? (
              <div className="rounded-lg bg-[#fff5f8] px-4 py-3 text-sm font-bold text-[#f1416c]">
                <div>{getApiErrorMessage(saveBank.error || deleteBank.error)}</div>
                {getApiErrorDetail(saveBank.error || deleteBank.error) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(saveBank.error || deleteBank.error)}</div> : null}
              </div>
            ) : null}
            <button className="btn btn-primary w-full justify-center" disabled={!canSave || saveBank.isPending} onClick={() => saveBank.mutate()}>
              <Plus size={18} />
              {saveBank.isPending ? 'Menyimpan...' : editing ? 'Update Bank' : 'Tambah Bank'}
            </button>
          </div>
        </aside>
      </section>
    </div>
  );
}

function Field({ label, children }) {
  return (
    <label className="block">
      <span className="mb-2 block text-sm font-bold text-[#3f4254]">{label}</span>
      {children}
    </label>
  );
}
