import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link2, RefreshCcw, UserCheck } from 'lucide-react';
import { api } from '../lib/api';
import { getApiErrorDetail, getApiErrorMessage } from '../lib/apiError';

export default function EnrollmentPage() {
  const queryClient = useQueryClient();
  const [studentId, setStudentId] = useState('');
  const [classRoomId, setClassRoomId] = useState('');
  const [search, setSearch] = useState('');

  const students = useQuery({
    queryKey: ['students', 'select'],
    queryFn: async () => (await api.get('/students/', { params: { page: 1, limit: 200 } })).data.data.items,
  });
  const classes = useQuery({
    queryKey: ['class-rooms', 'select'],
    queryFn: async () => (await api.get('/class-rooms/', { params: { page: 1, limit: 200 } })).data.data.items,
  });
  const enrollments = useQuery({
    queryKey: ['enrollment', search, studentId],
    queryFn: async () => {
      const params = { page: 1, limit: 50, search };
      if (studentId) params.student_id = studentId;
      return (await api.get('/enrollment/', { params })).data.data;
    },
  });

  const selectedStudent = useMemo(() => students.data?.find((item) => item.id === studentId), [students.data, studentId]);

  const saveMutation = useMutation({
    mutationFn: async () => {
      const payload = {
        student_id: studentId,
        class_room_id: classRoomId,
        status: 'active',
        description: 'Enrollment aktif',
        metadata: {},
      };
      return (await api.post('/enrollment/', payload)).data.data;
    },
    onSuccess: () => {
      setClassRoomId('');
      queryClient.invalidateQueries({ queryKey: ['enrollment'] });
      queryClient.invalidateQueries({ queryKey: ['students'] });
    },
  });

  const closeMutation = useMutation({
    mutationFn: async (item) => api.delete(`/enrollment/${item.id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['enrollment'] }),
  });

  return (
    <div className="space-y-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <div className="text-sm font-bold text-[#a1a5b7]">Academic Relationship</div>
          <h2 className="mt-1 text-2xl font-extrabold text-[#181c32]">Enrollment Siswa ke Kelas</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">
            Satu kelas dapat memiliki banyak siswa, dan setiap perpindahan kelas siswa disimpan sebagai riwayat.
          </p>
        </div>
        <div className="panel flex items-center p-4 sm:w-72">
          <div className="grid h-11 w-11 place-items-center rounded-lg bg-[#f1faff] text-[#009ef7]">
            <UserCheck size={21} />
          </div>
          <div className="ml-3">
            <div className="text-xs font-bold uppercase text-[#a1a5b7]">Enrollment</div>
            <div className="text-xl font-extrabold text-[#181c32]">{enrollments.data?.total || 0}</div>
          </div>
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-[380px_minmax(0,1fr)]">
        <form
          className="panel h-fit overflow-hidden"
          onSubmit={(event) => {
            event.preventDefault();
            if (studentId && classRoomId) saveMutation.mutate();
          }}
        >
          <div className="border-b border-[#eff2f5] p-5">
            <div className="text-sm font-bold text-[#a1a5b7]">Tambah / Pindah Kelas</div>
            <h3 className="mt-1 text-xl font-extrabold text-[#181c32]">Assign Siswa</h3>
          </div>
          <div className="space-y-4 p-5">
            <Field label="Siswa">
              <select className="input" value={studentId} onChange={(event) => setStudentId(event.target.value)}>
                <option value="">Pilih siswa</option>
                {students.data?.map((student) => (
                  <option key={student.id} value={student.id}>
                    {student.code} - {student.name}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Kelas">
              <select className="input" value={classRoomId} onChange={(event) => setClassRoomId(event.target.value)}>
                <option value="">Pilih kelas</option>
                {classes.data?.map((classRoom) => (
                  <option key={classRoom.id} value={classRoom.id}>
                    {classRoom.code} - {classRoom.name}
                  </option>
                ))}
              </select>
            </Field>
            {selectedStudent ? (
              <div className="rounded-lg bg-[#f5f8fa] p-4 text-sm font-semibold text-[#7e8299]">
                Riwayat di bawah otomatis difilter untuk {selectedStudent.name}.
              </div>
            ) : null}
            {saveMutation.error ? (
              <div className="rounded-lg bg-[#fff5f8] px-4 py-3 text-sm font-bold text-[#f1416c]">
                <div>{getApiErrorMessage(saveMutation.error)}</div>
                {getApiErrorDetail(saveMutation.error) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(saveMutation.error)}</div> : null}
              </div>
            ) : null}
            <button className="btn btn-primary w-full justify-center" disabled={!studentId || !classRoomId || saveMutation.isPending}>
              <Link2 size={18} />
              {saveMutation.isPending ? 'Menyimpan...' : 'Simpan Enrollment'}
            </button>
            {closeMutation.error ? (
              <div className="rounded-lg bg-[#fff5f8] px-4 py-3 text-sm font-bold text-[#f1416c]">
                <div>{getApiErrorMessage(closeMutation.error)}</div>
                {getApiErrorDetail(closeMutation.error) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(closeMutation.error)}</div> : null}
              </div>
            ) : null}
          </div>
        </form>

        <div className="panel overflow-hidden">
          <div className="flex flex-col gap-4 border-b border-[#eff2f5] p-5 md:flex-row md:items-center md:justify-between">
            <div>
              <h3 className="text-lg font-extrabold text-[#181c32]">Riwayat Kelas Siswa</h3>
              <p className="text-sm font-medium text-[#a1a5b7]">Aktif dan histori perpindahan kelas.</p>
            </div>
            <div className="flex flex-col gap-3 sm:flex-row">
              <label className="sm:w-72">
                <input className="input h-11" placeholder="Cari siswa atau kelas" value={search} onChange={(event) => setSearch(event.target.value)} />
              </label>
              <button className="btn btn-ghost justify-center" onClick={() => enrollments.refetch()}>
                <RefreshCcw size={17} />
                Refresh
              </button>
            </div>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full min-w-[820px] text-left text-sm">
              <thead className="border-b border-[#eff2f5] text-xs font-extrabold uppercase text-[#a1a5b7]">
                <tr>
                  <th className="px-5 py-4">Siswa</th>
                  <th className="px-5 py-4">Kelas</th>
                  <th className="px-5 py-4">Mulai</th>
                  <th className="px-5 py-4">Selesai</th>
                  <th className="px-5 py-4">Status</th>
                  <th className="px-5 py-4 text-right">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#eff2f5]">
                {enrollments.isLoading ? (
                  <tr>
                    <td className="px-5 py-10 text-center font-semibold text-[#7e8299]" colSpan={6}>Memuat data...</td>
                  </tr>
                ) : null}
                {enrollments.data?.items?.map((item) => (
                  <tr key={item.id} className="hover:bg-[#f9fafb]">
                    <td className="px-5 py-4">
                      <div className="font-extrabold text-[#181c32]">{item.student_name}</div>
                      <div className="text-xs font-semibold text-[#a1a5b7]">{item.student_code}</div>
                    </td>
                    <td className="px-5 py-4">
                      <div className="font-bold text-[#181c32]">{item.class_room_name}</div>
                      <div className="text-xs font-semibold text-[#a1a5b7]">{item.class_room_code}</div>
                    </td>
                    <td className="px-5 py-4 font-medium text-[#7e8299]">{formatDate(item.enrolled_at)}</td>
                    <td className="px-5 py-4 font-medium text-[#7e8299]">{item.exited_at ? formatDate(item.exited_at) : '-'}</td>
                    <td className="px-5 py-4">
                      <span className={item.active ? 'inline-flex rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold text-[#50cd89]' : 'inline-flex rounded-md bg-[#f5f8fa] px-2.5 py-1 text-xs font-extrabold text-[#7e8299]'}>
                        {item.active ? 'Aktif' : 'History'}
                      </span>
                    </td>
                    <td className="px-5 py-4 text-right">
                      <button className="btn btn-ghost" disabled={!item.active || closeMutation.isPending} onClick={() => closeMutation.mutate(item)}>
                        Tutup
                      </button>
                    </td>
                  </tr>
                ))}
                {!enrollments.isLoading && enrollments.data?.items?.length === 0 ? (
                  <tr>
                    <td className="px-5 py-12 text-center font-semibold text-[#7e8299]" colSpan={6}>
                      Belum ada enrollment.
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
    <label className="block">
      <span className="mb-2 block text-sm font-bold text-[#3f4254]">{label}</span>
      {children}
    </label>
  );
}

function formatDate(value) {
  if (!value) return '-';
  return new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value));
}
