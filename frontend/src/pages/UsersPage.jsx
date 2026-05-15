import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { KeyRound, RefreshCcw, ShieldCheck, Users } from 'lucide-react';
import { api } from '../lib/api';
import { getApiErrorDetail, getApiErrorMessage } from '../lib/apiError';

export default function UsersPage() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [selectedUser, setSelectedUser] = useState(null);
  const [password, setPassword] = useState('');

  const users = useQuery({
    queryKey: ['users', search],
    queryFn: async () => (await api.get('/users/', { params: { page: 1, limit: 50, search } })).data.data,
  });

  const passwordMutation = useMutation({
    mutationFn: async () => api.put(`/users/${selectedUser.id}/password`, { password }),
    onSuccess: () => {
      setPassword('');
      setSelectedUser(null);
      queryClient.invalidateQueries({ queryKey: ['users'] });
    },
  });

  return (
    <div className="space-y-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <div className="text-sm font-bold text-[#a1a5b7]">Super Admin Only</div>
          <h2 className="mt-1 text-2xl font-extrabold text-[#181c32]">Manajemen User</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">
            Kelola akun user lintas role dan reset password langsung. Akses halaman ini dikunci oleh permission `users:read` dan `users:write`.
          </p>
        </div>
        <div className="panel flex items-center p-4 sm:w-72">
          <div className="grid h-11 w-11 place-items-center rounded-lg bg-[#f1faff] text-[#009ef7]">
            <Users size={21} />
          </div>
          <div className="ml-3">
            <div className="text-xs font-bold uppercase text-[#a1a5b7]">Total Users</div>
            <div className="text-xl font-extrabold text-[#181c32]">{users.data?.total || 0}</div>
          </div>
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="panel overflow-hidden">
          <div className="flex flex-col gap-4 border-b border-[#eff2f5] p-5 md:flex-row md:items-center md:justify-between">
            <div>
              <h3 className="text-lg font-extrabold text-[#181c32]">Daftar User</h3>
              <p className="text-sm font-medium text-[#a1a5b7]">Email, role, status, dan audit waktu pembuatan.</p>
            </div>
            <div className="flex flex-col gap-3 sm:flex-row">
              <label className="relative sm:w-72">
                <input className="input h-11" placeholder="Cari nama atau email" value={search} onChange={(event) => setSearch(event.target.value)} />
              </label>
              <button className="btn btn-ghost justify-center" onClick={() => users.refetch()}>
                <RefreshCcw size={17} />
                Refresh
              </button>
            </div>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full min-w-[820px] text-left text-sm">
              <thead className="border-b border-[#eff2f5] text-xs font-extrabold uppercase text-[#a1a5b7]">
                <tr>
                  <th className="px-5 py-4">User</th>
                  <th className="px-5 py-4">Role</th>
                  <th className="px-5 py-4">Status</th>
                  <th className="px-5 py-4">Dibuat</th>
                  <th className="px-5 py-4 text-right">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#eff2f5]">
                {users.isLoading ? (
                  <tr>
                    <td className="px-5 py-10 text-center font-semibold text-[#7e8299]" colSpan={5}>Memuat user...</td>
                  </tr>
                ) : null}
                {users.data?.items?.map((user) => (
                  <tr key={user.id} className="hover:bg-[#f9fafb]">
                    <td className="px-5 py-4">
                      <div className="font-extrabold text-[#181c32]">{user.name}</div>
                      <div className="text-xs font-semibold text-[#a1a5b7]">{user.email || user.code}</div>
                    </td>
                    <td className="px-5 py-4">
                      <div className="flex flex-wrap gap-2">
                        {(user.roles?.length ? user.roles : ['No Role']).map((role) => (
                          <span key={role} className="inline-flex rounded-md bg-[#f1faff] px-2.5 py-1 text-xs font-extrabold text-[#009ef7]">
                            {role}
                          </span>
                        ))}
                      </div>
                    </td>
                    <td className="px-5 py-4">
                      <span className="inline-flex rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold capitalize text-[#50cd89]">{user.status}</span>
                    </td>
                    <td className="px-5 py-4 font-medium text-[#7e8299]">{formatDate(user.created_at)}</td>
                    <td className="px-5 py-4 text-right">
                      <button className="btn btn-ghost" onClick={() => setSelectedUser(user)}>
                        <KeyRound size={17} />
                        Ubah Password
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <form
          className="panel h-fit overflow-hidden"
          onSubmit={(event) => {
            event.preventDefault();
            if (selectedUser && password.length >= 8) passwordMutation.mutate();
          }}
        >
          <div className="border-b border-[#eff2f5] p-5">
            <div className="text-sm font-bold text-[#a1a5b7]">Credential Control</div>
            <h3 className="mt-1 text-xl font-extrabold text-[#181c32]">Reset Password</h3>
          </div>
          <div className="space-y-4 p-5">
            {selectedUser ? (
              <div className="rounded-lg bg-[#f5f8fa] p-4">
                <div className="font-extrabold text-[#181c32]">{selectedUser.name}</div>
                <div className="text-sm font-semibold text-[#7e8299]">{selectedUser.email}</div>
              </div>
            ) : (
              <div className="rounded-lg bg-[#f5f8fa] p-4 text-sm font-semibold text-[#7e8299]">
                Pilih user dari tabel untuk mengubah password.
              </div>
            )}
            <label className="block">
              <span className="mb-2 block text-sm font-bold text-[#3f4254]">Password Baru</span>
              <input className="input" type="password" placeholder="Minimal 8 karakter" value={password} onChange={(event) => setPassword(event.target.value)} />
            </label>
            {passwordMutation.error ? (
              <div className="rounded-lg bg-[#fff5f8] px-4 py-3 text-sm font-bold text-[#f1416c]">
                <div>{getApiErrorMessage(passwordMutation.error)}</div>
                {getApiErrorDetail(passwordMutation.error) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(passwordMutation.error)}</div> : null}
              </div>
            ) : null}
            <button className="btn btn-primary w-full justify-center" disabled={!selectedUser || password.length < 8 || passwordMutation.isPending}>
              <ShieldCheck size={18} />
              {passwordMutation.isPending ? 'Menyimpan...' : 'Update Password'}
            </button>
          </div>
        </form>
      </section>
    </div>
  );
}

function formatDate(value) {
  if (!value) return '-';
  return new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value));
}
