import { useQuery } from '@tanstack/react-query';
import {
  Activity,
  AlertTriangle,
  BarChart3,
  BookOpen,
  CalendarClock,
  CheckCircle2,
  CircleDot,
  Database,
  Eye,
  FileCheck2,
  GraduationCap,
  RefreshCcw,
  Server,
  ShieldAlert,
  Users,
} from 'lucide-react';
import { Link } from 'react-router-dom';
import { api } from '../lib/api';
import { getApiErrorDetail, getApiErrorMessage } from '../lib/apiError';
import { useAuthStore } from '../stores/authStore';

const metricIcons = {
  tenants: Server,
  users: Users,
  students: GraduationCap,
  lecturers: Users,
  questions: BookOpen,
  question_tags: CircleDot,
  exams: CalendarClock,
  active_sessions: Activity,
  draft_exams: FileCheck2,
  published_exams: CheckCircle2,
  manual_review: AlertTriangle,
  available_exams: CalendarClock,
  completed: FileCheck2,
  classes: Users,
  reconnecting: AlertTriangle,
  high_risk: ShieldAlert,
  events_today: Eye,
};

const roleCopy = {
  super_admin: ['Platform Command Center', 'Pantau tenant, penggunaan sistem, health service, dan aktivitas proctoring lintas kampus.'],
  tenant_admin: ['Dashboard Kampus', 'Ringkasan operasional kampus: siswa, guru, bank soal, ujian aktif, dan aktivitas mencurigakan terbaru.'],
  lecturer: ['Dashboard Guru', 'Fokus pada ujian, bank soal, sesi aktif, dan grading yang perlu diselesaikan.'],
  student: ['Dashboard Siswa', 'Lihat ujian yang tersedia, sesi berjalan, kelas yang diikuti, dan hasil ujian terakhir.'],
  proctor: ['Dashboard Proctor', 'Monitor peserta online, reconnect, high risk event, dan aktivitas anti-cheat terbaru.'],
};

export default function DashboardPage() {
  const user = useAuthStore((state) => state.user);
  const dashboard = useQuery({
    queryKey: ['dashboard-summary'],
    queryFn: async () => (await api.get('/dashboard/summary')).data.data,
    refetchInterval: 30000,
  });
  const data = dashboard.data || {};
  const [title, description] = roleCopy[data.role] || roleCopy.tenant_admin;

  return (
    <div className="space-y-6">
      <section className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <div className="text-sm font-bold text-[#a1a5b7]">{title}</div>
          <h2 className="mt-1 text-2xl font-extrabold text-[#181c32]">Halo, {user?.name || 'User'}</h2>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7e8299]">{description}</p>
        </div>
        <button className="btn btn-ghost justify-center" onClick={() => dashboard.refetch()}>
          <RefreshCcw size={17} />
          Refresh
        </button>
      </section>

      {dashboard.error ? (
        <section className="panel bg-[#fff5f8] p-5 text-sm font-bold text-[#f1416c]">
          <div>{getApiErrorMessage(dashboard.error)}</div>
          {getApiErrorDetail(dashboard.error) ? <div className="mt-1 text-xs font-semibold opacity-80">{getApiErrorDetail(dashboard.error)}</div> : null}
        </section>
      ) : null}

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {(dashboard.isLoading ? skeletonMetrics() : data.metrics || []).map((metric) => (
          <MetricCard key={metric.key} metric={metric} loading={dashboard.isLoading} />
        ))}
      </section>

      {data.role === 'student' ? <StudentDashboard data={data} /> : <OperatorDashboard data={data} />}
    </div>
  );
}

function OperatorDashboard({ data }) {
  return (
    <div className="grid gap-6 xl:grid-cols-[1.25fr_0.75fr]">
      <section className="space-y-6">
        <ActiveExamPanel exams={data.active_exams || []} />
        <TrendPanel trends={data.trends || []} />
      </section>
      <section className="space-y-6">
        {data.health ? <HealthPanel health={data.health} /> : null}
        <ActivityPanel activities={data.activities || []} />
      </section>
    </div>
  );
}

