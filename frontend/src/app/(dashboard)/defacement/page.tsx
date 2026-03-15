"use client"

import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  RefreshCw,
  AlertCircle,
  AlertTriangle,
  CheckCircle2,
  Shield,
  Skull,
  ExternalLink,
  Clock,
  Loader2,
  Radar,
  Filter,
  Search,
  Eye,
  User,
  Users,
} from "lucide-react"
import { toast } from "sonner"

import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
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
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog"
import { Textarea } from "@/components/ui/textarea"

import { defacementApi } from "@/lib/api"
import { formatDate } from "@/lib/utils"

interface DefacementStats {
  total_incidents: number
  unacknowledged_count: number
  websites_affected: number
  by_source: Record<string, number>
  recent_incidents?: DefacementIncident[]
  last_scan_at?: string
}

interface DefacementIncident {
  id: number
  website_id: number
  website_name?: string
  website_url?: string
  source: string
  source_id?: string
  defaced_url: string
  attacker?: string
  team?: string
  defaced_at?: string
  mirror_url?: string
  is_acknowledged: boolean
  acknowledged_at?: string
  acknowledged_by?: string
  notes?: string
  created_at: string
}

export default function DefacementPage() {
  const [search, setSearch] = useState("")
  const [filterSource, setFilterSource] = useState<string>("all")
  const [filterAck, setFilterAck] = useState<string>("all")
  const [scanLoading, setScanLoading] = useState(false)
  const [ackDialog, setAckDialog] = useState<DefacementIncident | null>(null)
  const [ackNotes, setAckNotes] = useState("")

  const queryClient = useQueryClient()

  const statsQuery = useQuery({
    queryKey: ["defacement-stats"],
    queryFn: () => defacementApi.stats(),
  })

  const incidentsQuery = useQuery({
    queryKey: ["defacement-incidents"],
    queryFn: () => defacementApi.incidents(),
  })

  const ackMutation = useMutation({
    mutationFn: ({ id, notes }: { id: number; notes: string }) =>
      defacementApi.acknowledge(id, notes),
    onSuccess: () => {
      toast.success("Incident berhasil di-acknowledge")
      queryClient.invalidateQueries({ queryKey: ["defacement-incidents"] })
      queryClient.invalidateQueries({ queryKey: ["defacement-stats"] })
      setAckDialog(null)
      setAckNotes("")
    },
    onError: () => {
      toast.error("Gagal acknowledge incident")
    },
  })

  const stats = (statsQuery.data?.data as DefacementStats) || null
  const incidents = (incidentsQuery.data?.data as DefacementIncident[] | null) || []
  const isLoading = statsQuery.isLoading || incidentsQuery.isLoading
  const error = statsQuery.error || incidentsQuery.error

  const handleScan = async () => {
    try {
      setScanLoading(true)
      await defacementApi.scan()
      toast.success("Defacement scan dimulai")
      setTimeout(() => {
        statsQuery.refetch()
        incidentsQuery.refetch()
      }, 5000)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menjalankan scan")
    } finally {
      setScanLoading(false)
    }
  }

  if (isLoading) return <DefacementSkeleton />

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-20">
        <div className="rounded-full bg-red-500/10 p-3">
          <AlertCircle className="h-5 w-5 text-red-500" />
        </div>
        <h2 className="text-sm font-medium">Gagal Memuat Data Defacement</h2>
        <p className="text-xs text-muted-foreground">
          {error instanceof Error ? error.message : "Gagal memuat data"}
        </p>
        <Button
          onClick={() => {
            statsQuery.refetch()
            incidentsQuery.refetch()
          }}
          variant="outline"
          className="h-8 text-xs"
        >
          <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
          Coba Lagi
        </Button>
      </div>
    )
  }

  const filteredIncidents = incidents.filter((inc) => {
    if (search) {
      const q = search.toLowerCase()
      const matchUrl = inc.defaced_url?.toLowerCase().includes(q)
      const matchAttacker = inc.attacker?.toLowerCase().includes(q)
      const matchTeam = inc.team?.toLowerCase().includes(q)
      const matchWebsite = inc.website_name?.toLowerCase().includes(q)
      if (!matchUrl && !matchAttacker && !matchTeam && !matchWebsite) return false
    }
    if (filterSource !== "all" && inc.source !== filterSource) return false
    if (filterAck === "yes" && !inc.is_acknowledged) return false
    if (filterAck === "no" && inc.is_acknowledged) return false
    return true
  })

  const sourceLabel = (s: string) => {
    switch (s) {
      case "zone_h":
        return "Zone-H"
      case "zone_xsec":
        return "Zone-XSEC"
      default:
        return s
    }
  }

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Defacement Archive</h1>
          <p className="text-[13px] text-muted-foreground mt-0.5">
            Monitoring arsip defacement dari Zone-H dan Zone-XSEC
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => {
              statsQuery.refetch()
              incidentsQuery.refetch()
            }}
            disabled={statsQuery.isFetching || incidentsQuery.isFetching}
            className="h-8 text-xs"
          >
            <RefreshCw
              className={`mr-1.5 h-3.5 w-3.5 ${
                statsQuery.isFetching || incidentsQuery.isFetching ? "animate-spin" : ""
              }`}
            />
            Refresh
          </Button>
          <Button onClick={handleScan} disabled={scanLoading} className="h-8 text-xs">
            {scanLoading ? (
              <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
            ) : (
              <Radar className="mr-1.5 h-3.5 w-3.5" />
            )}
            Scan Sekarang
          </Button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Card className="border-border/50 border-l-4 border-l-red-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-red-500/10">
                <Skull className="h-4 w-4 text-red-500" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">Total Insiden</p>
                <p className="text-2xl font-bold">{stats?.total_incidents || 0}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/50 border-l-4 border-l-amber-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-500/10">
                <AlertTriangle className="h-4 w-4 text-amber-500" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">Belum Ditangani</p>
                <p className="text-2xl font-bold">{stats?.unacknowledged_count || 0}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/50 border-l-4 border-l-purple-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-purple-500/10">
                <Shield className="h-4 w-4 text-purple-500" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">Website Terdampak</p>
                <p className="text-2xl font-bold">{stats?.websites_affected || 0}</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/50 border-l-4 border-l-blue-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-500/10">
                <Clock className="h-4 w-4 text-blue-500" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">Scan Terakhir</p>
                <p className="text-sm font-medium mt-0.5">
                  {stats?.last_scan_at ? formatDate(stats.last_scan_at) : "Belum pernah"}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Source Breakdown */}
      {stats?.by_source && Object.keys(stats.by_source).length > 0 && (
        <div className="flex items-center gap-3">
          <p className="text-xs text-muted-foreground">Sumber:</p>
          {Object.entries(stats.by_source).map(([source, count]) => (
            <Badge key={source} variant="outline" className="text-[11px]">
              {sourceLabel(source)}: {count}
            </Badge>
          ))}
        </div>
      )}

      {/* Search & Filter */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
          <Input
            placeholder="Cari URL, attacker, team..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="h-8 pl-8 text-[13px]"
          />
        </div>
        <div className="flex items-center gap-2">
          <Select value={filterSource} onValueChange={setFilterSource}>
            <SelectTrigger className="h-8 w-[140px] text-[13px]">
              <Filter className="mr-1 h-3 w-3" />
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua Sumber</SelectItem>
              <SelectItem value="zone_h">Zone-H</SelectItem>
              <SelectItem value="zone_xsec">Zone-XSEC</SelectItem>
            </SelectContent>
          </Select>
          <Select value={filterAck} onValueChange={setFilterAck}>
            <SelectTrigger className="h-8 w-[150px] text-[13px]">
              <Filter className="mr-1 h-3 w-3" />
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua Status</SelectItem>
              <SelectItem value="no">Belum Ditangani</SelectItem>
              <SelectItem value="yes">Sudah Ditangani</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Incidents Table */}
      <Card className="border-border/50">
        <div className="px-5 py-4 border-b border-border/50 flex items-center justify-between">
          <div>
            <h2 className="text-sm font-medium">Defacement Incidents</h2>
            <p className="text-xs text-muted-foreground mt-0.5">
              Riwayat insiden defacement yang terdeteksi
            </p>
          </div>
          <Badge variant="secondary" className="text-[11px]">
            {filteredIncidents.length} insiden
          </Badge>
        </div>
        <CardContent className="p-0">
          {filteredIncidents.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    URL
                  </TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Attacker
                  </TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Sumber
                  </TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Waktu Deface
                  </TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Status
                  </TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider text-right">
                    Aksi
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredIncidents.map((incident) => (
                  <TableRow key={incident.id} className="hover:bg-muted/30">
                    <TableCell className="py-3">
                      <div className="max-w-[300px]">
                        <p className="text-[13px] font-medium truncate">
                          {incident.defaced_url}
                        </p>
                        {incident.website_name && (
                          <p className="text-xs text-muted-foreground truncate">
                            {incident.website_name}
                          </p>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="py-3">
                      <div>
                        {incident.attacker && (
                          <div className="flex items-center gap-1 text-[13px]">
                            <User className="h-3 w-3 text-muted-foreground" />
                            {incident.attacker}
                          </div>
                        )}
                        {incident.team && (
                          <div className="flex items-center gap-1 text-xs text-muted-foreground">
                            <Users className="h-3 w-3" />
                            {incident.team}
                          </div>
                        )}
                        {!incident.attacker && !incident.team && (
                          <span className="text-xs text-muted-foreground">-</span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="py-3">
                      <Badge
                        variant="outline"
                        className={`text-[11px] ${
                          incident.source === "zone_h"
                            ? "bg-red-500/10 text-red-600 border-red-500/20"
                            : "bg-orange-500/10 text-orange-600 border-orange-500/20"
                        }`}
                      >
                        {sourceLabel(incident.source)}
                      </Badge>
                    </TableCell>
                    <TableCell className="py-3">
                      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                        <Clock className="h-3 w-3" />
                        {incident.defaced_at
                          ? formatDate(incident.defaced_at)
                          : formatDate(incident.created_at)}
                      </div>
                    </TableCell>
                    <TableCell className="py-3">
                      {incident.is_acknowledged ? (
                        <Badge
                          variant="outline"
                          className="bg-emerald-500/10 text-emerald-600 border-emerald-500/20 text-[11px]"
                        >
                          <CheckCircle2 className="mr-1 h-3 w-3" />
                          Ditangani
                        </Badge>
                      ) : (
                        <Badge
                          variant="outline"
                          className="bg-amber-500/10 text-amber-600 border-amber-500/20 text-[11px]"
                        >
                          <AlertTriangle className="mr-1 h-3 w-3" />
                          Pending
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className="py-3 text-right">
                      <div className="flex items-center justify-end gap-1">
                        {incident.mirror_url && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-7 w-7 p-0"
                            asChild
                          >
                            <a
                              href={incident.mirror_url}
                              target="_blank"
                              rel="noopener noreferrer"
                              title="Lihat mirror"
                            >
                              <ExternalLink className="h-3.5 w-3.5" />
                            </a>
                          </Button>
                        )}
                        {!incident.is_acknowledged && (
                          <Button
                            variant="outline"
                            size="sm"
                            className="h-7 text-[11px]"
                            onClick={() => {
                              setAckDialog(incident)
                              setAckNotes("")
                            }}
                          >
                            <Eye className="mr-1 h-3 w-3" />
                            Acknowledge
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <div className="flex flex-col items-center justify-center py-16">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted">
                <Shield className="h-5 w-5 text-muted-foreground" />
              </div>
              <p className="text-sm font-medium mt-3">
                {incidents.length === 0 ? "Tidak ada insiden defacement" : "Tidak ada hasil yang cocok"}
              </p>
              <p className="text-xs text-muted-foreground mt-0.5">
                {incidents.length === 0
                  ? "Belum ada insiden defacement yang terdeteksi dari arsip Zone-H / Zone-XSEC"
                  : "Coba ubah filter atau kata kunci pencarian"}
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Acknowledge Dialog */}
      <Dialog open={!!ackDialog} onOpenChange={() => setAckDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="text-sm">Acknowledge Incident</DialogTitle>
          </DialogHeader>
          {ackDialog && (
            <div className="space-y-3">
              <div className="rounded-md bg-muted/50 p-3 space-y-1">
                <p className="text-xs text-muted-foreground">URL yang di-deface:</p>
                <p className="text-[13px] font-mono break-all">{ackDialog.defaced_url}</p>
                {ackDialog.attacker && (
                  <>
                    <p className="text-xs text-muted-foreground mt-2">Attacker:</p>
                    <p className="text-[13px]">{ackDialog.attacker}</p>
                  </>
                )}
              </div>
              <div className="space-y-1.5">
                <label className="text-xs font-medium">Catatan (opsional)</label>
                <Textarea
                  value={ackNotes}
                  onChange={(e) => setAckNotes(e.target.value)}
                  placeholder="Tambahkan catatan penanganan..."
                  className="text-[13px] min-h-[80px]"
                />
              </div>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setAckDialog(null)} className="h-8 text-xs">
              Batal
            </Button>
            <Button
              onClick={() => ackDialog && ackMutation.mutate({ id: ackDialog.id, notes: ackNotes })}
              disabled={ackMutation.isPending}
              className="h-8 text-xs"
            >
              {ackMutation.isPending && <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />}
              Acknowledge
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function DefacementSkeleton() {
  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <Skeleton className="h-6 w-48 mb-1.5" />
          <Skeleton className="h-4 w-72" />
        </div>
        <div className="flex gap-2">
          <Skeleton className="h-8 w-24" />
          <Skeleton className="h-8 w-32" />
        </div>
      </div>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Card key={i} className="border-border/50">
            <CardContent className="px-5 py-4">
              <div className="flex items-center gap-3">
                <Skeleton className="h-7 w-7 rounded-lg" />
                <div className="flex-1">
                  <Skeleton className="h-3 w-24 mb-1.5" />
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
          <Skeleton className="h-3 w-56" />
        </div>
        <CardContent className="p-4">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex items-center gap-4 py-3">
              <Skeleton className="h-4 w-48" />
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-5 w-16 rounded-full" />
              <Skeleton className="h-3 w-28" />
              <Skeleton className="h-5 w-20 rounded-full" />
              <Skeleton className="h-7 w-24 ml-auto" />
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}
