"use client"

import { useState, useEffect } from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import {
  Search,
  RefreshCw,
  AlertCircle,
  Eye,
  EyeOff,
  CheckCircle2,
  Flag,
  Shield,
  Scan,
  CircleDot,
  Plus,
  Trash2,
  Pencil,
  AlertTriangle,
  Loader2,
  Filter,
  ChevronDown,
  X,
  Bot,
  Sparkles,
  Globe,
  ExternalLink,
  Info,
  FileSearch,
} from "lucide-react"
import { toast } from "sonner"

import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

import { dorkApi, defacementApi } from "@/lib/api"
import type { DorkCategory, DorkDetectionDetail } from "@/lib/api"
import { formatDate } from "@/lib/utils"
import { useMutationAction } from "@/hooks/use-mutation-action"

interface DorkStats {
  total_scans: number
  active_detections: number
  false_positives: number
  total_patterns: number
  unresolved_count?: number
  websites_affected?: number
}

interface DorkPattern {
  id: number
  name: string
  pattern: string
  pattern_type: string
  category: string
  severity: string
  description?: string
  is_active: boolean
  is_default: boolean
  keywords?: string[]
}

interface DorkDetection {
  id: number
  website_id: number
  website_name?: string
  website_url?: string
  pattern_id: number
  pattern_name?: string
  pattern_matched?: string
  matched_content?: string
  matched_text?: string
  context?: string
  location?: string
  category: string
  severity: string
  confidence?: number
  ai_verified?: boolean
  url?: string
  detected_at: string
  status: string
  is_false_positive: boolean
  is_resolved: boolean
  resolved_at?: string
  resolved_by?: string
  notes?: string
}

const FALLBACK_CATEGORIES = [
  { value: "gambling", label: "Gambling (Judol)", color: "bg-purple-500/10 text-purple-700 dark:text-purple-400 border-purple-500/20" },
  { value: "defacement", label: "Defacement", color: "bg-red-500/10 text-red-700 dark:text-red-400 border-red-500/20" },
  { value: "malware", label: "Malware", color: "bg-orange-500/10 text-orange-700 dark:text-orange-400 border-orange-500/20" },
  { value: "phishing", label: "Phishing", color: "bg-amber-500/10 text-amber-700 dark:text-amber-400 border-amber-500/20" },
  { value: "seo_spam", label: "SEO Spam", color: "bg-yellow-500/10 text-yellow-700 dark:text-yellow-400 border-yellow-500/20" },
  { value: "webshell", label: "Webshell", color: "bg-rose-500/10 text-rose-700 dark:text-rose-400 border-rose-500/20" },
  { value: "backdoor", label: "Backdoor", color: "bg-pink-500/10 text-pink-700 dark:text-pink-400 border-pink-500/20" },
  { value: "injection", label: "Injection", color: "bg-cyan-500/10 text-cyan-700 dark:text-cyan-400 border-cyan-500/20" },
]

const CATEGORY_COLORS: Record<string, string> = {
  gambling: "bg-purple-500/10 text-purple-700 dark:text-purple-400 border-purple-500/20",
  defacement: "bg-red-500/10 text-red-700 dark:text-red-400 border-red-500/20",
  malware: "bg-orange-500/10 text-orange-700 dark:text-orange-400 border-orange-500/20",
  phishing: "bg-amber-500/10 text-amber-700 dark:text-amber-400 border-amber-500/20",
  seo_spam: "bg-yellow-500/10 text-yellow-700 dark:text-yellow-400 border-yellow-500/20",
  webshell: "bg-rose-500/10 text-rose-700 dark:text-rose-400 border-rose-500/20",
  backdoor: "bg-pink-500/10 text-pink-700 dark:text-pink-400 border-pink-500/20",
  injection: "bg-cyan-500/10 text-cyan-700 dark:text-cyan-400 border-cyan-500/20",
}

const SEVERITIES = [
  { value: "critical", label: "Critical", color: "bg-red-500/10 text-red-600 border-red-500/20" },
  { value: "high", label: "High", color: "bg-orange-500/10 text-orange-600 border-orange-500/20" },
  { value: "medium", label: "Medium", color: "bg-amber-500/10 text-amber-600 border-amber-500/20" },
  { value: "low", label: "Low", color: "bg-blue-500/10 text-blue-600 border-blue-500/20" },
]

function getCategoryColor(category: string): string {
  return CATEGORY_COLORS[category?.toLowerCase()]
    || "bg-gray-500/10 text-gray-700 dark:text-gray-400 border-gray-500/20"
}

function getSeverityColor(severity: string): string {
  return SEVERITIES.find(s => s.value === severity?.toLowerCase())?.color
    || "bg-gray-500/10 text-gray-600 border-gray-500/20"
}

function getSeverityDot(detection: DorkDetection): string {
  if (detection.is_false_positive) return "bg-gray-400"
  if (detection.is_resolved || detection.status === "resolved") return "bg-emerald-500"
  switch (detection.severity) {
    case "critical": return "bg-red-500 animate-pulse"
    case "high": return "bg-orange-500"
    case "medium": return "bg-amber-500"
    default: return "bg-blue-500"
  }
}

