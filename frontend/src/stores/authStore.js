import { create } from 'zustand';

const anonymousUser = { name: 'User', permissions: [], roles: [] };
const initialSession = readStoredSession();

export const useAuthStore = create((set, get) => ({
  accessToken: initialSession.accessToken,
  refreshToken: initialSession.refreshToken,
  tenantId: initialSession.tenantId,
  user: initialSession.user,
  hydrate() {
    const raw = localStorage.getItem('cbt.auth');
    if (raw) set(JSON.parse(raw));
  },
  setSession(session) {
    const claims = decodeJwt(session.access_token);
    const permissions = claims.permissions || session.user?.permissions || [];
    const next = {
      accessToken: session.access_token,
      refreshToken: session.refresh_token,
      tenantId: session.tenant_id || claims.tenant_id || crypto.randomUUID(),
      user: {
        name: session.user?.name || claims.email || 'User',
        permissions,
        roles: session.user?.roles || inferRoles(permissions),
      },
    };
    localStorage.setItem('cbt.auth', JSON.stringify(next));
    set(next);
  },
  logout() {
    localStorage.removeItem('cbt.auth');
    set({ accessToken: null, refreshToken: null, tenantId: null, user: anonymousUser });
  },
  hasPermission(permission) {
    const perms = get().user?.permissions || [];
    return perms.includes('*') || perms.includes(permission);
  },
}));

function readStoredSession() {
  try {
    const raw = localStorage.getItem('cbt.auth');
    if (!raw) return { accessToken: null, refreshToken: null, tenantId: null, user: anonymousUser };
    const parsed = JSON.parse(raw);
    return {
      accessToken: parsed.accessToken || null,
      refreshToken: parsed.refreshToken || null,
      tenantId: parsed.tenantId || null,
      user: parsed.user || anonymousUser,
    };
  } catch {
    return { accessToken: null, refreshToken: null, tenantId: null, user: anonymousUser };
  }
}

function decodeJwt(token) {
  try {
    const payload = token.split('.')[1];
    const normalized = payload.replace(/-/g, '+').replace(/_/g, '/');
    return JSON.parse(atob(normalized));
  } catch {
    return {};
  }
}

function inferRoles(permissions) {
  if (permissions.includes('*')) return ['Super Admin'];
  if (permissions.includes('exams:invite')) return ['Lecturer'];
  if (permissions.includes('exams:join')) return ['Student'];
  return ['User'];
}
