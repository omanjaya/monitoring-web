"use client"

import { useState } from "react"
import {
    Building2,
    RefreshCw,
    AlertCircle,
    Plus,
    Trash2,
    Mail,
    Phone,
    Upload,
} from "lucide-react"
import { toast } from "sonner"
import { useQuery } from "@tanstack/react-query"

import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog"
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table"

import { opdApi, type OPD } from "@/lib/api"
import { useMutationAction } from "@/hooks/use-mutation-action"
import { BulkImportDialog } from "@/components/bulk-import-dialog"

export default function OPDPage() {
    const [dialogOpen, setDialogOpen] = useState(false)
    const [bulkImportOpen, setBulkImportOpen] = useState(false)

    const [form, setForm] = useState({
        name: "",
        code: "",
        contact_email: "",
        contact_phone: "",
    })

    const { data: items = [], isLoading: loading, error, refetch } = useQuery({
        queryKey: ["opd"],
        queryFn: async () => {
            const res = await opdApi.list()
            return (res.data || []) as OPD[]
        },
    })

    const createMutation = useMutationAction({
        mutationFn: (payload: { name: string; code: string; contact_email?: string; contact_phone?: string }) =>
            opdApi.create(payload),
        successMessage: "OPD berhasil ditambahkan",
        invalidateKeys: ["opd"],
        onSuccess: () => setDialogOpen(false),
    })

    const openCreate = () => {
        setForm({ name: "", code: "", contact_email: "", contact_phone: "" })
        setDialogOpen(true)
    }

    const handleSubmit = () => {
        if (!form.name || !form.code) {
            toast.error("Nama dan Kode OPD harus diisi")
            return
        }
        createMutation.mutate({
            name: form.name,
            code: form.code.toUpperCase(),
            contact_email: form.contact_email || undefined,
            contact_phone: form.contact_phone || undefined,
        })
    }

    const handleDelete = async () => {
        toast.error("Fitur hapus OPD belum diimplementasikan di sisi server")
    }

    if (loading) return <OPDSkeleton />

    if (error) {
        return (
            <div className="flex flex-col items-center justify-center gap-3 py-20">
                <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-500/10">
                    <AlertCircle className="h-6 w-6 text-red-500" />
                </div>
                <h2 className="text-sm font-medium">Gagal Memuat Data OPD</h2>
                <p className="text-xs text-muted-foreground">{error instanceof Error ? error.message : "Gagal memuat data OPD"}</p>
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
                    <h1 className="text-xl font-semibold tracking-tight">Organisasi Perangkat Daerah</h1>
                    <p className="text-xs text-muted-foreground mt-0.5">
                        Kelola master data instansi/OPD yang dinaungi
                    </p>
                </div>
                <div className="flex gap-2">
                    <Button variant="outline" onClick={() => setBulkImportOpen(true)} className="h-8 text-xs">
                        <Upload className="mr-1.5 h-3.5 w-3.5" />
                        Bulk Import
                    </Button>
                    <Button onClick={openCreate} className="h-8 text-xs">
                        <Plus className="mr-1.5 h-3.5 w-3.5" />
                        Tambah OPD
                    </Button>
                </div>
            </div>

            {/* OPD Table */}
            <Card className="border-border/50">
                <CardContent className="p-0">
                    {items.length > 0 ? (
                        <Table>
                            <TableHeader>
                                <TableRow className="hover:bg-transparent">
                                    <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider pl-4">Nama OPD</TableHead>
                                    <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Kode</TableHead>
                                    <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Kontak</TableHead>
                                    <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider text-right pr-4">Aksi</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {items.map((item) => (
                                    <TableRow key={item.id} className="group">
                                        <TableCell className="pl-4">
                                            <span className="text-[13px] font-medium">{item.name}</span>
                                        </TableCell>
                                        <TableCell>
                                            <span className="inline-flex items-center rounded px-1.5 py-0.5 bg-muted text-[11px] font-mono font-medium">
                                                {item.code}
                                            </span>
                                        </TableCell>
                                        <TableCell>
                                            <div className="flex flex-col gap-1">
                                                {item.contact_email && (
                                                    <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                                                        <Mail className="h-3 w-3 shrink-0" />
                                                        <span className="truncate">{item.contact_email}</span>
                                                    </div>
                                                )}
                                                {item.contact_phone && (
                                                    <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                                                        <Phone className="h-3 w-3 shrink-0" />
                                                        <span>{item.contact_phone}</span>
                                                    </div>
                                                )}
                                                {!item.contact_email && !item.contact_phone && (
                                                    <span className="text-xs text-muted-foreground/50">-</span>
                                                )}
                                            </div>
                                        </TableCell>
                                        <TableCell className="text-right pr-4">
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => handleDelete()}
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
                                <Building2 className="h-5 w-5 text-muted-foreground" />
                            </div>
                            <p className="text-sm font-medium mt-3">Belum ada data OPD</p>
                            <p className="text-xs text-muted-foreground mt-0.5">Tambahkan instansi baru untuk memulai</p>
                        </div>
                    )}
                </CardContent>
            </Card>

            {/* Add OPD Dialog */}
            <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                <DialogContent className="sm:max-w-[400px]">
                    <DialogHeader>
                        <DialogTitle className="text-sm font-medium">Tambah OPD</DialogTitle>
                        <DialogDescription className="text-xs">
                            Masukkan informasi OPD atau Dinas terkait.
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-3 py-1">
                        <div className="space-y-1.5">
                            <Label htmlFor="code" className="text-xs">Kode / Singkatan <span className="text-red-500">*</span></Label>
                            <Input
                                id="code"
                                value={form.code}
                                onChange={(e) => setForm((f) => ({ ...f, code: e.target.value.toUpperCase() }))}
                                placeholder="mis. BAPPEDA"
                                className="uppercase h-8 text-[13px]"
                            />
                        </div>
                        <div className="space-y-1.5">
                            <Label htmlFor="name" className="text-xs">Nama Lengkap OPD <span className="text-red-500">*</span></Label>
                            <Input
                                id="name"
                                value={form.name}
                                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                                placeholder="Badan Perencanaan Pembangunan Daerah"
                                className="h-8 text-[13px]"
                            />
                        </div>
                        <div className="space-y-1.5">
                            <Label htmlFor="email" className="text-xs">Email PIC (Opsional)</Label>
                            <Input
                                id="email"
                                type="email"
                                value={form.contact_email}
                                onChange={(e) => setForm((f) => ({ ...f, contact_email: e.target.value }))}
                                placeholder="pic@domain.go.id"
                                className="h-8 text-[13px]"
                            />
                        </div>
                        <div className="space-y-1.5">
                            <Label htmlFor="phone" className="text-xs">Telepon PIC (Opsional)</Label>
                            <Input
                                id="phone"
                                value={form.contact_phone}
                                onChange={(e) => setForm((f) => ({ ...f, contact_phone: e.target.value }))}
                                placeholder="08123456789"
                                className="h-8 text-[13px]"
                            />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDialogOpen(false)} className="h-8 text-xs">
                            Batal
                        </Button>
                        <Button onClick={handleSubmit} disabled={createMutation.isPending} className="h-8 text-xs">
                            {createMutation.isPending && <RefreshCw className="mr-1.5 h-3.5 w-3.5 animate-spin" />}
                            Simpan
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            <BulkImportDialog<{ name: string; code: string; contact_email?: string; contact_phone?: string }>
                open={bulkImportOpen}
                onOpenChange={setBulkImportOpen}
                onImport={async (data) => {
                    const res = await opdApi.bulkImport(data)
                    return res.data
                }}
                title="Bulk Import OPD"
                description="Import banyak OPD sekaligus via CSV"
                columns={[
                    { key: "code", label: "Kode" },
                    { key: "name", label: "Nama OPD" },
                    { key: "contact_email", label: "Email" },
                    { key: "contact_phone", label: "Telepon" },
                ]}
                exampleRows={[
                    ["BAPPEDA", "Badan Perencanaan Pembangunan Daerah", "bappeda@baliprov.go.id", "0361123456"],
                    ["DISKOMINFOS", "Dinas Komunikasi Informatika dan Statistik", "diskominfos@baliprov.go.id", ""],
                ]}
                parseRow={(row) => {
                    if (!row[0] || !row[1]) return null
                    return {
                        code: row[0].toUpperCase(),
                        name: row[1],
                        contact_email: row[2] || undefined,
                        contact_phone: row[3] || undefined,
                    }
                }}
                onSuccess={() => refetch()}
            />
        </div>
    )
}

function OPDSkeleton() {
    return (
        <div className="space-y-5">
            <div className="flex items-center justify-between">
                <div>
                    <Skeleton className="h-6 w-48 mb-1" />
                    <Skeleton className="h-3.5 w-56" />
                </div>
                <Skeleton className="h-8 w-28" />
            </div>
            <Card className="border-border/50">
                <CardContent className="p-0">
                    <div className="divide-y divide-border/50">
                        {Array.from({ length: 5 }).map((_, i) => (
                            <div key={i} className="flex items-center gap-4 px-4 py-3">
                                <Skeleton className="h-4 w-40" />
                                <Skeleton className="h-5 w-16" />
                                <div className="flex-1">
                                    <Skeleton className="h-3.5 w-32 mb-1" />
                                    <Skeleton className="h-3.5 w-24" />
                                </div>
                                <Skeleton className="h-7 w-7 rounded" />
                            </div>
                        ))}
                    </div>
                </CardContent>
            </Card>
        </div>
    )
}
