import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { useNavigate } from 'react-router-dom';
import { ArrowRight, ShieldCheck } from 'lucide-react';
import { api } from '../lib/api';
import { useAuthStore } from '../stores/authStore';

const schema = z.object({
  email: z.string().email(),
  password: z.string().min(6),
});

export default function LoginPage() {
  const navigate = useNavigate();
  const setSession = useAuthStore((state) => state.setSession);
  const [errorMessage, setErrorMessage] = useState('');
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm({
    resolver: zodResolver(schema),
    defaultValues: { email: 'admin@example.edu', password: 'ChangeMe123!' },
  });

  async function submit(values) {
    setErrorMessage('');
    try {
      const { data } = await api.post('/auth/login', {
        ...values,
        device_name: navigator.userAgent,
        fingerprint: navigator.userAgent,
      });
      setSession(data.data);
      navigate('/');
    } catch (error) {
      setErrorMessage(error.response?.data?.message || error.message || 'Login gagal.');
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center p-6">
      <div className="grid w-full max-w-6xl gap-8 lg:grid-cols-[1.15fr_0.85fr]">
        <section className="rounded-[2rem] border border-slate-800 bg-slate-950 p-8 text-white shadow-panel md:p-10">
          <div className="inline-flex items-center gap-2 rounded-2xl border border-cyan-300/20 bg-cyan-300/10 px-4 py-2 text-sm font-bold text-cyan-200">
            <ShieldCheck size={18} />
            Enterprise CBT SaaS
          </div>
          <h1 className="mt-8 max-w-2xl text-4xl font-extrabold leading-tight md:text-5xl">
            Command center ujian kampus untuk tenant, proctoring, recovery, dan realtime monitoring.
          </h1>
          <p className="mt-6 max-w-xl text-sm leading-6 text-slate-300">
            Template dashboard mengikuti gaya ERP: fokus pada operasional, panel padat, navigasi jelas, dan kontrol enterprise yang siap dipakai harian.
          </p>
          <div className="mt-8 grid gap-3 sm:grid-cols-3">
            {['Server timer', 'RBAC SaaS', 'Recovery ready'].map((item) => (
              <div key={item} className="rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-sm font-bold">
                {item}
              </div>
            ))}
          </div>
        </section>

        <form onSubmit={handleSubmit(submit)} className="panel p-8 md:p-9">
          <p className="text-xs font-bold uppercase tracking-[0.32em] text-cyan-600">Secure Access</p>
          <h2 className="mt-3 text-3xl font-extrabold">Masuk CBT Kampus</h2>
          <p className="mt-2 text-sm text-slate-500">Gunakan akun enterprise untuk membuka dashboard operasional.</p>

          {errorMessage ? (
            <div className="mt-6 rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm font-semibold text-rose-700">
              {errorMessage}
            </div>
          ) : null}

          <div className="mt-8 space-y-4">
            <div>
              <label className="mb-2 block text-sm font-bold">Email</label>
              <input className="input" placeholder="admin@example.edu" {...register('email')} />
              {errors.email ? <p className="mt-2 text-sm font-semibold text-danger">{errors.email.message}</p> : null}
            </div>
            <div>
              <label className="mb-2 block text-sm font-bold">Password</label>
              <input className="input" type="password" placeholder="Password" {...register('password')} />
              {errors.password ? <p className="mt-2 text-sm font-semibold text-danger">{errors.password.message}</p> : null}
            </div>
            <button className="btn btn-primary w-full justify-center" disabled={isSubmitting}>
              {isSubmitting ? 'Memproses...' : 'Access Dashboard'}
              <ArrowRight size={18} />
            </button>
          </div>
        </form>
      </div>
    </main>
  );
}
