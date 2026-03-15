"use client"

import { useQuery } from "@tanstack/react-query"
import {
  AlertCircle,
  CheckCircle2,
  Clock,
  Database,
  Globe,
  Loader2,
  Lock,
  Play,
  Radar,
  RefreshCw,
  Search,
  Server,
  Shield,
  ShieldAlert,
  Bug,
  FileWarning,
} from "lucide-react"

import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import { adminApi, type SystemStatus } from "@/lib/api"
import { useMutationAction } from "@/hooks/use-mutation-action"

const CHECK_TYPES = [
  { type: "uptime", label: "Uptime Check", icon: Globe, description: "Cek status up/down semua website", color: "text-emerald-500", bg: "bg-emerald-500/10" },
  { type: "ssl", label: "SSL Check", icon: Lock, description: "Cek validitas sertifikat SSL", color: "text-blue-500", bg: "bg-blue-500/10" },
  { type: "content", label: "Content Scan", icon: Search, description: "Scan konten gambling & judi", color: "text-amber-500", bg: "bg-amber-500/10" },
  { type: "dork", label: "Dork Scan", icon: Radar, description: "Scan Google dork patterns", color: "text-purple-500", bg: "bg-purple-500/10" },
  { type: "vulnerability", label: "Vulnerability Scan", icon: Bug, description: "Scan kerentanan website", color: "text-red-500", bg: "bg-red-500/10" },
  { type: "dns", label: "DNS Check", icon: Database, description: "Cek DNS records & subdomain", color: "text-cyan-500", bg: "bg-cyan-500/10" },
  { type: "security", label: "Security Headers", icon: Shield, description: "Cek security headers", color: "text-orange-500", bg: "bg-orange-500/10" },
  { type: "defacement", label: "Defacement Scan", icon: FileWarning, description: "Deteksi perubahan tampilan", color: "text-pink-500", bg: "bg-pink-500/10" },
] as const

