import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'

// Auth
import LoginPage from '@/pages/auth/LoginPage'

// Peserta (exam flow)
import ExamLobby  from '@/pages/exam/ExamLobby'
import ExamRoom   from '@/pages/exam/ExamRoom'
import ExamResult from '@/pages/exam/ExamResult'

// Admin
import AdminDashboard from '@/pages/admin/AdminDashboard'
import ProctoringLog  from '@/pages/admin/ProctoringLog'
import ProfileSetting from '@/pages/admin/ProfileSetting'

// Guru / Teacher
import TeacherDashboard from '@/pages/teacher/TeacherDashboard'
import BankSoal         from '@/pages/guru/BankSoal'
import BuatSoal         from '@/pages/guru/BuatSoal'
import ManajemenPeserta from '@/pages/guru/ManajemenPeserta'
import PenilaianEssay   from '@/pages/guru/PenilaianEssay'

function RequireAuth({ children, roles }: { children: React.ReactNode; roles?: string[] }) {
  const { token, user } = useAuthStore()
  if (!token) return <Navigate to="/login" replace />
  if (roles && user && !roles.includes(user.role)) return <Navigate to="/" replace />
  return <>{children}</>
}

function RootRedirect() {
  const { user } = useAuthStore()
  if (!user) return <Navigate to="/login" replace />
  if (user.role === 'admin') return <Navigate to="/admin"   replace />
  if (user.role === 'guru')  return <Navigate to="/teacher" replace />
  return <Navigate to="/exam" replace />
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/"      element={<RootRedirect />} />
        <Route path="/login" element={<LoginPage />} />

        {/* ── Peserta ───────────────────────────────────────────────── */}
        <Route path="/exam" element={
          <RequireAuth roles={['peserta']}><ExamLobby /></RequireAuth>
        }/>
        <Route path="/exam/room" element={
          <RequireAuth roles={['peserta']}><ExamRoom /></RequireAuth>
        }/>
        <Route path="/exam/result" element={
          <RequireAuth roles={['peserta']}><ExamResult /></RequireAuth>
        }/>

        {/* ── Admin ─────────────────────────────────────────────────── */}
        <Route path="/admin" element={
          <RequireAuth roles={['admin']}><AdminDashboard /></RequireAuth>
        }/>
        <Route path="/admin/proctoring" element={
          <RequireAuth roles={['admin']}><ProctoringLog /></RequireAuth>
        }/>
        <Route path="/admin/settings" element={
          <RequireAuth roles={['admin']}><ProfileSetting /></RequireAuth>
        }/>

        {/* ── Guru ──────────────────────────────────────────────────── */}
        <Route path="/teacher" element={
          <RequireAuth roles={['guru']}><TeacherDashboard /></RequireAuth>
        }/>
        <Route path="/teacher/bank-soal" element={
          <RequireAuth roles={['guru']}><BankSoal /></RequireAuth>
        }/>
        <Route path="/teacher/soal/baru" element={
          <RequireAuth roles={['guru']}><BuatSoal /></RequireAuth>
        }/>
        <Route path="/teacher/soal/:id" element={
          <RequireAuth roles={['guru']}><BuatSoal /></RequireAuth>
        }/>
        <Route path="/teacher/peserta" element={
          <RequireAuth roles={['guru']}><ManajemenPeserta /></RequireAuth>
        }/>
        <Route path="/teacher/essay" element={
          <RequireAuth roles={['guru']}><PenilaianEssay /></RequireAuth>
        }/>

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
