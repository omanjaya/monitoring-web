"use client"

import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { websiteApi, opdApi } from "@/lib/api"
import type { Website, OPD } from "@/lib/api"
import { usePaginatedQuery } from "@/hooks/use-paginated-query"
import { useMutationAction } from "@/hooks/use-mutation-action"
import { formatRelativeTime } from "@/lib/utils"
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { WebsiteDetailDialog } from "@/components/websites/website-detail-dialog"
import { WebsiteFormDialog } from "@/components/websites/website-form-dialog"
import { BulkImportDialog } from "@/components/bulk-import-dialog"
import type { WebsiteCreate } from "@/lib/api"
import {
  Plus,
  Search,
  MoreHorizontal,
  Pencil,
  Trash2,
  Power,
  PowerOff,
  CheckCircle2,
  XCircle,
  Globe,
  ChevronLeft,
  ChevronRight,
  Loader2,
  AlertTriangle,
  Upload,
  Download,
} from "lucide-react"

interface WebsiteFilters {
  search: string
  status: string
  opd_id: string
  is_active: string
}

export default function WebsitesPage() {
  const {
    data: websites,
    total,
    totalPages,
    page,
    setPage,
    filters,
    setFilter,
    isLoading,
    refetch,
    pageSize,
    setPageSize,
  } = usePaginatedQuery<Website, WebsiteFilters>({
    queryKey: "websites",
    queryFn: (params) => websiteApi.list(params),
    pageSize: 25,
    initialFilters: { search: "", status: "all", opd_id: "all", is_active: "all" },
  })

  const { data: opdList = [] } = useQuery<OPD[]>({
    queryKey: ["opd-list"],
    queryFn: async () => {
      const res = await opdApi.list()
      return res.data || []
    },
  })

  // Selection
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())

  // Dialogs
  const [detailId, setDetailId] = useState<number | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)
  const [formOpen, setFormOpen] = useState(false)
  const [editingWebsite, setEditingWebsite] = useState<Website | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null)
  const [bulkAction, setBulkAction] = useState<string | null>(null)
  const [bulkImportOpen, setBulkImportOpen] = useState(false)

  const deleteMutation = useMutationAction({
    mutationFn: (id: number) => websiteApi.delete(id),
    successMessage: "Website berhasil dihapus",
    invalidateKeys: ["websites"],
    onSuccess: () => setDeleteConfirm(null),
  })

  const bulkMutation = useMutationAction({
    mutationFn: async ({ ids, action }: { ids: number[]; action: string }) => {
      if (action === "delete") {
        for (const id of ids) await websiteApi.delete(id)
        return { affected: ids.length }
      }
      const res = await websiteApi.bulkAction(ids, action)
      return res.data
    },
    successMessage: "Aksi bulk berhasil",
    invalidateKeys: ["websites"],
    onSuccess: () => {
      setSelectedIds(new Set())
      setBulkAction(null)
    },
  })

  const [isExporting, setIsExporting] = useState(false)

  const handleExportCSV = async () => {
    setIsExporting(true)
    try {
      const res = await websiteApi.list("limit=1000")
      const data = res.data || []
      const headers = ["ID", "Nama", "URL", "OPD", "Status", "SSL Expiry", "Terakhir Dicek"]
      const rows = data.map((w) => [
        w.id,
        w.name,
        w.url,
        w.opd_name || "-",
        w.status || "-",
        w.ssl_expiry_date ? new Date(w.ssl_expiry_date).toLocaleDateString("id-ID") : "-",
        w.last_checked_at ? new Date(w.last_checked_at).toLocaleString("id-ID") : "-",
      ])
      const csv = [headers, ...rows].map((r) => r.map((v) => `"${v}"`).join(",")).join("\n")
      const blob = new Blob(["\uFEFF" + csv], { type: "text/csv;charset=utf-8" })
      const url = URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = url
      a.download = `websites-${new Date().toISOString().split("T")[0]}.csv`
      a.click()
      URL.revokeObjectURL(url)
    } finally {
      setIsExporting(false)
    }
  }

  const allSelected =
    websites.length > 0 && websites.every((w) => selectedIds.has(w.id))

  const toggleSelectAll = () => {
    setSelectedIds(allSelected ? new Set() : new Set(websites.map((w) => w.id)))
  }

  const toggleSelect = (id: number) => {
    const next = new Set(selectedIds)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    setSelectedIds(next)
  }

  const openEdit = (website: Website) => {
    setEditingWebsite(website)
    setFormOpen(true)
  }

  const openCreate = () => {
    setEditingWebsite(null)
    setFormOpen(true)
  }

  const getResponseTimeColor = (ms: number | null | undefined) => {
    if (ms == null) return "text-muted-foreground"
    if (ms < 1000) return "text-emerald-600 dark:text-emerald-400"
    if (ms < 3000) return "text-amber-600 dark:text-amber-400"
    return "text-red-600 dark:text-red-400"
  }

  const getStatusDot = (status: string) => {
    switch (status) {
      case "up":
        return "bg-emerald-500"
      case "down":
        return "bg-red-500"
      case "degraded":
        return "bg-amber-500"
      default:
        return "bg-muted-foreground"
    }
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Websites</h1>
          <p className="text-sm text-muted-foreground">
            Kelola dan monitor website yang dipantau
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={handleExportCSV} disabled={isExporting} className="h-8 text-xs">
            {isExporting ? (
              <Loader2 className="size-3.5 mr-1 animate-spin" />
            ) : (
              <Download className="size-3.5 mr-1" />
            )}
            Export CSV
          </Button>
          <Button variant="outline" onClick={() => setBulkImportOpen(true)} className="h-8 text-xs">
            <Upload className="size-3.5 mr-1" />
            Bulk Import
          </Button>
          <Button onClick={openCreate} className="h-8 text-xs">
            <Plus className="size-3.5 mr-1" />
            Add Website
          </Button>
        </div>
      </div>

      {/* Filter Bar */}
      <div className="flex items-center gap-2 flex-wrap">
        <div className="relative flex-1 min-w-[200px] max-w-xs">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground" />
          <Input
            placeholder="Cari nama atau URL..."
            value={filters.search}
            onChange={(e) => setFilter("search", e.target.value)}
            className="pl-8 h-8 text-xs"
          />
        </div>

        <Select value={filters.status} onValueChange={(v) => setFilter("status", v)}>
          <SelectTrigger className="w-[130px] h-8 text-xs">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua Status</SelectItem>
            <SelectItem value="up">Up</SelectItem>
            <SelectItem value="down">Down</SelectItem>
            <SelectItem value="degraded">Degraded</SelectItem>
          </SelectContent>
        </Select>

        <Select value={filters.opd_id} onValueChange={(v) => setFilter("opd_id", v)}>
          <SelectTrigger className="w-[160px] h-8 text-xs">
            <SelectValue placeholder="OPD" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua OPD</SelectItem>
            {opdList.map((opd) => (
              <SelectItem key={opd.id} value={String(opd.id)}>
                {opd.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select value={filters.is_active} onValueChange={(v) => setFilter("is_active", v)}>
          <SelectTrigger className="w-[120px] h-8 text-xs">
            <SelectValue placeholder="Aktif" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Semua</SelectItem>
            <SelectItem value="true">Aktif</SelectItem>
            <SelectItem value="false">Nonaktif</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Bulk Action Confirmation */}
      {bulkAction && (
        <div className="flex items-center gap-3 rounded-lg border border-red-200 dark:border-red-500/20 bg-red-50 dark:bg-red-500/5 px-4 py-3">
          <AlertTriangle className="size-4 text-red-500 shrink-0" />
          <span className="text-[13px]">
            Yakin ingin <strong>{bulkAction}</strong> {selectedIds.size} website?
          </span>
          <div className="ml-auto flex gap-2">
            <Button variant="outline" className="h-7 text-[11px]" onClick={() => setBulkAction(null)}>
              Batal
            </Button>
            <Button
              variant="destructive"
              className="h-7 text-[11px]"
              disabled={bulkMutation.isPending}
              onClick={() => bulkMutation.mutate({ ids: Array.from(selectedIds), action: bulkAction })}
            >
              {bulkMutation.isPending && <Loader2 className="size-3 animate-spin mr-1" />}
              Konfirmasi
            </Button>
          </div>
        </div>
      )}

      {/* Table */}
      <div className="rounded-lg border border-border/50">
        <Table>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead className="w-10">
                <Checkbox checked={allSelected} onCheckedChange={toggleSelectAll} aria-label="Select all" />
              </TableHead>
              <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Nama</TableHead>
              <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">URL</TableHead>
              <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Status</TableHead>
              <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Response</TableHead>
              <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">SSL</TableHead>
              <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Content</TableHead>
              <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Terakhir Dicek</TableHead>
              <TableHead className="w-10" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={9} className="h-32 text-center">
                  <Loader2 className="size-5 animate-spin mx-auto text-muted-foreground" />
                </TableCell>
              </TableRow>
            ) : websites.length === 0 ? (
              <TableRow>
                <TableCell colSpan={9} className="h-40 text-center">
                  <div className="flex flex-col items-center gap-3">
                    <div className="rounded-full bg-muted p-3">
                      <Globe className="size-5 text-muted-foreground" />
                    </div>
                    <div>
                      <p className="text-sm font-medium">Tidak ada website ditemukan</p>
                      <p className="text-xs text-muted-foreground mt-0.5">Tambahkan website untuk mulai monitoring</p>
                    </div>
                    <Button variant="outline" className="h-7 text-[11px]" onClick={openCreate}>
                      <Plus className="size-3 mr-1" /> Tambah Website
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              websites.map((website) => (
                <TableRow
                  key={website.id}
                  className="hover:bg-muted/30"
                  data-state={selectedIds.has(website.id) ? "selected" : undefined}
                >
                  <TableCell>
                    <Checkbox
                      checked={selectedIds.has(website.id)}
                      onCheckedChange={() => toggleSelect(website.id)}
                      aria-label={`Select ${website.name}`}
                    />
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <button
                        className="text-[13px] font-medium hover:underline text-left"
                        onClick={() => { setDetailId(website.id); setDetailOpen(true) }}
                      >
                        {website.name}
                      </button>
                      {!website.is_active && (
                        <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium bg-muted text-muted-foreground">
                          Nonaktif
                        </span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="text-[13px] text-muted-foreground truncate max-w-[200px] block">
                      {website.url}
                    </span>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1.5">
                      <span className={`size-2 rounded-full ${getStatusDot(website.status)}`} />
                      <span className="text-[13px] capitalize">{website.status}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className={`text-[13px] font-medium tabular-nums ${getResponseTimeColor(website.last_response_time)}`}>
                      {website.last_response_time != null ? `${website.last_response_time}ms` : "-"}
                    </span>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      {website.ssl_valid === true ? (
                        <CheckCircle2 className="size-4 text-emerald-500" />
                      ) : website.ssl_valid === false ? (
                        <XCircle className="size-4 text-red-500" />
                      ) : (
                        <span className="text-[13px] text-muted-foreground">-</span>
                      )}
                      {website.ssl_expiry_date && (
                        <span className="text-[11px] text-muted-foreground ml-1">
                          {new Date(website.ssl_expiry_date).toLocaleDateString("id-ID")}
                        </span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    {website.content_clean === true ? (
                      <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium bg-emerald-500/10 text-emerald-600 dark:text-emerald-400">
                        Clean
                      </span>
                    ) : website.content_clean === false ? (
                      <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium bg-red-500/10 text-red-600 dark:text-red-400">
                        Issue
                      </span>
                    ) : (
                      <span className="text-[13px] text-muted-foreground">-</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <span className="text-[13px] text-muted-foreground">
                      {formatRelativeTime(website.last_checked_at)}
                    </span>
                  </TableCell>
                  <TableCell>
                    {deleteConfirm === website.id ? (
                      <div className="flex items-center gap-1">
                        <Button
                          variant="destructive"
                          className="h-7 text-[11px]"
                          disabled={deleteMutation.isPending}
                          onClick={() => deleteMutation.mutate(website.id)}
                        >
                          {deleteMutation.isPending ? <Loader2 className="size-3 animate-spin" /> : "Ya"}
                        </Button>
                        <Button variant="outline" className="h-7 text-[11px]" onClick={() => setDeleteConfirm(null)}>
                          Batal
                        </Button>
                      </div>
                    ) : (
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon" className="size-7">
                            <MoreHorizontal className="size-4" />
                            <span className="sr-only">Actions</span>
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => openEdit(website)}>
                            <Pencil className="size-3.5 mr-2" /> Edit
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            className="text-destructive focus:text-destructive"
                            onClick={() => setDeleteConfirm(website.id)}
                          >
                            <Trash2 className="size-3.5 mr-2" /> Hapus
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    )}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* Pagination */}
      {total > 0 && (
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <p className="text-xs text-muted-foreground">
              Menampilkan {(page - 1) * pageSize + 1}–{Math.min(page * pageSize, total)} dari {total} website
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
              {Array.from({ length: totalPages }, (_, i) => i + 1)
                .filter((p) => p === 1 || p === totalPages || (p >= page - 1 && p <= page + 1))
                .reduce<(number | string)[]>((acc, p, idx, arr) => {
                  if (idx > 0 && p - (arr[idx - 1] as number) > 1) acc.push("...")
                  acc.push(p)
                  return acc
                }, [])
                .map((item, idx) =>
                  item === "..." ? (
                    <span key={`ellipsis-${idx}`} className="px-1.5 text-xs text-muted-foreground">...</span>
                  ) : (
                    <Button
                      key={item}
                      variant={page === item ? "default" : "outline"}
                      size="icon"
                      className="size-7 text-xs"
                      onClick={() => setPage(item as number)}
                    >
                      {item}
                    </Button>
                  )
                )}
              <Button variant="outline" size="icon" className="size-7" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>
                <ChevronRight className="size-3.5" />
              </Button>
            </div>
          )}
        </div>
      )}

      {/* Floating Bulk Action Bar */}
      {selectedIds.size > 0 && !bulkAction && (
        <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50">
          <div className="flex items-center gap-3 rounded-xl border border-border/50 bg-background/95 backdrop-blur-sm shadow-lg px-5 py-3">
            <span className="text-[13px] font-medium">
              {selectedIds.size} dipilih
            </span>
            <div className="h-4 w-px bg-border" />
            <div className="flex items-center gap-1.5">
              <Button variant="outline" className="h-7 text-[11px]" onClick={() => setBulkAction("enable")}>
                <Power className="size-3 mr-1" /> Enable
              </Button>
              <Button variant="outline" className="h-7 text-[11px]" onClick={() => setBulkAction("disable")}>
                <PowerOff className="size-3 mr-1" /> Disable
              </Button>
              <Button variant="destructive" className="h-7 text-[11px]" onClick={() => setBulkAction("delete")}>
                <Trash2 className="size-3 mr-1" /> Delete
              </Button>
            </div>
            <div className="h-4 w-px bg-border" />
            <Button
              variant="ghost"
              className="h-7 text-[11px] text-muted-foreground"
              onClick={() => setSelectedIds(new Set())}
            >
              Batal
            </Button>
          </div>
        </div>
      )}

      {/* Dialogs */}
      <WebsiteDetailDialog websiteId={detailId} open={detailOpen} onOpenChange={setDetailOpen} />
      <WebsiteFormDialog website={editingWebsite} open={formOpen} onOpenChange={setFormOpen} onSuccess={() => refetch()} />
      <BulkImportDialog<WebsiteCreate>
        open={bulkImportOpen}
        onOpenChange={setBulkImportOpen}
        onImport={async (data) => {
          const res = await websiteApi.bulkImport(data)
          return res.data
        }}
        title="Bulk Import Websites"
        description="Import banyak website sekaligus via CSV"
        columns={[
          { key: "url", label: "URL" },
          { key: "name", label: "Nama" },
          { key: "description", label: "Deskripsi" },
        ]}
        exampleRows={[
          ["https://baliprov.go.id", "Portal Pemprov Bali", "Website utama"],
          ["https://diskominfos.baliprov.go.id", "Diskominfos Bali", "Dinas Kominfos"],
        ]}
        parseRow={(row) => {
          if (!row[0]) return null
          let rawUrl = row[0].trim()
          // Auto-prepend https:// if no scheme
          if (!/^https?:\/\//i.test(rawUrl)) {
            rawUrl = "https://" + rawUrl
          }
          let name = row[1]
          if (!name) {
            try {
              name = new URL(rawUrl).hostname
            } catch {
              name = rawUrl
            }
          }
          return {
            url: rawUrl,
            name,
            description: row[2] || undefined,
          } as WebsiteCreate
        }}
        onSuccess={() => refetch()}
      />
    </div>
  )
}
