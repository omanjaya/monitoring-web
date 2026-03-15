"use client"

import { useState } from "react"
import {
  Tag,
  RefreshCw,
  AlertCircle,
  Plus,
  Trash2,
  Code2,
  Upload,
} from "lucide-react"
import { toast } from "sonner"
import { useQuery } from "@tanstack/react-query"

import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
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

import { keywordApi, type Keyword } from "@/lib/api"
import { useMutationAction } from "@/hooks/use-mutation-action"
import { BulkImportDialog } from "@/components/bulk-import-dialog"

const CATEGORIES = [
  { value: "all", label: "All Categories" },
  { value: "gambling", label: "Gambling" },
  { value: "defacement", label: "Defacement" },
  { value: "malware", label: "Malware" },
  { value: "phishing", label: "Phishing" },
  { value: "porn", label: "Porn" },
  { value: "custom", label: "Custom" },
]

function getCategoryColor(category: string): string {
  switch (category?.toLowerCase()) {
    case "gambling":
      return "bg-red-500/10 text-red-700 dark:text-red-400"
    case "defacement":
      return "bg-amber-500/10 text-amber-700 dark:text-amber-400"
    case "malware":
      return "bg-purple-500/10 text-purple-700 dark:text-purple-400"
    case "phishing":
      return "bg-blue-500/10 text-blue-700 dark:text-blue-400"
    case "porn":
      return "bg-pink-500/10 text-pink-700 dark:text-pink-400"
    case "custom":
      return "bg-gray-500/10 text-gray-700 dark:text-gray-400"
    default:
      return "bg-gray-500/10 text-gray-700 dark:text-gray-400"
  }
}

