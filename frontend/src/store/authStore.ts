import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface AuthUser { id:string; nama:string; email:string; role:'admin'|'guru'|'peserta' }

interface AuthState {
  token: string | null
  user: AuthUser | null
  setAuth: (token: string, user: AuthUser) => void
  logout: () => void
  isAuthenticated: () => boolean
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null, user: null,
      setAuth: (token, user) => { localStorage.setItem('cbt_token', token); set({ token, user }) },
      logout: () => { localStorage.removeItem('cbt_token'); set({ token: null, user: null }) },
      isAuthenticated: () => !!get().token,
    }),
    { name: 'cbt-auth' }
  )
)
