"use client"

import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import {
  Shield,
  RefreshCw,
  AlertCircle,
  CheckCircle2,
  AlertTriangle,
  Globe,
  Lock,
  Clock,
  Eye,
  XCircle,
  History,
  TrendingUp,
  TrendingDown,
} from "lucide-react"

import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

import { securityApi } from "@/lib/api"
import { formatDate } from "@/lib/utils"
import { useMutationAction } from "@/hooks/use-mutation-action"

type SecurityTabType = "detail" | "history"

interface SecurityHistoryItem {
  id: number
  score: number
  grade: string
  checked_at: string
}

interface SecurityStats {
  total_websites: number
  average_score: number
  grade_distribution: Record<string, number>
  common_issues: { header_name: string; missing_count: number; percentage: number }[] | null
  ssl_valid: number
  ssl_expiring_soon: number
  missing_headers: number
}

interface SecuritySummaryItem {
  id: number
  name: string
  url: string
  ssl_valid: boolean
  ssl_days_until_expiry?: number
  security_score: number
  grade: string
  last_checked_at: string
}

interface HeaderResult {
  name: string
  present: boolean
  value?: string
  expected: boolean
  description: string
  impact: string
  points: number
  max_points: number
}

interface SecurityFinding {
  type: string
  severity: string
  title: string
  description: string
  recommendation: string
}

interface SecurityDetail {
  id: number
  website_id: number
  score: number
  grade: string
  headers?: string // JSON string of HeaderResult[]
  findings?: string // JSON string of SecurityFinding[]
  checked_at: string
}

function parseJSON<T>(str: string | undefined | null): T | null {
  if (!str) return null
  try {
    return JSON.parse(str)
  } catch {
    return null
  }
}

function getGradeColor(grade: string): string {
  switch (grade?.toUpperCase()) {
    case "A":
    case "A+":
      return "bg-emerald-500/10 text-emerald-600 border-emerald-500/20"
    case "B":
    case "B+":
      return "bg-blue-500/10 text-blue-600 border-blue-500/20"
    case "C":
    case "C+":
      return "bg-amber-500/10 text-amber-600 border-amber-500/20"
    case "D":
    case "F":
      return "bg-red-500/10 text-red-600 border-red-500/20"
    default:
      return "bg-gray-500/10 text-gray-600 border-gray-500/20"
  }
}

function getScoreBarColor(score: number): string {
  if (score >= 80) return "bg-emerald-500"
  if (score >= 60) return "bg-blue-500"
  if (score >= 40) return "bg-amber-500"
  return "bg-red-500"
}

function getScoreTextColor(score: number): string {
  if (score >= 80) return "text-emerald-600"
  if (score >= 60) return "text-blue-600"
  if (score >= 40) return "text-amber-600"
  return "text-red-600"
}

