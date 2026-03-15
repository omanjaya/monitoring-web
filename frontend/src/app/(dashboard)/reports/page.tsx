"use client"

import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import {
  FileText,
  RefreshCw,
  AlertCircle,
  Download,
  FileBarChart,
  Calendar,
  ClipboardList,
  Bell,
  ArrowRight,
  Loader2,
  Clock,
  Settings2,
  Shield,
} from "lucide-react"
import { toast } from "sonner"

import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

import { reportApi } from "@/lib/api"

interface ReportType {
  type: string
  name: string
  description: string
  formats: string[]
}

interface ScheduleOption {
  value: string
  label: string
  description?: string
  cron?: string
}

interface ScheduleOptionsResponse {
  frequencies?: ScheduleOption[]
  formats?: string[]
  report_types?: string[]
  delivery_methods?: { label: string; value: string }[]
}

// Helper to get auth token for direct fetch calls
function getToken(): string | null {
  if (typeof window === "undefined") return null
  return localStorage.getItem("token")
}

export default function ReportsPage() {
  const [quickLoading, setQuickLoading] = useState<string | null>(null)
  const [generateLoading, setGenerateLoading] = useState(false)
  const [allReportsLoading, setAllReportsLoading] = useState(false)
  const [allReportsProgress, setAllReportsProgress] = useState({ current: 0, total: 0, label: "" })

  const [form, setForm] = useState({
    type: "",
    start_date: "",
    end_date: "",
    format: "pdf",
  })

  const reportTypesQuery = useQuery({
    queryKey: ["report-types"],
    queryFn: () => reportApi.types(),
  })

  const scheduleOptionsQuery = useQuery({
    queryKey: ["report-schedule-options"],
    queryFn: async () => {
      const res = await reportApi.scheduleOptions()
      return (res as unknown as ScheduleOptionsResponse) || {}
    },
  })

  const reportTypes = (reportTypesQuery.data?.data as ReportType[] | null) || []
  const scheduleOptions = scheduleOptionsQuery.data
  const isLoading = reportTypesQuery.isLoading
  const error = reportTypesQuery.error

  // Download file from a binary response
  const downloadFile = async (response: Response, fallbackName: string) => {
    const blob = await response.blob()
    const disposition = response.headers.get("Content-Disposition")
    let filename = fallbackName
    if (disposition) {
      const match = disposition.match(/filename=(.+)/)
      if (match) filename = match[1].replace(/"/g, "")
    }
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    window.URL.revokeObjectURL(url)
  }

  const handleQuickReport = async (type: string, period: string, label: string) => {
    const key = `${type}-${period}`
    try {
      setQuickLoading(key)
      const token = getToken()
      const response = await fetch(`/api/reports/quick/${type}/${period}?format=pdf`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      if (!response.ok) {
        const err = await response.json().catch(() => ({ error: "Download gagal" }))
        throw new Error(err.error || "Download gagal")
      }
      await downloadFile(response, `${type}-${period}-report.pdf`)
      toast.success(`${label} berhasil di-download`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : `Gagal generate ${label}`)
    } finally {
      setQuickLoading(null)
    }
  }

  const handleAllReports = async () => {
    const allTypes = [
      { type: "comprehensive", period: "month", label: "Comprehensive" },
      { type: "uptime", period: "month", label: "Uptime" },
      { type: "ssl", period: "month", label: "SSL" },
      { type: "security", period: "month", label: "Security" },
      { type: "alerts", period: "month", label: "Alerts" },
      { type: "content_scan", period: "month", label: "Content Scan" },
    ]

    try {
      setAllReportsLoading(true)
      const token = getToken()
      let successCount = 0
      let failCount = 0

      for (let i = 0; i < allTypes.length; i++) {
        const report = allTypes[i]
        setAllReportsProgress({ current: i + 1, total: allTypes.length, label: report.label })

        try {
          const response = await fetch(`/api/reports/quick/${report.type}/${report.period}?format=pdf`, {
            headers: token ? { Authorization: `Bearer ${token}` } : {},
          })
          if (!response.ok) {
            failCount++
            continue
          }
          await downloadFile(response, `${report.type}-${report.period}-report.pdf`)
          successCount++
          // Small delay between downloads so browser doesn't block them
          if (i < allTypes.length - 1) {
            await new Promise((r) => setTimeout(r, 500))
          }
        } catch {
          failCount++
        }
      }

      if (failCount === 0) {
        toast.success(`Semua ${successCount} laporan berhasil di-download`)
      } else {
        toast.warning(`${successCount} berhasil, ${failCount} gagal di-download`)
      }
    } catch (err) {
      toast.error("Gagal mendownload semua laporan")
    } finally {
      setAllReportsLoading(false)
      setAllReportsProgress({ current: 0, total: 0, label: "" })
    }
  }

  const handleGenerate = async () => {
    if (!form.type) {
      toast.error("Pilih tipe report terlebih dahulu")
      return
    }
    if (!form.start_date || !form.end_date) {
      toast.error("Tanggal mulai dan akhir harus diisi")
      return
    }

    try {
      setGenerateLoading(true)
      const token = getToken()

      // Convert date-only to RFC3339 format
      const startDate = `${form.start_date}T00:00:00+08:00`
      const endDate = `${form.end_date}T23:59:59+08:00`

      const response = await fetch("/api/reports/generate", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          type: form.type,
          start_date: startDate,
          end_date: endDate,
          format: form.format,
        }),
      })

      if (!response.ok) {
        const err = await response.json().catch(() => ({ error: "Generate gagal" }))
        throw new Error(err.error || "Generate gagal")
      }

      const ext = form.format === "csv" ? "csv" : form.format === "pdf" ? "pdf" : "xlsx"
      await downloadFile(response, `${form.type}-report.${ext}`)
      toast.success("Report berhasil di-download")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal generate report")
    } finally {
      setGenerateLoading(false)
    }
  }

  if (isLoading) return <ReportsSkeleton />

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-20">
        <div className="rounded-full bg-red-500/10 p-3">
          <AlertCircle className="h-5 w-5 text-red-500" />
        </div>
        <h2 className="text-sm font-medium">Gagal Memuat Data Reports</h2>
        <p className="text-xs text-muted-foreground">{error instanceof Error ? error.message : "Gagal memuat tipe report"}</p>
        <Button onClick={() => reportTypesQuery.refetch()} variant="outline" className="h-8 text-xs mt-1">
          <RefreshCw className="mr-1.5 h-3 w-3" />
          Coba Lagi
        </Button>
      </div>
    )
  }

  const quickReports = [
    {
      type: "comprehensive",
      period: "week",
      label: "Weekly Summary",
      icon: Calendar,
      description: "Ringkasan komprehensif minggu ini",
      color: "text-blue-600 dark:text-blue-400",
      bg: "bg-blue-500/10",
    },
    {
      type: "comprehensive",
      period: "month",
      label: "Monthly Summary",
      icon: FileBarChart,
      description: "Ringkasan komprehensif bulan ini",
      color: "text-emerald-600 dark:text-emerald-400",
      bg: "bg-emerald-500/10",
    },
    {
      type: "uptime",
      period: "month",
      label: "Uptime Report",
      icon: ClipboardList,
      description: "Laporan uptime bulanan",
      color: "text-purple-600 dark:text-purple-400",
      bg: "bg-purple-500/10",
    },
    {
      type: "alerts",
      period: "week",
      label: "Alert Report",
      icon: Bell,
      description: "Laporan alert minggu ini",
      color: "text-amber-600 dark:text-amber-400",
      bg: "bg-amber-500/10",
    },
    {
      type: "ssl",
      period: "month",
      label: "SSL Report",
      icon: Shield,
      description: "Laporan sertifikat SSL bulanan",
      color: "text-cyan-600 dark:text-cyan-400",
      bg: "bg-cyan-500/10",
    },
    {
      type: "security",
      period: "month",
      label: "Security Report",
      icon: FileText,
      description: "Laporan security headers bulanan",
      color: "text-red-600 dark:text-red-400",
      bg: "bg-red-500/10",
    },
  ]

  const scheduleIntervals = scheduleOptions?.frequencies || []

  // Get available formats for selected report type
  const selectedType = reportTypes.find((rt) => rt.type === form.type)
  const availableFormats = selectedType?.formats || ["excel", "csv"]

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Reports</h1>
          <p className="text-xs text-muted-foreground mt-0.5">
            Generate dan download laporan monitoring website
          </p>
        </div>
        <Button
          onClick={handleAllReports}
          disabled={allReportsLoading || quickLoading !== null}
          className="h-9 text-xs gap-1.5"
        >
          {allReportsLoading ? (
            <>
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              {allReportsProgress.total > 0
                ? `${allReportsProgress.current}/${allReportsProgress.total} - ${allReportsProgress.label}`
                : "Memproses..."}
            </>
          ) : (
            <>
              <Download className="h-3.5 w-3.5" />
              Download Semua Laporan
            </>
          )}
        </Button>
      </div>

      {/* Quick Reports */}
      <div className="space-y-3">
        <h2 className="text-sm font-medium">Quick Reports</h2>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
          {quickReports.map((report) => {
            const Icon = report.icon
            const key = `${report.type}-${report.period}`
            const isItemLoading = quickLoading === key
            return (
              <Card
                key={key}
                className="border-border/50 cursor-pointer transition-all hover:border-border hover:shadow-sm group"
                onClick={() => !isItemLoading && handleQuickReport(report.type, report.period, report.label)}
              >
                <CardContent className="p-4">
                  <div className="flex items-start gap-3">
                    <div className={`rounded-lg ${report.bg} p-2 shrink-0`}>
                      {isItemLoading ? (
                        <Loader2 className={`h-4 w-4 ${report.color} animate-spin`} />
                      ) : (
                        <Icon className={`h-4 w-4 ${report.color}`} />
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between">
                        <h3 className="text-[13px] font-medium">{report.label}</h3>
                        <ArrowRight className="h-3.5 w-3.5 text-muted-foreground/0 group-hover:text-muted-foreground/60 transition-colors shrink-0" />
                      </div>
                      <p className="text-xs text-muted-foreground mt-0.5 line-clamp-1">
                        {report.description}
                      </p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      </div>

      {/* Custom Report */}
      <div className="space-y-3">
        <h2 className="text-sm font-medium">Custom Report</h2>
        <Card className="border-border/50">
          <CardContent className="p-4">
            <p className="text-xs text-muted-foreground mb-4">
              Generate laporan kustom dengan rentang waktu dan format tertentu
            </p>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-5 items-end">
              <div className="space-y-1.5">
                <Label className="text-xs">Tipe Report</Label>
                <Select
                  value={form.type}
                  onValueChange={(value) => setForm((f) => ({ ...f, type: value }))}
                >
                  <SelectTrigger className="h-8 text-[13px]">
                    <SelectValue placeholder="Pilih tipe" />
                  </SelectTrigger>
                  <SelectContent>
                    {reportTypes.length > 0 ? (
                      reportTypes.map((rt) => (
                        <SelectItem key={rt.type} value={rt.type}>
                          {rt.name}
                        </SelectItem>
                      ))
                    ) : (
                      <>
                        <SelectItem value="uptime">Uptime</SelectItem>
                        <SelectItem value="ssl">SSL</SelectItem>
                        <SelectItem value="security">Security</SelectItem>
                        <SelectItem value="alerts">Alerts</SelectItem>
                        <SelectItem value="comprehensive">Comprehensive</SelectItem>
                      </>
                    )}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs">Tanggal Mulai</Label>
                <Input
                  type="date"
                  value={form.start_date}
                  onChange={(e) => setForm((f) => ({ ...f, start_date: e.target.value }))}
                  className="h-8 text-[13px]"
                />
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs">Tanggal Akhir</Label>
                <Input
                  type="date"
                  value={form.end_date}
                  onChange={(e) => setForm((f) => ({ ...f, end_date: e.target.value }))}
                  className="h-8 text-[13px]"
                />
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs">Format</Label>
                <Select
                  value={form.format}
                  onValueChange={(value) => setForm((f) => ({ ...f, format: value }))}
                >
                  <SelectTrigger className="h-8 text-[13px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {availableFormats.includes("pdf") && <SelectItem value="pdf">PDF</SelectItem>}
                    {availableFormats.includes("excel") && <SelectItem value="excel">Excel</SelectItem>}
                    {availableFormats.includes("csv") && <SelectItem value="csv">CSV</SelectItem>}
                  </SelectContent>
                </Select>
              </div>
              <Button onClick={handleGenerate} disabled={generateLoading} className="h-8 text-xs">
                {generateLoading ? (
                  <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />
                ) : (
                  <Download className="mr-1.5 h-3 w-3" />
                )}
                Generate
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Scheduled Report Options */}
      <div className="space-y-3">
        <h2 className="text-sm font-medium">Opsi Laporan Terjadwal</h2>
        <Card className="border-border/50">
          <CardContent className="p-4">
            {scheduleOptionsQuery.isLoading ? (
              <div className="space-y-3">
                <Skeleton className="h-3 w-64" />
                <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
                  {Array.from({ length: 3 }).map((_, i) => (
                    <Skeleton key={i} className="h-16 rounded-md" />
                  ))}
                </div>
              </div>
            ) : scheduleIntervals.length > 0 ? (
              <>
                <p className="text-xs text-muted-foreground mb-4">
                  Interval yang tersedia untuk laporan terjadwal otomatis
                </p>
                <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
                  {scheduleIntervals.map((option) => (
                    <div
                      key={option.value}
                      className="flex items-start gap-3 rounded-md border border-border/50 p-3 hover:bg-muted/50 transition-colors"
                    >
                      <div className="rounded-lg bg-muted p-2 shrink-0">
                        <Clock className="h-4 w-4 text-muted-foreground" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <h3 className="text-[13px] font-medium">{option.label}</h3>
                        {option.description && (
                          <p className="text-xs text-muted-foreground mt-0.5 line-clamp-2">
                            {option.description}
                          </p>
                        )}
                        {option.cron && (
                          <p className="text-[11px] text-muted-foreground/70 font-mono mt-1">
                            {option.cron}
                          </p>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
                {scheduleOptions?.formats && scheduleOptions.formats.length > 0 && (
                  <div className="mt-3 pt-3 border-t border-border/50">
                    <p className="text-xs text-muted-foreground">
                      Format tersedia:{" "}
                      {scheduleOptions.formats.map((fmt, idx) => (
                        <span key={fmt}>
                          <span className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-[11px] font-medium text-muted-foreground uppercase">
                            {fmt}
                          </span>
                          {idx < scheduleOptions.formats!.length - 1 && " "}
                        </span>
                      ))}
                    </p>
                  </div>
                )}
              </>
            ) : (
              <div className="flex items-center gap-3 py-4">
                <div className="rounded-lg bg-muted p-2.5">
                  <Settings2 className="h-4 w-4 text-muted-foreground" />
                </div>
                <div>
                  <p className="text-[13px] font-medium">Laporan Terjadwal</p>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Konfigurasi laporan terjadwal tersedia melalui pengaturan scheduler di backend.
                    Report dapat di-generate secara otomatis sesuai interval yang dikonfigurasi (harian, mingguan, bulanan).
                  </p>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function ReportsSkeleton() {
  return (
    <div className="space-y-5">
      <div>
        <Skeleton className="h-6 w-24 mb-1" />
        <Skeleton className="h-3 w-52" />
      </div>
      <div className="space-y-3">
        <Skeleton className="h-4 w-28" />
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Card key={i} className="border-border/50">
              <CardContent className="p-4">
                <div className="flex items-start gap-3">
                  <Skeleton className="h-8 w-8 rounded-lg" />
                  <div className="flex-1 space-y-1.5">
                    <Skeleton className="h-4 w-28" />
                    <Skeleton className="h-3 w-36" />
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
      <div className="space-y-3">
        <Skeleton className="h-4 w-28" />
        <Card className="border-border/50">
          <CardContent className="p-4">
            <Skeleton className="h-3 w-64 mb-4" />
            <div className="flex gap-3">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-8 flex-1" />
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
      <div className="space-y-3">
        <Skeleton className="h-4 w-40" />
        <Card className="border-border/50">
          <CardContent className="p-4">
            <Skeleton className="h-16 w-full" />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
