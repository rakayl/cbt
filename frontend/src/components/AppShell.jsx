import { NavLink, Outlet, useLocation } from 'react-router-dom';
import {
  BarChart3,
  Bell,
  BookOpen,
  CalendarClock,
  CreditCard,
  FileText,
  GraduationCap,
  History,
  LayoutDashboard,
  LibraryBig,
  ListChecks,
  LogOut,
  Menu,
  ShieldAlert,
  ShieldCheck,
  Users,
  UserRound,
} from 'lucide-react';
import { TenantSelector } from './TenantSelector';
import { useAuthStore } from '../stores/authStore';

const nav = [
  ['/', 'Dashboard', LayoutDashboard, '*'],
  ['/academic/programs', 'Program Akademik', LibraryBig, 'study.programs:read'],
  ['/academic/classes', 'Kelas', Users, 'class.rooms:read'],
  ['/academic/enrollment', 'Enrollment', ListChecks, 'enrollment:read'],
  ['/academic/students', 'Siswa', GraduationCap, 'students:read', ['Lecturer']],
  ['/academic/lecturers', 'Guru', UserRound, 'lecturers:read'],
  ['/academic/courses', 'Mapel', LibraryBig, 'courses:read'],
  ['/question-banks', 'Bank Soal', BookOpen, 'question.banks:read'],
  ['/questions', 'Create Soal', BookOpen, 'questions:read'],
  ['/exams', 'Jadwal Ujian', CalendarClock, 'exams:read'],
  ['/student/exams', 'Ujian Saya', CalendarClock, 'exams:join'],
  ['/student/history', 'History Ujian', History, 'exams:join'],
  ['/student/classes', 'Kelas Saya', GraduationCap, 'exams:join'],
  ['/analytics', 'Analitik', BarChart3, 'analytics:read'],
  ['/monitoring', 'Proctoring', ShieldAlert, 'proctoring:read'],
  ['/reports', 'Laporan', FileText, 'reports:read'],
  ['/rbac', 'RBAC Permission', ShieldCheck, 'roles:read'],
  ['/billing', 'Billing SaaS', CreditCard, 'billing:read'],
  ['/users', 'Manajemen User', Users, 'users:read'],
];

const pageHeaders = {
  '/': { title: 'Dashboard', eyebrow: 'CBT Kampus Enterprise' },
  '/academic/programs': { title: 'Program Akademik', eyebrow: 'Master Data Akademik' },
  '/academic/classes': { title: 'Manajemen Kelas', eyebrow: 'Master Data Akademik' },
  '/academic/enrollment': { title: 'Enrollment Siswa', eyebrow: 'Relasi Akademik' },
  '/academic/students': { title: 'Manajemen Siswa', eyebrow: 'Master Data Akademik' },
  '/academic/lecturers': { title: 'Manajemen Guru', eyebrow: 'Master Data Akademik' },
  '/academic/courses': { title: 'Manajemen Mapel', eyebrow: 'Master Data Akademik' },
  '/question-banks': { title: 'Bank Soal', eyebrow: 'Manajemen Soal' },
  '/questions': { title: 'Create Soal', eyebrow: 'Manajemen Soal' },
  '/exams': { title: 'Jadwal Ujian', eyebrow: 'Exam Management' },
  '/student/exams': { title: 'Ujian Saya', eyebrow: 'Student Exam Center' },
  '/student/history': { title: 'History Ujian', eyebrow: 'Riwayat Siswa' },
  '/student/classes': { title: 'Kelas Saya', eyebrow: 'Akademik Siswa' },
  '/analytics': { title: 'Analitik', eyebrow: 'Insight & Monitoring' },
  '/monitoring': { title: 'Proctoring', eyebrow: 'Keamanan Ujian' },
  '/reports': { title: 'Laporan', eyebrow: 'Reporting Center' },
  '/rbac': { title: 'RBAC Permission', eyebrow: 'Admin Security' },
  '/billing': { title: 'Billing SaaS', eyebrow: 'Tenant Subscription' },
  '/users': { title: 'Manajemen User', eyebrow: 'Super Admin' },
};

function getHeaderMeta(pathname) {
  const path = Object.keys(pageHeaders)
    .sort((a, b) => b.length - a.length)
    .find((item) => (item === '/' ? pathname === '/' : pathname.startsWith(item)));
  return pageHeaders[path] || pageHeaders['/'];
}