function SecurityDetailDialog({
  websiteId,
  websiteName,
  open,
  onOpenChange,
}: {
  websiteId: number | null
  websiteName?: string
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const [activeTab, setActiveTab] = useState<SecurityTabType>("detail")

  const detailQuery = useQuery({
    queryKey: ["security-detail", websiteId],
    queryFn: () => securityApi.website(websiteId!),
    enabled: !!websiteId && open,
  })

  const historyQuery = useQuery({
    queryKey: ["security-history", websiteId],
    queryFn: () => securityApi.history(websiteId!, 10),
    enabled: !!websiteId && open && activeTab === "history",
  })

  const checkMutation = useMutationAction({
    mutationFn: () => securityApi.check(websiteId!),
    successMessage: "Security check dimulai",
    errorMessage: "Gagal memulai security check",
    invalidateKeys: ["security-detail", "security-history", "security-stats", "security-summary"],
  })

  const detail = detailQuery.data?.data as SecurityDetail | null
  const headers = parseJSON<HeaderResult[]>(detail?.headers)
  const findings = parseJSON<SecurityFinding[]>(detail?.findings)

  const presentHeaders = headers?.filter((h) => h.present) ?? []
  const missingHeaders = headers?.filter((h) => !h.present) ?? []

  const historyData = historyQuery.data?.data
  const history: SecurityHistoryItem[] = Array.isArray(historyData) ? historyData : []

  return (
    <Dialog open={open} onOpenChange={(o) => {
      onOpenChange(o)
      if (!o) setActiveTab("detail")
    }}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <DialogTitle className="text-base">
              Security Detail — {websiteName || `Website #${websiteId}`}
            </DialogTitle>
            <Button
              variant="outline"
              size="sm"
              onClick={() => checkMutation.mutate()}
              disabled={checkMutation.isPending || !websiteId}
              className="h-7 text-[11px]"
            >
              {checkMutation.isPending ? (
                <RefreshCw className="mr-1.5 h-3 w-3 animate-spin" />
              ) : (
                <Shield className="mr-1.5 h-3 w-3" />
              )}
              Scan Now
            </Button>
          </div>
        </DialogHeader>

        {/* Tab Switcher */}
        <div className="flex gap-1 border-b border-border/50">
          <button
            onClick={() => setActiveTab("detail")}
            className={`px-3 py-2 text-xs font-medium border-b-2 transition-colors ${
              activeTab === "detail"
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            <div className="flex items-center gap-1.5">
              <Shield className="h-3.5 w-3.5" />
              Detail
            </div>
          </button>
          <button
            onClick={() => setActiveTab("history")}
            className={`px-3 py-2 text-xs font-medium border-b-2 transition-colors ${
              activeTab === "history"
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            <div className="flex items-center gap-1.5">
              <History className="h-3.5 w-3.5" />
              Riwayat Check
            </div>
          </button>
        </div>

        {/* Detail Tab */}
        {activeTab === "detail" && (
          <>
            {detailQuery.isLoading ? (
              <div className="space-y-3 py-4">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
                <Skeleton className="h-4 w-1/2" />
              </div>
            ) : detailQuery.error ? (
              <div className="flex flex-col items-center py-8 gap-2">
                <AlertCircle className="h-8 w-8 text-muted-foreground" />
                <p className="text-sm text-muted-foreground">
                  Belum ada data security check. Jalankan check terlebih dahulu.
                </p>
              </div>
            ) : detail ? (
              <div className="space-y-5">
                {/* Score & Grade */}
                <div className="flex items-center gap-4">
                  <div className="flex flex-col items-center gap-1">
                    <span className={`text-3xl font-bold ${getScoreTextColor(detail.score)}`}>
                      {detail.score}
                    </span>
                    <span className="text-[11px] text-muted-foreground">/ 100</span>
                  </div>
                  <div className="h-12 w-px bg-border" />
                  <span
                    className={`inline-flex items-center rounded-full px-3 py-1 text-sm font-semibold border ${getGradeColor(detail.grade)}`}
                  >
                    Grade {detail.grade}
                  </span>
                  <span className="text-xs text-muted-foreground ml-auto">
                    Checked: {formatDate(detail.checked_at)}
                  </span>
                </div>

                {/* Missing Headers */}
                {missingHeaders.length > 0 && (
                  <div>
                    <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                      Missing Headers ({missingHeaders.length})
                    </h3>
                    <div className="space-y-1.5">
                      {missingHeaders.map((h) => (
                        <div
                          key={h.name}
                          className="flex items-start gap-2 rounded-md bg-red-500/5 border border-red-500/10 px-3 py-2"
                        >
                          <XCircle className="h-3.5 w-3.5 text-red-500 mt-0.5 shrink-0" />
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <span className="text-[12px] font-medium text-red-600">{h.name}</span>
                              <Badge variant="outline" className="text-[10px] px-1.5 py-0 h-4 border-red-500/20 text-red-500">
                                {h.impact}
                              </Badge>
                            </div>
                            <p className="text-[11px] text-muted-foreground mt-0.5">{h.description}</p>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* Present Headers */}
                {presentHeaders.length > 0 && (
                  <div>
                    <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                      Present Headers ({presentHeaders.length})
                    </h3>
                    <div className="rounded-lg border border-border/50 overflow-hidden">
                      <table className="w-full text-[12px]">
                        <tbody>
                          {presentHeaders.map((h) => (
                            <tr key={h.name} className="border-b border-border/30 last:border-0">
                              <td className="px-3 py-2 font-medium text-emerald-600 whitespace-nowrap bg-emerald-500/5 w-1/3">
                                <div className="flex items-center gap-1.5">
                                  <CheckCircle2 className="h-3 w-3" />
                                  {h.name}
                                </div>
                              </td>
                              <td className="px-3 py-2 text-muted-foreground break-all">
                                {h.value || "-"}
                              </td>
                              <td className="px-3 py-2 text-right text-[11px] text-muted-foreground whitespace-nowrap">
                                {h.points}/{h.max_points}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>
                )}

                {/* Findings / Recommendations */}
                {findings && findings.length > 0 && (
                  <div>
                    <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                      Findings ({findings.length})
                    </h3>
                    <ul className="space-y-2">
                      {findings.map((f, i) => (
                        <li
                          key={i}
                          className="flex items-start gap-2 text-xs rounded-md bg-muted/30 border border-border/50 px-3 py-2"
                        >
                          <AlertTriangle className="h-3.5 w-3.5 text-amber-500 mt-0.5 shrink-0" />
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <span className="font-medium">{f.title}</span>
                              <Badge variant="outline" className="text-[10px] px-1.5 py-0 h-4">
                                {f.severity}
                              </Badge>
                            </div>
                            {f.description && (
                              <p className="text-muted-foreground mt-0.5">{f.description}</p>
                            )}
                            {f.recommendation && (
                              <p className="text-primary/80 mt-1">{f.recommendation}</p>
                            )}
                          </div>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            ) : (
              <div className="flex flex-col items-center py-8 gap-2">
                <Shield className="h-8 w-8 text-muted-foreground" />
                <p className="text-sm text-muted-foreground">Belum ada data security check</p>
              </div>
            )}
          </>
        )}

        {/* History Tab */}
        {activeTab === "history" && (
          <>
            {historyQuery.isLoading ? (
              <div className="space-y-3 py-4">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
                <Skeleton className="h-4 w-1/2" />
              </div>
            ) : history.length > 0 ? (
              <div className="space-y-4 mt-2">
                {/* Score Trend Bar Chart */}
                <div>
                  <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-3">
                    Tren Skor Keamanan (10 check terakhir)
                  </h3>
                  <div className="flex items-end gap-1.5 h-32 px-1">
                    {[...history].reverse().map((item, i) => {
                      const score = item.score ?? 0
                      const height = score
                      return (
                        <div key={item.id || i} className="flex-1 flex flex-col items-center gap-1 group relative">
                          {/* Tooltip */}
                          <div className="absolute -top-8 left-1/2 -translate-x-1/2 bg-popover border border-border rounded px-2 py-1 text-[10px] whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-10 shadow-sm">
                            Score: {score} | Grade: {item.grade} | {formatDate(item.checked_at)}
                          </div>
                          <span className={`text-[10px] font-medium ${getScoreTextColor(score)}`}>
                            {score}
                          </span>
                          <div
                            className={`w-full rounded-t transition-all ${getScoreBarColor(score)}`}
                            style={{ height: `${Math.max(height, 4)}%` }}
                          />
                          <span className="text-[9px] text-muted-foreground truncate w-full text-center">
                            {item.checked_at ? new Date(item.checked_at).toLocaleDateString("id-ID", { day: "2-digit", month: "short" }) : "-"}
                          </span>
                        </div>
                      )
                    })}
                  </div>
                  {/* Trend Indicator */}
                  {history.length >= 2 && (() => {
                    const latest = history[0]?.score ?? 0
                    const previous = history[1]?.score ?? 0
                    const diff = latest - previous
                    if (diff === 0) return null
                    return (
                      <div className={`flex items-center gap-1 mt-2 text-[11px] ${diff > 0 ? "text-emerald-600" : "text-red-600"}`}>
                        {diff > 0 ? (
                          <TrendingUp className="h-3.5 w-3.5" />
                        ) : (
                          <TrendingDown className="h-3.5 w-3.5" />
                        )}
                        <span>
                          {diff > 0 ? "Naik" : "Turun"} {Math.abs(diff)} poin dari check sebelumnya
                        </span>
                      </div>
                    )
                  })()}
                </div>

                {/* History List */}
                <div>
                  <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">
                    Detail Riwayat
                  </h3>
                  <div className="space-y-1.5">
                    {history.map((item, i) => {
                      const score = item.score ?? 0
                      return (
                        <div
                          key={item.id || i}
                          className="flex items-center gap-3 rounded-md bg-muted/30 border border-border/50 px-3 py-2"
                        >
                          <Clock className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                          <span className="text-xs text-muted-foreground min-w-[100px]">
                            {formatDate(item.checked_at)}
                          </span>
                          <div className="flex-1 flex items-center gap-2">
                            <div className="h-1.5 w-16 rounded-full bg-muted overflow-hidden">
                              <div
                                className={`h-full rounded-full transition-all ${getScoreBarColor(score)}`}
                                style={{ width: `${score}%` }}
                              />
                            </div>
                            <span className={`text-xs font-semibold ${getScoreTextColor(score)}`}>
                              {score}
                            </span>
                          </div>
                          <span
                            className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium border ${getGradeColor(item.grade)}`}
                          >
                            {item.grade || "-"}
                          </span>
                        </div>
                      )
                    })}
                  </div>
                </div>
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center py-14 mt-2">
                <div className="flex h-14 w-14 items-center justify-center rounded-full bg-muted">
                  <History className="h-6 w-6 text-muted-foreground" />
                </div>
                <p className="text-sm font-medium mt-3">Belum Ada Riwayat</p>
                <p className="text-xs text-muted-foreground mt-0.5">Belum ada riwayat check untuk website ini.</p>
              </div>
            )}
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}

const SECURITY_PAGE_SIZE = 20

export default function SecurityPage() {
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [selectedName, setSelectedName] = useState<string>("")
  const [detailOpen, setDetailOpen] = useState(false)
  const [page, setPage] = useState(1)

  const statsQuery = useQuery({
    queryKey: ["security-stats"],
    queryFn: () => securityApi.stats(),
  })

  const summaryQuery = useQuery({
    queryKey: ["security-summary", page],
    queryFn: () => {
      const params = new URLSearchParams()
      params.set("page", String(page))
      params.set("limit", String(SECURITY_PAGE_SIZE))
      return securityApi.summary(params.toString())
    },
  })

  const checkAllMutation = useMutationAction({
    mutationFn: () => securityApi.checkAll(),
    successMessage: "Security check dimulai untuk semua website",
    errorMessage: "Gagal memulai security check",
    invalidateKeys: ["security-stats", "security-summary"],
  })

  const stats = statsQuery.data?.data as SecurityStats | null
  const summary = (summaryQuery.data?.data as SecuritySummaryItem[]) || []
  const summaryTotal = summaryQuery.data?.total ?? 0
  const isLoading = statsQuery.isLoading || summaryQuery.isLoading
  const error = statsQuery.error || summaryQuery.error

  const refetch = () => {
    statsQuery.refetch()
    summaryQuery.refetch()
  }

  if (isLoading) return <SecuritySkeleton />

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-20">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-500/10">
          <AlertCircle className="h-6 w-6 text-red-500" />
        </div>
        <h2 className="text-sm font-medium">Gagal Memuat Data Security</h2>
        <p className="text-xs text-muted-foreground">{error instanceof Error ? error.message : "Gagal memuat data security"}</p>
        <Button onClick={() => refetch()} variant="outline" className="h-8 text-xs">
          <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
          Coba Lagi
        </Button>
      </div>
    )
  }

  const avgScore = stats?.average_score ?? 0
  const gradeDist = stats?.grade_distribution ?? {}
  const gradeA = (gradeDist["A"] ?? 0) + (gradeDist["A+"] ?? 0)
  const gradeB = (gradeDist["B"] ?? 0) + (gradeDist["B+"] ?? 0)
  const gradeC = (gradeDist["C"] ?? 0) + (gradeDist["C+"] ?? 0)
  const gradeLow = (gradeDist["D"] ?? 0) + (gradeDist["F"] ?? 0)

  return (
    <div className="space-y-5 px-2 lg:px-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Security Overview</h1>
          <p className="text-[13px] text-muted-foreground mt-0.5">
            Monitoring keamanan header dan konfigurasi website
          </p>
        </div>
        <Button
          variant="outline"
          onClick={() => checkAllMutation.mutate()}
          disabled={checkAllMutation.isPending}
          className="h-8 text-xs"
        >
          {checkAllMutation.isPending ? (
            <RefreshCw className="mr-1.5 h-3.5 w-3.5 animate-spin" />
          ) : (
            <Shield className="mr-1.5 h-3.5 w-3.5" />
          )}
          Check All
        </Button>
      </div>

      {/* Stats Cards - Row 1 */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {/* Avg Score */}
        <Card className="border-border/50 border-l-4 border-l-blue-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-500/10">
                <Shield className="h-4 w-4 text-blue-500" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">Avg Score</p>
                <div className="flex items-baseline gap-1.5">
                  <p className="text-2xl font-bold">{avgScore.toFixed(1)}</p>
                  <span className="text-xs text-muted-foreground">/ 100</span>
                </div>
              </div>
            </div>
            <div className="mt-2.5 h-1.5 w-full rounded-full bg-muted overflow-hidden">
              <div
                className={`h-full rounded-full transition-all ${getScoreBarColor(avgScore)}`}
                style={{ width: `${avgScore}%` }}
              />
            </div>
          </CardContent>
        </Card>

        {/* SSL Valid */}
        <Card className="border-border/50 border-l-4 border-l-emerald-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-500/10">
                <Lock className="h-4 w-4 text-emerald-500" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">SSL Valid</p>
                <p className="text-2xl font-bold">{stats?.ssl_valid ?? 0}</p>
              </div>
            </div>
            <p className="text-xs text-muted-foreground mt-1">sertifikat aktif</p>
          </CardContent>
        </Card>

        {/* SSL Expiring */}
        <Card className="border-border/50 border-l-4 border-l-amber-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-500/10">
                <Clock className="h-4 w-4 text-amber-500" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">SSL Expiring</p>
                <p className="text-2xl font-bold">{stats?.ssl_expiring_soon ?? 0}</p>
              </div>
            </div>
            <p className="text-xs text-muted-foreground mt-1">expires &lt; 30 hari</p>
          </CardContent>
        </Card>

        {/* Total */}
        <Card className="border-border/50 border-l-4 border-l-slate-400">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-slate-500/10">
                <Globe className="h-4 w-4 text-slate-500" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">Total Website</p>
                <p className="text-2xl font-bold">{stats?.total_websites ?? 0}</p>
              </div>
            </div>
            <p className="text-xs text-muted-foreground mt-1">dipantau</p>
          </CardContent>
        </Card>
      </div>

      {/* Grade Distribution */}
      {(gradeA + gradeB + gradeC + gradeLow) > 0 && (
        <Card className="border-border/50">
          <div className="px-5 py-4 border-b border-border/50">
            <h2 className="text-sm font-medium">Grade Distribution</h2>
            <p className="text-xs text-muted-foreground mt-0.5">Distribusi grade keamanan website</p>
          </div>
          <CardContent className="px-5 py-4">
            <div className="grid grid-cols-4 gap-3">
              <div className="flex items-center gap-2.5 rounded-lg bg-emerald-500/5 border border-emerald-500/10 px-3 py-2.5">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-emerald-500/10 text-sm font-bold text-emerald-600">A</div>
                <div>
                  <p className="text-lg font-bold">{gradeA}</p>
                  <p className="text-[11px] text-muted-foreground">Excellent</p>
                </div>
              </div>
              <div className="flex items-center gap-2.5 rounded-lg bg-blue-500/5 border border-blue-500/10 px-3 py-2.5">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-blue-500/10 text-sm font-bold text-blue-600">B</div>
                <div>
                  <p className="text-lg font-bold">{gradeB}</p>
                  <p className="text-[11px] text-muted-foreground">Good</p>
                </div>
              </div>
              <div className="flex items-center gap-2.5 rounded-lg bg-amber-500/5 border border-amber-500/10 px-3 py-2.5">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-amber-500/10 text-sm font-bold text-amber-600">C</div>
                <div>
                  <p className="text-lg font-bold">{gradeC}</p>
                  <p className="text-[11px] text-muted-foreground">Fair</p>
                </div>
              </div>
              <div className="flex items-center gap-2.5 rounded-lg bg-red-500/5 border border-red-500/10 px-3 py-2.5">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-red-500/10 text-sm font-bold text-red-600">D/F</div>
                <div>
                  <p className="text-lg font-bold">{gradeLow}</p>
                  <p className="text-[11px] text-muted-foreground">Poor</p>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Common Issues */}
      {stats?.common_issues && stats.common_issues.length > 0 && (
        <Card className="border-border/50">
          <div className="px-5 py-4 border-b border-border/50">
            <h2 className="text-sm font-medium">Common Security Issues</h2>
            <p className="text-xs text-muted-foreground mt-0.5">Header keamanan yang paling sering hilang</p>
          </div>
          <CardContent className="px-5 py-4">
            <div className="space-y-3">
              {stats.common_issues.map((issue) => (
                <div key={issue.header_name} className="flex items-center gap-3">
                  <XCircle className="h-4 w-4 text-red-500 shrink-0" />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-[13px] font-medium">{issue.header_name}</span>
                      <span className="text-xs text-muted-foreground">
                        {issue.missing_count} website ({issue.percentage.toFixed(0)}%)
                      </span>
                    </div>
                    <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
                      <div
                        className="h-full rounded-full bg-red-500/60 transition-all"
                        style={{ width: `${issue.percentage}%` }}
                      />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Security Summary Table */}
      <Card className="border-border/50">
        <div className="px-5 py-4 border-b border-border/50">
          <h2 className="text-sm font-medium">Security Summary</h2>
          <p className="text-xs text-muted-foreground mt-0.5">Skor keamanan per website</p>
        </div>
        <CardContent className="p-0">
          {summary.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider w-[30%] pl-4">Website</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider w-[10%]">SSL</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider w-[20%]">Security Score</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider w-[8%]">Grade</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider w-[18%]">Last Checked</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider text-right w-[10%]">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {summary.map((item, idx) => {
                  const score = item.security_score ?? 0
                  return (
                    <TableRow key={item.id ?? idx} className="group hover:bg-muted/30 transition-colors">
                      <TableCell className="py-3 pl-4">
                        <div className="min-w-0">
                          <div className="text-[13px] font-medium truncate">{item.name}</div>
                          <div className="text-xs text-muted-foreground truncate">{item.url}</div>
                        </div>
                      </TableCell>
                      <TableCell className="py-3">
                        {item.ssl_valid ? (
                          <div className="flex items-center gap-1.5">
                            <Lock className="h-3.5 w-3.5 text-emerald-500" />
                            <span className="text-[11px] text-emerald-600 font-medium">Valid</span>
                            {item.ssl_days_until_expiry !== undefined && item.ssl_days_until_expiry <= 30 && (
                              <Badge variant="outline" className="text-[10px] px-1.5 py-0 h-4 border-amber-500/30 text-amber-600">
                                {item.ssl_days_until_expiry}d
                              </Badge>
                            )}
                          </div>
                        ) : (
                          <div className="flex items-center gap-1.5">
                            <XCircle className="h-3.5 w-3.5 text-red-500" />
                            <span className="text-[11px] text-red-600 font-medium">Invalid</span>
                          </div>
                        )}
                      </TableCell>
                      <TableCell className="py-3">
                        <div className="flex items-center gap-2.5">
                          <div className="h-1.5 w-20 rounded-full bg-muted overflow-hidden">
                            <div
                              className={`h-full rounded-full transition-all ${getScoreBarColor(score)}`}
                              style={{ width: `${score}%` }}
                            />
                          </div>
                          <span className={`text-[13px] font-semibold ${item.last_checked_at ? getScoreTextColor(score) : "text-muted-foreground"}`}>
                            {item.last_checked_at ? score : "—"}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell className="py-3">
                        <span
                          className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium border ${item.last_checked_at ? getGradeColor(item.grade) : "bg-gray-500/10 text-gray-500 border-gray-500/20"}`}
                        >
                          {item.last_checked_at ? item.grade : "—"}
                        </span>
                      </TableCell>
                      <TableCell className="py-3 text-xs text-muted-foreground">
                        {formatDate(item.last_checked_at)}
                      </TableCell>
                      <TableCell className="py-3 text-right">
                        <Button
                          variant="ghost"
                          onClick={() => {
                            setSelectedId(item.id)
                            setSelectedName(item.name)
                            setDetailOpen(true)
                          }}
                          className="h-7 text-[11px] hover:bg-primary/10 hover:text-primary"
                        >
                          <Eye className="mr-1 h-3 w-3" />
                          Detail
                        </Button>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          ) : (
            <div className="flex flex-col items-center justify-center py-16">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted">
                <Shield className="h-5 w-5 text-muted-foreground" />
              </div>
              <p className="text-sm font-medium mt-3">Belum ada data</p>
              <p className="text-xs text-muted-foreground mt-0.5">Jalankan security check untuk melihat hasil</p>
            </div>
          )}
        </CardContent>
        {summaryTotal > SECURITY_PAGE_SIZE && (
          <div className="flex items-center justify-between px-5 py-3 border-t border-border/50">
            <span className="text-sm text-muted-foreground">
              Menampilkan {((page - 1) * SECURITY_PAGE_SIZE) + 1}–{Math.min(page * SECURITY_PAGE_SIZE, summaryTotal)} dari {summaryTotal}
            </span>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" disabled={page === 1} onClick={() => setPage(p => p - 1)}>
                Sebelumnya
              </Button>
              <Button variant="outline" size="sm" disabled={page * SECURITY_PAGE_SIZE >= summaryTotal} onClick={() => setPage(p => p + 1)}>
                Selanjutnya
              </Button>
            </div>
          </div>
        )}
      </Card>

      {/* Security Detail Dialog */}
      <SecurityDetailDialog
        websiteId={selectedId}
        websiteName={selectedName}
        open={detailOpen}
        onOpenChange={(open) => {
          setDetailOpen(open)
          if (!open) {
            setSelectedId(null)
            setSelectedName("")
          }
        }}
      />
    </div>
  )
}

function SecuritySkeleton() {
  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <Skeleton className="h-6 w-44 mb-1.5" />
          <Skeleton className="h-4 w-64" />
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
          <Skeleton className="h-4 w-36 mb-1" />
          <Skeleton className="h-3 w-48" />
        </div>
        <CardContent className="p-4">
          <div className="space-y-4">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="flex items-center gap-4">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-4 w-12" />
                <Skeleton className="h-1.5 w-20 rounded-full" />
                <Skeleton className="h-5 w-10 rounded-full" />
                <Skeleton className="h-3 w-24" />
                <Skeleton className="h-6 w-14 ml-auto rounded" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
