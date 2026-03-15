"use client"

import { useState, useCallback, useRef } from "react"
import { toast } from "sonner"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Upload,
  FileText,
  Download,
  Loader2,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  X,
  ClipboardPaste,
} from "lucide-react"
import type { BulkImportResult } from "@/lib/api"

interface BulkImportDialogProps<T> {
  open: boolean
  onOpenChange: (open: boolean) => void
  onImport: (data: T[]) => Promise<BulkImportResult>
  title: string
  description: string
  columns: { key: string; label: string }[]
  exampleRows: string[][]
  parseRow: (row: string[]) => T | null
  onSuccess?: () => void
}

type Step = "input" | "preview" | "result"

export function BulkImportDialog<T>({
  open,
  onOpenChange,
  onImport,
  title,
  description,
  columns,
  exampleRows,
  parseRow,
  onSuccess,
}: BulkImportDialogProps<T>) {
  const [step, setStep] = useState<Step>("input")
  const [mode, setMode] = useState<"paste" | "file">("paste")
  const [csvText, setCsvText] = useState("")
  const [parsedRows, setParsedRows] = useState<string[][]>([])
  const [parsedData, setParsedData] = useState<T[]>([])
  const [importing, setImporting] = useState(false)
  const [result, setResult] = useState<BulkImportResult | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const reset = useCallback(() => {
    setStep("input")
    setCsvText("")
    setParsedRows([])
    setParsedData([])
    setImporting(false)
    setResult(null)
  }, [])

  const handleOpenChange = (open: boolean) => {
    if (!open) reset()
    onOpenChange(open)
  }

  const parseCSV = (text: string): string[][] => {
    const lines = text.trim().split("\n").filter(line => line.trim())
    return lines.map(line => {
      // Try tab first (tabs are unambiguous)
      if (line.includes("\t")) return line.split("\t").map(s => s.trim())
      // Handle quoted CSV fields (e.g., "Badan Perencanaan, Pembangunan")
      if (line.includes('"')) {
        const fields: string[] = []
        let current = ""
        let inQuotes = false
        for (let i = 0; i < line.length; i++) {
          const ch = line[i]
          if (ch === '"') {
            inQuotes = !inQuotes
          } else if ((ch === "," || ch === ";") && !inQuotes) {
            fields.push(current.trim())
            current = ""
          } else {
            current += ch
          }
        }
        fields.push(current.trim())
        return fields
      }
      // Simple split for unquoted fields
      if (line.includes(",")) return line.split(",").map(s => s.trim())
      if (line.includes(";")) return line.split(";").map(s => s.trim())
      return [line.trim()]
    })
  }

  const handleParse = () => {
    const rows = parseCSV(csvText)
    if (rows.length === 0) {
      toast.error("Tidak ada data yang bisa diparsing")
      return
    }

    // Check if first row is a header (matches column names)
    const firstRow = rows[0].map(s => s.toLowerCase().replace(/[^a-z0-9_]/g, ""))
    const colKeys = columns.map(c => c.key.toLowerCase())
    const isHeader = firstRow.some(cell => colKeys.includes(cell))

    const dataRows = isHeader ? rows.slice(1) : rows
    if (dataRows.length === 0) {
      toast.error("Tidak ada data (hanya header yang terdeteksi)")
      return
    }

    const parsed: T[] = []
    const validRows: string[][] = []

    for (const row of dataRows) {
      const item = parseRow(row)
      if (item) {
        parsed.push(item)
        validRows.push(row)
      }
    }

    if (parsed.length === 0) {
      toast.error("Tidak ada data valid yang bisa diimport")
      return
    }

    setParsedRows(validRows)
    setParsedData(parsed)
    setStep("preview")
  }

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = (event) => {
      const text = event.target?.result as string
      setCsvText(text)
    }
    reader.readAsText(file)
    // Reset file input
    if (fileInputRef.current) fileInputRef.current.value = ""
  }

  const handleImport = async () => {
    setImporting(true)
    try {
      const res = await onImport(parsedData)
      setResult(res)
      setStep("result")
      if ((res.created?.length ?? 0) > 0) {
        onSuccess?.()
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Import gagal")
    } finally {
      setImporting(false)
    }
  }

  const downloadTemplate = () => {
    const header = columns.map(c => c.label).join(",")
    const rows = exampleRows.map(r => r.join(",")).join("\n")
    const csv = `${header}\n${rows}`
    const blob = new Blob([csv], { type: "text/csv" })
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = `template_${title.toLowerCase().replace(/\s+/g, "_")}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  const createdCount = result?.created?.length ?? 0
  const skippedCount = result?.skipped?.length ?? 0
  const failedCount = result?.failed?.length ?? 0

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[85vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="text-sm font-medium">{title}</DialogTitle>
          <DialogDescription className="text-xs">{description}</DialogDescription>
        </DialogHeader>

        {step === "input" && (
          <div className="space-y-4 flex-1 overflow-hidden">
            {/* Mode tabs */}
            <div className="flex gap-1 border-b border-border/50">
              <button
                className={`flex items-center gap-1.5 px-3 py-2 text-xs font-medium border-b-2 transition-colors -mb-px ${
                  mode === "paste"
                    ? "border-primary text-primary"
                    : "border-transparent text-muted-foreground hover:text-foreground"
                }`}
                onClick={() => setMode("paste")}
              >
                <ClipboardPaste className="size-3.5" />
                Paste Data
              </button>
              <button
                className={`flex items-center gap-1.5 px-3 py-2 text-xs font-medium border-b-2 transition-colors -mb-px ${
                  mode === "file"
                    ? "border-primary text-primary"
                    : "border-transparent text-muted-foreground hover:text-foreground"
                }`}
                onClick={() => setMode("file")}
              >
                <Upload className="size-3.5" />
                Upload CSV
              </button>
            </div>

            {mode === "paste" ? (
              <div className="space-y-2">
                <Textarea
                  placeholder={`Paste data CSV/TSV di sini...\nContoh:\n${exampleRows.map(r => r.join("\t")).join("\n")}`}
                  value={csvText}
                  onChange={(e) => setCsvText(e.target.value)}
                  className="min-h-[200px] font-mono text-xs resize-none"
                />
                <p className="text-[11px] text-muted-foreground">
                  Format: pisahkan kolom dengan Tab, Koma, atau Titik Koma. Satu baris per item.
                </p>
              </div>
            ) : (
              <div className="space-y-3">
                <div
                  className="flex flex-col items-center justify-center gap-3 rounded-lg border-2 border-dashed border-border/60 p-8 cursor-pointer hover:border-primary/40 transition-colors"
                  onClick={() => fileInputRef.current?.click()}
                >
                  <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
                    <FileText className="size-5 text-muted-foreground" />
                  </div>
                  <div className="text-center">
                    <p className="text-xs font-medium">Klik untuk upload file CSV</p>
                    <p className="text-[11px] text-muted-foreground mt-0.5">
                      pilih file .csv dari komputer
                    </p>
                  </div>
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".csv,.txt,.tsv"
                    onChange={handleFileUpload}
                    className="hidden"
                  />
                </div>
                {csvText && (
                  <div className="flex items-center gap-2 rounded-lg bg-muted/50 px-3 py-2">
                    <FileText className="size-3.5 text-muted-foreground shrink-0" />
                    <span className="text-xs flex-1 truncate">File loaded ({csvText.split("\n").filter(l => l.trim()).length} baris)</span>
                    <button onClick={() => setCsvText("")} className="text-muted-foreground hover:text-foreground">
                      <X className="size-3.5" />
                    </button>
                  </div>
                )}
              </div>
            )}

            {/* Column info */}
            <div className="rounded-lg border border-border/50 bg-muted/30 p-3">
              <p className="text-[11px] font-medium text-muted-foreground mb-1.5">Format kolom:</p>
              <div className="flex flex-wrap gap-1.5">
                {columns.map((col, i) => (
                  <span
                    key={col.key}
                    className="inline-flex items-center rounded px-1.5 py-0.5 bg-background text-[11px] font-mono border border-border/50"
                  >
                    <span className="text-muted-foreground mr-1">{i + 1}.</span>
                    {col.label}
                  </span>
                ))}
              </div>
            </div>
          </div>
        )}

        {step === "preview" && (
          <div className="space-y-3 flex-1 overflow-hidden">
            <div className="flex items-center justify-between">
              <p className="text-xs text-muted-foreground">
                <span className="font-medium text-foreground">{parsedData.length}</span> item siap diimport
              </p>
              <Button variant="ghost" className="h-7 text-[11px]" onClick={() => setStep("input")}>
                Kembali
              </Button>
            </div>
            <div className="rounded-lg border border-border/50 overflow-auto max-h-[300px]">
              <Table>
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="text-[11px] font-medium text-muted-foreground w-8">#</TableHead>
                    {columns.map(col => (
                      <TableHead key={col.key} className="text-[11px] font-medium text-muted-foreground">
                        {col.label}
                      </TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {parsedRows.slice(0, 50).map((row, i) => (
                    <TableRow key={i}>
                      <TableCell className="text-[11px] text-muted-foreground">{i + 1}</TableCell>
                      {columns.map((col, j) => (
                        <TableCell key={col.key} className="text-xs">
                          {row[j] || <span className="text-muted-foreground/50">-</span>}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
            {parsedRows.length > 50 && (
              <p className="text-[11px] text-muted-foreground text-center">
                ...dan {parsedRows.length - 50} item lainnya
              </p>
            )}
          </div>
        )}

        {step === "result" && result && (
          <div className="space-y-4 flex-1 overflow-hidden">
            {/* Summary cards */}
            <div className="grid grid-cols-3 gap-2">
              <div className="rounded-lg border border-emerald-200 dark:border-emerald-500/20 bg-emerald-50 dark:bg-emerald-500/5 p-3 text-center">
                <CheckCircle2 className="size-4 text-emerald-600 dark:text-emerald-400 mx-auto mb-1" />
                <p className="text-lg font-semibold text-emerald-700 dark:text-emerald-300">{createdCount}</p>
                <p className="text-[11px] text-emerald-600 dark:text-emerald-400">Berhasil</p>
              </div>
              <div className="rounded-lg border border-amber-200 dark:border-amber-500/20 bg-amber-50 dark:bg-amber-500/5 p-3 text-center">
                <AlertTriangle className="size-4 text-amber-600 dark:text-amber-400 mx-auto mb-1" />
                <p className="text-lg font-semibold text-amber-700 dark:text-amber-300">{skippedCount}</p>
                <p className="text-[11px] text-amber-600 dark:text-amber-400">Dilewati</p>
              </div>
              <div className="rounded-lg border border-red-200 dark:border-red-500/20 bg-red-50 dark:bg-red-500/5 p-3 text-center">
                <XCircle className="size-4 text-red-600 dark:text-red-400 mx-auto mb-1" />
                <p className="text-lg font-semibold text-red-700 dark:text-red-300">{failedCount}</p>
                <p className="text-[11px] text-red-600 dark:text-red-400">Gagal</p>
              </div>
            </div>

            {/* Details */}
            <div className="space-y-2 overflow-auto max-h-[200px]">
              {result.created && result.created.length > 0 && (
                <div className="rounded-lg bg-emerald-50 dark:bg-emerald-500/5 p-2.5">
                  <p className="text-[11px] font-medium text-emerald-700 dark:text-emerald-400 mb-1">Berhasil ditambahkan:</p>
                  <div className="flex flex-wrap gap-1">
                    {result.created.map((item, i) => (
                      <span key={i} className="inline-flex rounded px-1.5 py-0.5 bg-emerald-100 dark:bg-emerald-500/10 text-[11px] text-emerald-800 dark:text-emerald-300">
                        {item}
                      </span>
                    ))}
                  </div>
                </div>
              )}
              {result.skipped && result.skipped.length > 0 && (
                <div className="rounded-lg bg-amber-50 dark:bg-amber-500/5 p-2.5">
                  <p className="text-[11px] font-medium text-amber-700 dark:text-amber-400 mb-1">Dilewati (sudah ada):</p>
                  <div className="flex flex-wrap gap-1">
                    {result.skipped.map((item, i) => (
                      <span key={i} className="inline-flex rounded px-1.5 py-0.5 bg-amber-100 dark:bg-amber-500/10 text-[11px] text-amber-800 dark:text-amber-300">
                        {item}
                      </span>
                    ))}
                  </div>
                </div>
              )}
              {result.failed && result.failed.length > 0 && (
                <div className="rounded-lg bg-red-50 dark:bg-red-500/5 p-2.5">
                  <p className="text-[11px] font-medium text-red-700 dark:text-red-400 mb-1">Gagal:</p>
                  <div className="space-y-1">
                    {result.failed.map((item, i) => (
                      <div key={i} className="text-[11px] text-red-700 dark:text-red-300">
                        <span className="font-mono">{item.keyword || item.name || item.url}</span>
                        <span className="text-red-500"> — {item.error}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        <DialogFooter className="flex-row justify-between sm:justify-between">
          {step === "input" && (
            <>
              <Button
                variant="outline"
                className="h-8 text-xs"
                onClick={downloadTemplate}
              >
                <Download className="size-3.5 mr-1.5" />
                Download Template
              </Button>
              <div className="flex gap-2">
                <Button variant="outline" className="h-8 text-xs" onClick={() => handleOpenChange(false)}>
                  Batal
                </Button>
                <Button
                  className="h-8 text-xs"
                  disabled={!csvText.trim()}
                  onClick={handleParse}
                >
                  Parse Data
                </Button>
              </div>
            </>
          )}
          {step === "preview" && (
            <>
              <div />
              <div className="flex gap-2">
                <Button variant="outline" className="h-8 text-xs" onClick={() => setStep("input")}>
                  Kembali
                </Button>
                <Button
                  className="h-8 text-xs"
                  disabled={importing}
                  onClick={handleImport}
                >
                  {importing && <Loader2 className="size-3.5 animate-spin mr-1.5" />}
                  Import {parsedData.length} Item
                </Button>
              </div>
            </>
          )}
          {step === "result" && (
            <>
              <div />
              <Button className="h-8 text-xs" onClick={() => handleOpenChange(false)}>
                Selesai
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