export default function KeywordsPage() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [bulkImportOpen, setBulkImportOpen] = useState(false)
  const [filter, setFilter] = useState("all")

  const [form, setForm] = useState({
    keyword: "",
    category: "custom",
    is_regex: false,
    weight: 1,
  })

  const { data: keywords = [], isLoading: loading, error, refetch } = useQuery({
    queryKey: ["keywords"],
    queryFn: async () => {
      const res = await keywordApi.list()
      return (res.data || []) as Keyword[]
    },
  })

  const createMutation = useMutationAction({
    mutationFn: (payload: { keyword: string; category: string; is_regex: boolean; weight: number }) =>
      keywordApi.create(payload),
    successMessage: "Keyword berhasil ditambahkan",
    invalidateKeys: ["keywords"],
    onSuccess: () => setDialogOpen(false),
  })

  const deleteMutation = useMutationAction({
    mutationFn: (id: number) => keywordApi.delete(id),
    successMessage: "Keyword dihapus",
    invalidateKeys: ["keywords"],
  })

  const openCreate = () => {
    setForm({ keyword: "", category: "custom", is_regex: false, weight: 1 })
    setDialogOpen(true)
  }

  const handleSubmit = () => {
    if (!form.keyword.trim()) {
      toast.error("Keyword harus diisi")
      return
    }
    createMutation.mutate({
      keyword: form.keyword,
      category: form.category,
      is_regex: form.is_regex,
      weight: form.weight,
    })
  }

  const handleDelete = (id: number) => {
    deleteMutation.mutate(id)
  }

  const filteredKeywords =
    filter === "all"
      ? keywords
      : keywords.filter((k) => k.category === filter)

  if (loading) return <KeywordsSkeleton />

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-20">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-500/10">
          <AlertCircle className="h-6 w-6 text-red-500" />
        </div>
        <h2 className="text-sm font-medium">Gagal Memuat Data Keywords</h2>
        <p className="text-xs text-muted-foreground">{error instanceof Error ? error.message : "Gagal memuat data keywords"}</p>
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
          <h1 className="text-xl font-semibold tracking-tight">Keywords</h1>
          <p className="text-xs text-muted-foreground mt-0.5">
            Kelola kata kunci untuk deteksi konten berbahaya
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setBulkImportOpen(true)} className="h-8 text-xs">
            <Upload className="mr-1.5 h-3.5 w-3.5" />
            Bulk Import
          </Button>
          <Button onClick={openCreate} className="h-8 text-xs">
            <Plus className="mr-1.5 h-3.5 w-3.5" />
            Add Keyword
          </Button>
        </div>
      </div>

      {/* Filter + Count */}
      <div className="flex items-center gap-2">
        <Select value={filter} onValueChange={setFilter}>
          <SelectTrigger className="w-[160px] h-8 text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {CATEGORIES.map((cat) => (
              <SelectItem key={cat.value} value={cat.value} className="text-xs">
                {cat.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <span className="text-xs text-muted-foreground">
          {filteredKeywords.length} keyword{filteredKeywords.length !== 1 ? "s" : ""}
        </span>
      </div>

      {/* Keywords Table */}
      <Card className="border-border/50">
        <CardContent className="p-0">
          {filteredKeywords.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider pl-4">Keyword</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Category</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Type</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Weight</TableHead>
                  <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider text-right pr-4">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredKeywords.map((kw) => (
                  <TableRow key={kw.id} className="group">
                    <TableCell className="pl-4">
                      <span className="text-[13px] font-mono bg-muted/60 px-1.5 py-0.5 rounded text-[12px]">
                        {kw.keyword}
                      </span>
                    </TableCell>
                    <TableCell>
                      <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ${getCategoryColor(kw.category)}`}>
                        {kw.category}
                      </span>
                    </TableCell>
                    <TableCell>
                      {kw.is_regex ? (
                        <span className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 bg-violet-500/10 text-violet-700 dark:text-violet-400 text-[11px] font-mono">
                          <Code2 className="h-3 w-3" />
                          regex
                        </span>
                      ) : (
                        <span className="text-xs text-muted-foreground">plain</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <span className="inline-flex items-center justify-center h-5 min-w-[20px] rounded bg-muted px-1 text-[11px] font-medium">
                        {kw.weight}
                      </span>
                    </TableCell>
                    <TableCell className="text-right pr-4">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDelete(kw.id)}
                        className="h-7 w-7 p-0 text-muted-foreground hover:text-red-600 opacity-0 group-hover:opacity-100 transition-opacity"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <div className="flex flex-col items-center justify-center py-12">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
                <Tag className="h-5 w-5 text-muted-foreground" />
              </div>
              <p className="text-sm font-medium mt-3">
                {filter !== "all"
                  ? `Tidak ada keyword "${filter}"`
                  : "Belum ada keyword"}
              </p>
              <p className="text-xs text-muted-foreground mt-0.5">
                {filter !== "all"
                  ? "Coba pilih kategori lain"
                  : "Tambahkan keyword baru untuk memulai"}
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Add Keyword Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader>
            <DialogTitle className="text-sm font-medium">Add Keyword</DialogTitle>
            <DialogDescription className="text-xs">
              Tambahkan kata kunci baru untuk deteksi konten
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3 py-1">
            <div className="space-y-1.5">
              <Label htmlFor="keyword" className="text-xs">Keyword</Label>
              <Input
                id="keyword"
                value={form.keyword}
                onChange={(e) => setForm((f) => ({ ...f, keyword: e.target.value }))}
                placeholder="Masukkan kata kunci atau regex pattern"
                className="h-8 text-[13px]"
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs">Category</Label>
              <Select
                value={form.category}
                onValueChange={(value) => setForm((f) => ({ ...f, category: value }))}
              >
                <SelectTrigger className="h-8 text-[13px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="gambling" className="text-xs">Gambling</SelectItem>
                  <SelectItem value="defacement" className="text-xs">Defacement</SelectItem>
                  <SelectItem value="malware" className="text-xs">Malware</SelectItem>
                  <SelectItem value="phishing" className="text-xs">Phishing</SelectItem>
                  <SelectItem value="porn" className="text-xs">Porn</SelectItem>
                  <SelectItem value="custom" className="text-xs">Custom</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center space-x-2">
              <Checkbox
                id="is_regex"
                checked={form.is_regex}
                onCheckedChange={(checked) =>
                  setForm((f) => ({ ...f, is_regex: checked === true }))
                }
              />
              <Label htmlFor="is_regex" className="text-xs">
                Is Regex Pattern
              </Label>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="weight" className="text-xs">Weight</Label>
              <Input
                id="weight"
                type="number"
                min={1}
                max={10}
                value={form.weight}
                onChange={(e) =>
                  setForm((f) => ({ ...f, weight: parseInt(e.target.value) || 1 }))
                }
                className="h-8 text-[13px]"
              />
              <p className="text-[11px] text-muted-foreground">
                Bobot keyword (1-10), semakin tinggi semakin prioritas
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)} className="h-8 text-xs">
              Batal
            </Button>
            <Button onClick={handleSubmit} disabled={createMutation.isPending} className="h-8 text-xs">
              {createMutation.isPending && <RefreshCw className="mr-1.5 h-3.5 w-3.5 animate-spin" />}
              Add
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <BulkImportDialog<{ keyword: string; category: string; is_regex?: boolean; weight?: number }>
        open={bulkImportOpen}
        onOpenChange={setBulkImportOpen}
        onImport={async (data) => {
          const res = await keywordApi.bulkImport(data)
          return res.data
        }}
        title="Bulk Import Keywords"
        description="Import banyak keyword sekaligus via CSV"
        columns={[
          { key: "keyword", label: "Keyword" },
          { key: "category", label: "Category" },
          { key: "is_regex", label: "Is Regex" },
          { key: "weight", label: "Weight" },
        ]}
        exampleRows={[
          ["judi online", "gambling", "false", "5"],
          ["slot\\s*gacor", "gambling", "true", "8"],
          ["hacked by", "defacement", "false", "10"],
        ]}
        parseRow={(row) => {
          if (!row[0]) return null
          return {
            keyword: row[0],
            category: row[1] || "custom",
            is_regex: row[2]?.toLowerCase() === "true",
            weight: parseInt(row[3]) || 5,
          }
        }}
        onSuccess={() => refetch()}
      />
    </div>
  )
}

function KeywordsSkeleton() {
  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <Skeleton className="h-6 w-28 mb-1" />
          <Skeleton className="h-3.5 w-56" />
        </div>
        <Skeleton className="h-8 w-28" />
      </div>
      <div className="flex items-center gap-2">
        <Skeleton className="h-8 w-[160px]" />
        <Skeleton className="h-3.5 w-20" />
      </div>
      <Card className="border-border/50">
        <CardContent className="p-0">
          <div className="divide-y divide-border/50">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="flex items-center gap-4 px-4 py-3">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-5 w-16 rounded-full" />
                <Skeleton className="h-4 w-12" />
                <Skeleton className="h-5 w-6" />
                <Skeleton className="h-7 w-7 ml-auto rounded" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
