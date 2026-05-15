import { useForm } from 'react-hook-form';
import { Link } from 'react-router-dom';
import { ArrowLeft, Building2 } from 'lucide-react';

export default function RegisterPage() {
  const { register, handleSubmit } = useForm();

  return (
    <main className="flex min-h-screen items-center justify-center p-6">
      <div className="grid w-full max-w-5xl gap-8 lg:grid-cols-[0.9fr_1.1fr]">
        <section className="rounded-[2rem] border border-slate-800 bg-slate-950 p-8 text-white shadow-panel md:p-10">
          <div className="grid h-14 w-14 place-items-center rounded-2xl bg-cyan-300 text-slate-950">
            <Building2 size={26} />
          </div>
          <p className="mt-8 text-xs font-bold uppercase tracking-[0.36em] text-cyan-300">Tenant Onboarding</p>
          <h1 className="mt-4 text-4xl font-extrabold leading-tight">Siapkan tenant kampus baru dengan baseline SaaS enterprise.</h1>
          <p className="mt-5 text-sm leading-6 text-slate-300">
            Alur ini disiapkan untuk plan, limit fitur, custom domain, dan strategi database shared atau dedicated.
          </p>
        </section>

        <form onSubmit={handleSubmit(console.log)} className="panel p-8 md:p-9">
          <h2 className="text-3xl font-extrabold">Registrasi Tenant</h2>
          <p className="mt-2 text-sm text-slate-500">Data awal kampus dan administrator tenant.</p>

          <div className="mt-8 grid gap-4 md:grid-cols-2">
            <div className="md:col-span-2">
              <label className="mb-2 block text-sm font-bold">Nama Kampus</label>
              <input className="input" placeholder="Universitas Contoh" {...register('tenant_name')} />
            </div>
            <div>
              <label className="mb-2 block text-sm font-bold">Nama Admin</label>
              <input className="input" placeholder="Nama lengkap" {...register('name')} />
            </div>
            <div>
              <label className="mb-2 block text-sm font-bold">Email Admin</label>
              <input className="input" placeholder="admin@kampus.ac.id" {...register('email')} />
            </div>
            <div className="md:col-span-2">
              <label className="mb-2 block text-sm font-bold">Password</label>
              <input className="input" type="password" placeholder="Password" {...register('password')} />
            </div>
          </div>

          <button className="btn btn-primary mt-6 w-full justify-center">Register Tenant</button>
          <Link to="/login" className="mt-6 inline-flex items-center gap-2 text-sm font-bold text-cyan-700">
            <ArrowLeft size={16} />
            Kembali login
          </Link>
        </form>
      </div>
    </main>
  );
}
