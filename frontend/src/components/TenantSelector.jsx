import { Building2 } from 'lucide-react';
import { useAuthStore } from '../stores/authStore';

export function TenantSelector() {
  const tenantId = useAuthStore((state) => state.tenantId);

  return (
    <div className="hidden h-10 max-w-[260px] items-center gap-2 rounded-lg bg-[#f5f8fa] px-3 text-sm font-bold text-[#3f4254] md:flex" title="Tenant aktif">
      <Building2 size={18} />
      <span className="max-w-44 truncate">{tenantId || 'Campus Tenant'}</span>
    </div>
  );
}
