import { Navigate } from 'react-router-dom';
import { useAuthStore } from '../stores/authStore';
export function ProtectedRoute({children}){ const accessToken=useAuthStore(s=>s.accessToken); return accessToken ? children : <Navigate to="/login" replace/>; }
