"use client"

import { useEffect, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog"
import { Badge } from "@/components/ui/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { websiteApi, alertApi } from "@/lib/api"
import type { Website, UptimeEntry, Alert, ResponseTimePercentiles } from "@/lib/api"
import { formatRelativeTime, getStatusBgColor, getSeverityColor } from "@/lib/utils"
import {
  Clock,
  Shield,
  ShieldCheck,
  ShieldAlert,
  AlertTriangle,
  Loader2,
  ExternalLink,
  CheckCircle2,
  XCircle,
  RefreshCw,
} from "lucide-react"
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts"

interface WebsiteDetailDialogProps {
  websiteId: number | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

function getStatusDotColor(status: string): string {
  switch (status) {
    case "up":
    case "operational":
      return "bg-green-500"
    case "down":
    case "major_outage":
      return "bg-red-500"
    case "degraded":
    case "partial_outage":
      return "bg-yellow-500"
    default:
      return "bg-gray-400"
  }
}

function getStatusLabel(status: string): string {
  switch (status) {
    case "up":
    case "operational":
      return "Online"
    case "down":
    case "major_outage":
      return "Offline"
    case "degraded":
    case "partial_outage":
      return "Degraded"
    default:
      return "Unknown"
  }
}

function getGradeColor(grade: string): string {
  switch (grade) {
    case "A":
    case "A+":
      return "bg-emerald-500/10 text-emerald-600 border-emerald-500/20"
    case "B":
      return "bg-blue-500/10 text-blue-600 border-blue-500/20"
    case "C":
      return "bg-amber-500/10 text-amber-600 border-amber-500/20"
    case "D":
      return "bg-orange-500/10 text-orange-600 border-orange-500/20"
    default:
      return "bg-red-500/10 text-red-600 border-red-500/20"
  }
}

function getP95Color(value: number | undefined): string {
  if (value == null) return ""
  if (value < 1000) return "text-emerald-600"
  if (value < 3000) return "text-amber-600"
  return "text-red-600"
}

export function WebsiteDetailDialog({
  websiteId,
  open,
  onOpenChange,
}: WebsiteDetailDialogProps) {
  const [website, setWebsite] = useState<Website | null>(null)
  const [uptimeData, setUptimeData] = useState<UptimeEntry[]>([])
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [loading, setLoading] = useState(false)
  const [activeTab, setActiveTab] = useState<"overview" | "performance" | "dns">("overview")

  useEffect(() => {
    if (!websiteId || !open) return

    setActiveTab("overview")

    const fetchData = async () => {
      setLoading(true)
      try {
        const [wsRes, uptimeRes, alertsRes] = await Promise.all([
          websiteApi.get(websiteId),
          websiteApi.uptime(websiteId, 24),
          alertApi.list(`website_id=${websiteId}&limit=5`),
        ])
        setWebsite(wsRes.data)
        setUptimeData(uptimeRes.data || [])
        setAlerts(alertsRes.data || [])
      } catch {
        // silently fail
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [websiteId, open])

  const { data: metrics, isLoading: metricsLoading } = useQuery({
    queryKey: ["website-metrics", websiteId],
    queryFn: async () => {
      const res = await websiteApi.metrics(websiteId!)
      return res.data
    },
    enabled: !!websiteId && open && activeTab === "performance",
  })

  const { data: dnsData, isLoading: dnsLoading, refetch: refetchDns } = useQuery({
    queryKey: ["website-dns", websiteId],
    queryFn: async () => {
      const res = await websiteApi.dns(websiteId!)
      return res.data
    },
    enabled: !!websiteId && open && activeTab === "dns",
  })

  const chartData = uptimeData.map((entry) => ({
    time: new Date(entry.checked_at).toLocaleTimeString("id-ID", {
      hour: "2-digit",
      minute: "2-digit",
    }),
    responseTime: entry.response_time,
    status: entry.status,
  }))

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[700px] max-h-[85vh] overflow-y-auto p-0">
        <DialogHeader className="px-6 pt-6 pb-0">
          <DialogTitle className="text-base">Detail Website</DialogTitle>
          <DialogDescription className="text-[13px]">
            Informasi lengkap dan monitoring website
          </DialogDescription>
        </DialogHeader>

        {loading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 className="size-5 animate-spin text-muted-foreground" />
          </div>
        ) : website ? (
          <div className="px-6 pb-6 space-y-4">
            {/* Tabs */}
            <div className="flex gap-1 border-b border-border/50 mb-4">
              {(["overview", "performance", "dns"] as const).map((tab) => (
                <button
                  key={tab}
                  className={`px-3 py-2 text-xs font-medium border-b-2 transition-colors -mb-px ${
                    activeTab === tab
                      ? "border-primary text-primary"
                      : "border-transparent text-muted-foreground hover:text-foreground"
                  }`}
                  onClick={() => setActiveTab(tab)}
                >
                  {tab === "overview" ? "Overview" : tab === "performance" ? "Performance" : "DNS & Network"}
                </button>
              ))}
            </div>

            {/* Overview Tab */}
            {activeTab === "overview" && (
              <div className="space-y-4">
                {/* Website Info Card */}
                <div className="rounded-lg border border-border/50 p-4">
                  <div className="grid grid-cols-2 gap-x-6 gap-y-3">
                    <div className="space-y-0.5">
                      <p className="text-xs text-muted-foreground">Nama</p>
                      <p className="text-[13px] font-medium">{website.name}</p>
                    </div>
                    <div className="space-y-0.5">
                      <p className="text-xs text-muted-foreground">Status</p>
                      <div className="flex items-center gap-1.5">
                        <span className={`size-2 rounded-full ${getStatusDotColor(website.status)}`} />
                        <span className="text-[13px] font-medium">{getStatusLabel(website.status)}</span>
                      </div>
                    </div>
                    <div className="space-y-0.5">
                      <p className="text-xs text-muted-foreground">URL</p>
                      <a
                        href={website.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-[13px] text-primary hover:underline inline-flex items-center gap-1"
                      >
                        {website.url}
                        <ExternalLink className="size-3 opacity-50" />
                      </a>
                    </div>
                    <div className="space-y-0.5">
                      <p className="text-xs text-muted-foreground">OPD</p>
                      <p className="text-[13px] font-medium">{website.opd_name || "-"}</p>
                    </div>
                    <div className="space-y-0.5">
                      <p className="text-xs text-muted-foreground">Response Time</p>
                      <p className="text-[13px] font-medium">
                        {website.last_response_time
                          ? `${website.last_response_time} ms`
                          : "-"}
                      </p>
                    </div>
                    <div className="space-y-0.5">
                      <p className="text-xs text-muted-foreground">Terakhir Dicek</p>
                      <p className="text-[13px] font-medium inline-flex items-center gap-1">
                        <Clock className="size-3 text-muted-foreground" />
                        {formatRelativeTime(website.last_checked_at)}
                      </p>
                    </div>
                  </div>
                </div>

                {/* Response Time Chart */}
                {chartData.length > 0 && (
                  <div className="rounded-lg border border-border/50 p-4">
                    <h4 className="text-xs font-medium text-muted-foreground mb-3">
                      Response Time (24 Jam Terakhir)
                    </h4>
                    <div className="h-44 w-full">
                      <ResponsiveContainer width="100%" height="100%">
                        <AreaChart data={chartData}>
                          <defs>
                            <linearGradient id="responseTimeGradient" x1="0" y1="0" x2="0" y2="1">
                              <stop offset="0%" stopColor="hsl(var(--primary))" stopOpacity={0.15} />
                              <stop offset="100%" stopColor="hsl(var(--primary))" stopOpacity={0.01} />
                            </linearGradient>
                          </defs>
                          <CartesianGrid
                            strokeDasharray="3 3"
                            stroke="hsl(var(--border))"
                            strokeOpacity={0.5}
                            vertical={false}
                          />
                          <XAxis
                            dataKey="time"
                            tick={{ fontSize: 10, fill: "hsl(var(--muted-foreground))" }}
                            tickLine={false}
                            axisLine={false}
                            interval="preserveStartEnd"
                          />
                          <YAxis
                            tick={{ fontSize: 10, fill: "hsl(var(--muted-foreground))" }}
                            tickLine={false}
                            axisLine={false}
                            unit=" ms"
                            width={60}
                          />
                          <Tooltip
                            contentStyle={{
                              backgroundColor: "hsl(var(--popover))",
                              border: "1px solid hsl(var(--border))",
                              borderRadius: "8px",
                              fontSize: "12px",
                              boxShadow: "0 4px 6px -1px rgb(0 0 0 / 0.1)",
                              padding: "8px 12px",
                            }}
                            formatter={(value) => [`${value} ms`, "Response Time"]}
                          />
                          <Area
                            type="monotone"
                            dataKey="responseTime"
                            stroke="hsl(var(--primary))"
                            fill="url(#responseTimeGradient)"
                            strokeWidth={1.5}
                            dot={false}
                            activeDot={{ r: 3, strokeWidth: 1.5 }}
                          />
                        </AreaChart>
                      </ResponsiveContainer>
                    </div>
                  </div>
                )}

                {/* SSL & Security Row */}
                <div className="grid grid-cols-2 gap-3">
                  {/* SSL Info */}
                  <div className="rounded-lg border border-border/50 p-4">
                    <h4 className="text-xs font-medium text-muted-foreground mb-2.5 flex items-center gap-1.5">
                      <Shield className="size-3.5" />
                      SSL / TLS
                    </h4>
                    <div className="space-y-2">
                      {website.ssl_valid ? (
                        <div className="flex items-center gap-2">
                          <ShieldCheck className="size-4 text-emerald-500" />
                          <span className="text-[13px] font-medium text-emerald-600">Valid</span>
                        </div>
                      ) : website.ssl_valid === false ? (
                        <div className="flex items-center gap-2">
                          <ShieldAlert className="size-4 text-red-500" />
                          <span className="text-[13px] font-medium text-red-600">Invalid</span>
                        </div>
                      ) : (
                        <span className="text-[13px] text-muted-foreground">N/A</span>
                      )}
                      {website.ssl_expiry_date && (
                        <p className="text-xs text-muted-foreground">
                          Expires {new Date(website.ssl_expiry_date).toLocaleDateString("id-ID", {
                            day: "numeric",
                            month: "short",
                            year: "numeric",
                          })}
                        </p>
                      )}
                    </div>
                  </div>

                  {/* Security Score */}
                  <div className="rounded-lg border border-border/50 p-4">
                    <h4 className="text-xs font-medium text-muted-foreground mb-2.5">
                      Security Score
                    </h4>
                    {website.security_score != null ? (
                      <div className="flex items-center gap-3">
                        <div className="flex items-baseline gap-1">
                          <span className="text-xl font-bold tracking-tight">{website.security_score}</span>
                          <span className="text-xs text-muted-foreground">/100</span>
                        </div>
                        {website.security_grade && (
                          <Badge className={`rounded-full px-2 py-0.5 text-[11px] font-medium ${getGradeColor(website.security_grade)}`}>
                            {website.security_grade}
                          </Badge>
                        )}
                      </div>
                    ) : (
                      <span className="text-[13px] text-muted-foreground">N/A</span>
                    )}
                  </div>
                </div>

                {/* Recent Alerts */}
                <div className="rounded-lg border border-border/50 p-4">
                  <h4 className="text-xs font-medium text-muted-foreground mb-2.5 flex items-center gap-1.5">
                    <AlertTriangle className="size-3.5" />
                    Alert Terbaru
                  </h4>
                  {alerts.length === 0 ? (
                    <p className="text-[13px] text-muted-foreground py-2">
                      Tidak ada alert terbaru.
                    </p>
                  ) : (
                    <div className="space-y-1">
                      {alerts.map((alert) => (
                        <div
                          key={alert.id}
                          className="flex items-center justify-between py-2 border-b border-border/40 last:border-0"
                        >
                          <div className="flex items-center gap-2 min-w-0">
                            <Badge className={`rounded-full px-2 py-0.5 text-[11px] font-medium shrink-0 ${getSeverityColor(alert.severity)}`}>
                              {alert.severity}
                            </Badge>
                            <div className="min-w-0">
                              <p className="text-[13px] font-medium truncate">
                                {alert.title}
                              </p>
                              <p className="text-xs text-muted-foreground truncate">
                                {alert.message}
                              </p>
                            </div>
                          </div>
                          <span className="text-[11px] text-muted-foreground whitespace-nowrap ml-3 shrink-0">
                            {formatRelativeTime(alert.created_at)}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Performance Tab */}
            {activeTab === "performance" && (
              <div className="space-y-4">
                {metricsLoading ? (
                  <div className="flex items-center justify-center py-16">
                    <Loader2 className="size-5 animate-spin text-muted-foreground" />
                  </div>
                ) : metrics ? (
                  <div className="rounded-lg border border-border/50 overflow-hidden">
                    <Table>
                      <TableHeader>
                        <TableRow className="hover:bg-transparent">
                          <TableHead className="text-[11px]">Metrik</TableHead>
                          <TableHead className="text-[11px] text-center">24 Jam</TableHead>
                          <TableHead className="text-[11px] text-center">7 Hari</TableHead>
                          <TableHead className="text-[11px] text-center">30 Hari</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {[
                          { label: "P50 (Median)", key: "p50" },
                          { label: "P95", key: "p95" },
                          { label: "P99", key: "p99" },
                          { label: "Rata-rata", key: "avg" },
                          { label: "Minimum", key: "min" },
                          { label: "Maximum", key: "max" },
                        ].map(({ label, key }) => (
                          <TableRow key={key}>
                            <TableCell className="text-xs font-medium">{label}</TableCell>
                            <TableCell className={`text-xs text-center tabular-nums ${key === "p95" ? getP95Color(metrics?.last_24h?.[key as keyof ResponseTimePercentiles] as number) : ""}`}>
                              {metrics?.last_24h?.[key as keyof ResponseTimePercentiles] != null
                                ? `${(metrics.last_24h[key as keyof ResponseTimePercentiles] as number).toFixed(0)}ms`
                                : "-"}
                            </TableCell>
                            <TableCell className={`text-xs text-center tabular-nums ${key === "p95" ? getP95Color(metrics?.last_7d?.[key as keyof ResponseTimePercentiles] as number) : ""}`}>
                              {metrics?.last_7d?.[key as keyof ResponseTimePercentiles] != null
                                ? `${(metrics.last_7d[key as keyof ResponseTimePercentiles] as number).toFixed(0)}ms`
                                : "-"}
                            </TableCell>
                            <TableCell className={`text-xs text-center tabular-nums ${key === "p95" ? getP95Color(metrics?.last_30d?.[key as keyof ResponseTimePercentiles] as number) : ""}`}>
                              {metrics?.last_30d?.[key as keyof ResponseTimePercentiles] != null
                                ? `${(metrics.last_30d[key as keyof ResponseTimePercentiles] as number).toFixed(0)}ms`
                                : "-"}
                            </TableCell>
                          </TableRow>
                        ))}
                        <TableRow>
                          <TableCell className="text-xs font-medium">Total Checks</TableCell>
                          <TableCell className="text-xs text-center tabular-nums">{metrics?.last_24h?.count ?? "-"}</TableCell>
                          <TableCell className="text-xs text-center tabular-nums">{metrics?.last_7d?.count ?? "-"}</TableCell>
                          <TableCell className="text-xs text-center tabular-nums">{metrics?.last_30d?.count ?? "-"}</TableCell>
                        </TableRow>
                      </TableBody>
                    </Table>
                  </div>
                ) : (
                  <p className="text-center text-[13px] text-muted-foreground py-12">
                    Data metrik tidak tersedia.
                  </p>
                )}
              </div>
            )}

            {/* DNS & Network Tab */}
            {activeTab === "dns" && (
              <div className="space-y-4">
                {dnsLoading ? (
                  <div className="flex items-center justify-center py-16">
                    <Loader2 className="size-5 animate-spin text-muted-foreground" />
                  </div>
                ) : dnsData ? (
                  <>
                    {/* Scan DNS Button */}
                    <div className="flex justify-end">
                      <button
                        onClick={() => refetchDns()}
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md border border-border/50 hover:bg-accent transition-colors"
                      >
                        <RefreshCw className="size-3" />
                        Refresh DNS
                      </button>
                    </div>

                    {/* Email Security */}
                    <div className="grid grid-cols-2 gap-3">
                      <div className="rounded-lg border border-border/50 p-3">
                        <div className="flex items-center gap-2 mb-1">
                          {dnsData?.spf_record ? (
                            <CheckCircle2 className="size-3.5 text-emerald-500" />
                          ) : (
                            <XCircle className="size-3.5 text-red-500" />
                          )}
                          <span className="text-xs font-medium">SPF Record</span>
                        </div>
                        <p className="text-[11px] text-muted-foreground font-mono break-all">
                          {dnsData?.spf_record || "Tidak ditemukan"}
                        </p>
                      </div>
                      <div className="rounded-lg border border-border/50 p-3">
                        <div className="flex items-center gap-2 mb-1">
                          {dnsData?.dmarc_record ? (
                            <CheckCircle2 className="size-3.5 text-emerald-500" />
                          ) : (
                            <XCircle className="size-3.5 text-red-500" />
                          )}
                          <span className="text-xs font-medium">DMARC Record</span>
                        </div>
                        <p className="text-[11px] text-muted-foreground font-mono break-all">
                          {dnsData?.dmarc_record || "Tidak ditemukan"}
                        </p>
                      </div>
                    </div>

                    {/* DNS Records */}
                    {dnsData?.records && dnsData.records.length > 0 && (
                      <div>
                        <h4 className="text-xs font-medium mb-2">DNS Records</h4>
                        <div className="rounded-lg border border-border/50 overflow-hidden">
                          <Table>
                            <TableHeader>
                              <TableRow className="hover:bg-transparent">
                                <TableHead className="text-[11px] w-16">Type</TableHead>
                                <TableHead className="text-[11px]">Value</TableHead>
                              </TableRow>
                            </TableHeader>
                            <TableBody>
                              {dnsData.records.map((record, i) => (
                                <TableRow key={i}>
                                  <TableCell>
                                    <span className="inline-flex rounded px-1.5 py-0.5 bg-primary/10 text-primary text-[11px] font-mono font-medium">
                                      {record.type}
                                    </span>
                                  </TableCell>
                                  <TableCell className="text-xs font-mono break-all">{record.value}</TableCell>
                                </TableRow>
                              ))}
                            </TableBody>
                          </Table>
                        </div>
                      </div>
                    )}

                    {/* Nameservers */}
                    {dnsData?.nameservers && dnsData.nameservers.length > 0 && (
                      <div>
                        <h4 className="text-xs font-medium mb-2">Nameservers</h4>
                        <div className="rounded-lg border border-border/50 p-3 space-y-1">
                          {dnsData.nameservers.map((ns, i) => (
                            <p key={i} className="text-xs font-mono text-muted-foreground">{ns}</p>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* MX Records */}
                    {dnsData?.mx_records && dnsData.mx_records.length > 0 && (
                      <div>
                        <h4 className="text-xs font-medium mb-2">MX Records</h4>
                        <div className="rounded-lg border border-border/50 p-3 space-y-1">
                          {dnsData.mx_records.map((mx, i) => (
                            <p key={i} className="text-xs font-mono text-muted-foreground">{mx}</p>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* Subdomains */}
                    {dnsData?.subdomains && dnsData.subdomains.length > 0 && (
                      <div>
                        <h4 className="text-xs font-medium mb-2">
                          Subdomains Ditemukan ({dnsData.subdomains.length})
                        </h4>
                        <div className="rounded-lg border border-border/50 overflow-hidden max-h-[250px] overflow-y-auto">
                          <Table>
                            <TableHeader>
                              <TableRow className="hover:bg-transparent">
                                <TableHead className="text-[11px]">Subdomain</TableHead>
                                <TableHead className="text-[11px]">IP</TableHead>
                                <TableHead className="text-[11px]">Status</TableHead>
                                <TableHead className="text-[11px]">Title</TableHead>
                              </TableRow>
                            </TableHeader>
                            <TableBody>
                              {dnsData.subdomains.map((sub, i) => (
                                <TableRow key={i}>
                                  <TableCell className="text-xs font-mono">{sub.subdomain}</TableCell>
                                  <TableCell className="text-xs font-mono text-muted-foreground">{sub.ip || "-"}</TableCell>
                                  <TableCell>
                                    {sub.status_code ? (
                                      <span className={`text-[11px] font-medium ${
                                        sub.status_code < 400 ? "text-emerald-600" : "text-red-600"
                                      }`}>
                                        {sub.status_code}
                                      </span>
                                    ) : (
                                      <span className="text-xs text-muted-foreground">-</span>
                                    )}
                                  </TableCell>
                                  <TableCell className="text-xs text-muted-foreground truncate max-w-[150px]">
                                    {sub.title || "-"}
                                  </TableCell>
                                </TableRow>
                              ))}
                            </TableBody>
                          </Table>
                        </div>
                      </div>
                    )}

                    {/* Scan Info */}
                    {dnsData?.scanned_at && (
                      <p className="text-[11px] text-muted-foreground text-right">
                        Terakhir scan: {new Date(dnsData.scanned_at).toLocaleString("id-ID")}
                        {dnsData.scan_duration_ms ? ` (${dnsData.scan_duration_ms}ms)` : ""}
                      </p>
                    )}
                  </>
                ) : (
                  <p className="text-center text-[13px] text-muted-foreground py-12">
                    Data DNS tidak tersedia.
                  </p>
                )}
              </div>
            )}
          </div>
        ) : (
          <p className="text-center text-[13px] text-muted-foreground py-12">
            Data tidak ditemukan.
          </p>
        )}
      </DialogContent>
    </Dialog>
  )
}
