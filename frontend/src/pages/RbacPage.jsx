import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CheckCircle2, LockKeyhole, RefreshCcw, Save, ShieldCheck } from 'lucide-react';
import { api } from '../lib/api';
import { getApiErrorDetail, getApiErrorMessage } from '../lib/apiError';

export default function RbacPage() {
  const queryClient = useQueryClient();
  const [roleSearch, setRoleSearch] = useState('');
  const [permissionSearch, setPermissionSearch] = useState('');
  const [selectedRoleId, setSelectedRoleId] = useState('');
  const [selectedPermissionIds, setSelectedPermissionIds] = useState([]);

  const roles = useQuery({
    queryKey: ['roles', roleSearch],
    queryFn: async () => (await api.get('/roles/', { params: { page: 1, limit: 100, search: roleSearch } })).data.data.items,
  });

  const permissions = useQuery({
    queryKey: ['permissions', selectedRoleId],
    enabled: Boolean(selectedRoleId),
    queryFn: async () => (await api.get(`/roles/${selectedRoleId}/permissions`)).data.data,
  });

  const selectedRole = useMemo(() => roles.data?.find((role) => role.id === selectedRoleId), [roles.data, selectedRoleId]);
  const filteredPermissions = useMemo(() => {
    const items = permissions.data?.permissions || [];
    const clean = permissionSearch.trim().toLowerCase();
    if (!clean) return items;
    return items.filter((item) => `${item.code} ${item.name} ${item.description || ''}`.toLowerCase().includes(clean));
  }, [permissions.data, permissionSearch]);

  const groupedPermissions = useMemo(() => {
    const groups = new Map();
    for (const permission of filteredPermissions) {
      const groupName = permission.code === '*' ? 'super' : permission.code.split(':')[0] || 'other';
      if (!groups.has(groupName)) groups.set(groupName, []);
      groups.get(groupName).push(permission);
    }
    return Array.from(groups.entries()).sort(([a], [b]) => a.localeCompare(b));
  }, [filteredPermissions]);

  const savePermissions = useMutation({
    mutationFn: async () => (await api.put(`/roles/${selectedRoleId}/permissions`, { permission_ids: selectedPermissionIds })).data.data,
    onSuccess: (data) => {
      setSelectedPermissionIds(data.permissions.filter((permission) => permission.assigned).map((permission) => permission.id));
      queryClient.invalidateQueries({ queryKey: ['permissions', selectedRoleId] });
    },
  });

  function chooseRole(role) {
    setSelectedRoleId(role.id);
    setPermissionSearch('');
    setSelectedPermissionIds([]);
  }

  function togglePermission(permissionId) {
    setSelectedPermissionIds((current) => (
      current.includes(permissionId)
        ? current.filter((id) => id !== permissionId)
        : [...current, permissionId]
    ));
  }

  function selectAllVisible() {
    setSelectedPermissionIds((current) => Array.from(new Set([...current, ...filteredPermissions.map((permission) => permission.id)])));
  }

  function clearVisible() {
    const visibleIds = new Set(filteredPermissions.map((permission) => permission.id));
    setSelectedPermissionIds((current) => current.filter((id) => !visibleIds.has(id)));
  }

  useEffect(() => {
    if (permissions.data?.permissions) {
      setSelectedPermissionIds(permissions.data.permissions.filter((permission) => permission.assigned).map((permission) => permission.id));
    }
  }, [permissions.data]);

  return (
    <div className="space-y-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <div className="text-sm font-bold text-[#a1a5b7]">Admin Security</div>
          <h2 className="mt-1 text-2xl font-extrabold text-[#181c32]">RBAC Permission Management</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">
            Atur permission granular untuk setiap role. Perubahan berlaku saat user login ulang atau token di-refresh.
          </p>
        </div>
        <div className="panel flex items-center p-4 sm:w-72">
          <div className="grid h-11 w-11 place-items-center rounded-lg bg-[#f1faff] text-[#009ef7]">
            <ShieldCheck size={21} />
          </div>
          <div className="ml-3">
            <div className="text-xs font-bold uppercase text-[#a1a5b7]">Permission Dipilih</div>
            <div className="text-xl font-extrabold text-[#181c32]">{selectedPermissionIds.length}</div>
          </div>
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-[360px_minmax(0,1fr)]">
        <aside className="panel h-fit overflow-hidden">
          <div className="border-b border-[#eff2f5] p-5">
            <h3 className="text-lg font-extrabold text-[#181c32]">Role</h3>
            <p className="text-sm font-medium text-[#a1a5b7]">Pilih role yang akan diatur.</p>
            <label className="mt-4 block">
              <input className="input h-11" placeholder="Cari role" value={roleSearch} onChange={(event) => setRoleSearch(event.target.value)} />
            </label>
          </div>
          <div className="max-h-[640px] overflow-y-auto p-3">
            {roles.isLoading ? <div className="p-4 text-sm font-semibold text-[#7e8299]">Memuat role...</div> : null}
            {roles.data?.map((role) => (
              <button
                key={role.id}
                className={role.id === selectedRoleId ? 'mb-2 flex w-full items-center justify-between rounded-lg bg-[#f1faff] px-4 py-3 text-left' : 'mb-2 flex w-full items-center justify-between rounded-lg px-4 py-3 text-left hover:bg-[#f5f8fa]'}
                onClick={() => chooseRole(role)}
              >
                <span>
                  <span className="block font-extrabold text-[#181c32]">{role.name}</span>
                  <span className="mt-1 block text-xs font-semibold text-[#a1a5b7]">{role.code}</span>
                </span>
                {role.id === selectedRoleId ? <CheckCircle2 className="text-[#009ef7]" size={18} /> : null}
              </button>
            ))}
          </div>
        </aside>

        <div className="panel overflow-hidden">
          <div className="flex flex-col gap-4 border-b border-[#eff2f5] p-5 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h3 className="text-lg font-extrabold text-[#181c32]">{selectedRole ? selectedRole.name : 'Permission Matrix'}</h3>
              <p className="text-sm font-medium text-[#a1a5b7]">{selectedRole ? selectedRole.code : 'Pilih role untuk mulai mengatur permission.'}</p>
            </div>
            <div className="flex flex-col gap-3 sm:flex-row">
              <label className="sm:w-80">
                <input className="input h-11" placeholder="Cari permission" value={permissionSearch} onChange={(event) => setPermissionSearch(event.target.value)} disabled={!selectedRoleId} />
              </label>
              <button className="btn btn-ghost justify-center" disabled={!selectedRoleId} onClick={() => permissions.refetch()}>
                <RefreshCcw size={17} />
                Refresh
              </button>
            </div>
          </div>

          {selectedRoleId ? (
            <div className="space-y-5 p-5">
              <div className="flex flex-col gap-3 rounded-lg border border-[#eff2f5] bg-[#f9fafb] p-4 sm:flex-row sm:items-center sm:justify-between">
                <div className="text-sm font-semibold text-[#7e8299]">
                  Centang permission yang boleh digunakan oleh role <span className="font-extrabold text-[#181c32]">{selectedRole?.name}</span>.
                </div>
                <div className="flex flex-wrap gap-2">
                  <button className="btn btn-ghost" type="button" onClick={selectAllVisible}>Pilih Visible</button>
                  <button className="btn btn-ghost" type="button" onClick={clearVisible}>Clear Visible</button>
                  <button className="btn btn-primary" type="button" disabled={savePermissions.isPending} onClick={() => savePermissions.mutate()}>
                    <Save size={17} />
                    {savePermissions.isPending ? 'Menyimpan...' : 'Simpan'}
                  </button>
                </div>
              </div>

              {savePermissions.error ? (
                <div className="rounded-lg bg-[#fff5f8] px-4 py-3 text-sm font-bold text-[#f1416c]">
                  <div>{getApiErrorMessage(savePermissions.error)}</div>
                  {getApiErrorDetail(savePermissions.error) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(savePermissions.error)}</div> : null}
                </div>
              ) : null}

              {permissions.isLoading ? (
                <div className="rounded-lg bg-[#f5f8fa] p-6 text-sm font-semibold text-[#7e8299]">Memuat permission...</div>
              ) : null}

              {groupedPermissions.map(([group, items]) => (
                <div key={group} className="overflow-hidden rounded-lg border border-[#eff2f5]">
                  <div className="flex items-center justify-between border-b border-[#eff2f5] bg-[#f9fafb] px-4 py-3">
                    <div className="font-extrabold capitalize text-[#181c32]">{group.replaceAll('.', ' ')}</div>
                    <div className="text-xs font-bold uppercase text-[#a1a5b7]">{items.length} permission</div>
                  </div>
                  <div className="grid gap-0 md:grid-cols-2 2xl:grid-cols-3">
                    {items.map((permission) => {
                      const checked = selectedPermissionIds.includes(permission.id);
                      return (
                        <label key={permission.id} className="flex cursor-pointer items-start gap-3 border-b border-r border-[#eff2f5] p-4 hover:bg-[#f9fafb]">
                          <input
                            className="mt-1 h-4 w-4 accent-[#009ef7]"
                            type="checkbox"
                            checked={checked}
                            onChange={() => togglePermission(permission.id)}
                          />
                          <span className="min-w-0">
                            <span className="block break-words font-extrabold text-[#181c32]">{permission.code}</span>
                            <span className="mt-1 block text-xs font-semibold text-[#7e8299]">{permission.name}</span>
                          </span>
                        </label>
                      );
                    })}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="flex min-h-[420px] items-center justify-center p-8">
              <div className="text-center">
                <div className="mx-auto grid h-14 w-14 place-items-center rounded-xl bg-[#f1faff] text-[#009ef7]">
                  <LockKeyhole size={24} />
                </div>
                <h3 className="mt-4 text-lg font-extrabold text-[#181c32]">Pilih Role</h3>
                <p className="mt-2 max-w-sm text-sm font-medium leading-6 text-[#7e8299]">Permission matrix akan muncul setelah kamu memilih role dari panel kiri.</p>
              </div>
            </div>
          )}
        </div>
      </section>
    </div>
  );
}
