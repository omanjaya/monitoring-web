"use client"

import { useState } from "react"
import {
  Wrench,
  RefreshCw,
  AlertCircle,
  Plus,
  Ban,
  CheckCircle2,
  Trash2,
  Pencil,
  ArrowRight,
  Calendar,
  Loader2,
  Clock,
  Zap,
} from "lucide-react"
import { toast } from "sonner"
import { useQuery } from "@tanstack/react-query"

import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Checkbox } from "@/components/ui/checkbox"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { maintenanceApi, websiteApi, type Maintenance } from "@/lib/api"
import { formatDate } from "@/lib/utils"
import { useMutationAction } from "@/hooks/use-mutation-action"

const statusConfig: Record<string, { border: string; bg: string; text: string; label: string; icon: typeof Zap }> = {
  scheduled: {
    border: "border-l-blue-500",
    bg: "bg-blue-500/10",
    text: "text-blue-700 dark:text-blue-400",
    label: "Scheduled",
    icon: Calendar,
  },
  in_progress: {
    border: "border-l-amber-500",
    bg: "bg-amber-500/10",
    text: "text-amber-700 dark:text-amber-400",
    label: "In Progress",
    icon: Zap,
  },
  completed: {
    border: "border-l-emerald-500",
    bg: "bg-emerald-500/10",
    text: "text-emerald-700 dark:text-emerald-400",
    label: "Completed",
    icon: CheckCircle2,
  },
  cancelled: {
    border: "border-l-gray-400",
    bg: "bg-gray-500/10",
    text: "text-gray-600 dark:text-gray-400",
    label: "Cancelled",
    icon: Ban,
  },
}

function getStatus(status: string) {
  return statusConfig[status] || statusConfig.cancelled
}

