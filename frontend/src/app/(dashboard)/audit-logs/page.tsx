"use client"

import { useState, Fragment } from "react"
import { auditApi, type AuditLog } from "@/lib/api"
import { formatRelativeTime } from "@/lib/utils"
import {
  ChevronDown,
  ChevronRight,
  FileText,
  Loader2,
  Search,
  Clock,
  User,
  Globe,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { usePaginatedQuery } from "@/hooks/use-paginated-query"

const PAGE_SIZE = 20

interface AuditFilters {
  action: string
  resource_type: string
  username: string
}

export default function AuditLogsPage() {
  const [expandedRow, setExpandedRow] = useState<number | null>(null)

  const {
    data: logs,
    total,
    totalPages,
    page,
    setPage,
    filters,
    setFilter,
    isLoading: loading,
  } = usePaginatedQuery<AuditLog, AuditFilters>({
    queryKey: "audit-logs",
    queryFn: (params) => auditApi.list(params),
    pageSize: PAGE_SIZE,
    initialFilters: {
      action: "all",
      resource_type: "all",
      username: "",
    },
  })

  const getActionBadge = (action: string) => {
    if (action.includes("create") || action.includes("add")) {
      return (
        <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium bg-emerald-500/10 text-emerald-600 border border-emerald-500/20">
          {action}
        </span>
      )
    }
    if (action.includes("update") || action.includes("edit") || action.includes("change")) {
      return (
        <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium bg-blue-500/10 text-blue-600 border border-blue-500/20">
          {action}
        </span>
      )
    }
    if (action.includes("delete") || action.includes("remove")) {
      return (
        <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium bg-red-500/10 text-red-600 border border-red-500/20">
          {action}
        </span>
      )
    }
    if (action.includes("login") || action.includes("logout") || action.includes("auth")) {
      return (
        <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium bg-purple-500/10 text-purple-600 border border-purple-500/20">
          {action}
        </span>
      )
    }
    return (
      <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium bg-muted text-muted-foreground border border-border/50">
        {action}
      </span>
    )
  }

  const toggleRow = (id: number) => {
    setExpandedRow(expandedRow === id ? null : id)
  }

  return (
    <div className="space-y-5">
      {/* Header */}
      <div>
        <h1 className="text-xl font-semibold tracking-tight">Audit Logs</h1>
        <p className="text-[13px] text-muted-foreground mt-0.5">Riwayat aktivitas pengguna dalam sistem</p>
      </div>

      {/* Compact Filter Bar */}
      <Card className="border-border/50">
        <CardContent className="px-4 py-3">
          <div className="flex flex-wrap items-center gap-2">
            <Select
              value={filters.action}
              onValueChange={(v) => setFilter("action", v)}
            >
              <SelectTrigger className="h-8 w-[140px] text-[13px]">
                <SelectValue placeholder="Action" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all" className="text-[13px]">Semua Action</SelectItem>
                <SelectItem value="create" className="text-[13px]">Create</SelectItem>
                <SelectItem value="update" className="text-[13px]">Update</SelectItem>
                <SelectItem value="delete" className="text-[13px]">Delete</SelectItem>
                <SelectItem value="login" className="text-[13px]">Login</SelectItem>
                <SelectItem value="logout" className="text-[13px]">Logout</SelectItem>
              </SelectContent>
            </Select>

            <Select
              value={filters.resource_type}
              onValueChange={(v) => setFilter("resource_type", v)}
            >
              <SelectTrigger className="h-8 w-[150px] text-[13px]">
                <SelectValue placeholder="Resource Type" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all" className="text-[13px]">Semua Resource</SelectItem>
                <SelectItem value="website" className="text-[13px]">Website</SelectItem>
                <SelectItem value="user" className="text-[13px]">User</SelectItem>
                <SelectItem value="alert" className="text-[13px]">Alert</SelectItem>
                <SelectItem value="keyword" className="text-[13px]">Keyword</SelectItem>
                <SelectItem value="maintenance" className="text-[13px]">Maintenance</SelectItem>
                <SelectItem value="setting" className="text-[13px]">Setting</SelectItem>
              </SelectContent>
            </Select>

            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground" />
              <Input
                placeholder="Cari username..."
                className="h-8 pl-8 w-[180px] text-[13px]"
                value={filters.username}
                onChange={(e) => setFilter("username", e.target.value)}
              />
            </div>

            {total > 0 && (
              <span className="text-xs text-muted-foreground ml-auto">
                {total.toLocaleString()} log
              </span>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Table */}
      <Card className="border-border/50">
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="w-[32px] text-xs font-medium text-muted-foreground uppercase tracking-wider" />
                <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Waktu</TableHead>
                <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">User</TableHead>
                <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Action</TableHead>
                <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Resource</TableHead>
                <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">ID</TableHead>
                <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">IP Address</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-16">
                    <Loader2 className="size-5 animate-spin mx-auto text-muted-foreground" />
                  </TableCell>
                </TableRow>
              ) : logs.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-16">
                    <div className="flex flex-col items-center gap-2">
                      <div className="size-10 rounded-full bg-muted flex items-center justify-center">
                        <FileText className="size-5 text-muted-foreground" />
                      </div>
                      <p className="text-sm font-medium text-foreground">Tidak ada audit log</p>
                      <p className="text-xs text-muted-foreground">Log aktivitas akan muncul di sini</p>
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                logs.map((log) => (
                  <Fragment key={log.id}>
                    <TableRow
                      className="cursor-pointer group"
                      onClick={() => toggleRow(log.id)}
                    >
                      <TableCell className="py-2.5 pl-3">
                        <div className="size-5 rounded flex items-center justify-center text-muted-foreground group-hover:bg-muted transition-colors">
                          {expandedRow === log.id ? (
                            <ChevronDown className="size-3.5" />
                          ) : (
                            <ChevronRight className="size-3.5" />
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="py-2.5">
                        <div className="flex items-center gap-1.5 text-[13px] text-muted-foreground">
                          <Clock className="size-3 shrink-0" />
                          {formatRelativeTime(log.created_at)}
                        </div>
                      </TableCell>
                      <TableCell className="py-2.5">
                        <div className="flex items-center gap-1.5">
                          <div className="size-5 rounded-full bg-muted flex items-center justify-center">
                            <User className="size-3 text-muted-foreground" />
                          </div>
                          <span className="text-[13px] font-medium">
                            {log.username || `User #${log.user_id}` || "System"}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell className="py-2.5">{getActionBadge(log.action)}</TableCell>
                      <TableCell className="py-2.5">
                        <span className="text-[13px] text-muted-foreground capitalize">
                          {log.resource_type.replace(/_/g, " ")}
                        </span>
                      </TableCell>
                      <TableCell className="py-2.5">
                        <span className="text-[13px] text-muted-foreground font-mono">
                          {log.resource_id ?? "-"}
                        </span>
                      </TableCell>
                      <TableCell className="py-2.5">
                        <span className="text-[11px] text-muted-foreground font-mono bg-muted/50 rounded px-1.5 py-0.5">
                          {log.ip_address || "-"}
                        </span>
                      </TableCell>
                    </TableRow>
                    {expandedRow === log.id && (
                      <TableRow key={`${log.id}-detail`} className="hover:bg-transparent">
                        <TableCell colSpan={7} className="bg-muted/20 border-t-0 px-4 py-3">
                          <div className="ml-6 space-y-3">
                            {log.details != null && (
                              <div>
                                <p className="text-xs font-medium text-muted-foreground mb-1.5">
                                  Detail
                                </p>
                                <pre className="text-xs font-mono bg-background border border-border/50 rounded-md p-3 overflow-auto max-h-[280px] text-muted-foreground leading-relaxed">
                                  {JSON.stringify(log.details as Record<string, unknown>, null, 2)}
                                </pre>
                              </div>
                            )}
                            {log.user_agent && (
                              <div>
                                <p className="text-xs font-medium text-muted-foreground mb-1">
                                  User Agent
                                </p>
                                <p className="text-xs text-muted-foreground/70 break-all font-mono leading-relaxed">
                                  {log.user_agent}
                                </p>
                              </div>
                            )}
                            {!log.details && !log.user_agent && (
                              <p className="text-xs text-muted-foreground/60 italic">
                                Tidak ada detail tambahan
                              </p>
                            )}
                          </div>
                        </TableCell>
                      </TableRow>
                    )}
                  </Fragment>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-xs text-muted-foreground">
            {(page - 1) * PAGE_SIZE + 1}–{Math.min(page * PAGE_SIZE, total)} dari {total} log
          </p>
          <div className="flex items-center gap-1.5">
            <Button
              variant="outline"
              className="h-7 text-[11px] px-2.5"
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
            >
              Sebelumnya
            </Button>
            <div className="flex items-center gap-1">
              {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => {
                let pageNum: number
                if (totalPages <= 5) {
                  pageNum = i + 1
                } else if (page <= 3) {
                  pageNum = i + 1
                } else if (page >= totalPages - 2) {
                  pageNum = totalPages - 4 + i
                } else {
                  pageNum = page - 2 + i
                }
                return (
                  <Button
                    key={pageNum}
                    variant={page === pageNum ? "default" : "ghost"}
                    className="h-7 w-7 text-[11px] p-0"
                    onClick={() => setPage(pageNum)}
                  >
                    {pageNum}
                  </Button>
                )
              })}
            </div>
            <Button
              variant="outline"
              className="h-7 text-[11px] px-2.5"
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
            >
              Selanjutnya
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