export function AppShell() {
  const location = useLocation();
  const logout = useAuthStore((state) => state.logout);
  const hasPermission = useAuthStore((state) => state.hasPermission);
  const user = useAuthStore((state) => state.user);
  const roleNames = (user?.roles || []).map((role) => String(role).toLowerCase());
  const visibleNav = nav.filter(([, , , permission, hideForRoles]) => {
    const hidden = (hideForRoles || []).some((role) => roleNames.includes(String(role).toLowerCase()));
    return !hidden && (permission === '*' || hasPermission(permission));
  });
  const headerMeta = getHeaderMeta(location.pathname);
  const initials = (user?.name || 'Admin')
    .split(' ')
    .map((part) => part[0])
    .join('')
    .slice(0, 2)
    .toUpperCase();

  return (
    <div className="min-h-screen bg-[#f5f8fa] text-[#181c32]">
      <aside className="fixed inset-y-0 left-0 z-30 hidden w-[265px] flex-col bg-[#1e1e2d] lg:flex">
        <div className="flex h-[72px] items-center border-b border-white/5 px-7">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-[#009ef7] text-lg font-extrabold text-white">C</div>
          <div className="ml-3">
            <div className="text-base font-extrabold text-white">CBT Kampus</div>
            <div className="text-xs font-semibold text-[#7e8299]">Enterprise Suite</div>
          </div>
        </div>

        <div className="sidebar-scroll relative flex-1 overflow-y-auto px-4 py-5">
          <div className="sticky top-0 z-10 -mx-1 mb-2 bg-[#1e1e2d]/95 px-4 pb-3 pt-1 text-[11px] font-bold uppercase tracking-wider text-[#5e6278] backdrop-blur">
            Main Menu
          </div>
          <nav className="space-y-1">
            {visibleNav.map(([to, label, Icon]) => (
              <NavLink
                key={to}
                to={to}
                end={to === '/'}
                className={({ isActive }) =>
                  [
                    'group flex h-11 items-center rounded-lg px-3 text-sm font-semibold transition',
                    isActive ? 'bg-[#2a2a3c] text-white' : 'text-[#9899ac] hover:bg-[#2a2a3c] hover:text-white',
                  ].join(' ')
                }
              >
                <Icon className="mr-3 text-[#565674] group-hover:text-[#009ef7]" size={18} />
                <span>{label}</span>
              </NavLink>
            ))}
          </nav>
        </div>

        <div className="border-t border-white/5 p-4">
          <button className="flex h-11 w-full items-center rounded-lg px-3 text-sm font-semibold text-[#9899ac] hover:bg-[#2a2a3c] hover:text-white" onClick={logout}>
            <LogOut className="mr-3" size={18} />
            Logout
          </button>
        </div>
      </aside>

      <div className="lg:pl-[265px]">
        <header className="sticky top-0 z-20 flex min-h-[72px] items-center border-b border-[#eff2f5] bg-white px-4 shadow-sm md:px-8">
          <button className="mr-3 grid h-10 w-10 place-items-center rounded-lg border border-[#e4e6ef] bg-white text-[#5e6278] lg:hidden">
            <Menu size={20} />
          </button>

          <div className="min-w-0 flex-1">
            <div className="text-xs font-bold uppercase tracking-wide text-[#a1a5b7]">{headerMeta.eyebrow}</div>
            <h1 className="truncate text-lg font-extrabold text-[#181c32]">{headerMeta.title}</h1>
          </div>

          <div className="hidden min-w-[300px] max-w-md flex-1 px-6 xl:block">
            <label className="block">
              <input className="input h-11 bg-[#f5f8fa]" placeholder="Search exams, students, questions" />
            </label>
          </div>

          <div className="flex items-center gap-2">
            <TenantSelector />
            <button className="grid h-10 w-10 place-items-center rounded-lg bg-[#f5f8fa] text-[#5e6278]" title="Notifications">
              <Bell size={18} />
            </button>
            <div className="ml-1 flex items-center rounded-lg bg-[#f5f8fa] px-2 py-1.5">
              <div className="grid h-9 w-9 place-items-center rounded-lg bg-[#009ef7] text-sm font-extrabold text-white">{initials}</div>
              <div className="hidden px-3 sm:block">
                <div className="text-sm font-extrabold leading-tight text-[#181c32]">{user?.name || 'Administrator'}</div>
                <div className="text-xs font-semibold text-[#a1a5b7]">{user?.roles?.[0] || 'Super Admin'}</div>
              </div>
            </div>
          </div>
        </header>

        <nav className="flex gap-2 overflow-x-auto border-b border-[#eff2f5] bg-white px-4 py-3 lg:hidden">
          {visibleNav.map(([to, label, Icon]) => (
            <NavLink
              key={to}
              to={to}
              end={to === '/'}
              className={({ isActive }) =>
                [
                  'flex h-10 shrink-0 items-center rounded-lg px-3 text-sm font-bold',
                  isActive ? 'bg-[#009ef7] text-white' : 'bg-[#f5f8fa] text-[#5e6278]',
                ].join(' ')
              }
            >
              <Icon className="mr-2" size={16} />
              {label}
            </NavLink>
          ))}
        </nav>

        <main className="p-4 md:p-8">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
