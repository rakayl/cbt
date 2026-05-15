import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { Edit3, Plus, RefreshCcw, Trash2, Users, X } from 'lucide-react';
import { api } from '../lib/api';
import { applyApiFieldErrors, getApiErrorDetail, getApiErrorMessage } from '../lib/apiError';
import { useAuthStore } from '../stores/authStore';

const schema = z.object({
  code: z.string().min(2, 'Kode minimal 2 karakter').max(80, 'Kode maksimal 80 karakter'),
  name: z.string().min(2, 'Nama minimal 2 karakter').max(160, 'Nama maksimal 160 karakter'),
  description: z.string().max(2000, 'Deskripsi maksimal 2000 karakter').optional(),
  status: z.enum(['active', 'inactive', 'draft', 'published', 'completed', 'suspended']),
  metadata_label: z.string().max(120).optional(),
  lecturer_id: z.string().optional(),
  account_email: z.string().optional(),
  account_password: z.string().optional(),
});

const defaultValues = {
  code: '',
  name: '',
  description: '',
  status: 'active',
  metadata_label: '',
  lecturer_id: '',
  account_email: '',
  account_password: '',
};

export default function AcademicResourcePage({ config }) {
  const queryClient = useQueryClient();
  const hasPermission = useAuthStore((state) => state.hasPermission);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');
  const [editing, setEditing] = useState(null);
  const [detailItem, setDetailItem] = useState(null);
  const queryKey = [config.resource, page, search];
  const canWrite = !config.writePermission || hasPermission(config.writePermission);
  const canManageOwners = hasPermission('*') || hasPermission('users:read') || hasPermission('tenants:read');

  const listQuery = useQuery({
    queryKey,
    queryFn: async () => {
      const { data } = await api.get(`${config.endpoint}/`, { params: { page, limit: 10, search } });
      return data.data;
    },
  });

  const lecturersQuery = useQuery({
    queryKey: ['lecturers', config.resource, 'owner-select'],
    enabled: Boolean(config.ownerLecturer && canManageOwners),
    queryFn: async () => (await api.get('/lecturers/', { params: { page: 1, limit: 300 } })).data.data.items,
  });

  const classStudentsQuery = useQuery({
    queryKey: ['class-room-students', detailItem?.id],
    enabled: Boolean(config.detailStudents && detailItem?.id),
    queryFn: async () => (await api.get(`${config.endpoint}/${detailItem.id}/students`, { params: { page: 1, limit: 200 } })).data.data,
  });

  const form = useForm({ resolver: zodResolver(schema), defaultValues });
  const totalPages = useMemo(() => Math.max(1, Math.ceil((listQuery.data?.total || 0) / (listQuery.data?.limit || 10))), [listQuery.data]);

  const saveMutation = useMutation({
    mutationFn: async (values) => {
      const payload = toPayload(values);
      const url = editing ? `${config.endpoint}/${editing.id}` : `${config.endpoint}/`;
      const { data } = editing ? await api.put(url, payload) : await api.post(url, payload);
      return data.data;
    },
    onSuccess: () => {
      form.reset(defaultValues);
      setEditing(null);
      queryClient.invalidateQueries({ queryKey: [config.resource] });
    },
    onError: (error) => {
      applyApiFieldErrors(error, form.setError, {
        email: 'account_email',
        password: 'account_password',
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (item) => api.delete(`${config.endpoint}/${item.id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [config.resource] }),
  });

  function startEdit(item) {
    setEditing(item);
    form.reset({
      code: item.code || '',
      name: item.name || '',
      description: item.description || '',
      status: item.status || 'active',
      metadata_label: item.metadata?.label || '',
      account_email: item.metadata?.account_email || '',
      lecturer_id: item.metadata?.lecturer_id || '',
      account_password: '',
    });
  }

  function cancelEdit() {
    setEditing(null);
    form.reset(defaultValues);
  }

  function submit(values) {
    if (config.ownerLecturer && canManageOwners && !values.lecturer_id) {
      form.setError('lecturer_id', { message: 'Guru pemilik wajib dipilih' });
      return;
    }
    if (config.accountRequired && !editing) {
      if (!values.account_email) {
        form.setError('account_email', { message: 'Email akun wajib diisi' });
        return;
      }
      if (!values.account_email.includes('@')) {
        form.setError('account_email', { message: 'Format email tidak valid' });
        return;
      }
      if (!values.account_password || values.account_password.length < 8) {
        form.setError('account_password', { message: 'Password minimal 8 karakter' });
        return;
      }
    }
    saveMutation.mutate(values);
  }

  return (
    <div className="space-y-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <div className="text-sm font-bold text-[#a1a5b7]">{config.eyebrow}</div>
          <h2 className="mt-1 text-2xl font-extrabold text-[#181c32]">{config.title}</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">{config.description}</p>
        </div>
        <div className="grid grid-cols-2 gap-4 sm:w-[360px]">
          <Metric label="Total" value={listQuery.data?.total || 0} icon={config.icon} />
          <Metric label="Status" value="Active" icon={RefreshCcw} />
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
        <div className="panel overflow-hidden">
          <div className="flex flex-col gap-4 border-b border-[#eff2f5] p-5 md:flex-row md:items-center md:justify-between">
            <div>
              <h3 className="text-lg font-extrabold text-[#181c32]">Data {config.singular}</h3>
              <p className="text-sm font-medium text-[#a1a5b7]">{canWrite ? 'Search, pagination, update, dan soft delete.' : 'Mode baca. Penambahan dan perubahan data dibatasi oleh RBAC.'}</p>
            </div>
            <div className="flex flex-col gap-3 sm:flex-row">
              <label className="sm:w-72">
                <input
                  className="input h-11"
                  placeholder={`Cari ${config.singular.toLowerCase()}`}
                  value={search}
                  onChange={(event) => {
                    setPage(1);
                    setSearch(event.target.value);
                  }}
                />
              </label>
              <button className="btn btn-ghost justify-center" onClick={() => listQuery.refetch()}>
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
                  <th className="px-5 py-4">Status</th>
                  <th className="px-5 py-4">Keterangan</th>
                  <th className="px-5 py-4 text-right">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#eff2f5]">
                {listQuery.isLoading ? (
                  <tr>
                    <td className="px-5 py-10 text-center font-semibold text-[#7e8299]" colSpan={5}>Memuat data...</td>
                  </tr>
                ) : null}
                {listQuery.data?.items?.map((item) => (
                  <tr key={item.id} className="hover:bg-[#f9fafb]">
                    <td className="px-5 py-4 font-extrabold text-[#181c32]">{item.code}</td>
                    <td className="px-5 py-4">
                      <div className="font-bold text-[#181c32]">{item.name}</div>
                      {item.metadata?.label ? <div className="mt-1 text-xs font-semibold text-[#a1a5b7]">{item.metadata.label}</div> : null}
                      {item.metadata?.lecturer_name ? <div className="mt-1 text-xs font-semibold text-[#7e8299]">Guru: {item.metadata.lecturer_name}</div> : null}
                    </td>
                    <td className="px-5 py-4">
                      <span className="inline-flex rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold capitalize text-[#50cd89]">
                        {item.status}
                      </span>
                    </td>
                    <td className="max-w-sm px-5 py-4 font-medium text-[#7e8299]">{item.description || '-'}</td>
                    <td className="px-5 py-4">
                      <div className="flex justify-end gap-2">
                        {config.detailStudents ? (
                          <button className="grid h-9 w-9 place-items-center rounded-lg bg-[#f1faff] text-[#009ef7]" title="Detail siswa kelas" onClick={() => setDetailItem(item)}>
                            <Users size={16} />
                          </button>
                        ) : null}
                        {canWrite ? (
                          <>
                            <button className="grid h-9 w-9 place-items-center rounded-lg bg-[#f5f8fa] text-[#7e8299] hover:text-[#009ef7]" title="Edit" onClick={() => startEdit(item)}>
                              <Edit3 size={16} />
                            </button>
                            <button
                              className="grid h-9 w-9 place-items-center rounded-lg bg-[#fff5f8] text-[#f1416c]"
                              title="Delete"
                              disabled={deleteMutation.isPending}
                              onClick={() => deleteMutation.mutate(item)}
                            >
                              <Trash2 size={16} />
                            </button>
                          </>
                        ) : (
                          <span className="rounded-md bg-[#f5f8fa] px-3 py-2 text-xs font-extrabold text-[#7e8299]">Read only</span>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
                {!listQuery.isLoading && listQuery.data?.items?.length === 0 ? (
                  <tr>
                    <td className="px-5 py-12 text-center font-semibold text-[#7e8299]" colSpan={5}>
                      Belum ada data {config.singular.toLowerCase()}.
                    </td>
                  </tr>
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

        {canWrite ? (
        <form className="panel h-fit overflow-hidden" onSubmit={form.handleSubmit(submit)}>
          <div className="flex items-start justify-between border-b border-[#eff2f5] p-5">
            <div>
              <div className="text-sm font-bold text-[#a1a5b7]">{editing ? 'Edit Data' : 'Tambah Data'}</div>
              <h3 className="mt-1 text-xl font-extrabold text-[#181c32]">{editing ? editing.name : config.singular}</h3>
            </div>
            {editing ? (
              <button className="grid h-9 w-9 place-items-center rounded-lg bg-[#f5f8fa] text-[#7e8299]" type="button" onClick={cancelEdit}>
                <X size={18} />
              </button>
            ) : null}
          </div>

          <div className="space-y-4 p-5">
            <Field label="Kode" error={form.formState.errors.code?.message}>
              <input className="input" placeholder={config.codePlaceholder} {...form.register('code')} />
            </Field>
            <Field label="Nama" error={form.formState.errors.name?.message}>
              <input className="input" placeholder={config.namePlaceholder} {...form.register('name')} />
            </Field>
            <Field label="Status" error={form.formState.errors.status?.message}>
              <select className="input" {...form.register('status')}>
                <option value="active">Active</option>
                <option value="inactive">Inactive</option>
                <option value="draft">Draft</option>
                <option value="suspended">Suspended</option>
              </select>
            </Field>
            <Field label={config.metadataLabel} error={form.formState.errors.metadata_label?.message}>
              <input className="input" placeholder={config.metadataPlaceholder} {...form.register('metadata_label')} />
            </Field>
            {config.ownerLecturer && canManageOwners ? (
              <Field label="Guru Pemilik" error={form.formState.errors.lecturer_id?.message}>
                <select className="input" {...form.register('lecturer_id')}>
                  <option value="">Pilih guru pemilik</option>
                  {lecturersQuery.data?.map((lecturer) => (
                    <option key={lecturer.id} value={lecturer.id}>{lecturer.name}</option>
                  ))}
                </select>
              </Field>
            ) : null}
            {config.accountRequired ? (
              <div className="rounded-lg border border-[#eff2f5] bg-[#f9fafb] p-4">
                <div className="mb-4">
                  <div className="text-sm font-extrabold text-[#181c32]">Akun Login {config.singular}</div>
                  <div className="text-xs font-semibold text-[#a1a5b7]">
                    Akun dibuat otomatis saat data {config.singular.toLowerCase()} disimpan.
                  </div>
                </div>
                <div className="space-y-4">
                  <Field label="Email Akun" error={form.formState.errors.account_email?.message}>
                    <input className="input bg-white" disabled={Boolean(editing)} placeholder={config.emailPlaceholder} {...form.register('account_email')} />
                  </Field>
                  {!editing ? (
                    <Field label="Password Awal" error={form.formState.errors.account_password?.message}>
                      <input className="input bg-white" type="password" placeholder="Minimal 8 karakter" {...form.register('account_password')} />
                    </Field>
                  ) : null}
                </div>
              </div>
            ) : null}
            <Field label="Keterangan" error={form.formState.errors.description?.message}>
              <textarea className="input min-h-28" placeholder="Catatan internal" {...form.register('description')} />
            </Field>

            {saveMutation.error ? (
              <div className="rounded-lg bg-[#fff5f8] px-4 py-3 text-sm font-bold text-[#f1416c]">
                <div>{getApiErrorMessage(saveMutation.error)}</div>
                {getApiErrorDetail(saveMutation.error) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(saveMutation.error)}</div> : null}
              </div>
            ) : null}

            <button className="btn btn-primary w-full justify-center" disabled={saveMutation.isPending}>
              <Plus size={18} />
              {saveMutation.isPending ? 'Menyimpan...' : editing ? 'Update Data' : 'Tambah Data'}
            </button>
          </div>
        </form>
        ) : (
          <aside className="panel h-fit p-5">
            <div className="text-sm font-bold uppercase text-[#a1a5b7]">RBAC</div>
            <h3 className="mt-1 text-xl font-extrabold text-[#181c32]">Akses Baca Saja</h3>
            <p className="mt-2 text-sm font-medium leading-6 text-[#7e8299]">
              Role kamu dapat melihat data {config.singular.toLowerCase()}, tetapi tidak dapat menambah, edit, atau menghapus data ini.
            </p>
          </aside>
        )}
      </section>
      {config.detailStudents && detailItem ? (
        <ClassStudentsModal
          item={detailItem}
          query={classStudentsQuery}
          onClose={() => setDetailItem(null)}
        />
      ) : null}
    </div>
  );
}

function ClassStudentsModal({ item, query, onClose }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-[#181c32]/40 p-4">
      <div className="max-h-[86vh] w-full max-w-4xl overflow-hidden rounded-xl bg-white shadow-xl">
        <div className="flex items-start justify-between border-b border-[#eff2f5] p-5">
          <div>
            <div className="text-sm font-bold uppercase text-[#a1a5b7]">Detail Kelas</div>
            <h3 className="mt-1 text-xl font-extrabold text-[#181c32]">{item.name}</h3>
            <p className="mt-1 text-sm font-semibold text-[#7e8299]">Daftar siswa yang pernah atau sedang tergabung dalam kelas ini.</p>
          </div>
          <button className="grid h-9 w-9 place-items-center rounded-lg bg-[#f5f8fa] text-[#7e8299]" type="button" onClick={onClose}>
            <X size={18} />
          </button>
        </div>
        <div className="max-h-[64vh] overflow-auto">
          <table className="w-full min-w-[720px] text-left text-sm">
            <thead className="sticky top-0 border-b border-[#eff2f5] bg-white text-xs font-extrabold uppercase text-[#a1a5b7]">
              <tr>
                <th className="px-5 py-4">Kode</th>
                <th className="px-5 py-4">Nama Siswa</th>
                <th className="px-5 py-4">Program</th>
                <th className="px-5 py-4">Status</th>
                <th className="px-5 py-4">Tanggal Masuk</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#eff2f5]">
              {query.isLoading ? (
                <tr><td className="px-5 py-10 text-center font-semibold text-[#7e8299]" colSpan={5}>Memuat siswa...</td></tr>
              ) : null}
              {query.data?.items?.map((student) => (
                <tr key={student.enrollment_id} className="hover:bg-[#f9fafb]">
                  <td className="px-5 py-4 font-extrabold text-[#181c32]">{student.student_code}</td>
                  <td className="px-5 py-4 font-bold text-[#181c32]">{student.student_name}</td>
                  <td className="px-5 py-4 font-semibold text-[#7e8299]">{student.study_program_name || '-'}</td>
                  <td className="px-5 py-4">
                    <span className={student.active ? 'inline-flex rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold text-[#50cd89]' : 'inline-flex rounded-md bg-[#f5f8fa] px-2.5 py-1 text-xs font-extrabold text-[#7e8299]'}>
                      {student.active ? 'Aktif' : student.status}
                    </span>
                  </td>
                  <td className="px-5 py-4 font-semibold text-[#7e8299]">{student.enrolled_at ? new Date(student.enrolled_at).toLocaleDateString('id-ID') : '-'}</td>
                </tr>
              ))}
              {!query.isLoading && query.data?.items?.length === 0 ? (
                <tr><td className="px-5 py-12 text-center font-semibold text-[#7e8299]" colSpan={5}>Belum ada siswa di kelas ini.</td></tr>
              ) : null}
            </tbody>
          </table>
        </div>
        <div className="border-t border-[#eff2f5] p-5 text-sm font-semibold text-[#7e8299]">
          Total siswa: {query.data?.total || 0}
        </div>
      </div>
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

function Field({ label, error, children }) {
  return (
    <label className="block">
      <span className="mb-2 block text-sm font-bold text-[#3f4254]">{label}</span>
      {children}
      {error ? <span className="mt-2 block text-sm font-semibold text-danger">{error}</span> : null}
    </label>
  );
}

function toPayload(values) {
  const payload = {
    code: values.code,
    name: values.name,
    description: values.description || '',
    status: values.status,
    metadata: values.metadata_label ? { label: values.metadata_label } : {},
  };
  if (values.lecturer_id) {
    payload.lecturer_id = values.lecturer_id;
  }
  if (values.account_email) {
    payload.email = values.account_email;
  }
  if (values.account_password) {
    payload.password = values.account_password;
  }
  return payload;
}