function StudentDashboard({ data }) {
  return (
    <div className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
      <section className="space-y-6">
        <UpcomingExamPanel exams={data.upcoming_exams || []} />
      </section>
      <section className="space-y-6">
        <RecentResultPanel results={data.recent_results || []} />
      </section>
    </div>
  );
}

function MetricCard({ metric, loading }) {
  const Icon = metricIcons[metric.key] || BarChart3;
  const tone = toneClass(metric.tone);
  return (
    <article className="panel p-5">
      <div className="flex items-center justify-between">
        <div className={`grid h-12 w-12 place-items-center rounded-lg ${tone.iconBg} ${tone.iconText}`}>
          <Icon size={22} />
        </div>
        <span className={`rounded-md px-2.5 py-1 text-xs font-extrabold ${tone.badge}`}>{metric.tone || 'info'}</span>
      </div>
      <div className="mt-5 text-sm font-bold text-[#a1a5b7]">{metric.label}</div>
      <div className={loading ? 'mt-2 h-9 w-28 animate-pulse rounded bg-[#f5f8fa]' : 'mt-2 text-3xl font-extrabold text-[#181c32]'}>
        {loading ? '' : metric.display}
      </div>
    </article>
  );
}

function ActiveExamPanel({ exams }) {
  return (
    <section className="panel overflow-hidden">
      <PanelHeader eyebrow="Realtime Monitoring" title="Ujian Berjalan" action={<Link className="btn btn-ghost h-10" to="/monitoring"><Eye size={17} /> Monitoring</Link>} />
      <div className="overflow-x-auto">
        <table className="w-full min-w-[680px] text-left text-sm">
          <thead className="border-b border-[#eff2f5] text-xs font-extrabold uppercase text-[#a1a5b7]">
            <tr>
              <th className="px-5 py-4">Ujian</th>
              <th className="px-5 py-4">Status</th>
              <th className="px-5 py-4">Peserta</th>
              <th className="px-5 py-4">Published</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[#eff2f5]">
            {exams.map((exam) => (
              <tr key={exam.id} className="hover:bg-[#f9fafb]">
                <td className="px-5 py-4">
                  <div className="font-extrabold text-[#181c32]">{exam.name}</div>
                  <div className="mt-1 text-xs font-semibold text-[#a1a5b7]">{exam.code}</div>
                </td>
                <td className="px-5 py-4"><StatusBadge status={exam.status} /></td>
                <td className="px-5 py-4 font-extrabold text-[#181c32]">{exam.participant_count || 0}</td>
                <td className="px-5 py-4 font-semibold text-[#7e8299]">{formatDate(exam.published_at)}</td>
              </tr>
            ))}
            {exams.length === 0 ? <EmptyRow colSpan={4} text="Belum ada ujian yang sedang berjalan." /> : null}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function UpcomingExamPanel({ exams }) {
  return (
    <section className="panel overflow-hidden">
      <PanelHeader eyebrow="Student Exam Center" title="Ujian Tersedia" action={<Link className="btn btn-primary h-10" to="/student/exams"><CalendarClock size={17} /> Ujian Saya</Link>} />
      <div className="divide-y divide-[#eff2f5]">
        {exams.map((exam) => (
          <div key={exam.id} className="flex flex-col gap-3 p-5 md:flex-row md:items-center md:justify-between">
            <div>
              <div className="font-extrabold text-[#181c32]">{exam.name}</div>
              <div className="mt-1 text-xs font-semibold text-[#a1a5b7]">
                {exam.code} · Kode undangan {exam.invitation_code || '-'}
              </div>
            </div>
            <StatusBadge status={exam.status} />
          </div>
        ))}
        {exams.length === 0 ? <div className="p-10 text-center font-semibold text-[#7e8299]">Belum ada ujian yang bisa dikerjakan.</div> : null}
      </div>
    </section>
  );
}

function RecentResultPanel({ results }) {
  return (
    <section className="panel overflow-hidden">
      <PanelHeader eyebrow="Riwayat Nilai" title="Hasil Terakhir" action={<Link className="btn btn-ghost h-10" to="/student/history"><FileCheck2 size={17} /> History</Link>} />
      <div className="divide-y divide-[#eff2f5]">
        {results.map((item) => (
          <Link key={item.session_id} to={`/student/history/${item.session_id}`} className="block p-5 hover:bg-[#f9fafb]">
            <div className="flex items-start justify-between gap-3">
              <div>
                <div className="font-extrabold text-[#181c32]">{item.exam_name}</div>
                <div className="mt-1 text-xs font-semibold text-[#a1a5b7]">{formatDate(item.submitted_at)}</div>
              </div>
              <ResultBadge passed={item.passed} status={item.status} />
            </div>
            <div className="mt-4 flex items-center justify-between rounded-lg bg-[#f5f8fa] px-4 py-3">
              <span className="text-xs font-bold uppercase text-[#a1a5b7]">Skor</span>
              <span className="font-extrabold text-[#181c32]">{formatNumber(item.score)} / {formatNumber(item.max_score)} · {formatNumber(item.percentage)}%</span>
            </div>
          </Link>
        ))}
        {results.length === 0 ? <div className="p-10 text-center font-semibold text-[#7e8299]">Belum ada hasil ujian.</div> : null}
      </div>
    </section>
  );
}

function ActivityPanel({ activities }) {
  return (
    <section className="panel overflow-hidden">
      <PanelHeader eyebrow="Anti Cheat" title="Aktivitas Terbaru" />
      <div className="divide-y divide-[#eff2f5]">
        {activities.map((item) => (
          <div key={item.id} className="p-5">
            <div className="flex items-start justify-between gap-3">
              <div>
                <div className="font-extrabold text-[#181c32]">{item.title}</div>
                <div className="mt-1 text-xs font-semibold text-[#a1a5b7]">{formatDate(item.created_at)}</div>
              </div>
              <SeverityBadge severity={item.severity} />
            </div>
            <div className="mt-3 text-sm font-semibold text-[#7e8299]">Score {formatNumber(item.score || 0)}</div>
          </div>
        ))}
        {activities.length === 0 ? <div className="p-10 text-center font-semibold text-[#7e8299]">Belum ada aktivitas proctoring.</div> : null}
      </div>
    </section>
  );
}

function TrendPanel({ trends }) {
  const max = Math.max(1, ...trends.map((item) => Number(item.value || 0)));
  return (
    <section className="panel p-5">
      <PanelHeader eyebrow="7 Hari Terakhir" title="Trend Sesi Ujian" flush />
      <div className="mt-6 flex h-44 items-end gap-3">
        {trends.map((item) => (
          <div key={item.date} className="flex min-w-0 flex-1 flex-col items-center gap-2">
            <div className="w-full rounded-t-lg bg-[#009ef7]" style={{ height: `${Math.max(8, (Number(item.value || 0) / max) * 150)}px` }} />
            <div className="truncate text-[11px] font-bold text-[#a1a5b7]">{shortDate(item.date)}</div>
          </div>
        ))}
      </div>
      {trends.length === 0 ? <div className="py-10 text-center font-semibold text-[#7e8299]">Belum ada trend sesi.</div> : null}
    </section>
  );
}

function HealthPanel({ health }) {
  const entries = Object.entries(health);
  return (
    <section className="panel p-5">
      <PanelHeader eyebrow="Infrastructure" title="Service Health" flush />
      <div className="mt-4 grid gap-3">
        {entries.map(([key, value]) => (
          <div key={key} className="flex items-center justify-between rounded-lg bg-[#f5f8fa] px-4 py-3">
            <span className="inline-flex items-center gap-2 text-sm font-extrabold capitalize text-[#181c32]">
              <Database size={16} />
              {key}
            </span>
            <HealthBadge value={value} />
          </div>
        ))}
      </div>
    </section>
  );
}

function PanelHeader({ eyebrow, title, action, flush = false }) {
  return (
    <div className={flush ? 'flex items-center justify-between gap-3' : 'flex items-center justify-between gap-3 border-b border-[#eff2f5] p-5'}>
      <div>
        <div className="text-xs font-bold uppercase text-[#a1a5b7]">{eyebrow}</div>
        <h3 className="mt-1 text-lg font-extrabold text-[#181c32]">{title}</h3>
      </div>
      {action}
    </div>
  );
}

function StatusBadge({ status }) {
  const active = status === 'published' || status === 'active';
  return <span className={active ? 'inline-flex rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold capitalize text-[#50cd89]' : 'inline-flex rounded-md bg-[#f1faff] px-2.5 py-1 text-xs font-extrabold capitalize text-[#009ef7]'}>{status || '-'}</span>;
}

function ResultBadge({ passed, status }) {
  if (status !== 'completed') return <StatusBadge status={status} />;
  return <span className={passed ? 'inline-flex rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold text-[#50cd89]' : 'inline-flex rounded-md bg-[#fff5f8] px-2.5 py-1 text-xs font-extrabold text-[#f1416c]'}>{passed ? 'Lulus' : 'Tidak Lulus'}</span>;
}

function SeverityBadge({ severity }) {
  const tone = severity === 'critical' || severity === 'high' ? 'bg-[#fff5f8] text-[#f1416c]' : severity === 'medium' ? 'bg-[#fff8dd] text-[#a46a00]' : 'bg-[#e8fff3] text-[#50cd89]';
  return <span className={`inline-flex rounded-md px-2.5 py-1 text-xs font-extrabold capitalize ${tone}`}>{severity || 'low'}</span>;
}

function HealthBadge({ value }) {
  const healthy = value === 'healthy' || value === 'configured';
  return <span className={healthy ? 'inline-flex rounded-md bg-[#e8fff3] px-2.5 py-1 text-xs font-extrabold capitalize text-[#50cd89]' : 'inline-flex rounded-md bg-[#fff5f8] px-2.5 py-1 text-xs font-extrabold capitalize text-[#f1416c]'}>{value}</span>;
}

function EmptyRow({ colSpan, text }) {
  return <tr><td className="px-5 py-12 text-center font-semibold text-[#7e8299]" colSpan={colSpan}>{text}</td></tr>;
}

function toneClass(tone) {
  const map = {
    primary: { iconBg: 'bg-[#f1faff]', iconText: 'text-[#009ef7]', badge: 'bg-[#f1faff] text-[#009ef7]' },
    success: { iconBg: 'bg-[#e8fff3]', iconText: 'text-[#50cd89]', badge: 'bg-[#e8fff3] text-[#50cd89]' },
    warning: { iconBg: 'bg-[#fff8dd]', iconText: 'text-[#ffc700]', badge: 'bg-[#fff8dd] text-[#a46a00]' },
    danger: { iconBg: 'bg-[#fff5f8]', iconText: 'text-[#f1416c]', badge: 'bg-[#fff5f8] text-[#f1416c]' },
    muted: { iconBg: 'bg-[#f5f8fa]', iconText: 'text-[#7e8299]', badge: 'bg-[#f5f8fa] text-[#7e8299]' },
  };
  return map[tone] || { iconBg: 'bg-[#f1faff]', iconText: 'text-[#009ef7]', badge: 'bg-[#f1faff] text-[#009ef7]' };
}

function skeletonMetrics() {
  return ['a', 'b', 'c', 'd'].map((key) => ({ key, label: 'Memuat', display: '', tone: 'muted' }));
}

function formatNumber(value) {
  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(Number(value || 0));
}

function formatDate(value) {
  if (!value) return '-';
  return new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value));
}

function shortDate(value) {
  if (!value) return '-';
  return new Intl.DateTimeFormat('id-ID', { day: '2-digit', month: 'short' }).format(new Date(value));
}
