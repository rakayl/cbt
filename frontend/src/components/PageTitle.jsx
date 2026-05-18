import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';

const appName = 'Enterprise CBT Kampus';

const routeTitles = [
  { path: '/login', title: 'Login' },
  { path: '/register', title: 'Register' },
  { path: '/exam/', title: 'Secure Exam', startsWith: true },
  { path: '/student/exams', title: 'Ujian Saya' },
  { path: '/student/history', title: 'History Ujian', startsWith: true },
  { path: '/student/classes', title: 'Kelas Saya' },
  { path: '/academic/programs', title: 'Program Akademik' },
  { path: '/academic/classes', title: 'Manajemen Kelas' },
  { path: '/academic/enrollment', title: 'Enrollment Siswa' },
  { path: '/academic/students', title: 'Manajemen Siswa' },
  { path: '/academic/lecturers', title: 'Manajemen Guru' },
  { path: '/academic/courses', title: 'Manajemen Mapel' },
  { path: '/question-banks', title: 'Bank Soal' },
  { path: '/questions', title: 'Create Soal' },
  { path: '/exams/', title: 'Detail Ujian', startsWith: true },
  { path: '/exams', title: 'Jadwal Ujian' },
  { path: '/analytics', title: 'Analitik' },
  { path: '/monitoring', title: 'Proctoring' },
  { path: '/grading-review', title: 'Review Hasil' },
  { path: '/exam-rankings', title: 'Peringkat Ujian' },
  { path: '/reports', title: 'Laporan' },
  { path: '/rbac', title: 'RBAC Permission' },
  { path: '/billing', title: 'Billing SaaS' },
  { path: '/users', title: 'Manajemen User' },
  { path: '/', title: 'Dashboard' },
];

export function PageTitle() {
  const location = useLocation();

  useEffect(() => {
    const current = routeTitles.find((route) => (
      route.startsWith ? location.pathname.startsWith(route.path) : location.pathname === route.path
    ));
    document.title = current ? `${current.title} | ${appName}` : appName;
  }, [location.pathname]);

  return null;
}
