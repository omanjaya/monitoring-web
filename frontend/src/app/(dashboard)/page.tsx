"use client"

import { useQuery } from "@tanstack/react-query"
import { useRouter } from "next/navigation"
import {
  Globe,
  ArrowUpRight,
  ArrowDownRight,
  Bell,
  TrendingUp,
  AlertTriangle,
  RefreshCw,
  CheckCircle2,
  XCircle,
  MinusCircle,
  HelpCircle,
  Clock,
} from "lucide-react"
import {
  AreaChart,
  Area,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"

import {
  dashboardApi,
  alertApi,
  type DashboardOverview,
  type DashboardTrends,
  type AlertSummary,
} from "@/lib/api"
import { formatRelativeTime, getSeverityColor } from "@/lib/utils"

const REFRESH_INTERVAL = 60_000

export default function DashboardPage() {
  const router = useRouter()

  const overviewQuery = useQuery<DashboardOverview>({
    queryKey: ["dashboard-overview"],
    queryFn: async () => {
      const res = await dashboardApi.overview()
      return res.data
    },
    refetchInterval: REFRESH_INTERVAL,
  })

  const trendsQuery = useQuery<DashboardTrends>({
    queryKey: ["dashboard-trends"],
    queryFn: async () => {
      const res = await dashboardApi.trends()
      return res.data
    },
    refetchInterval: REFRESH_INTERVAL,
  })

  const alertQuery = useQuery<AlertSummary>({
    queryKey: ["alert-summary"],
    queryFn: async () => {
      const res = await alertApi.summary()
      return res.data
    },
    refetchInterval: REFRESH_INTERVAL,
  })

  const isLoading = overviewQuery.isLoading || trendsQuery.isLoading
  const isRefreshing = overviewQuery.isFetching && !overviewQuery.isLoading
  const error = overviewQuery.error || trendsQuery.error

  const handleRefresh = () => {
    overviewQuery.refetch()
    trendsQuery.refetch()
    alertQuery.refetch()
  }

  if (isLoading) return <DashboardSkeleton />

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-4 py-24">
        <div className="flex h-14 w-14 items-center justify-center rounded-full bg-destructive/10">
          <AlertTriangle className="h-7 w-7 text-destructive" />
        </div>
        <div className="text-center">
          <h2 className="text-base font-semibold">Gagal Memuat Dashboard</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {error instanceof Error ? error.message : "Terjadi kesalahan"}
          </p>
        </div>
        <button
          onClick={handleRefresh}
          className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
        >
          <RefreshCw className="h-3.5 w-3.5" />
          Coba Lagi
        </button>
      </div>
    )
  }

  const overview = overviewQuery.data
  const trends = trendsQuery.data
  const alertSummary = alertQuery.data

  if (!overview || !trends) return null

  const { stats, recent_alerts, status_distribution } = overview

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Dashboard</h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            Ringkasan monitoring website pemerintah
          </p>
        </div>
        <button
          onClick={handleRefresh}
          disabled={isRefreshing}
          className="inline-flex items-center gap-1.5 rounded-lg border border-border/60 bg-card px-3 py-1.5 text-xs font-medium text-muted-foreground hover:text-foreground hover:bg-accent transition-colors disabled:opacity-50"
        >
          <RefreshCw className={`h-3 w-3 ${isRefreshing ? "animate-spin" : ""}`} />
          Refresh
        </button>
      </div>

      {/* Stats Row */}
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-5">
        <MetricCard
          label="Total Website"
          value={stats.total_websites}
          icon={<Globe className="h-4 w-4" />}
          iconColor="text-primary bg-primary/10"
        />
        <MetricCard
          label="Website Up"
          value={stats.total_up}
          icon={<ArrowUpRight className="h-4 w-4" />}
          iconColor="text-emerald-600 bg-emerald-500/10 dark:text-emerald-400"
          valueColor="text-emerald-600 dark:text-emerald-400"
          sub={stats.total_websites > 0 ? `${((stats.total_up / stats.total_websites) * 100).toFixed(0)}%` : undefined}
        />
        <MetricCard
          label="Website Down"
          value={stats.total_down}
          icon={<ArrowDownRight className="h-4 w-4" />}
          iconColor="text-red-600 bg-red-500/10 dark:text-red-400"
          valueColor={stats.total_down > 0 ? "text-red-600 dark:text-red-400" : undefined}
          sub={stats.total_down > 0 ? "Perlu perhatian" : "Semua baik"}
        />
        <MetricCard
          label="Alert Aktif"
          value={alertSummary?.total_active ?? 0}
          icon={<Bell className="h-4 w-4" />}
          iconColor="text-amber-600 bg-amber-500/10 dark:text-amber-400"
          badge={
            alertSummary && alertSummary.critical > 0 ? (
              <span className="inline-flex items-center rounded-full bg-red-500/10 px-1.5 py-0.5 text-[10px] font-semibold text-red-600 dark:text-red-400">
                {alertSummary.critical} kritis
              </span>
            ) : undefined
          }
        />
        <MetricCard
          label="Uptime"
          value={`${stats.overall_uptime.toFixed(1)}%`}
          icon={<TrendingUp className="h-4 w-4" />}
          iconColor="text-primary bg-primary/10"
          sub={`${stats.avg_response_time.toFixed(0)}ms avg`}
        />
      </div>

      {/* Status Bars */}
      <Card className="border-border/50">
        <CardContent className="py-4 px-5">
          <div className="flex items-center justify-between mb-3">
            <p className="text-sm font-medium">Status Website</p>
            <p className="text-xs text-muted-foreground">{stats.total_websites} total</p>
          </div>
          {stats.total_websites > 0 ? (
            <>
              <div className="flex h-2.5 w-full overflow-hidden rounded-full bg-muted">
                {(status_distribution?.up ?? 0) > 0 && (
                  <div
                    className="bg-emerald-500 transition-all duration-500"
                    style={{ width: `${((status_distribution?.up ?? 0) / stats.total_websites) * 100}%` }}
                  />
                )}
                {(status_distribution?.degraded ?? 0) > 0 && (
                  <div
                    className="bg-amber-500 transition-all duration-500"
                    style={{ width: `${((status_distribution?.degraded ?? 0) / stats.total_websites) * 100}%` }}
                  />
                )}
                {(status_distribution?.down ?? 0) > 0 && (
                  <div
                    className="bg-red-500 transition-all duration-500"
                    style={{ width: `${((status_distribution?.down ?? 0) / stats.total_websites) * 100}%` }}
                  />
                )}
                {(status_distribution?.unknown ?? 0) > 0 && (
                  <div
                    className="bg-muted-foreground/30 transition-all duration-500"
                    style={{ width: `${((status_distribution?.unknown ?? 0) / stats.total_websites) * 100}%` }}
                  />
                )}
              </div>
              <div className="mt-2.5 flex flex-wrap gap-x-5 gap-y-1">
                <StatusLegend icon={<CheckCircle2 className="h-3 w-3 text-emerald-500" />} label="Up" count={status_distribution?.up ?? 0} />
                <StatusLegend icon={<MinusCircle className="h-3 w-3 text-amber-500" />} label="Degraded" count={status_distribution?.degraded ?? 0} />
                <StatusLegend icon={<XCircle className="h-3 w-3 text-red-500" />} label="Down" count={status_distribution?.down ?? 0} />
                <StatusLegend icon={<HelpCircle className="h-3 w-3 text-muted-foreground/50" />} label="Unknown" count={status_distribution?.unknown ?? 0} />
              </div>
            </>
          ) : (
            <p className="text-xs text-muted-foreground">Belum ada website yang dimonitor</p>
          )}
        </CardContent>
      </Card>

      {/* Charts */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card className="border-border/50">
          <CardHeader className="pb-2 px-5 pt-4">
            <CardTitle className="text-sm font-medium">Response Time</CardTitle>
            <p className="text-xs text-muted-foreground">Rata-rata harian (ms)</p>
          </CardHeader>
          <CardContent className="px-5 pb-4">
            {trends.response_times && trends.response_times.length > 0 ? (
              <ResponsiveContainer width="100%" height={220}>
                <AreaChart data={trends.response_times}>
                  <defs>
                    <linearGradient id="responseGrad" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="oklch(0.55 0.20 260)" stopOpacity={0.2} />
                      <stop offset="100%" stopColor="oklch(0.55 0.20 260)" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="oklch(0.80 0 0 / 0.15)" vertical={false} />
                  <XAxis
                    dataKey="date"
                    tick={{ fontSize: 11, fill: "oklch(0.50 0 0)" }}
                    tickLine={false}
                    axisLine={false}
                    tickFormatter={(v) => { const d = new Date(v); return `${d.getDate()}/${d.getMonth() + 1}` }}
                  />
                  <YAxis
                    tick={{ fontSize: 11, fill: "oklch(0.50 0 0)" }}
                    tickLine={false}
                    axisLine={false}
                    tickFormatter={(v) => `${v}`}
                    width={40}
                  />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: "var(--card)",
                      borderColor: "var(--border)",
                      borderRadius: "8px",
                      fontSize: "12px",
                      boxShadow: "0 4px 12px rgba(0,0,0,0.08)",
                    }}
                    formatter={(value) => [`${Number(value).toFixed(0)}ms`, "Response Time"]}
                    labelFormatter={(label) => new Date(label).toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" })}
                  />
                  <Area
                    type="monotone"
                    dataKey="avg"
                    stroke="oklch(0.55 0.20 260)"
                    strokeWidth={2}
                    fill="url(#responseGrad)"
                    dot={false}
                    activeDot={{ r: 4, strokeWidth: 0 }}
                  />
                </AreaChart>
              </ResponsiveContainer>
            ) : (
              <EmptyChart message="Belum ada data response time" />
            )}
          </CardContent>
        </Card>

        <Card className="border-border/50">
          <CardHeader className="pb-2 px-5 pt-4">
            <CardTitle className="text-sm font-medium">Uptime History</CardTitle>
            <p className="text-xs text-muted-foreground">Persentase uptime harian</p>
          </CardHeader>
          <CardContent className="px-5 pb-4">
            {trends.uptime_history && trends.uptime_history.length > 0 ? (
              <ResponsiveContainer width="100%" height={220}>
                <LineChart data={trends.uptime_history}>
                  <CartesianGrid strokeDasharray="3 3" stroke="oklch(0.80 0 0 / 0.15)" vertical={false} />
                  <XAxis
                    dataKey="date"
                    tick={{ fontSize: 11, fill: "oklch(0.50 0 0)" }}
                    tickLine={false}
                    axisLine={false}
                    tickFormatter={(v) => { const d = new Date(v); return `${d.getDate()}/${d.getMonth() + 1}` }}
                  />
                  <YAxis
                    tick={{ fontSize: 11, fill: "oklch(0.50 0 0)" }}
                    tickLine={false}
                    axisLine={false}
                    domain={[90, 100]}
                    tickFormatter={(v) => `${v}%`}
                    width={40}
                  />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: "var(--card)",
                      borderColor: "var(--border)",
                      borderRadius: "8px",
                      fontSize: "12px",
                      boxShadow: "0 4px 12px rgba(0,0,0,0.08)",
                    }}
                    formatter={(value) => [`${Number(value).toFixed(2)}%`, "Uptime"]}
                    labelFormatter={(label) => new Date(label).toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric" })}
                  />
                  <Line
                    type="monotone"
                    dataKey="uptime"
                    stroke="oklch(0.60 0.16 150)"
                    strokeWidth={2}
                    dot={false}
                    activeDot={{ r: 4, strokeWidth: 0, fill: "oklch(0.60 0.16 150)" }}
                  />
                </LineChart>
              </ResponsiveContainer>
            ) : (
              <EmptyChart message="Belum ada data uptime" />
            )}
          </CardContent>
        </Card>
      </div>

      {/* Recent Alerts */}
      <Card className="border-border/50">
        <CardHeader className="flex flex-row items-center justify-between pb-3 px-5 pt-4">
          <div>
            <CardTitle className="text-sm font-medium">Alert Terbaru</CardTitle>
            <p className="text-xs text-muted-foreground mt-0.5">Alert aktif yang perlu ditindaklanjuti</p>
          </div>
          <button
            onClick={() => router.push("/alerts")}
            className="text-xs font-medium text-primary hover:text-primary/80 transition-colors"
          >
            Lihat semua
          </button>
        </CardHeader>
        <CardContent className="px-5 pb-4">
          {recent_alerts && recent_alerts.length > 0 ? (
            <div className="space-y-2">
              {recent_alerts.slice(0, 8).map((alert) => (
                <div
                  key={alert.id}
                  className="flex items-start gap-3 rounded-lg border border-border/40 bg-card px-3.5 py-2.5 transition-colors hover:bg-accent/30 cursor-pointer"
                  onClick={() => router.push("/alerts")}
                >
                  <Badge variant="outline" className={`shrink-0 text-[10px] px-1.5 py-0 mt-0.5 ${getSeverityColor(alert.severity)}`}>
                    {alert.severity}
                  </Badge>
                  <div className="flex-1 min-w-0">
                    <p className="text-[13px] font-medium leading-tight truncate">{alert.title}</p>
                    <p className="text-xs text-muted-foreground mt-0.5 truncate">
                      {alert.website_name || `Website #${alert.website_id}`}
                      <span className="mx-1.5 text-border">|</span>
                      {alert.type}
                    </p>
                  </div>
                  <div className="flex items-center gap-1 shrink-0 text-muted-foreground">
                    <Clock className="h-3 w-3" />
                    <span className="text-[11px]">{formatRelativeTime(alert.created_at)}</span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center py-10 text-muted-foreground">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted mb-2">
                <Bell className="h-4 w-4" />
              </div>
              <p className="text-sm font-medium">Tidak ada alert aktif</p>
              <p className="text-xs mt-0.5">Semua sistem berjalan normal</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function MetricCard({ label, value, icon, iconColor, valueColor, sub, badge }: {
  label: string
  value: number | string
  icon: React.ReactNode
  iconColor: string
  valueColor?: string
  sub?: string
  badge?: React.ReactNode
}) {
  return (
    <Card className="border-border/50">
      <CardContent className="p-4">
        <div className="flex items-center justify-between mb-2.5">
          <span className="text-xs font-medium text-muted-foreground">{label}</span>
          <div className={`flex h-7 w-7 items-center justify-center rounded-lg ${iconColor}`}>
            {icon}
          </div>
        </div>
        <div className="flex items-end gap-2">
          <span className={`text-2xl font-bold leading-none tracking-tight ${valueColor ?? ""}`}>
            {value}
          </span>
          {badge}
        </div>
        {sub && (
          <p className="text-[11px] text-muted-foreground mt-1">{sub}</p>
        )}
      </CardContent>
    </Card>
  )
}

function StatusLegend({ icon, label, count }: { icon: React.ReactNode; label: string; count: number }) {
  return (
    <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
      {icon}
      <span>{label}</span>
      <span className="font-semibold text-foreground">{count}</span>
    </div>
  )
}

function EmptyChart({ message }: { message: string }) {
  return (
    <div className="flex h-[220px] items-center justify-center">
      <p className="text-xs text-muted-foreground">{message}</p>
    </div>
  )
}

function DashboardSkeleton() {
  return (
    <div className="space-y-6">
      <div>
        <Skeleton className="h-6 w-32 mb-1.5" />
        <Skeleton className="h-4 w-56" />
      </div>
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-5">
        {Array.from({ length: 5 }).map((_, i) => (
          <Card key={i} className="border-border/50">
            <CardContent className="p-4">
              <div className="flex items-center justify-between mb-2.5">
                <Skeleton className="h-3 w-16" />
                <Skeleton className="h-7 w-7 rounded-lg" />
              </div>
              <Skeleton className="h-7 w-12 mb-1" />
              <Skeleton className="h-3 w-20" />
            </CardContent>
          </Card>
        ))}
      </div>
      <Card className="border-border/50">
        <CardContent className="py-4 px-5">
          <Skeleton className="h-4 w-28 mb-3" />
          <Skeleton className="h-2.5 w-full rounded-full" />
        </CardContent>
      </Card>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {Array.from({ length: 2 }).map((_, i) => (
          <Card key={i} className="border-border/50">
            <CardHeader className="pb-2 px-5 pt-4">
              <Skeleton className="h-4 w-32" />
              <Skeleton className="h-3 w-44" />
            </CardHeader>
            <CardContent className="px-5 pb-4">
              <Skeleton className="h-[220px] w-full" />
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
