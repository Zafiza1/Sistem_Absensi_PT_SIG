"use client";

import { useState, type FormEvent } from "react";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { Eye, EyeOff, ShieldCheck, Users, ClipboardCheck } from "lucide-react";

import { useAuth } from "@/lib/auth-context";
import { ApiError } from "@/lib/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const HIGHLIGHTS = [
  { icon: Users, text: "Kelola data karyawan, divisi, dan jabatan" },
  { icon: ClipboardCheck, text: "Pantau kehadiran dan laporan secara real-time" },
  { icon: ShieldCheck, text: "Akses berbasis peran — Admin, HR, dan Management" },
];

export default function LoginPage() {
  const { login } = useAuth();
  const router = useRouter();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await login(email, password);
      router.push("/dashboard");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Terjadi kesalahan. Silakan coba lagi.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="grid min-h-screen lg:grid-cols-2">
      {/* Brand panel — hidden on small screens, the form alone is the mobile experience. */}
      <div className="relative hidden flex-col justify-between overflow-hidden bg-[#0f172a] p-10 text-white lg:flex">
        <div
          className="pointer-events-none absolute inset-0 opacity-40"
          style={{
            backgroundImage:
              "radial-gradient(circle at 15% 20%, rgba(56,189,248,0.25), transparent 45%), radial-gradient(circle at 85% 80%, rgba(56,189,248,0.15), transparent 40%)",
          }}
          aria-hidden
        />
        <div className="relative flex items-center gap-2.5">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-white">
            <Image src="/logo.png" alt="" width={26} height={17} aria-hidden />
          </div>
          <span className="text-sm font-semibold">PT Surya Inti Gas</span>
        </div>

        <div className="relative max-w-md">
          <h1 className="text-3xl font-semibold tracking-tight text-balance">Sistem Absensi Digital</h1>
          <p className="mt-3 text-sm leading-relaxed text-slate-300">
            Dashboard terpadu untuk mengelola karyawan, jadwal kerja, perangkat tablet absensi, dan laporan
            kehadiran perusahaan.
          </p>
          <ul className="mt-8 space-y-4">
            {HIGHLIGHTS.map(({ icon: Icon, text }) => (
              <li key={text} className="flex items-start gap-3 text-sm text-slate-300">
                <span className="mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-md bg-white/10">
                  <Icon className="size-3.5 text-sky-400" />
                </span>
                {text}
              </li>
            ))}
          </ul>
        </div>

        <p className="relative text-xs text-slate-500">© {new Date().getFullYear()} PT Surya Inti Gas</p>
      </div>

      {/* Form */}
      <div className="flex items-center justify-center bg-background px-6 py-12">
        <div className="w-full max-w-sm">
          <div className="mb-8 flex flex-col items-center gap-3 text-center lg:hidden">
            <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-white shadow-sm ring-1 ring-border">
              <Image src="/logo.png" alt="PT Surya Inti Gas" width={40} height={26} priority />
            </div>
            <div>
              <h1 className="text-lg font-semibold">PT SURYA INTI GAS</h1>
              <p className="text-sm text-muted-foreground">Sistem Absensi Digital</p>
            </div>
          </div>

          <div className="mb-6 hidden lg:block">
            <h2 className="text-xl font-semibold tracking-tight">Masuk ke akun Anda</h2>
            <p className="mt-1 text-sm text-muted-foreground">Gunakan email dan password yang terdaftar.</p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                autoComplete="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="nama@suryaintigas.com"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  className="absolute top-1/2 right-0 flex h-8 w-9 -translate-y-1/2 items-center justify-center text-muted-foreground hover:text-foreground"
                  aria-label={showPassword ? "Sembunyikan password" : "Tampilkan password"}
                >
                  {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                </button>
              </div>
            </div>
            {error && (
              <p role="alert" className="rounded-md bg-destructive/10 px-3 py-2 text-sm font-medium text-destructive">
                {error}
              </p>
            )}
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? "Memproses..." : "Masuk"}
            </Button>
          </form>
        </div>
      </div>
    </div>
  );
}
