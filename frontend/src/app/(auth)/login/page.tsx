"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { Activity, Loader2, Eye, EyeOff } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { useAuthStore } from "@/stores/auth"

export default function LoginPage() {
  const router = useRouter()
  const { login } = useAuthStore()
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError("")
    setLoading(true)

    try {
      await login(username, password)
      router.replace("/")
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "Login gagal. Silakan coba lagi."
      setError(message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="w-full max-w-sm mx-auto">
      {/* Logo & branding */}
      <div className="flex flex-col items-center mb-8">
        <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary text-primary-foreground mb-4">
          <Activity className="h-6 w-6" />
        </div>
        <h1 className="text-xl font-semibold tracking-tight">Monitoring Website</h1>
        <p className="text-sm text-muted-foreground mt-1">Pemerintah Provinsi Bali</p>
      </div>

      {/* Login form */}
      <div className="rounded-xl border border-border/60 bg-card p-6 shadow-sm">
        <div className="mb-5">
          <h2 className="text-base font-semibold">Masuk</h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            Masuk ke dashboard monitoring website
          </p>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          {error && (
            <div className="flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-900/50 dark:bg-red-950/30">
              <div className="h-4 w-4 shrink-0 rounded-full bg-red-500/15 flex items-center justify-center mt-0.5">
                <span className="text-red-600 dark:text-red-400 text-[10px] font-bold">!</span>
              </div>
              <p className="text-[13px] text-red-700 dark:text-red-300">{error}</p>
            </div>
          )}

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="username" className="text-[13px]">Username</Label>
            <Input
              id="username"
              type="text"
              placeholder="Masukkan username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              autoComplete="username"
              autoFocus
              disabled={loading}
              className="h-10"
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="password" className="text-[13px]">Password</Label>
            <div className="relative">
              <Input
                id="password"
                type={showPassword ? "text" : "password"}
                placeholder="Masukkan password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                autoComplete="current-password"
                disabled={loading}
                className="h-10 pr-10"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                tabIndex={-1}
              >
                {showPassword ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </button>
            </div>
          </div>

          <Button type="submit" className="w-full h-10 mt-1" disabled={loading}>
            {loading ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Masuk...
              </>
            ) : (
              "Masuk"
            )}
          </Button>
        </form>
      </div>

      <p className="text-center text-[11px] text-muted-foreground/60 mt-6">
        Dinas Komunikasi Informatika dan Statistik Provinsi Bali
      </p>
    </div>
  )
}
