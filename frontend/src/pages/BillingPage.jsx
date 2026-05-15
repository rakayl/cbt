import { useQuery } from '@tanstack/react-query';
import { CreditCard } from 'lucide-react';
import { api } from '../lib/api';

const fallback = {
  plan_code: 'FREE',
  plan_name: 'Free',
  students: 0,
  max_students: 100,
  active_exam_sessions: 0,
  max_concurrent_exams: 10,
  student_quota_percent: 0,
  exam_quota_percent: 0,
  features: {},
};

export default function BillingPage() {
  const { data } = useQuery({
    queryKey: ['billing-usage'],
    queryFn: async () => (await api.get('/billing/usage')).data.data,
    retry: false,
  });
  const usage = data || fallback;
  const plans = ['FREE', 'BASIC', 'PRO', 'ENT'];

  async function createCheckoutIntent(planCode) {
    await api.post('/billing/checkout-intent', { plan_code: planCode, billing_email: 'billing@example.edu' }).catch(() => null);
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <CreditCard size={24} />
        <h1 className="text-2xl font-bold">SaaS Billing</h1>
      </div>
      <section className="grid-auto">
        <div className="panel p-5">
          <div className="text-sm text-slate-500">Current Plan</div>
          <div className="text-3xl font-bold mt-2">{usage.plan_name}</div>
          <div className="mt-1 text-sm">{usage.plan_code}</div>
        </div>
        <QuotaCard title="Students" used={usage.students} limit={usage.max_students} percent={usage.student_quota_percent} />
        <QuotaCard title="Concurrent Exams" used={usage.active_exam_sessions} limit={usage.max_concurrent_exams} percent={usage.exam_quota_percent} />
      </section>
      <section className="grid-auto">
        {plans.map((plan) => (
          <div className="panel p-5 space-y-4" key={plan}>
            <div>
              <div className="font-bold">{plan}</div>
              <div className="text-sm text-slate-500">Payment integration ready</div>
            </div>
            <button className="btn btn-primary" onClick={() => createCheckoutIntent(plan)}>Choose Plan</button>
          </div>
        ))}
      </section>
    </div>
  );
}

function QuotaCard({ title, used, limit, percent }) {
  const width = Math.min(100, Math.round(percent || 0));
  return (
    <div className="panel p-5">
      <div className="text-sm text-slate-500">{title}</div>
      <div className="text-3xl font-bold mt-2">{used}/{limit}</div>
      <div className="h-2 bg-line rounded mt-4 overflow-hidden">
        <div className="h-full bg-accent" style={{ width: `${width}%` }} />
      </div>
    </div>
  );
}
