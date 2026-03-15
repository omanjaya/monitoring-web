"use client"

import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import {
  RefreshCw,
  AlertCircle,
  CheckCircle2,
  XCircle,
  Globe,
  Shield,
  Server,
  Mail,
  ChevronDown,
  ChevronRight,
  Clock,
  Search,
  Loader2,
  Radar,
  Filter,
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

import { dnsApi, adminApi, websiteApi, type DNSScanRecord } from "@/lib/api"
import { formatDate } from "@/lib/utils"

export default function DNSPage() {
  const [expandedDomain, setExpandedDomain] = useState<number | null>(null)
  const [search, setSearch] = useState("")
  const [scanLoading, setScanLoading] = useState(false)
  const [filterSPF, setFilterSPF] = useState<string>("all")
  const [filterDMARC, setFilterDMARC] = useState<string>("all")

  const summaryQuery = useQuery({
    queryKey: ["dns-summary"],
    queryFn: () => dnsApi.summary(),
  })

  const websitesQuery = useQuery({
    queryKey: ["websites-count"],
    queryFn: () => websiteApi.list(),
  })

  const scans = (summaryQuery.data?.data as DNSScanRecord[]) || []
  const totalWebsites = (() => {
    const resp = websitesQuery.data as unknown
    if (resp && typeof resp === "object") {
      const obj = resp as Record<string, unknown>
      if (typeof obj.total === "number") return obj.total
      if (Array.isArray(obj.data)) return obj.data.length
    }
    return 0
  })()
  const isLoading = summaryQuery.isLoading
  const error = summaryQuery.error

  const handleScanAll = async () => {
    try {
      setScanLoading(true)
      await adminApi.trigger("dns")
      toast.success("DNS scan sedang dijalankan untuk semua website")
      // Refetch after a delay to get updated data
      setTimeout(() => summaryQuery.refetch(), 5000)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menjalankan DNS scan")
    } finally {
      setScanLoading(false)
    }
  }

  if (isLoading) return <DNSSkeleton />

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-20">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-500/10">
          <AlertCircle className="h-6 w-6 text-red-500" />
        </div>
        <h2 className="text-sm font-medium">Gagal Memuat Data DNS</h2>
        <p className="text-xs text-muted-foreground">{error instanceof Error ? error.message : "Gagal memuat data DNS"}</p>
        <Button onClick={() => summaryQuery.refetch()} variant="outline" className="h-8 text-xs">
          <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
          Coba Lagi
        </Button>
      </div>
    )
  }

  const totalDomains = scans.length
  const withSPF = scans.filter(s => s.has_spf).length
  const withDMARC = scans.filter(s => s.has_dmarc).length
  const totalSubdomains = scans.reduce((sum, s) => sum + (s.subdomain_count || 0), 0)
  const notScanned = totalWebsites > 0 ? totalWebsites - totalDomains : 0

  // Filter scans
  const filteredScans = scans.filter(s => {
    if (search && !s.domain_name.toLowerCase().includes(search.toLowerCase())) return false
    if (filterSPF === "yes" && !s.has_spf) return false
    if (filterSPF === "no" && s.has_spf) return false
    if (filterDMARC === "yes" && !s.has_dmarc) return false
    if (filterDMARC === "no" && s.has_dmarc) return false
    return true
  })

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">DNS Monitoring</h1>
          <p className="text-[13px] text-muted-foreground mt-0.5">
            Monitoring DNS records, SPF, DMARC, dan subdomain
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => summaryQuery.refetch()}
            disabled={summaryQuery.isFetching}
            className="h-8 text-xs"
          >
            <RefreshCw className={`mr-1.5 h-3.5 w-3.5 ${summaryQuery.isFetching ? "animate-spin" : ""}`} />
            Refresh
          </Button>
          <Button
            onClick={handleScanAll}
            disabled={scanLoading}
            className="h-8 text-xs"
          >
            {scanLoading ? (
              <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
            ) : (
              <Radar className="mr-1.5 h-3.5 w-3.5" />
            )}
            Scan All DNS
          </Button>
        </div>
      </div>

      {/* Warning if not all scanned */}
      {notScanned > 0 && (
        <div className="flex items-center gap-3 rounded-lg border border-amber-500/30 bg-amber-500/5 px-4 py-3">
          <AlertCircle className="h-4 w-4 text-amber-500 shrink-0" />
          <p className="text-xs text-amber-700 dark:text-amber-400">
            <span className="font-medium">{notScanned} dari {totalWebsites} website</span> belum memiliki data DNS scan.
            Klik &quot;Scan All DNS&quot; untuk memulai scan semua website.
          </p>
        </div>
      )}

      {/* Stats Cards */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Card className="border-border/50 border-l-4 border-l-blue-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-500/10">
                <Globe className="h-4 w-4 text-blue-500" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">Total Domain</p>
                <p className="text-2xl font-bold">{totalDomains}</p>
              </div>
            </div>
            <p className="text-xs text-muted-foreground mt-1">domain dipantau</p>
          </CardContent>
        </Card>

        <Card className="border-border/50 border-l-4 border-l-emerald-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-500/10">
                <Shield className="h-4 w-4 text-emerald-500" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">SPF Configured</p>
                <div className="flex items-baseline gap-1.5">
                  <p className="text-2xl font-bold">{withSPF}</p>
                  <span className="text-xs text-muted-foreground">/ {totalDomains}</span>
                </div>
              </div>
            </div>
            <div className="mt-2 h-1.5 w-full rounded-full bg-muted overflow-hidden">
              <div
                className="h-full rounded-full bg-emerald-500 transition-all"
                style={{ width: `${totalDomains > 0 ? (withSPF / totalDomains) * 100 : 0}%` }}
              />
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/50 border-l-4 border-l-purple-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-purple-500/10">
                <Mail className="h-4 w-4 text-purple-500" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">DMARC Configured</p>
                <div className="flex items-baseline gap-1.5">
                  <p className="text-2xl font-bold">{withDMARC}</p>
                  <span className="text-xs text-muted-foreground">/ {totalDomains}</span>
                </div>
              </div>
            </div>
            <div className="mt-2 h-1.5 w-full rounded-full bg-muted overflow-hidden">
              <div
                className="h-full rounded-full bg-purple-500 transition-all"
                style={{ width: `${totalDomains > 0 ? (withDMARC / totalDomains) * 100 : 0}%` }}
              />
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/50 border-l-4 border-l-amber-500">
          <CardContent className="px-5 py-4">
            <div className="flex items-center gap-3">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-500/10">
                <Server className="h-4 w-4 text-amber-500" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">Subdomain Ditemukan</p>
                <p className="text-2xl font-bold">{totalSubdomains}</p>
              </div>
            </div>
            <p className="text-xs text-muted-foreground mt-1">total subdomain</p>
          </CardContent>
        </Card>
      </div>

      {/* Search & Filter */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
          <Input
            placeholder="Cari domain..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="h-8 pl-8 text-[13px]"
          />
        </div>
        <div className="flex items-center gap-2">
          <Select value={filterSPF} onValueChange={setFilterSPF}>
            <SelectTrigger className="h-8 w-[130px] text-[13px]">
              <Filter className="mr-1 h-3 w-3" />
              <SelectValue placeholder="SPF" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">SPF: Semua</SelectItem>
              <SelectItem value="yes">SPF: Ada</SelectItem>
              <SelectItem value="no">SPF: Tidak Ada</SelectItem>
            </SelectContent>
          </Select>
          <Select value={filterDMARC} onValueChange={setFilterDMARC}>
            <SelectTrigger className="h-8 w-[145px] text-[13px]">
              <Filter className="mr-1 h-3 w-3" />
              <SelectValue placeholder="DMARC" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">DMARC: Semua</SelectItem>
              <SelectItem value="yes">DMARC: Ada</SelectItem>
              <SelectItem value="no">DMARC: Tidak Ada</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* DNS Summary Table */}
      <Card className="border-border/50">
        <div className="px-5 py-4 border-b border-border/50 flex items-center justify-between">
          <div>
            <h2 className="text-sm font-medium">DNS Scan Results</h2>
            <p className="text-xs text-muted-foreground mt-0.5">Hasil scan DNS terakhir per domain</p>
          </div>
          {search || filterSPF !== "all" || filterDMARC !== "all" ? (
            <Badge variant="secondary" className="text-[11px]">
              {filteredScans.length} dari {scans.length} domain
            </Badge>
          ) : (
            <Badge variant="secondary" className="text-[11px]">
              {scans.length} domain
            </Badge>
          )}
        </div>
        <CardContent className="p-0">
          {filteredScans.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider w-8"></TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Domain</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">SPF</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">DMARC</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Nameservers</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">MX</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Subdomains</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider text-right">Scan Time</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredScans.map((scan) => {
                  const isExpanded = expandedDomain === scan.id
                  return (
                    <DNSRow
                      key={scan.id}
                      scan={scan}
                      isExpanded={isExpanded}
                      onToggle={() => setExpandedDomain(isExpanded ? null : scan.id)}
                    />
                  )
                })}
              </TableBody>
            </Table>
          ) : (
            <div className="flex flex-col items-center justify-center py-16">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted">
                <Globe className="h-5 w-5 text-muted-foreground" />
              </div>
              <p className="text-sm font-medium mt-3">Belum ada data DNS</p>
              <p className="text-xs text-muted-foreground mt-0.5">DNS scan akan berjalan otomatis setiap 12 jam</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function DNSRow({ scan, isExpanded, onToggle }: { scan: DNSScanRecord; isExpanded: boolean; onToggle: () => void }) {
  const nameservers = scan.nameservers || []
  const mxRecords = scan.mx_records || []
  const subdomains = scan.subdomains || []
  const dnsRecords = scan.dns_records || []

  return (
    <>
      <TableRow
        className="group hover:bg-muted/30 transition-colors cursor-pointer"
        onClick={onToggle}
      >
        <TableCell className="py-3 w-8">
          {isExpanded ? (
            <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
          )}
        </TableCell>
        <TableCell className="py-3">
          <div className="text-[13px] font-medium">{scan.domain_name}</div>
        </TableCell>
        <TableCell className="py-3">
          {scan.has_spf ? (
            <Badge variant="outline" className="bg-emerald-500/10 text-emerald-600 border-emerald-500/20 text-[11px]">
              <CheckCircle2 className="mr-1 h-3 w-3" />
              Yes
            </Badge>
          ) : (
            <Badge variant="outline" className="bg-red-500/10 text-red-600 border-red-500/20 text-[11px]">
              <XCircle className="mr-1 h-3 w-3" />
              No
            </Badge>
          )}
        </TableCell>
        <TableCell className="py-3">
          {scan.has_dmarc ? (
            <Badge variant="outline" className="bg-emerald-500/10 text-emerald-600 border-emerald-500/20 text-[11px]">
              <CheckCircle2 className="mr-1 h-3 w-3" />
              Yes
            </Badge>
          ) : (
            <Badge variant="outline" className="bg-red-500/10 text-red-600 border-red-500/20 text-[11px]">
              <XCircle className="mr-1 h-3 w-3" />
              No
            </Badge>
          )}
        </TableCell>
        <TableCell className="py-3">
          <span className="text-[13px]">{nameservers.length}</span>
        </TableCell>
        <TableCell className="py-3">
          <span className="text-[13px]">{mxRecords.length}</span>
        </TableCell>
        <TableCell className="py-3">
          <Badge variant="secondary" className="text-[11px]">
            {scan.subdomain_count || 0}
          </Badge>
        </TableCell>
        <TableCell className="py-3 text-right">
          <div className="flex items-center justify-end gap-1.5 text-xs text-muted-foreground">
            <Clock className="h-3 w-3" />
            {formatDate(scan.created_at)}
          </div>
        </TableCell>
      </TableRow>

      {/* Expanded Detail */}
      {isExpanded && (
        <TableRow className="hover:bg-transparent">
          <TableCell colSpan={8} className="p-0">
            <div className="bg-muted/20 border-t border-border/30 px-6 py-4 space-y-4">
              {/* SPF & DMARC Records */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                {scan.spf_record && (
                  <div>
                    <p className="text-xs font-medium text-muted-foreground mb-1">SPF Record</p>
                    <code className="block text-xs bg-background rounded-md p-2.5 border border-border/50 break-all">
                      {scan.spf_record}
                    </code>
                  </div>
                )}
                {scan.dmarc_record && (
                  <div>
                    <p className="text-xs font-medium text-muted-foreground mb-1">DMARC Record</p>
                    <code className="block text-xs bg-background rounded-md p-2.5 border border-border/50 break-all">
                      {scan.dmarc_record}
                    </code>
                  </div>
                )}
              </div>

              {/* Nameservers */}
              {nameservers.length > 0 && (
                <div>
                  <p className="text-xs font-medium text-muted-foreground mb-1.5">Nameservers</p>
                  <div className="flex flex-wrap gap-1.5">
                    {nameservers.map((ns, i) => (
                      <Badge key={i} variant="outline" className="text-[11px] font-mono">
                        {ns}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}

              {/* MX Records */}
              {mxRecords.length > 0 && (
                <div>
                  <p className="text-xs font-medium text-muted-foreground mb-1.5">MX Records</p>
                  <div className="flex flex-wrap gap-1.5">
                    {mxRecords.map((mx, i) => (
                      <Badge key={i} variant="outline" className="text-[11px] font-mono">
                        {mx}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}

              {/* DNS Records */}
              {dnsRecords.length > 0 && (
                <div>
                  <p className="text-xs font-medium text-muted-foreground mb-1.5">DNS Records ({dnsRecords.length})</p>
                  <div className="bg-background rounded-md border border-border/50 overflow-hidden">
                    <table className="w-full text-xs">
                      <thead>
                        <tr className="border-b border-border/30">
                          <th className="text-left px-3 py-1.5 text-muted-foreground font-medium">Type</th>
                          <th className="text-left px-3 py-1.5 text-muted-foreground font-medium">Name</th>
                          <th className="text-left px-3 py-1.5 text-muted-foreground font-medium">Value</th>
                          <th className="text-right px-3 py-1.5 text-muted-foreground font-medium">TTL</th>
                        </tr>
                      </thead>
                      <tbody>
                        {dnsRecords.slice(0, 20).map((rec, i) => (
                          <tr key={i} className="border-b border-border/20 last:border-0">
                            <td className="px-3 py-1.5">
                              <Badge variant="secondary" className="text-[10px] font-mono">{rec.type}</Badge>
                            </td>
                            <td className="px-3 py-1.5 font-mono text-muted-foreground">{rec.name}</td>
                            <td className="px-3 py-1.5 font-mono break-all max-w-[300px]">{rec.value}</td>
                            <td className="px-3 py-1.5 text-right text-muted-foreground">{rec.ttl}s</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                    {dnsRecords.length > 20 && (
                      <div className="px-3 py-1.5 text-xs text-muted-foreground border-t border-border/30">
                        ... dan {dnsRecords.length - 20} record lainnya
                      </div>
                    )}
                  </div>
                </div>
              )}

              {/* Subdomains */}
              {subdomains.length > 0 && (
                <div>
                  <p className="text-xs font-medium text-muted-foreground mb-1.5">Subdomains ({subdomains.length})</p>
                  <div className="bg-background rounded-md border border-border/50 overflow-hidden">
                    <table className="w-full text-xs">
                      <thead>
                        <tr className="border-b border-border/30">
                          <th className="text-left px-3 py-1.5 text-muted-foreground font-medium">Subdomain</th>
                          <th className="text-left px-3 py-1.5 text-muted-foreground font-medium">IP</th>
                          <th className="text-left px-3 py-1.5 text-muted-foreground font-medium">Status</th>
                          <th className="text-left px-3 py-1.5 text-muted-foreground font-medium">Title</th>
                          <th className="text-left px-3 py-1.5 text-muted-foreground font-medium">Source</th>
                        </tr>
                      </thead>
                      <tbody>
                        {subdomains.slice(0, 30).map((sub, i) => (
                          <tr key={i} className="border-b border-border/20 last:border-0">
                            <td className="px-3 py-1.5 font-mono">{sub.subdomain}</td>
                            <td className="px-3 py-1.5 font-mono text-muted-foreground">{sub.ip || "-"}</td>
                            <td className="px-3 py-1.5">
                              {sub.status_code ? (
                                <Badge
                                  variant="outline"
                                  className={`text-[10px] ${
                                    sub.status_code < 400
                                      ? "bg-emerald-500/10 text-emerald-600 border-emerald-500/20"
                                      : "bg-red-500/10 text-red-600 border-red-500/20"
                                  }`}
                                >
                                  {sub.status_code}
                                </Badge>
                              ) : (
                                <span className="text-muted-foreground">-</span>
                              )}
                            </td>
                            <td className="px-3 py-1.5 text-muted-foreground max-w-[200px] truncate">{sub.title || "-"}</td>
                            <td className="px-3 py-1.5">
                              <Badge variant="secondary" className="text-[10px]">{sub.source}</Badge>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                    {subdomains.length > 30 && (
                      <div className="px-3 py-1.5 text-xs text-muted-foreground border-t border-border/30">
                        ... dan {subdomains.length - 30} subdomain lainnya
                      </div>
                    )}
                  </div>
                </div>
              )}

              {/* Scan Duration */}
              <div className="text-xs text-muted-foreground">
                Scan duration: {scan.scan_duration_ms}ms
              </div>
            </div>
          </TableCell>
        </TableRow>
      )}
    </>
  )
}

function DNSSkeleton() {
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
                  <Skeleton className="h-3 w-20 mb-1.5" />
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
                <Skeleton className="h-4 w-4" />
                <Skeleton className="h-4 w-40" />
                <Skeleton className="h-5 w-12 rounded-full" />
                <Skeleton className="h-5 w-12 rounded-full" />
                <Skeleton className="h-4 w-8" />
                <Skeleton className="h-4 w-8" />
                <Skeleton className="h-5 w-10 rounded-full" />
                <Skeleton className="h-3 w-24 ml-auto" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