function TriggerButton({ type, label, icon: Icon, description, color, bg }: typeof CHECK_TYPES[number]) {
  const mutation = useMutationAction({
    mutationFn: () => adminApi.trigger(type),
    successMessage: `${label} berhasil di-trigger`,
    errorMessage: `Gagal menjalankan ${label}`,
  })

  return (
    <Card className="border-border/50 hover:border-border transition-colors">
      <CardContent className="px-5 py-4">
        <div className="flex items-start gap-3">
          <div className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg ${bg}`}>
            <Icon className={`h-4.5 w-4.5 ${color}`} />
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-[13px] font-medium">{label}</p>
            <p className="text-xs text-muted-foreground mt-0.5">{description}</p>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => mutation.mutate()}
            disabled={mutation.isPending}
            className="h-8 text-xs shrink-0"
          >
            {mutation.isPending ? (
              <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
            ) : (
              <Play className="mr-1.5 h-3.5 w-3.5" />
            )}
            Trigger
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

function formatLastCheck(dateStr?: string): string {
  if (!dateStr || dateStr === "0001-01-01T00:00:00Z") return "Belum pernah"
  const date = new Date(dateStr)
  if (isNaN(date.getTime())) return "Belum pernah"
  const now = new Date()
  const diff = Math.floor((now.getTime() - date.getTime()) / 1000)
  if (diff < 60) return `${diff} detik lalu`
  if (diff < 3600) return `${Math.floor(diff / 60)} menit lalu`
  if (diff < 86400) return `${Math.floor(diff / 3600)} jam lalu`
  return `${Math.floor(diff / 86400)} hari lalu`
}

export default function AdminPage() {
  const statusQuery = useQuery({
    queryKey: ["admin-status"],
    queryFn: () => adminApi.status(),
    refetchInterval: 30000,
  })

  const status = statusQuery.data?.data as SystemStatus | null
  const isLoading = statusQuery.isLoading
  const error = statusQuery.error

  if (isLoading) return <AdminSkeleton />

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-20">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-500/10">
          <AlertCircle className="h-6 w-6 text-red-500" />
        </div>
        <h2 className="text-sm font-medium">Gagal Memuat Status Sistem</h2>
        <p className="text-xs text-muted-foreground">
          {error instanceof Error ? error.message : "Gagal memuat data"}
        </p>
        <Button onClick={() => statusQuery.refetch()} variant="outline" className="h-8 text-xs">
          <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
          Coba Lagi
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Admin Panel</h1>
          <p className="text-[13px] text-muted-foreground mt-0.5">
            Status sistem & kontrol monitoring manual
          </p>
        </div>
        <Button
          variant="outline"
          onClick={() => statusQuery.refetch()}
          disabled={statusQuery.isFetching}
          className="h-8 text-xs"
        >
          {statusQuery.isFetching ? (
            <RefreshCw className="mr-1.5 h-3.5 w-3.5 animate-spin" />
          ) : (
            <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
          )}
          Refresh
        </Button>
      </div>

      {/* System Status Cards */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {/* System Status */}
        <Card className="border-border/50 border-l-4 border-l-emerald-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-500/10">
                <Server className="h-4 w-4 text-emerald-500" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">System Status</p>
                <div className="flex items-center gap-1.5 mt-0.5">
                  <Badge variant={status?.status === "ok" ? "default" : "destructive"} className="text-[11px] h-5">
                    {status?.status === "ok" ? "Healthy" : status?.status || "Unknown"}
                  </Badge>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Monitor Status */}
        <Card className="border-border/50 border-l-4 border-l-blue-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-500/10">
                {status?.monitor_running ? (
                  <CheckCircle2 className="h-4 w-4 text-blue-500" />
                ) : (
                  <ShieldAlert className="h-4 w-4 text-red-500" />
                )}
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">Scheduler</p>
                <p className="text-sm font-semibold mt-0.5">
                  {status?.monitor_running ? "Running" : "Stopped"}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Total Websites */}
        <Card className="border-border/50 border-l-4 border-l-purple-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-purple-500/10">
                <Globe className="h-4 w-4 text-purple-500" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">Total Websites</p>
                <p className="text-2xl font-bold">{status?.total_websites ?? 0}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Last Check */}
        <Card className="border-border/50 border-l-4 border-l-amber-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-500/10">
                <Clock className="h-4 w-4 text-amber-500" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">Last Check</p>
                <p className="text-sm font-semibold mt-0.5">
                  {formatLastCheck(status?.last_check)}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* System Info */}
      <Card className="border-border/50">
        <div className="px-5 py-4 border-b border-border/50">
          <h2 className="text-sm font-medium">System Information</h2>
          <p className="text-xs text-muted-foreground mt-0.5">Detail status komponen sistem</p>
        </div>
        <CardContent className="px-5 py-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="flex items-center justify-between py-2 border-b border-border/30">
              <span className="text-xs text-muted-foreground">Version</span>
              <span className="text-xs font-medium">{status?.version || "-"}</span>
            </div>
            <div className="flex items-center justify-between py-2 border-b border-border/30">
              <span className="text-xs text-muted-foreground">Database</span>
              <Badge variant={status?.db_status === "connected" ? "default" : "destructive"} className="text-[11px] h-5">
                {status?.db_status || "Unknown"}
              </Badge>
            </div>
            <div className="flex items-center justify-between py-2 border-b border-border/30">
              <span className="text-xs text-muted-foreground">Telegram</span>
              <Badge variant={status?.telegram_enabled ? "default" : "secondary"} className="text-[11px] h-5">
                {status?.telegram_enabled ? "Enabled" : "Disabled"}
              </Badge>
            </div>
            <div className="flex items-center justify-between py-2 border-b border-border/30">
              <span className="text-xs text-muted-foreground">Server Uptime</span>
              <span className="text-xs font-medium">{status?.server_uptime || "-"}</span>
            </div>
            <div className="flex items-center justify-between py-2 border-b border-border/30">
              <span className="text-xs text-muted-foreground">Scheduler</span>
              <span className="text-xs font-medium">{status?.scheduler || "-"}</span>
            </div>
            {status?.active_jobs && status.active_jobs.length > 0 && (
              <div className="flex items-center justify-between py-2 border-b border-border/30">
                <span className="text-xs text-muted-foreground">Active Jobs</span>
                <span className="text-xs font-medium">{status.active_jobs.length}</span>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Manual Trigger Section */}
      <div>
        <div className="mb-3">
          <h2 className="text-sm font-medium">Manual Trigger</h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            Jalankan pemindaian secara manual untuk setiap jenis pengecekan
          </p>
        </div>
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          {CHECK_TYPES.map((check) => (
            <TriggerButton key={check.type} {...check} />
          ))}
        </div>
      </div>
    </div>
  )
}

function AdminSkeleton() {
  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <Skeleton className="h-6 w-36 mb-1.5" />
          <Skeleton className="h-4 w-56" />
        </div>
        <Skeleton className="h-8 w-24" />
      </div>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Card key={i} className="border-border/50">
            <CardContent className="px-5 py-4">
              <div className="flex items-center gap-3">
                <Skeleton className="h-7 w-7 rounded-lg" />
                <div className="flex-1">
                  <Skeleton className="h-3 w-16 mb-1.5" />
                  <Skeleton className="h-7 w-12" />
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
      <Card className="border-border/50">
        <div className="px-5 py-4 border-b border-border/50">
          <Skeleton className="h-4 w-40 mb-1" />
          <Skeleton className="h-3 w-52" />
        </div>
        <CardContent className="px-5 py-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="flex items-center justify-between py-2">
                <Skeleton className="h-3 w-20" />
                <Skeleton className="h-5 w-16" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
      <div>
        <Skeleton className="h-4 w-28 mb-1" />
        <Skeleton className="h-3 w-64 mb-3" />
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          {Array.from({ length: 8 }).map((_, i) => (
            <Card key={i} className="border-border/50">
              <CardContent className="px-5 py-4">
                <div className="flex items-center gap-3">
                  <Skeleton className="h-9 w-9 rounded-lg" />
                  <div className="flex-1">
                    <Skeleton className="h-3.5 w-28 mb-1" />
                    <Skeleton className="h-3 w-40" />
                  </div>
                  <Skeleton className="h-8 w-20" />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    </div>
  )
}