export default function DorkPage() {
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState<"detections" | "patterns" | "defacement">("detections")
  const [showCreatePattern, setShowCreatePattern] = useState(false)
  const [editingPattern, setEditingPattern] = useState<null | { id: number; name: string; pattern: string; category: string; severity: string; pattern_type: string; description: string }>(null)
  const [expandedDetection, setExpandedDetection] = useState<number | null>(null)
  const [detailDialogId, setDetailDialogId] = useState<number | null>(null)
  const [scanningWebsiteId, setScanningWebsiteId] = useState<number | null>(null)

  // Detection filters
  const [filterCategory, setFilterCategory] = useState("")
  const [filterSeverity, setFilterSeverity] = useState("")
  const [filterStatus, setFilterStatus] = useState<"" | "unresolved" | "resolved" | "false_positive">("")
  const [detectionLimit, setDetectionLimit] = useState(50)
  const [detectionOffset, setDetectionOffset] = useState(0)

  // Build detection query params
  const detectionParams = new URLSearchParams()
  if (filterCategory) detectionParams.set("category", filterCategory)
  if (filterSeverity) detectionParams.set("severity", filterSeverity)
  if (filterStatus === "unresolved") detectionParams.set("is_resolved", "false")
  if (filterStatus === "resolved") { detectionParams.set("is_resolved", "true"); detectionParams.set("is_false_positive", "false") }
  if (filterStatus === "false_positive") detectionParams.set("is_false_positive", "true")
  detectionParams.set("limit", String(detectionLimit))
  detectionParams.set("offset", String(detectionOffset))

  const statsQuery = useQuery({
    queryKey: ["dork-stats"],
    queryFn: () => dorkApi.stats(),
  })

  const categoriesQuery = useQuery({
    queryKey: ["dork-categories"],
    queryFn: () => dorkApi.categories(),
  })

  const patternsQuery = useQuery({
    queryKey: ["dork-patterns"],
    queryFn: () => dorkApi.patterns(),
  })

  const detectionsQuery = useQuery({
    queryKey: ["dork-detections", filterCategory, filterSeverity, filterStatus, detectionLimit, detectionOffset],
    queryFn: () => dorkApi.detections(detectionParams.toString()),
  })

  // Defacement archive queries
  const defacementStatsQuery = useQuery({
    queryKey: ["defacement-stats"],
    queryFn: () => defacementApi.stats(),
  })

  const defacementQuery = useQuery({
    queryKey: ["defacement-incidents"],
    queryFn: () => defacementApi.incidents("limit=50"),
  })

  const defacementScanMutation = useMutationAction({
    mutationFn: () => defacementApi.scan(),
    successMessage: "Defacement archive scan dimulai",
    errorMessage: "Gagal memulai scan",
    invalidateKeys: ["defacement-stats", "defacement-incidents"],
  })

  const defacementStats = defacementStatsQuery.data?.data as {
    total_incidents: number; unacknowledged_count: number; websites_affected: number;
    by_source: Record<string, number>; last_scan_at?: string
  } | null
  const defacementIncidents = (defacementQuery.data?.data || []) as Array<{
    id: number; website_id: number; website_name?: string; website_url?: string;
    source: string; defaced_url: string; attacker?: string; team?: string;
    defaced_at?: string; mirror_url?: string; is_acknowledged: boolean;
    acknowledged_by?: string; notes?: string; created_at: string
  }>

  const scanAllMutation = useMutationAction({
    mutationFn: () => dorkApi.scanAll(),
    successMessage: "Scan defacement dimulai untuk semua website",
    errorMessage: "Gagal memulai scan",
    invalidateKeys: ["dork-stats", "dork-detections"],
  })

  const verifyAIMutation = useMutationAction({
    mutationFn: () => dorkApi.verifyAI(),
    successMessage: "AI verification selesai",
    errorMessage: "Gagal menjalankan AI verification",
    invalidateKeys: ["dork-stats", "dork-detections"],
  })

  const clearAllMutation = useMutationAction({
    mutationFn: () => dorkApi.clearAll(),
    successMessage: "Semua deteksi berhasil dihapus",
    errorMessage: "Gagal menghapus deteksi",
    invalidateKeys: ["dork-stats", "dork-detections"],
  })

  const stats = statsQuery.data?.data as DorkStats | null
  const aiStatus = (statsQuery.data as { ai_status?: { enabled: boolean; provider: string; model: string } } | undefined)?.ai_status
  const patterns = (patternsQuery.data?.data as DorkPattern[]) || []
  const detections = (detectionsQuery.data?.data as DorkDetection[]) || []
  const detectionsTotal = (detectionsQuery.data as { total?: number } | undefined)?.total
  const isLoading = statsQuery.isLoading || patternsQuery.isLoading || detectionsQuery.isLoading
  const error = statsQuery.error || patternsQuery.error || detectionsQuery.error

  // Build categories list from API or fallback
  const apiCategories = (categoriesQuery.data?.data || []) as DorkCategory[]
  const categories = apiCategories.length > 0
    ? apiCategories.map(c => ({
        value: c.value || c.name,
        label: c.name || c.value,
        color: getCategoryColor(c.value || c.name),
        count: c.count,
      }))
    : FALLBACK_CATEGORIES

  const refetch = () => {
    statsQuery.refetch()
    patternsQuery.refetch()
    detectionsQuery.refetch()
  }

  const handleAcknowledge = async (id: number) => {
    try {
      await defacementApi.acknowledge(id)
      toast.success("Incident acknowledged")
      queryClient.invalidateQueries({ queryKey: ["defacement-incidents"] })
      queryClient.invalidateQueries({ queryKey: ["defacement-stats"] })
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal acknowledge incident")
    }
  }

  const handleResolve = async (id: number) => {
    try {
      await dorkApi.resolveDetection(id)
      toast.success("Detection berhasil di-resolve")
      queryClient.invalidateQueries({ queryKey: ["dork-detections"] })
      queryClient.invalidateQueries({ queryKey: ["dork-stats"] })
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal resolve detection")
    }
  }

  const handleFalsePositive = async (id: number) => {
    try {
      await dorkApi.markFalsePositive(id)
      toast.success("Detection ditandai sebagai false positive")
      queryClient.invalidateQueries({ queryKey: ["dork-detections"] })
      queryClient.invalidateQueries({ queryKey: ["dork-stats"] })
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menandai false positive")
    }
  }

  const handleDeletePattern = async (id: number) => {
    try {
      await dorkApi.deletePattern(id)
      toast.success("Pattern berhasil dihapus")
      queryClient.invalidateQueries({ queryKey: ["dork-patterns"] })
      queryClient.invalidateQueries({ queryKey: ["dork-stats"] })
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menghapus pattern")
    }
  }

  const handleWebsiteScan = async (websiteId: number) => {
    try {
      setScanningWebsiteId(websiteId)
      await dorkApi.websiteScan(websiteId)
      toast.success("Scan dimulai untuk website ini")
      queryClient.invalidateQueries({ queryKey: ["dork-detections"] })
      queryClient.invalidateQueries({ queryKey: ["dork-stats"] })
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memulai scan website")
    } finally {
      setScanningWebsiteId(null)
    }
  }

  const hasActiveFilters = filterCategory || filterSeverity || filterStatus
  const clearFilters = () => {
    setFilterCategory("")
    setFilterSeverity("")
    setFilterStatus("")
    setDetectionOffset(0)
  }

  if (isLoading) return <DorkSkeleton />

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-20">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-500/10">
          <AlertCircle className="h-6 w-6 text-red-500" />
        </div>
        <h2 className="text-sm font-medium">Gagal Memuat Data Dork</h2>
        <p className="text-xs text-muted-foreground">{error instanceof Error ? error.message : "Gagal memuat data dork"}</p>
        <Button onClick={() => refetch()} variant="outline" className="h-8 text-xs">
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
          <h1 className="text-xl font-semibold tracking-tight">Dork / Defacement Detection</h1>
          <p className="text-xs text-muted-foreground mt-0.5">
            Deteksi defacement, gambling, malware, dan konten berbahaya
          </p>
        </div>
        <div className="flex items-center gap-2">
          {detections.length > 0 && (
            <Button
              variant="outline"
              onClick={() => {
                if (confirm("Hapus semua deteksi? Data akan hilang permanen.")) {
                  clearAllMutation.mutate()
                }
              }}
              disabled={clearAllMutation.isPending}
              className="h-8 text-xs text-destructive hover:text-destructive"
            >
              {clearAllMutation.isPending ? (
                <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
              ) : (
                <Trash2 className="mr-1.5 h-3.5 w-3.5" />
              )}
              Hapus Semua
            </Button>
          )}
          {aiStatus?.enabled && detections.length > 0 && (
            <Button
              variant="outline"
              onClick={() => verifyAIMutation.mutate()}
              disabled={verifyAIMutation.isPending}
              className="h-8 text-xs text-emerald-600 hover:text-emerald-700"
            >
              {verifyAIMutation.isPending ? (
                <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
              ) : (
                <Bot className="mr-1.5 h-3.5 w-3.5" />
              )}
              AI Verify All
            </Button>
          )}
          <Button
            variant="outline"
            onClick={() => scanAllMutation.mutate()}
            disabled={scanAllMutation.isPending}
            className="h-8 text-xs"
          >
            {scanAllMutation.isPending ? (
              <RefreshCw className="mr-1.5 h-3.5 w-3.5 animate-spin" />
            ) : (
              <Scan className="mr-1.5 h-3.5 w-3.5" />
            )}
            Scan All
          </Button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Card className="border-border/50 border-l-4 border-l-blue-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-500/10">
                <Search className="h-3.5 w-3.5 text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Total Scans</p>
                <p className="text-2xl font-bold leading-none mt-0.5">{stats?.total_scans ?? 0}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/50 border-l-4 border-l-red-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-red-500/10">
                <AlertTriangle className="h-3.5 w-3.5 text-red-600 dark:text-red-400" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Deteksi Aktif</p>
                <p className="text-2xl font-bold leading-none mt-0.5 text-red-600 dark:text-red-400">
                  {stats?.active_detections ?? 0}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/50 border-l-4 border-l-gray-400">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-gray-500/10">
                <EyeOff className="h-3.5 w-3.5 text-muted-foreground" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">False Positives</p>
                <p className="text-2xl font-bold leading-none mt-0.5 text-muted-foreground">
                  {stats?.false_positives ?? 0}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/50 border-l-4 border-l-purple-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-purple-500/10">
                <Eye className="h-3.5 w-3.5 text-purple-600 dark:text-purple-400" />
              </div>
              <div>
                <p className="text-xs text-muted-foreground">Total Patterns</p>
                <p className="text-2xl font-bold leading-none mt-0.5">{stats?.total_patterns ?? patterns.length}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* AI Verification Status Banner */}
      {aiStatus && (
        <div className={`flex items-center gap-2 rounded-lg border px-4 py-2.5 text-[12px] ${
          aiStatus.enabled
            ? "border-emerald-500/30 bg-emerald-500/5 text-emerald-700 dark:text-emerald-400"
            : "border-border/50 bg-muted/30 text-muted-foreground"
        }`}>
          <Bot className="h-4 w-4 shrink-0" />
          {aiStatus.enabled ? (
            <p>
              <span className="font-medium">AI Verification aktif</span>
              {" — "}
              <span className="capitalize">{aiStatus.provider}</span>
              {aiStatus.model && ` (${aiStatus.model})`}
              {". Deteksi false positive otomatis difilter sebelum disimpan."}
            </p>
          ) : (
            <p>
              <span className="font-medium">AI Verification nonaktif</span>
              {" — Aktifkan di "}
              <a href="/settings" className="underline hover:text-foreground">Pengaturan &rarr; AI Verification</a>
              {" untuk mengurangi false positive secara otomatis."}
            </p>
          )}
        </div>
      )}

      {/* Tabs */}
      <div>
        <div className="flex items-center justify-between border-b border-border/50">
          <div className="flex items-center gap-4">
            <button
              onClick={() => setActiveTab("detections")}
              className={`pb-2 text-[13px] font-medium border-b-2 transition-colors ${
                activeTab === "detections"
                  ? "border-primary text-foreground"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              }`}
            >
              Detections
              {(stats?.active_detections ?? 0) > 0 && (
                <span className="ml-1.5 inline-flex items-center rounded-full bg-red-500/10 px-1.5 py-0.5 text-[10px] font-medium text-red-600 dark:text-red-400">
                  {stats?.active_detections}
                </span>
              )}
            </button>
            <button
              onClick={() => setActiveTab("patterns")}
              className={`pb-2 text-[13px] font-medium border-b-2 transition-colors ${
                activeTab === "patterns"
                  ? "border-primary text-foreground"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              }`}
            >
              Patterns
              <span className="ml-1.5 inline-flex items-center rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
                {patterns.length}
              </span>
            </button>
            <button
              onClick={() => setActiveTab("defacement")}
              className={`pb-2 text-[13px] font-medium border-b-2 transition-colors ${
                activeTab === "defacement"
                  ? "border-primary text-foreground"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              }`}
            >
              <Globe className="inline h-3.5 w-3.5 mr-1 -mt-0.5" />
              Zone-H / XSEC
              {(defacementStats?.unacknowledged_count ?? 0) > 0 && (
                <span className="ml-1.5 inline-flex items-center rounded-full bg-orange-500/10 px-1.5 py-0.5 text-[10px] font-medium text-orange-600 dark:text-orange-400">
                  {defacementStats?.unacknowledged_count}
                </span>
              )}
            </button>
          </div>
          {activeTab === "patterns" && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowCreatePattern(true)}
              className="h-7 text-[11px] mb-1"
            >
              <Plus className="h-3 w-3 mr-1" />
              Add Pattern
            </Button>
          )}
        </div>

        {/* Detections Content */}
        {activeTab === "detections" && (
          <div className="mt-3 space-y-3">
            {/* Filters */}
            <div className="flex items-center gap-2 flex-wrap">
              <Filter className="h-3.5 w-3.5 text-muted-foreground" />
              <Select value={filterCategory} onValueChange={(v) => { setFilterCategory(v === "all" ? "" : v); setDetectionOffset(0) }}>
                <SelectTrigger className="h-7 w-[160px] text-[11px]">
                  <SelectValue placeholder="Kategori" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Semua Kategori</SelectItem>
                  {categories.map(c => (
                    <SelectItem key={c.value} value={c.value}>
                      <span className="flex items-center gap-1.5">
                        {c.label}
                        {"count" in c && c.count != null && c.count !== undefined && (
                          <span className="text-[10px] text-muted-foreground">({String(c.count)})</span>
                        )}
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Select value={filterSeverity} onValueChange={(v) => { setFilterSeverity(v === "all" ? "" : v); setDetectionOffset(0) }}>
                <SelectTrigger className="h-7 w-[120px] text-[11px]">
                  <SelectValue placeholder="Severity" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Semua Severity</SelectItem>
                  {SEVERITIES.map(s => (
                    <SelectItem key={s.value} value={s.value}>{s.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Select value={filterStatus} onValueChange={(v) => { setFilterStatus(v === "all" ? "" : v as typeof filterStatus); setDetectionOffset(0) }}>
                <SelectTrigger className="h-7 w-[130px] text-[11px]">
                  <SelectValue placeholder="Status" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Semua Status</SelectItem>
                  <SelectItem value="unresolved">Unresolved</SelectItem>
                  <SelectItem value="resolved">Resolved</SelectItem>
                  <SelectItem value="false_positive">False Positive</SelectItem>
                </SelectContent>
              </Select>
              {hasActiveFilters && (
                <Button variant="ghost" size="sm" onClick={clearFilters} className="h-7 text-[11px] text-muted-foreground">
                  <X className="h-3 w-3 mr-1" />
                  Clear
                </Button>
              )}
              {detectionsQuery.isFetching && !detectionsQuery.isLoading && (
                <Loader2 className="h-3.5 w-3.5 text-muted-foreground animate-spin ml-auto" />
              )}
            </div>

            {/* Detection List */}
            {detections.length > 0 ? (
              <>
                <div className="space-y-2">
                  {detections.map((detection) => {
                    const isExpanded = expandedDetection === detection.id
                    const matchedText = detection.matched_content || detection.matched_text || detection.pattern_matched || detection.pattern_name
                    return (
                      <Card key={detection.id} className="border-border/50">
                        <CardContent className="px-4 py-3">
                          <div className="flex items-start gap-3">
                            <div className="mt-1.5">
                              <div className={`h-2 w-2 rounded-full ${getSeverityDot(detection)}`} />
                            </div>
                            <div
                              className="flex-1 min-w-0 cursor-pointer"
                              onClick={() => setExpandedDetection(isExpanded ? null : detection.id)}
                            >
                              <div className="flex items-start justify-between gap-2">
                                <div>
                                  <p className="text-[13px] font-medium leading-tight">
                                    {detection.website_name || `Website #${detection.website_id}`}
                                  </p>
                                  {detection.website_url && (
                                    <p className="text-xs text-muted-foreground mt-0.5 truncate">
                                      {detection.url || detection.website_url}
                                    </p>
                                  )}
                                </div>
                                <div className="flex items-center gap-1.5 shrink-0">
                                  {detection.severity && (
                                    <Badge variant="outline" className={`text-[10px] ${getSeverityColor(detection.severity)}`}>
                                      {detection.severity}
                                    </Badge>
                                  )}
                                  {detection.ai_verified && (
                                    <span className="inline-flex items-center gap-0.5 rounded-full px-1.5 py-0.5 text-[10px] font-medium bg-violet-500/10 text-violet-600 dark:text-violet-400">
                                      <Sparkles className="h-2.5 w-2.5" />
                                      AI
                                    </span>
                                  )}
                                  <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ${
                                    detection.is_false_positive
                                      ? "bg-gray-500/10 text-gray-600 dark:text-gray-400"
                                      : (detection.is_resolved || detection.status === "resolved")
                                      ? "bg-emerald-500/10 text-emerald-700 dark:text-emerald-400"
                                      : detection.ai_verified
                                      ? "bg-red-500/10 text-red-700 dark:text-red-400"
                                      : "bg-amber-500/10 text-amber-700 dark:text-amber-400"
                                  }`}>
                                    {detection.is_false_positive
                                      ? (detection.ai_verified ? "AI: False Positive" : "False Positive")
                                      : (detection.is_resolved || detection.status === "resolved")
                                      ? "resolved"
                                      : detection.ai_verified
                                      ? "Confirmed"
                                      : "unverified"}
                                  </span>
                                </div>
                              </div>
                              <div className="flex items-center gap-2 mt-2 flex-wrap">
                                {detection.category && (
                                  <Badge variant="outline" className={`text-[10px] ${getCategoryColor(detection.category)}`}>
                                    {detection.category}
                                  </Badge>
                                )}
                                <code className="text-[11px] bg-muted px-1.5 py-0.5 rounded font-mono max-w-[300px] truncate">
                                  {matchedText}
                                </code>
                                {detection.confidence != null && detection.confidence > 0 && (
                                  <span className="text-[10px] text-muted-foreground">
                                    {(detection.confidence * 100).toFixed(0)}% confidence
                                  </span>
                                )}
                                <span className="text-[11px] text-muted-foreground ml-auto">
                                  {formatDate(detection.detected_at)}
                                </span>
                                <ChevronDown className={`h-3 w-3 text-muted-foreground transition-transform ${isExpanded ? "rotate-180" : ""}`} />
                              </div>

                              {/* Expanded Detail */}
                              {isExpanded && (
                                <div className="mt-3 pt-3 border-t border-border/30 space-y-2">
                                  {detection.context && (
                                    <div>
                                      <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider mb-1">Context</p>
                                      <code className="block text-[11px] bg-muted/50 rounded-md p-2 font-mono break-all whitespace-pre-wrap">
                                        {detection.context}
                                      </code>
                                    </div>
                                  )}
                                  {detection.location && (
                                    <div className="flex items-center gap-2 text-xs">
                                      <span className="text-muted-foreground">Location:</span>
                                      <code className="text-[11px] bg-muted px-1.5 py-0.5 rounded font-mono">{detection.location}</code>
                                    </div>
                                  )}
                                  {detection.resolved_by && (
                                    <div className="flex items-center gap-2 text-xs">
                                      <span className="text-muted-foreground">Resolved by:</span>
                                      <span>{detection.resolved_by}</span>
                                      {detection.resolved_at && <span className="text-muted-foreground">({formatDate(detection.resolved_at)})</span>}
                                    </div>
                                  )}
                                  {detection.notes && (
                                    <div className="flex items-center gap-2 text-xs">
                                      <span className="text-muted-foreground">Notes:</span>
                                      <span>{detection.notes}</span>
                                    </div>
                                  )}
                                </div>
                              )}
                            </div>
                            <div className="flex items-center gap-0.5 shrink-0">
                              {/* Detail button */}
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-7 w-7 p-0 text-muted-foreground hover:text-foreground"
                                onClick={(e) => { e.stopPropagation(); setDetailDialogId(detection.id) }}
                                title="Lihat detail"
                              >
                                <Info className="h-3.5 w-3.5" />
                              </Button>
                              {/* Per-website scan button */}
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-7 w-7 p-0 text-muted-foreground hover:text-blue-600"
                                onClick={(e) => { e.stopPropagation(); handleWebsiteScan(detection.website_id) }}
                                disabled={scanningWebsiteId === detection.website_id}
                                title="Scan ulang website ini"
                              >
                                {scanningWebsiteId === detection.website_id ? (
                                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                                ) : (
                                  <FileSearch className="h-3.5 w-3.5" />
                                )}
                              </Button>
                              {!detection.is_false_positive && !detection.is_resolved && detection.status !== "resolved" && (
                                <>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="h-7 text-[11px] text-emerald-600 hover:text-emerald-700 px-2"
                                    onClick={(e) => { e.stopPropagation(); handleResolve(detection.id) }}
                                  >
                                    <CheckCircle2 className="h-3.5 w-3.5 mr-1" />
                                    Resolve
                                  </Button>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="h-7 text-[11px] text-muted-foreground px-2"
                                    onClick={(e) => { e.stopPropagation(); handleFalsePositive(detection.id) }}
                                  >
                                    <Flag className="h-3.5 w-3.5 mr-1" />
                                    False +
                                  </Button>
                                </>
                              )}
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    )
                  })}
                </div>

                {/* Pagination */}
                {detectionsTotal != null && detectionsTotal > detectionLimit && (
                  <div className="flex items-center justify-between pt-2">
                    <p className="text-xs text-muted-foreground">
                      Menampilkan {detectionOffset + 1}-{Math.min(detectionOffset + detectionLimit, detectionsTotal)} dari {detectionsTotal}
                    </p>
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 text-[11px]"
                        disabled={detectionOffset === 0}
                        onClick={() => setDetectionOffset(Math.max(0, detectionOffset - detectionLimit))}
                      >
                        Previous
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 text-[11px]"
                        disabled={detectionOffset + detectionLimit >= detectionsTotal}
                        onClick={() => setDetectionOffset(detectionOffset + detectionLimit)}
                      >
                        Next
                      </Button>
                    </div>
                  </div>
                )}
              </>
            ) : (
              <div className="flex flex-col items-center justify-center py-12">
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
                  <Shield className="h-5 w-5 text-muted-foreground" />
                </div>
                <p className="text-sm font-medium mt-3">Tidak ada deteksi</p>
                <p className="text-xs text-muted-foreground mt-0.5">
                  {hasActiveFilters ? "Tidak ada hasil dengan filter ini" : "Belum ada konten berbahaya terdeteksi"}
                </p>
              </div>
            )}
          </div>
        )}

        {/* Patterns Content */}
        {activeTab === "patterns" && (
          <div className="mt-3 space-y-2">
            {patterns.length > 0 ? (
              patterns.map((pattern) => (
                <Card key={pattern.id} className="border-border/50">
                  <CardContent className="px-4 py-3">
                    <div className="flex items-center justify-between gap-3">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          {pattern.name && (
                            <span className="text-[13px] font-medium">{pattern.name}</span>
                          )}
                          <Badge variant="outline" className={`text-[10px] ${getCategoryColor(pattern.category)}`}>
                            {pattern.category}
                          </Badge>
                          {pattern.severity && (
                            <Badge variant="outline" className={`text-[10px] ${getSeverityColor(pattern.severity)}`}>
                              {pattern.severity}
                            </Badge>
                          )}
                          {pattern.is_default && (
                            <Badge variant="secondary" className="text-[10px]">default</Badge>
                          )}
                        </div>
                        <code className="block text-[11px] bg-muted px-1.5 py-0.5 rounded font-mono mt-1.5 truncate max-w-[600px]">
                          {pattern.pattern}
                        </code>
                        {pattern.description && (
                          <p className="text-xs text-muted-foreground mt-1">{pattern.description}</p>
                        )}
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ${
                          pattern.is_active
                            ? "bg-emerald-500/10 text-emerald-700 dark:text-emerald-400"
                            : "bg-gray-500/10 text-gray-600 dark:text-gray-400"
                        }`}>
                          <CircleDot className="h-2.5 w-2.5 mr-1" />
                          {pattern.is_active ? "Active" : "Inactive"}
                        </span>
                        {!pattern.is_default && (
                          <>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-7 w-7 p-0 text-muted-foreground hover:text-foreground"
                              onClick={() => setEditingPattern({
                                id: pattern.id,
                                name: pattern.name || "",
                                pattern: pattern.pattern,
                                category: pattern.category,
                                severity: pattern.severity || "medium",
                                pattern_type: pattern.pattern_type || "regex",
                                description: pattern.description || "",
                              })}
                            >
                              <Pencil className="h-3.5 w-3.5" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-7 w-7 p-0 text-muted-foreground hover:text-destructive"
                              onClick={() => handleDeletePattern(pattern.id)}
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </Button>
                          </>
                        )}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))
            ) : (
              <div className="flex flex-col items-center justify-center py-12">
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
                  <Search className="h-5 w-5 text-muted-foreground" />
                </div>
                <p className="text-sm font-medium mt-3">Belum ada pattern</p>
                <p className="text-xs text-muted-foreground mt-0.5">Pattern deteksi belum terdaftar</p>
              </div>
            )}
          </div>
        )}

        {/* Defacement Archive Content */}
        {activeTab === "defacement" && (
          <div className="mt-3 space-y-3">
            {/* Defacement Stats */}
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <Card className="border-border/50 border-l-4 border-l-orange-500">
                <CardContent className="px-4 py-3">
                  <div className="flex items-center gap-3">
                    <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-orange-500/10">
                      <AlertTriangle className="h-3.5 w-3.5 text-orange-600 dark:text-orange-400" />
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Total Insiden</p>
                      <p className="text-xl font-bold mt-0.5">{defacementStats?.total_incidents ?? 0}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>
              <Card className="border-border/50 border-l-4 border-l-red-500">
                <CardContent className="px-4 py-3">
                  <div className="flex items-center gap-3">
                    <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-red-500/10">
                      <AlertCircle className="h-3.5 w-3.5 text-red-600 dark:text-red-400" />
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Belum Acknowledged</p>
                      <p className="text-xl font-bold mt-0.5 text-red-600 dark:text-red-400">{defacementStats?.unacknowledged_count ?? 0}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>
              <Card className="border-border/50 border-l-4 border-l-blue-500">
                <CardContent className="px-4 py-3">
                  <div className="flex items-center gap-3">
                    <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-500/10">
                      <Globe className="h-3.5 w-3.5 text-blue-600 dark:text-blue-400" />
                    </div>
                    <div>
                      <p className="text-xs text-muted-foreground">Website Terdampak</p>
                      <p className="text-xl font-bold mt-0.5">{defacementStats?.websites_affected ?? 0}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Source breakdown if available */}
            {defacementStats?.by_source && Object.keys(defacementStats.by_source).length > 0 && (
              <div className="flex items-center gap-3 text-[11px] text-muted-foreground">
                <span>Sumber:</span>
                {Object.entries(defacementStats.by_source).map(([source, count]) => (
                  <Badge key={source} variant="outline" className="text-[10px]">
                    {source === "zone_xsec" ? "Zone-XSEC" : source === "zone_h" ? "Zone-H" : source}: {count as number}
                  </Badge>
                ))}
              </div>
            )}

            <div className="flex items-center justify-between">
              <p className="text-xs text-muted-foreground">
                Data dari Zone-XSEC defacement archive
                {defacementStats?.last_scan_at && ` \u2022 Scan terakhir: ${formatDate(defacementStats.last_scan_at)}`}
              </p>
              <Button
                variant="outline"
                size="sm"
                className="h-7 text-[11px]"
                onClick={() => defacementScanMutation.mutate()}
                disabled={defacementScanMutation.isPending}
              >
                {defacementScanMutation.isPending ? (
                  <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                ) : (
                  <Globe className="mr-1 h-3 w-3" />
                )}
                Scan Archive
              </Button>
            </div>

            {/* Incidents List */}
            {defacementIncidents.length > 0 ? (
              <div className="space-y-2">
                {defacementIncidents.map((inc) => (
                  <Card key={inc.id} className={`border-border/50 ${!inc.is_acknowledged ? "border-l-4 border-l-orange-500" : ""}`}>
                    <CardContent className="px-4 py-3">
                      <div className="flex items-start justify-between gap-3">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-[13px] font-medium">{inc.website_name || `Website #${inc.website_id}`}</span>
                            <Badge variant="outline" className="text-[10px] bg-orange-500/10 text-orange-600 border-orange-500/20">
                              {inc.source === "zone_xsec" ? "Zone-XSEC" : "Zone-H"}
                            </Badge>
                            {inc.is_acknowledged ? (
                              <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium bg-emerald-500/10 text-emerald-600">
                                <CheckCircle2 className="h-2.5 w-2.5 mr-0.5" />
                                Acknowledged
                              </span>
                            ) : (
                              <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium bg-red-500/10 text-red-600 animate-pulse">
                                Baru
                              </span>
                            )}
                          </div>
                          <p className="text-xs text-muted-foreground mt-1 truncate">{inc.defaced_url}</p>
                          <div className="flex items-center gap-3 mt-1.5 text-[11px] text-muted-foreground flex-wrap">
                            {inc.attacker && <span>Attacker: <span className="text-foreground font-medium">{inc.attacker}</span></span>}
                            {inc.team && <span>Team: <span className="text-foreground font-medium">{inc.team}</span></span>}
                            {inc.defaced_at && <span>{formatDate(inc.defaced_at)}</span>}
                            {inc.mirror_url && (
                              <a href={inc.mirror_url} target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-0.5 text-primary hover:underline">
                                <ExternalLink className="h-3 w-3" />
                                Mirror
                              </a>
                            )}
                          </div>
                          {inc.acknowledged_by && (
                            <p className="text-[10px] text-muted-foreground mt-1">
                              Acknowledged oleh: {inc.acknowledged_by}
                            </p>
                          )}
                          {inc.notes && (
                            <p className="text-[10px] text-muted-foreground mt-0.5">
                              Catatan: {inc.notes}
                            </p>
                          )}
                        </div>
                        {!inc.is_acknowledged && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-7 text-[11px] text-emerald-600 hover:text-emerald-700 shrink-0"
                            onClick={() => handleAcknowledge(inc.id)}
                          >
                            <CheckCircle2 className="h-3.5 w-3.5 mr-1" />
                            Acknowledge
                          </Button>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center py-12">
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
                  <Shield className="h-5 w-5 text-muted-foreground" />
                </div>
                <p className="text-sm font-medium mt-3">Tidak ada insiden defacement</p>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Klik &quot;Scan Archive&quot; untuk mengecek Zone-H dan Zone-XSEC
                </p>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Detection Detail Dialog */}
      <DetectionDetailDialog
        detectionId={detailDialogId}
        onClose={() => setDetailDialogId(null)}
        onResolve={(id) => { handleResolve(id); setDetailDialogId(null) }}
        onFalsePositive={(id) => { handleFalsePositive(id); setDetailDialogId(null) }}
        onScanWebsite={(websiteId) => handleWebsiteScan(websiteId)}
        scanningWebsiteId={scanningWebsiteId}
      />

      {/* Create Pattern Dialog */}
      <CreatePatternDialog
        open={showCreatePattern}
        onOpenChange={setShowCreatePattern}
        categories={categories}
        onCreated={() => {
          queryClient.invalidateQueries({ queryKey: ["dork-patterns"] })
          queryClient.invalidateQueries({ queryKey: ["dork-stats"] })
        }}
      />

      {/* Edit Pattern Dialog */}
      <EditPatternDialog
        pattern={editingPattern}
        onOpenChange={(open) => { if (!open) setEditingPattern(null) }}
        categories={categories}
        onUpdated={() => {
          queryClient.invalidateQueries({ queryKey: ["dork-patterns"] })
          setEditingPattern(null)
        }}
      />
    </div>
  )
}

// --- Detection Detail Dialog ---
function DetectionDetailDialog({
  detectionId,
  onClose,
  onResolve,
  onFalsePositive,
  onScanWebsite,
  scanningWebsiteId,
}: {
  detectionId: number | null
  onClose: () => void
  onResolve: (id: number) => void
  onFalsePositive: (id: number) => void
  onScanWebsite: (websiteId: number) => void
  scanningWebsiteId: number | null
}) {
  const detailQuery = useQuery({
    queryKey: ["dork-detection-detail", detectionId],
    queryFn: () => dorkApi.detection(detectionId!),
    enabled: detectionId != null,
  })

  const detail = detailQuery.data?.data as DorkDetectionDetail | undefined
  const isOpen = detectionId != null

  return (
    <Dialog open={isOpen} onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="sm:max-w-lg max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="text-sm flex items-center gap-2">
            <FileSearch className="h-4 w-4" />
            Detail Deteksi
          </DialogTitle>
          <DialogDescription className="text-xs">
            Informasi lengkap hasil deteksi dork
          </DialogDescription>
        </DialogHeader>

        {detailQuery.isLoading ? (
          <div className="space-y-3 py-4">
            <Skeleton className="h-4 w-48" />
            <Skeleton className="h-3 w-full" />
            <Skeleton className="h-3 w-3/4" />
            <Skeleton className="h-20 w-full" />
          </div>
        ) : detailQuery.error ? (
          <div className="flex flex-col items-center py-6 text-center">
            <AlertCircle className="h-8 w-8 text-red-500 mb-2" />
            <p className="text-sm text-muted-foreground">Gagal memuat detail deteksi</p>
          </div>
        ) : detail ? (
          <div className="space-y-4">
            {/* Website info */}
            <div className="space-y-1.5">
              <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Website</p>
              <p className="text-sm font-medium">{detail.website_name || `Website #${detail.website_id}`}</p>
              {detail.website_url && (
                <a href={detail.website_url} target="_blank" rel="noopener noreferrer" className="text-xs text-primary hover:underline inline-flex items-center gap-1">
                  {detail.website_url}
                  <ExternalLink className="h-3 w-3" />
                </a>
              )}
            </div>

            {/* Badges row */}
            <div className="flex items-center gap-2 flex-wrap">
              {detail.category && (
                <Badge variant="outline" className={`text-[10px] ${getCategoryColor(detail.category)}`}>
                  {detail.category}
                </Badge>
              )}
              {detail.severity && (
                <Badge variant="outline" className={`text-[10px] ${getSeverityColor(detail.severity)}`}>
                  {detail.severity}
                </Badge>
              )}
              {detail.ai_verified && (
                <span className="inline-flex items-center gap-0.5 rounded-full px-1.5 py-0.5 text-[10px] font-medium bg-violet-500/10 text-violet-600 dark:text-violet-400">
                  <Sparkles className="h-2.5 w-2.5" />
                  AI Verified
                </span>
              )}
              <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${
                detail.is_false_positive
                  ? "bg-gray-500/10 text-gray-600 dark:text-gray-400"
                  : (detail.is_resolved || detail.status === "resolved")
                  ? "bg-emerald-500/10 text-emerald-700 dark:text-emerald-400"
                  : "bg-amber-500/10 text-amber-700 dark:text-amber-400"
              }`}>
                {detail.is_false_positive ? "False Positive" : (detail.is_resolved || detail.status === "resolved") ? "Resolved" : "Unresolved"}
              </span>
              {detail.confidence != null && detail.confidence > 0 && (
                <span className="text-[10px] text-muted-foreground">
                  {(detail.confidence * 100).toFixed(0)}% confidence
                </span>
              )}
            </div>

            {/* Matched URL */}
            {detail.url && (
              <div className="space-y-1">
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider">URL Terdeteksi</p>
                <a href={detail.url} target="_blank" rel="noopener noreferrer" className="text-xs text-primary hover:underline break-all inline-flex items-start gap-1">
                  {detail.url}
                  <ExternalLink className="h-3 w-3 mt-0.5 shrink-0" />
                </a>
              </div>
            )}

            {/* Matched content / pattern */}
            <div className="space-y-1">
              <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Konten Terdeteksi</p>
              <code className="block text-[11px] bg-muted/50 rounded-md p-2.5 font-mono break-all whitespace-pre-wrap border border-border/30">
                {detail.matched_content || detail.matched_text || detail.pattern_matched || "-"}
              </code>
            </div>

            {/* Pattern name */}
            {detail.pattern_name && (
              <div className="space-y-1">
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Pattern</p>
                <p className="text-xs">{detail.pattern_name}</p>
              </div>
            )}

            {/* Snippet */}
            {detail.snippet && (
              <div className="space-y-1">
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Snippet</p>
                <code className="block text-[11px] bg-muted/50 rounded-md p-2.5 font-mono break-all whitespace-pre-wrap border border-border/30 max-h-40 overflow-y-auto">
                  {detail.snippet}
                </code>
              </div>
            )}

            {/* Context */}
            {detail.context && (
              <div className="space-y-1">
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Context</p>
                <code className="block text-[11px] bg-muted/50 rounded-md p-2.5 font-mono break-all whitespace-pre-wrap border border-border/30 max-h-40 overflow-y-auto">
                  {detail.context}
                </code>
              </div>
            )}

            {/* Location */}
            {detail.location && (
              <div className="space-y-1">
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Lokasi</p>
                <code className="text-[11px] bg-muted px-1.5 py-0.5 rounded font-mono">{detail.location}</code>
              </div>
            )}

            {/* Date */}
            <div className="space-y-1">
              <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Tanggal Terdeteksi</p>
              <p className="text-xs">{formatDate(detail.detected_at)}</p>
            </div>

            {/* Resolution info */}
            {detail.resolved_by && (
              <div className="space-y-1">
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Resolved oleh</p>
                <p className="text-xs">{detail.resolved_by} {detail.resolved_at && `- ${formatDate(detail.resolved_at)}`}</p>
              </div>
            )}
            {detail.notes && (
              <div className="space-y-1">
                <p className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Catatan</p>
                <p className="text-xs">{detail.notes}</p>
              </div>
            )}

            {/* Actions */}
            <div className="flex items-center gap-2 pt-2 border-t border-border/30">
              <Button
                variant="outline"
                size="sm"
                className="h-7 text-[11px]"
                onClick={() => onScanWebsite(detail.website_id)}
                disabled={scanningWebsiteId === detail.website_id}
              >
                {scanningWebsiteId === detail.website_id ? (
                  <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                ) : (
                  <FileSearch className="h-3 w-3 mr-1" />
                )}
                Scan Ulang Website
              </Button>
              {!detail.is_false_positive && !detail.is_resolved && detail.status !== "resolved" && (
                <>
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-7 text-[11px] text-emerald-600 hover:text-emerald-700"
                    onClick={() => onResolve(detail.id)}
                  >
                    <CheckCircle2 className="h-3 w-3 mr-1" />
                    Resolve
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-7 text-[11px] text-muted-foreground"
                    onClick={() => onFalsePositive(detail.id)}
                  >
                    <Flag className="h-3 w-3 mr-1" />
                    False Positive
                  </Button>
                </>
              )}
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}

// --- Edit Pattern Dialog ---
function EditPatternDialog({ pattern, onOpenChange, categories, onUpdated }: {
  pattern: null | { id: number; name: string; pattern: string; category: string; severity: string; pattern_type: string; description: string }
  onOpenChange: (open: boolean) => void
  categories: Array<{ value: string; label: string }>
  onUpdated: () => void
}) {
  const [name, setName] = useState(pattern?.name ?? "")
  const [patternStr, setPatternStr] = useState(pattern?.pattern ?? "")
  const [category, setCategory] = useState(pattern?.category ?? "")
  const [severity, setSeverity] = useState(pattern?.severity ?? "medium")
  const [patternType, setPatternType] = useState(pattern?.pattern_type ?? "regex")
  const [description, setDescription] = useState(pattern?.description ?? "")
  const [submitting, setSubmitting] = useState(false)

  // Sync state when pattern changes
  useEffect(() => {
    if (pattern) {
      setName(pattern.name)
      setPatternStr(pattern.pattern)
      setCategory(pattern.category)
      setSeverity(pattern.severity)
      setPatternType(pattern.pattern_type)
      setDescription(pattern.description)
    }
  }, [pattern])

  const handleSubmit = async () => {
    if (!pattern) return
    if (!name.trim()) { toast.error("Nama pattern harus diisi"); return }
    if (!patternStr.trim()) { toast.error("Pattern harus diisi"); return }
    if (!category) { toast.error("Kategori harus dipilih"); return }

    try {
      setSubmitting(true)
      await dorkApi.updatePattern(pattern.id, {
        name: name.trim(),
        pattern: patternStr.trim(),
        category,
        severity,
        pattern_type: patternType,
        description: description.trim() || undefined,
      })
      toast.success("Pattern berhasil diupdate")
      onUpdated()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal mengupdate pattern")
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={!!pattern} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="text-sm">Edit Pattern</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <div className="space-y-1.5">
            <Label className="text-xs">Nama</Label>
            <Input
              className="h-8 text-[13px]"
              placeholder="Nama pattern"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <Label className="text-xs">Pattern</Label>
            <Textarea
              className="text-[13px] font-mono min-h-[60px]"
              placeholder="(?i)slot\s*online|judi\s*bola"
              value={patternStr}
              onChange={(e) => setPatternStr(e.target.value)}
              rows={2}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-xs">Kategori</Label>
              <Select value={category} onValueChange={setCategory}>
                <SelectTrigger className="h-8 text-[13px]">
                  <SelectValue placeholder="Pilih kategori" />
                </SelectTrigger>
                <SelectContent>
                  {categories.map(c => (
                    <SelectItem key={c.value} value={c.value}>{c.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs">Severity</Label>
              <Select value={severity} onValueChange={setSeverity}>
                <SelectTrigger className="h-8 text-[13px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {SEVERITIES.map(s => (
                    <SelectItem key={s.value} value={s.value}>{s.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="space-y-1.5">
            <Label className="text-xs">Tipe Pattern</Label>
            <Select value={patternType} onValueChange={setPatternType}>
              <SelectTrigger className="h-8 text-[13px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="regex">Regex</SelectItem>
                <SelectItem value="keyword">Keyword</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label className="text-xs">Deskripsi (opsional)</Label>
            <Textarea
              className="text-[13px] min-h-[40px]"
              placeholder="Deskripsi pattern..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={2}
            />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="outline" className="h-8 text-xs" onClick={() => onOpenChange(false)}>
              Batal
            </Button>
            <Button className="h-8 text-xs" onClick={handleSubmit} disabled={submitting}>
              {submitting ? <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" /> : <Pencil className="h-3.5 w-3.5 mr-1.5" />}
              Update
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

// --- Create Pattern Dialog ---
function CreatePatternDialog({ open, onOpenChange, categories, onCreated }: {
  open: boolean
  onOpenChange: (open: boolean) => void
  categories: Array<{ value: string; label: string }>
  onCreated: () => void
}) {
  const [name, setName] = useState("")
  const [pattern, setPattern] = useState("")
  const [category, setCategory] = useState("")
  const [severity, setSeverity] = useState("medium")
  const [patternType, setPatternType] = useState("regex")
  const [description, setDescription] = useState("")
  const [submitting, setSubmitting] = useState(false)

  const resetForm = () => {
    setName("")
    setPattern("")
    setCategory("")
    setSeverity("medium")
    setPatternType("regex")
    setDescription("")
  }

  const handleSubmit = async () => {
    if (!name.trim()) { toast.error("Nama pattern harus diisi"); return }
    if (!pattern.trim()) { toast.error("Pattern harus diisi"); return }
    if (!category) { toast.error("Kategori harus dipilih"); return }

    try {
      setSubmitting(true)
      await dorkApi.createPattern({
        name: name.trim(),
        pattern: pattern.trim(),
        category,
        severity,
        pattern_type: patternType,
        description: description.trim() || undefined,
      })
      toast.success("Pattern berhasil dibuat")
      resetForm()
      onOpenChange(false)
      onCreated()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal membuat pattern")
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="text-sm">Tambah Custom Pattern</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <div className="space-y-1.5">
            <Label className="text-xs">Nama</Label>
            <Input
              className="h-8 text-[13px]"
              placeholder="Nama pattern"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <Label className="text-xs">Pattern (Regex)</Label>
            <Textarea
              className="text-[13px] font-mono min-h-[60px]"
              placeholder="(?i)slot\s*online|judi\s*bola"
              value={pattern}
              onChange={(e) => setPattern(e.target.value)}
              rows={2}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-xs">Kategori</Label>
              <Select value={category} onValueChange={setCategory}>
                <SelectTrigger className="h-8 text-[13px]">
                  <SelectValue placeholder="Pilih kategori" />
                </SelectTrigger>
                <SelectContent>
                  {categories.map(c => (
                    <SelectItem key={c.value} value={c.value}>{c.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs">Severity</Label>
              <Select value={severity} onValueChange={setSeverity}>
                <SelectTrigger className="h-8 text-[13px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {SEVERITIES.map(s => (
                    <SelectItem key={s.value} value={s.value}>{s.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="space-y-1.5">
            <Label className="text-xs">Tipe Pattern</Label>
            <Select value={patternType} onValueChange={setPatternType}>
              <SelectTrigger className="h-8 text-[13px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="regex">Regex</SelectItem>
                <SelectItem value="keyword">Keyword</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label className="text-xs">Deskripsi (opsional)</Label>
            <Textarea
              className="text-[13px] min-h-[40px]"
              placeholder="Deskripsi pattern..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={2}
            />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="outline" className="h-8 text-xs" onClick={() => onOpenChange(false)}>
              Batal
            </Button>
            <Button className="h-8 text-xs" onClick={handleSubmit} disabled={submitting}>
              {submitting ? <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" /> : <Plus className="h-3.5 w-3.5 mr-1.5" />}
              Simpan
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function DorkSkeleton() {
  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <Skeleton className="h-6 w-56 mb-1" />
          <Skeleton className="h-3.5 w-64" />
        </div>
        <Skeleton className="h-8 w-24" />
      </div>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Card key={i} className="border-border/50">
            <CardContent className="px-5 py-4">
              <div className="flex items-center gap-3">
                <Skeleton className="h-7 w-7 rounded-lg" />
                <div>
                  <Skeleton className="h-3 w-16 mb-1.5" />
                  <Skeleton className="h-7 w-10" />
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
      <Skeleton className="h-8 w-48" />
      <div className="space-y-2">
        {Array.from({ length: 4 }).map((_, i) => (
          <Card key={i} className="border-border/50">
            <CardContent className="px-4 py-3">
              <div className="flex items-center gap-3">
                <Skeleton className="h-2 w-2 rounded-full" />
                <div className="flex-1">
                  <Skeleton className="h-3.5 w-32 mb-1.5" />
                  <Skeleton className="h-3 w-48" />
                </div>
                <Skeleton className="h-5 w-16 rounded-full" />
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
