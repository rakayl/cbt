import axios from 'axios';
import { useAuthStore } from '../stores/authStore';
export const api = axios.create({ baseURL: import.meta.env.VITE_API_URL || '/api/v1', timeout: 30000 });
api.interceptors.request.use((config)=>{ const { accessToken, tenantId } = useAuthStore.getState(); if(accessToken) config.headers.Authorization = 'Bearer ' + accessToken; if(tenantId) config.headers['X-Tenant-ID'] = tenantId; return config; });
api.interceptors.response.use(r=>r, async error=>{
  const original = error.config || {};
  const store = useAuthStore.getState();
  if(error.response?.status===401 && !original._retry && store.refreshToken && !String(original.url || '').includes('/auth/refresh')){
    original._retry = true;
    try {
      const { data } = await axios.post(`${api.defaults.baseURL}/auth/refresh`, { refresh_token: store.refreshToken });
      store.setSession(data.data);
      original.headers = original.headers || {};
      original.headers.Authorization = 'Bearer ' + data.data.access_token;
      if(data.data.tenant_id) original.headers['X-Tenant-ID'] = data.data.tenant_id;
      return api(original);
    } catch {
      store.logout();
    }
  } else if(error.response?.status===401){
    store.logout();
  }
  throw error;
});
