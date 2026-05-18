import React, { useEffect } from 'react';
import { createRoot } from 'react-dom/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, HashRouter, Routes, Route, Navigate } from 'react-router-dom';
import { GraduationCap, LibraryBig, Users, UserRound } from 'lucide-react';
import './styles.css';
import { useAuthStore } from './stores/authStore';
import { AppShell } from './components/AppShell';
import { PageTitle } from './components/PageTitle';
import { ProtectedRoute } from './components/ProtectedRoute';
import AcademicResourcePage from './pages/AcademicResourcePage';
import EnrollmentPage from './pages/EnrollmentPage';
import LoginPage from './pages/LoginPage';
import RegisterPage from './pages/RegisterPage';
import DashboardPage from './pages/DashboardPage';
import QuestionBanksPage from './pages/QuestionBanksPage';
import QuestionsPage from './pages/QuestionsPage';
import ExamSchedulerPage from './pages/ExamSchedulerPage';
import ExamDetailPage from './pages/ExamDetailPage';
import ExamPage from './pages/ExamPage';
import StudentClassesPage from './pages/StudentClassesPage';
import StudentExamsPage from './pages/StudentExamsPage';
import StudentHistoryPage from './pages/StudentHistoryPage';
import StudentResultDetailPage from './pages/StudentResultDetailPage';
import AnalyticsPage from './pages/AnalyticsPage';
import MonitoringPage from './pages/MonitoringPage';
import ExamReviewPage from './pages/ExamReviewPage';
import ExamRankingsPage from './pages/ExamRankingsPage';
import ReportsPage from './pages/ReportsPage';
import RbacPage from './pages/RbacPage';
import BillingPage from './pages/BillingPage';
import UsersPage from './pages/UsersPage';
const queryClient = new QueryClient({ defaultOptions:{ queries:{ staleTime:30000, retry:1 } } });
const Router = import.meta.env.VITE_APP_PLATFORM === 'desktop' ? HashRouter : BrowserRouter;
const academicPages = {
  students: {
    resource: 'students',
    endpoint: '/students',
    singular: 'Siswa',
    title: 'Manajemen Siswa',
    eyebrow: 'Academic Master Data',
    description: 'Kelola identitas siswa dan akun login. Program dan kelas siswa dikelola terpisah melalui Enrollment.',
    codePlaceholder: 'NIS-0001',
    namePlaceholder: 'Nama siswa',
    metadataLabel: 'Nomor/Keterangan Siswa',
    metadataPlaceholder: 'NISN / catatan internal',
    emailPlaceholder: 'siswa@kampus.ac.id',
    accountRequired: true,
    writePermission: 'students:write',
    icon: GraduationCap,
  },
  lecturers: {
    resource: 'lecturers',
    endpoint: '/lecturers',
    singular: 'Guru',
    title: 'Manajemen Guru',
    eyebrow: 'Academic Master Data',
    description: 'Kelola guru atau dosen pengampu untuk bank soal, jadwal ujian, penilaian, dan laporan.',
    codePlaceholder: 'GR-0001',
    namePlaceholder: 'Nama guru',
    metadataLabel: 'Bidang Keahlian',
    metadataPlaceholder: 'Matematika / Rekayasa Perangkat Lunak',
    emailPlaceholder: 'guru@kampus.ac.id',
    accountRequired: true,
    writePermission: 'lecturers:write',
    icon: UserRound,
  },
  courses: {
    resource: 'courses',
    endpoint: '/courses',
    singular: 'Mapel',
    title: 'Manajemen Mapel',
    eyebrow: 'Academic Master Data',
    description: 'Kelola mata pelajaran atau mata kuliah sebagai dasar kelas, bank soal, dan jadwal ujian.',
    codePlaceholder: 'MTK-101',
    namePlaceholder: 'Nama mapel',
    metadataLabel: 'Kelompok Mapel',
    metadataPlaceholder: 'Wajib / Peminatan / Praktikum',
    writePermission: 'courses:write',
    icon: LibraryBig,
  },
  classes: {
    resource: 'class-rooms',
    endpoint: '/class-rooms',
    singular: 'Kelas',
    title: 'Manajemen Kelas',
    eyebrow: 'Academic Master Data',
    description: 'Kelola kelas/rombongan belajar. Satu kelas dapat berisi banyak siswa melalui Enrollment.',
    codePlaceholder: 'XII-RPL-1',
    namePlaceholder: 'XII RPL 1',
    metadataLabel: 'Program',
    metadataPlaceholder: 'RPL / IPA / IPS',
    writePermission: 'class.rooms:write',
    ownerLecturer: true,
    detailStudents: true,
    icon: Users,
  },
  programs: {
    resource: 'study-programs',
    endpoint: '/study-programs',
    singular: 'Program',
    title: 'Manajemen Program',
    eyebrow: 'Academic Master Data',
    description: 'Kelola program studi, jurusan, atau peminatan sebagai master data terpisah dari siswa.',
    codePlaceholder: 'RPL',
    namePlaceholder: 'Rekayasa Perangkat Lunak',
    metadataLabel: 'Fakultas/Bidang',
    metadataPlaceholder: 'Teknik / Kejuruan',
    writePermission: 'study.programs:write',
    icon: LibraryBig,
  },
};

function App(){ const hydrate=useAuthStore(s=>s.hydrate); useEffect(()=>{hydrate(); if(import.meta.env.VITE_APP_PLATFORM !== 'desktop' && 'serviceWorker' in navigator) navigator.serviceWorker.register('/sw.js').catch(()=>{});},[hydrate]); return <QueryClientProvider client={queryClient}><Router><PageTitle/><Routes><Route path="/login" element={<LoginPage/>}/><Route path="/register" element={<RegisterPage/>}/><Route path="/exam/:sessionId" element={<ProtectedRoute><ExamPage/></ProtectedRoute>}/><Route path="/" element={<ProtectedRoute><AppShell/></ProtectedRoute>}><Route index element={<DashboardPage/>}/><Route path="student/exams" element={<StudentExamsPage/>}/><Route path="student/history" element={<StudentHistoryPage/>}/><Route path="student/history/:sessionId" element={<StudentResultDetailPage/>}/><Route path="student/classes" element={<StudentClassesPage/>}/><Route path="academic/programs" element={<AcademicResourcePage config={academicPages.programs}/>}/><Route path="academic/classes" element={<AcademicResourcePage config={academicPages.classes}/>}/><Route path="academic/enrollment" element={<EnrollmentPage/>}/><Route path="academic/students" element={<AcademicResourcePage config={academicPages.students}/>}/><Route path="academic/lecturers" element={<AcademicResourcePage config={academicPages.lecturers}/>}/><Route path="academic/courses" element={<AcademicResourcePage config={academicPages.courses}/>}/><Route path="question-banks" element={<QuestionBanksPage/>}/><Route path="questions" element={<QuestionsPage/>}/><Route path="exams" element={<ExamSchedulerPage/>}/><Route path="exams/:examId" element={<ExamDetailPage/>}/><Route path="analytics" element={<AnalyticsPage/>}/><Route path="monitoring" element={<MonitoringPage/>}/><Route path="grading-review" element={<ExamReviewPage/>}/><Route path="exam-rankings" element={<ExamRankingsPage/>}/><Route path="reports" element={<ReportsPage/>}/><Route path="rbac" element={<RbacPage/>}/><Route path="billing" element={<BillingPage/>}/><Route path="users" element={<UsersPage/>}/></Route><Route path="*" element={<Navigate to="/"/>}/></Routes></Router></QueryClientProvider> }
createRoot(document.getElementById('root')).render(<App/>);
