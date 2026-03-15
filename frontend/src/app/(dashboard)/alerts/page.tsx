"use client"

import { useState, useEffect } from "react"
import { useQuery } from "@tanstack/react-query"
import { alertApi, type Alert, type AlertSummary } from "@/lib/api"
import { usePaginatedQuery } from "@/hooks/use-paginated-query"
import { useMutationAction } from "@/hooks/use-mutation-action"
import { formatDate } from "@/lib/utils"
import {
  AlertTriangle,
  Bell,
  CheckCircle2,
  Info,
  Loader2,
  XCircle,
  ChevronLeft,
  ChevronRight,
  Inbox,
  Search,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"

interface AlertFilters {
  type: string
  severity: string
  is_resolved: string
  search: string
}

export default function AlertsPage() {
  const {
    data: alerts,
    total,
    totalPages,
    page,
    setPage,
    filters,
    setFilter,
    isLoading,
    pageSize,
    setPageSize,
  } = usePaginatedQuery<Alert, AlertFilters>({
    queryKey: "alerts",
    queryFn: (params) => alertApi.list(params),
    pageSize: 25,
    initialFilters: { type: "all", severity: "all", is_resolved: "all", search: "" },
  })

  // Debounced website name search
  const [searchInput, setSearchInput] = useState("")
  useEffect(() => {
    const timer = setTimeout(() => {
      setFilter("search", searchInput)
    }, 300)
    return () => clearTimeout(timer)
  }, [searchInput, setFilter])

  const { data: summary } = useQuery<AlertSummary>({
    queryKey: ["alert-summary"],
    queryFn: async () => {
      const res = await alertApi.summary()
      return res.data
    },
  })

  // Bulk selection state
  const [selectedIds, setSelectedIds] = useState<number[]>([])
  const [bulkResolveDialogOpen, setBulkResolveDialogOpen] = useState(false)
  const [bulkResolveNote, setBulkResolveNote] = useState("")

  // Resolve dialog
  const [resolveDialogOpen, setResolveDialogOpen] = useState(false)
  const [resolveTarget, setResolveTarget] = useState<Alert | null>(null)
  const [resolveNote, setResolveNote] = useState("")

  const acknowledgeMutation = useMutationAction({
    mutationFn: (id: number) => alertApi.acknowledge(id),
    successMessage: "Alert berhasil di-acknowledge",
    invalidateKeys: ["alerts", "alert-summary"],
  })

  const resolveMutation = useMutationAction({
    mutationFn: ({ id, note }: { id: number; note?: string }) => alertApi.resolve(id, note),
    successMessage: "Alert berhasil di-resolve",
    invalidateKeys: ["alerts", "alert-summary"],
    onSuccess: () => setResolveDialogOpen(false),
  })

  const bulkResolveMutation = useMutationAction({
    mutationFn: ({ ids, note }: { ids: number[]; note: string }) => alertApi.bulkResolve(ids, note),
    successMessage: "Alert berhasil di-resolve",
    invalidateKeys: ["alerts", "alert-summary"],
    onSuccess: () => {
      setBulkResolveDialogOpen(false)
      setSelectedIds([])
      setBulkResolveNote("")
    },
  })

  const openResolveDialog = (alert: Alert) => {
    setResolveTarget(alert)
    setResolveNote("")
    setResolveDialogOpen(true)
  }

  // Unresolved alerts on current page (eligible for selection)
  const unresolvedAlerts = alerts.filter((a) => !a.is_resolved)
  const allUnresolvedSelected =
    unresolvedAlerts.length > 0 &&
    unresolvedAlerts.every((a) => selectedIds.includes(a.id))

  const toggleSelectAll = () => {
    if (allUnresolvedSelected) {
      setSelectedIds([])
    } else {
      setSelectedIds(unresolvedAlerts.map((a) => a.id))
    }
  }

  const toggleSelectOne = (id: number) => {
    setSelectedIds((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]
    )
  }

  const handleBulkResolve = () => {
    setBulkResolveNote("")
    setBulkResolveDialogOpen(true)
  }

  const getSeverityDot = (severity: string) => {
    switch (severity) {
      case "critical": return "bg-red-500"
      case "warning": return "bg-amber-500"
      case "info": return "bg-blue-500"
      default: return "bg-muted-foreground"
    }
  }

  const getSeverityBadgeClasses = (severity: string) => {
    switch (severity) {
      case "critical": return "bg-red-500/10 text-red-600 dark:text-red-400"
      case "warning": return "bg-amber-500/10 text-amber-600 dark:text-amber-400"
      case "info": return "bg-blue-500/10 text-blue-600 dark:text-blue-400"
      default: return "bg-muted text-muted-foreground"
    }
  }

  const getTypeBadgeClasses = (type: string) => {
    switch (type) {
      case "downtime": return "bg-red-500/10 text-red-600 dark:text-red-400"
      case "ssl_expiry": return "bg-amber-500/10 text-amber-600 dark:text-amber-400"
      case "content_issue": return "bg-purple-500/10 text-purple-600 dark:text-purple-400"
      case "performance": return "bg-blue-500/10 text-blue-600 dark:text-blue-400"
      case "security": return "bg-orange-500/10 text-orange-600 dark:text-orange-400"
      default: return "bg-muted text-muted-foreground"
    }
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div>
        <h1 className="text-xl font-semibold tracking-tight">Alerts</h1>
        <p className="text-sm text-muted-foreground">Monitor dan kelola alert dari semua website</p>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-3 grid-cols-2 lg:grid-cols-4">
        {/* Total Active */}
        <div className="rounded-lg border border-border/50 px-5 py-4 border-l-4 border-l-foreground/20">
          <div className="flex items-center gap-3">
            <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-muted">
              <Bell className="size-3.5 text-muted-foreground" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Total Aktif</p>
              <p className="text-2xl font-bold">{summary?.total_active ?? "-"}</p>
            </div>
          </div>
        </div>

        {/* Critical */}
        <div className="rounded-lg border border-border/50 px-5 py-4 border-l-4 border-l-red-500">
          <div className="flex items-center gap-3">
            <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-red-500/10">
              <XCircle className="size-3.5 text-red-500" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Critical</p>
              <p className="text-2xl font-bold text-red-600 dark:text-red-400">{summary?.critical ?? 0}</p>
            </div>
          </div>
        </div>

        {/* Warning */}
        <div className="rounded-lg border border-border/50 px-5 py-4 border-l-4 border-l-amber-500">
          <div className="flex items-center gap-3">
            <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-500/10">
              <AlertTriangle className="size-3.5 text-amber-500" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Warning</p>
              <p className="text-2xl font-bold text-amber-600 dark:text-amber-400">{summary?.warning ?? 0}</p>
            </div>
          </div>
        </div>

        {/* Info */}
        <div className="rounded-lg border border-border/50 px-5 py-4 border-l-4 border-l-blue-500">
          <div className="flex items-center gap-3">
            <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-500/10">
              <Info className="size-3.5 text-blue-500" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Info</p>
              <p className="text-2xl font-bold text-blue-600 dark:text-blue-400">{summary?.info ?? 0}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Filter Bar */}
      <div className="flex items-center gap-2 flex-wrap">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground pointer-events-none" />
          <Input
            type="text"
            placeholder="Cari nama website..."
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="h-8 pl-8 text-xs w-[200px]"
          />
        </div>

        <Select value={filters.type} onValueChange={(v) => setFilter("type", v)}>
          <SelectTrigger className="w-[140px] h-8 text-xs">
            <SelectValue placeholder="Type" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Type</SelectItem>
            <SelectItem value="downtime">Downtime</SelectItem>
            <SelectItem value="ssl_expiry">SSL Expiry</SelectItem>
            <SelectItem value="content_issue">Content Issue</SelectItem>
            <SelectItem value="performance">Performance</SelectItem>
            <SelectItem value="security">Security</SelectItem>
          </SelectContent>
        </Select>

        <Select value={filters.severity} onValueChange={(v) => setFilter("severity", v)}>
          <SelectTrigger className="w-[140px] h-8 text-xs">
            <SelectValue placeholder="Severity" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Severity</SelectItem>
            <SelectItem value="critical">Critical</SelectItem>
            <SelectItem value="warning">Warning</SelectItem>
            <SelectItem value="info">Info</SelectItem>
          </SelectContent>
        </Select>

        <Select value={filters.is_resolved} onValueChange={(v) => setFilter("is_resolved", v)}>
          <SelectTrigger className="w-[140px] h-8 text-xs">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Status</SelectItem>
            <SelectItem value="false">Active</SelectItem>
            <SelectItem value="true">Resolved</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Bulk Action Toolbar */}
      {selectedIds.length > 0 && (
        <div className="flex items-center gap-2 p-2 bg-muted rounded-md">
          <span className="text-sm text-muted-foreground">{selectedIds.length} dipilih</span>
          <Button size="sm" className="h-7 text-xs" onClick={handleBulkResolve}>
            Resolve Terpilih
          </Button>
          <Button size="sm" variant="ghost" className="h-7 text-xs" onClick={() => setSelectedIds([])}>
            Batal
          </Button>
        </div>
      )}

      {/* Alert List */}
      <div className="space-y-2">
        {/* Select-all row (only when there are unresolved alerts) */}
        {!isLoading && unresolvedAlerts.length > 0 && (
          <div className="flex items-center gap-2 px-4 py-1.5 rounded-lg border border-border/30 bg-muted/20">
            <Checkbox
              id="select-all"
              checked={allUnresolvedSelected}
              onCheckedChange={toggleSelectAll}
              className="size-3.5"
            />
            <label htmlFor="select-all" className="text-xs text-muted-foreground cursor-pointer select-none">
              Pilih semua alert aktif di halaman ini ({unresolvedAlerts.length})
            </label>
          </div>
        )}

        {isLoading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 className="size-5 animate-spin text-muted-foreground" />
          </div>
        ) : alerts.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 gap-3">
            <div className="rounded-full bg-muted p-3">
              <Inbox className="size-5 text-muted-foreground" />
            </div>
            <div className="text-center">
              <p className="text-sm font-medium">Tidak ada alert ditemukan</p>
              <p className="text-xs text-muted-foreground mt-0.5">Semua sistem berjalan normal</p>
            </div>
          </div>
        ) : (
          alerts.map((alert) => (
            <div
              key={alert.id}
              className={`rounded-lg border border-border/50 p-4 transition-colors hover:bg-muted/30 ${
                alert.is_resolved ? "opacity-60" : ""
              } ${selectedIds.includes(alert.id) ? "border-primary/40 bg-primary/5" : ""}`}
            >
              <div className="flex items-start gap-3">
                {/* Checkbox (only for unresolved alerts) */}
                {!alert.is_resolved && (
                  <div className="mt-1 shrink-0">
                    <Checkbox
                      checked={selectedIds.includes(alert.id)}
                      onCheckedChange={() => toggleSelectOne(alert.id)}
                      className="size-3.5"
                    />
                  </div>
                )}

                {/* Severity dot */}
                <div className={`shrink-0 ${alert.is_resolved ? "mt-1.5" : "mt-1.5"}`}>
                  <span className={`block size-2 rounded-full ${getSeverityDot(alert.severity)}`} />
                </div>

                {/* Content */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-[13px] font-medium truncate">{alert.title}</span>
                    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ${getSeverityBadgeClasses(alert.severity)}`}>
                      {alert.severity}
                    </span>
                    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ${getTypeBadgeClasses(alert.type)}`}>
                      {alert.type.replace(/_/g, " ")}
                    </span>
                    {alert.is_resolved ? (
                      <span className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                        <CheckCircle2 className="size-3" /> Resolved
                      </span>
                    ) : alert.is_acknowledged ? (
                      <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium bg-blue-500/10 text-blue-600 dark:text-blue-400">
                        Acknowledged
                      </span>
                    ) : (
                      <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium bg-red-500/10 text-red-600 dark:text-red-400">
                        Active
                      </span>
                    )}
                  </div>

                  <p className="text-[13px] text-muted-foreground mt-1 line-clamp-1">{alert.message}</p>

                  <div className="flex items-center gap-3 mt-2">
                    <span className="text-xs text-muted-foreground">
                      {alert.website_name || `#${alert.website_id}`}
                    </span>
                    <span className="text-xs text-muted-foreground/50">|</span>
                    <span className="text-xs text-muted-foreground">
                      {formatDate(alert.created_at)}
                    </span>
                  </div>
                </div>

                {/* Actions */}
                {!alert.is_resolved && (
                  <div className="flex items-center gap-1.5 shrink-0">
                    {!alert.is_acknowledged && (
                      <Button
                        variant="outline"
                        className="h-7 text-[11px]"
                        disabled={acknowledgeMutation.isPending}
                        onClick={() => acknowledgeMutation.mutate(alert.id)}
                      >
                        {acknowledgeMutation.isPending ? (
                          <Loader2 className="size-3 animate-spin" />
                        ) : (
                          "Acknowledge"
                        )}
                      </Button>
                    )}
                    <Button
                      variant="outline"
                      className="h-7 text-[11px] border-emerald-200 dark:border-emerald-500/20 text-emerald-600 dark:text-emerald-400 hover:bg-emerald-50 dark:hover:bg-emerald-500/10"
                      onClick={() => openResolveDialog(alert)}
                    >
                      Resolve
                    </Button>
                  </div>
                )}
              </div>
            </div>
          ))
        )}
      </div>

      {/* Pagination */}
      {total > 0 && (
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <p className="text-xs text-muted-foreground">
              Menampilkan {(page - 1) * pageSize + 1}–{Math.min(page * pageSize, total)} dari {total} alert
            </p>
            <select
              value={pageSize}
              onChange={(e) => setPageSize(Number(e.target.value))}
              className="h-7 rounded-md border border-input bg-background px-2 text-xs text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            >
              {[25, 50, 100].map((size) => (
                <option key={size} value={size}>{size} / halaman</option>
              ))}
            </select>
          </div>
          {totalPages > 1 && (
            <div className="flex items-center gap-1">
              <Button variant="outline" size="icon" className="size-7" disabled={page <= 1} onClick={() => setPage(page - 1)}>
                <ChevronLeft className="size-3.5" />
              </Button>
              <span className="text-xs text-muted-foreground px-2">{page} / {totalPages}</span>
              <Button variant="outline" size="icon" className="size-7" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>
                <ChevronRight className="size-3.5" />
              </Button>
            </div>
          )}
        </div>
      )}

      {/* Single Resolve Dialog */}
      <Dialog open={resolveDialogOpen} onOpenChange={setResolveDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-sm font-medium">Resolve Alert</DialogTitle>
            <DialogDescription className="text-xs">
              {resolveTarget?.title} — {resolveTarget?.website_name}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <div className="space-y-1.5">
              <Label htmlFor="resolve-note" className="text-xs">Resolution Note</Label>
              <Textarea
                id="resolve-note"
                placeholder="Jelaskan bagaimana alert ini di-resolve..."
                value={resolveNote}
                onChange={(e) => setResolveNote(e.target.value)}
                rows={3}
                className="text-[13px]"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" className="h-8 text-xs" onClick={() => setResolveDialogOpen(false)}>Batal</Button>
            <Button
              className="h-8 text-xs"
              onClick={() => resolveTarget && resolveMutation.mutate({ id: resolveTarget.id, note: resolveNote || undefined })}
              disabled={resolveMutation.isPending}
            >
              {resolveMutation.isPending && <Loader2 className="size-3 animate-spin mr-1.5" />}
              Resolve
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Bulk Resolve Dialog */}
      <Dialog open={bulkResolveDialogOpen} onOpenChange={setBulkResolveDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-sm font-medium">Bulk Resolve Alerts</DialogTitle>
            <DialogDescription className="text-xs">
              Resolve {selectedIds.length} alert yang dipilih sekaligus.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <div className="space-y-1.5">
              <Label htmlFor="bulk-resolve-note" className="text-xs">Resolution Note (opsional)</Label>
              <Textarea
                id="bulk-resolve-note"
                placeholder="Jelaskan alasan resolve alert ini..."
                value={bulkResolveNote}
                onChange={(e) => setBulkResolveNote(e.target.value)}
                rows={3}
                className="text-[13px]"
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              className="h-8 text-xs"
              onClick={() => setBulkResolveDialogOpen(false)}
            >
              Batal
            </Button>
            <Button
              className="h-8 text-xs"
              onClick={() =>
                bulkResolveMutation.mutate({ ids: selectedIds, note: bulkResolveNote })
              }
              disabled={bulkResolveMutation.isPending}
            >
              {bulkResolveMutation.isPending && (
                <Loader2 className="size-3 animate-spin mr-1.5" />
              )}
              Resolve {selectedIds.length} Alert
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
