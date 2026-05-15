import { useAuthStore } from '../stores/authStore';
export function PermissionGuard({permission, children, fallback=null}){ const ok=useAuthStore(s=>s.hasPermission(permission)); return ok ? children : fallback; }