function MaintenanceCard({
  item,
  onComplete,
  onCancel,
  onEdit,
  onDelete,
  compact = false,
}: {
  item: Maintenance
  onComplete: (id: number) => void
  onCancel: (id: number) => void
  onEdit: (item: Maintenance) => void
  onDelete: (id: number) => void
  compact?: boolean
}) {
  const status = getStatus(item.status)
  const isActive = item.status === "scheduled" || item.status === "in_progress"

  return (
    <Card
      className={`border-border/50 border-l-[3px] ${status.border} overflow-hidden`}
    >
      <CardContent className={compact ? "px-4 py-3" : "px-5 py-4"}>
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1 min-w-0 space-y-1.5">
            <div className="flex items-center gap-2.5">
              <span className={`${compact ? "text-xs" : "text-[13px]"} font-medium truncate`}>{item.title}</span>
              <span
                className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ${status.bg} ${status.text}`}
              >
                {status.label}
              </span>
            </div>
            {item.description && (
              <p className="text-xs text-muted-foreground line-clamp-1">
                {item.description}
              </p>
            )}
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <Calendar className="h-3 w-3 shrink-0" />
              <span>{formatDate(item.scheduled_start)}</span>
              <ArrowRight className="h-3 w-3 shrink-0 text-muted-foreground/50" />
              <span>{formatDate(item.scheduled_end)}</span>
            </div>
          </div>
          <div className="flex items-center gap-1 shrink-0">
            {isActive && (
              <>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onComplete(item.id)}
                  className="h-7 text-[11px] text-emerald-600 hover:text-emerald-700 hover:bg-emerald-500/10 px-2"
                >
                  <CheckCircle2 className="h-3.5 w-3.5 mr-1" />
                  Selesai
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onCancel(item.id)}
                  className="h-7 text-[11px] text-amber-600 hover:text-amber-700 hover:bg-amber-500/10 px-2"
                >
                  <Ban className="h-3.5 w-3.5 mr-1" />
                  Batal
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onEdit(item)}
                  className="h-7 text-[11px] px-2"
                >
                  <Pencil className="h-3.5 w-3.5 mr-1" />
                  Edit
                </Button>
              </>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={() => onDelete(item.id)}
              className="h-7 px-2 text-red-500 hover:text-red-600 hover:bg-red-500/10"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

export default function MaintenancePage() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingItem, setEditingItem] = useState<Maintenance | null>(null)

  const [form, setForm] = useState<{
    title: string;
    description: string;
    scheduled_start: string;
    scheduled_end: string;
    website_ids: number[];
  }>({
    title: "",
    description: "",
    scheduled_start: "",
    scheduled_end: "",
    website_ids: [],
  })

  const { data: items = [], isLoading: loadingMaintenance, error: maintenanceError, refetch } = useQuery({
    queryKey: ["maintenance"],
    queryFn: async () => {
      const res = await maintenanceApi.list()
      return (res.data || []) as Maintenance[]
    },
  })

  const { data: currentItems = [] } = useQuery({
    queryKey: ["maintenance-current"],
    queryFn: async () => {
      const res = await maintenanceApi.current()
      return (res.data || []) as Maintenance[]
    },
  })

  const { data: upcomingItems = [] } = useQuery({
    queryKey: ["maintenance-upcoming"],
    queryFn: async () => {
      const res = await maintenanceApi.upcoming()
      return (res.data || []) as Maintenance[]
    },
  })

  const { data: websites = [] } = useQuery({
    queryKey: ["maintenance-websites"],
    queryFn: async () => {
      const res = await websiteApi.list("limit=100")
      return (res.data || []) as { id: number; name: string }[]
    },
  })

  const loading = loadingMaintenance
  const error = maintenanceError

  const invalidateKeys = ["maintenance", "maintenance-current", "maintenance-upcoming"]

  const saveMutation = useMutationAction<unknown, { editingId?: number; payload: { title: string; description?: string; scheduled_start: string; scheduled_end: string; website_ids: number[] } }>({
    mutationFn: async ({ editingId, payload }) => {
      if (editingId) {
        return maintenanceApi.update(editingId, payload)
      }
      return maintenanceApi.create(payload)
    },
    successMessage: undefined,
    invalidateKeys,
    onSuccess: () => {
      toast.success(editingItem ? "Maintenance berhasil diperbarui" : "Maintenance berhasil dibuat")
      setDialogOpen(false)
    },
  })

  const cancelMutation = useMutationAction({
    mutationFn: (id: number) => maintenanceApi.cancel(id),
    successMessage: "Maintenance dibatalkan",
    invalidateKeys,
  })

  const completeMutation = useMutationAction({
    mutationFn: (id: number) => maintenanceApi.complete(id),
    successMessage: "Maintenance selesai",
    invalidateKeys,
  })

  const deleteMutation = useMutationAction({
    mutationFn: (id: number) => maintenanceApi.delete(id),
    successMessage: "Maintenance dihapus",
    invalidateKeys,
  })

  const openCreate = () => {
    setEditingItem(null)
    setForm({ title: "", description: "", scheduled_start: "", scheduled_end: "", website_ids: [] })
    setDialogOpen(true)
  }

  const openEdit = (item: Maintenance) => {
    setEditingItem(item)
    setForm({
      title: item.title,
      description: item.description || "",
      scheduled_start: item.scheduled_start?.slice(0, 16) || "",
      scheduled_end: item.scheduled_end?.slice(0, 16) || "",
      website_ids: ((item as unknown) as Record<string, unknown>).website_ids as number[] || [],
    })
    setDialogOpen(true)
  }

  const handleSubmit = () => {
    if (!form.title || !form.scheduled_start || !form.scheduled_end) {
      toast.error("Judul, tanggal mulai, dan tanggal selesai harus diisi")
      return
    }
    const payload = {
      title: form.title,
      description: form.description || undefined,
      scheduled_start: new Date(form.scheduled_start).toISOString(),
      scheduled_end: new Date(form.scheduled_end).toISOString(),
      website_ids: form.website_ids,
    }
    saveMutation.mutate({ editingId: editingItem?.id, payload })
  }

  const handleCancel = (id: number) => {
    cancelMutation.mutate(id)
  }

  const handleComplete = (id: number) => {
    completeMutation.mutate(id)
  }

  const handleDelete = (id: number) => {
    deleteMutation.mutate(id)
  }

  if (loading) return <MaintenanceSkeleton />

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-20">
        <div className="rounded-full bg-red-500/10 p-3">
          <AlertCircle className="h-5 w-5 text-red-500" />
        </div>
        <h2 className="text-sm font-medium">Gagal Memuat Data Maintenance</h2>
        <p className="text-xs text-muted-foreground">{error instanceof Error ? error.message : "Gagal memuat data"}</p>
        <Button onClick={() => refetch()} variant="outline" className="h-8 text-xs mt-1">
          <RefreshCw className="mr-1.5 h-3 w-3" />
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
          <h1 className="text-xl font-semibold tracking-tight">Maintenance</h1>
          <p className="text-xs text-muted-foreground mt-0.5">
            Kelola jadwal maintenance website
          </p>
        </div>
        <Button onClick={openCreate} className="h-8 text-xs">
          <Plus className="mr-1.5 h-3.5 w-3.5" />
          Buat Maintenance
        </Button>
      </div>

      {/* Current Active Maintenance */}
      {currentItems.length > 0 && (
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <div className="h-2 w-2 rounded-full bg-amber-500 animate-pulse" />
            <h2 className="text-sm font-medium">Sedang Berlangsung</h2>
            <span className="inline-flex items-center justify-center rounded-full bg-amber-500/10 text-amber-700 dark:text-amber-400 px-1.5 py-0.5 text-[10px] font-medium">
              {currentItems.length}
            </span>
          </div>
          <div className="space-y-2">
            {currentItems.map((item) => (
              <MaintenanceCard
                key={`current-${item.id}`}
                item={item}
                onComplete={handleComplete}
                onCancel={handleCancel}
                onEdit={openEdit}
                onDelete={handleDelete}
              />
            ))}
          </div>
        </div>
      )}

      {/* Upcoming Maintenance */}
      {upcomingItems.length > 0 && (
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <Clock className="h-3.5 w-3.5 text-blue-500" />
            <h2 className="text-sm font-medium">Akan Datang</h2>
            <span className="inline-flex items-center justify-center rounded-full bg-blue-500/10 text-blue-700 dark:text-blue-400 px-1.5 py-0.5 text-[10px] font-medium">
              {upcomingItems.length}
            </span>
          </div>
          <div className="space-y-2">
            {upcomingItems.map((item) => (
              <MaintenanceCard
                key={`upcoming-${item.id}`}
                item={item}
                onComplete={handleComplete}
                onCancel={handleCancel}
                onEdit={openEdit}
                onDelete={handleDelete}
                compact
              />
            ))}
          </div>
        </div>
      )}

      {/* Separator if there are current/upcoming items */}
      {(currentItems.length > 0 || upcomingItems.length > 0) && items.length > 0 && (
        <div className="border-b border-border/50" />
      )}

      {/* All Maintenance Items */}
      <div className="space-y-2">
        {(currentItems.length > 0 || upcomingItems.length > 0) && (
          <h2 className="text-sm font-medium">Semua Maintenance</h2>
        )}
        {items.length > 0 ? (
          <div className="space-y-2">
            {items.map((item) => (
              <MaintenanceCard
                key={item.id}
                item={item}
                onComplete={handleComplete}
                onCancel={handleCancel}
                onEdit={openEdit}
                onDelete={handleDelete}
              />
            ))}
          </div>
        ) : (
          <Card className="border-border/50">
            <CardContent className="py-16">
              <div className="flex flex-col items-center justify-center gap-2">
                <div className="rounded-full bg-muted p-3">
                  <Wrench className="h-5 w-5 text-muted-foreground" />
                </div>
                <p className="text-sm font-medium">Belum ada jadwal maintenance</p>
                <p className="text-xs text-muted-foreground">Buat jadwal maintenance baru untuk memulai</p>
              </div>
            </CardContent>
          </Card>
        )}
      </div>

      {/* Create/Edit Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-[480px]">
          <DialogHeader>
            <DialogTitle className="text-sm font-medium">
              {editingItem ? "Edit Maintenance" : "Buat Maintenance"}
            </DialogTitle>
            <DialogDescription className="text-xs">
              {editingItem
                ? "Perbarui jadwal maintenance"
                : "Buat jadwal maintenance baru"}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="title" className="text-xs">Judul</Label>
              <Input
                id="title"
                value={form.title}
                onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
                placeholder="Judul maintenance"
                className="h-8 text-[13px]"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="description" className="text-xs">Deskripsi</Label>
              <Textarea
                id="description"
                value={form.description}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                placeholder="Deskripsi maintenance (opsional)"
                rows={3}
                className="text-[13px] resize-none"
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label htmlFor="start" className="text-xs">Mulai</Label>
                <Input
                  id="start"
                  type="datetime-local"
                  value={form.scheduled_start}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, scheduled_start: e.target.value }))
                  }
                  className="h-8 text-[13px]"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="end" className="text-xs">Selesai</Label>
                <Input
                  id="end"
                  type="datetime-local"
                  value={form.scheduled_end}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, scheduled_end: e.target.value }))
                  }
                  className="h-8 text-[13px]"
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <Label className="text-xs">Target Website</Label>
              {websites.length > 0 ? (
                <ScrollArea className="h-[140px] rounded-md border border-border/50 p-3">
                  <div className="space-y-2.5">
                    {websites.map((website) => (
                      <div key={website.id} className="flex items-center space-x-2">
                        <Checkbox
                          id={`web-${website.id}`}
                          checked={form.website_ids.includes(website.id)}
                          onCheckedChange={(checked) => {
                            if (checked) {
                              setForm(f => ({ ...f, website_ids: [...f.website_ids, website.id] }))
                            } else {
                              setForm(f => ({ ...f, website_ids: f.website_ids.filter(id => id !== website.id) }))
                            }
                          }}
                        />
                        <Label
                          htmlFor={`web-${website.id}`}
                          className="text-[13px] font-normal cursor-pointer flex-1"
                        >
                          {website.name}
                        </Label>
                      </div>
                    ))}
                  </div>
                </ScrollArea>
              ) : (
                <div className="text-xs text-muted-foreground p-3 border border-border/50 rounded-md">
                  Memuat website...
                </div>
              )}
              {form.website_ids.length > 0 && (
                <p className="text-[11px] text-muted-foreground">
                  {form.website_ids.length} website dipilih
                </p>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)} className="h-8 text-xs">
              Batal
            </Button>
            <Button onClick={handleSubmit} disabled={saveMutation.isPending} className="h-8 text-xs">
              {saveMutation.isPending && <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />}
              {editingItem ? "Perbarui" : "Buat"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function MaintenanceSkeleton() {
  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <Skeleton className="h-6 w-32 mb-1" />
          <Skeleton className="h-3 w-52" />
        </div>
        <Skeleton className="h-8 w-36" />
      </div>
      {/* Current/Upcoming skeleton */}
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <Skeleton className="h-2 w-2 rounded-full" />
          <Skeleton className="h-4 w-32" />
        </div>
        <Card className="border-border/50 border-l-[3px] border-l-muted">
          <CardContent className="px-5 py-4">
            <div className="flex items-start justify-between">
              <div className="space-y-2 flex-1">
                <div className="flex items-center gap-2">
                  <Skeleton className="h-4 w-40" />
                  <Skeleton className="h-5 w-20 rounded-full" />
                </div>
                <Skeleton className="h-3 w-48" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
      <div className="space-y-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <Card key={i} className="border-border/50 border-l-[3px] border-l-muted">
            <CardContent className="px-5 py-4">
              <div className="flex items-start justify-between">
                <div className="space-y-2 flex-1">
                  <div className="flex items-center gap-2">
                    <Skeleton className="h-4 w-40" />
                    <Skeleton className="h-5 w-20 rounded-full" />
                  </div>
                  <Skeleton className="h-3 w-64" />
                  <Skeleton className="h-3 w-48" />
                </div>
                <div className="flex gap-1">
                  <Skeleton className="h-7 w-16" />
                  <Skeleton className="h-7 w-7" />
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
