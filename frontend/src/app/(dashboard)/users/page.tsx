"use client"

import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { userApi, type User, type UserCreate } from "@/lib/api"
import { useMutationAction } from "@/hooks/use-mutation-action"
import { formatDate } from "@/lib/utils"
import {
  Loader2,
  MoreHorizontal,
  Pencil,
  KeyRound,
  Trash2,
  UserPlus,
  Users,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Switch } from "@/components/ui/switch"
import { toast } from "sonner"

function getInitials(name: string) {
  return name
    .split(" ")
    .map((n) => n[0])
    .slice(0, 2)
    .join("")
    .toUpperCase()
}

function getAvatarColor(name: string) {
  const colors = [
    "bg-blue-500/15 text-blue-600",
    "bg-emerald-500/15 text-emerald-600",
    "bg-violet-500/15 text-violet-600",
    "bg-amber-500/15 text-amber-600",
    "bg-rose-500/15 text-rose-600",
    "bg-cyan-500/15 text-cyan-600",
    "bg-pink-500/15 text-pink-600",
    "bg-teal-500/15 text-teal-600",
  ]
  let hash = 0
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash)
  }
  return colors[Math.abs(hash) % colors.length]
}

function formatRelativeLogin(dateStr: string) {
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return "Baru saja"
  if (diffMins < 60) return `${diffMins} menit lalu`
  if (diffHours < 24) return `${diffHours} jam lalu`
  if (diffDays < 7) return `${diffDays} hari lalu`
  return formatDate(dateStr)
}

export default function UsersPage() {
  const { data: users = [], isLoading } = useQuery<User[]>({
    queryKey: ["users"],
    queryFn: async () => {
      const res = await userApi.list()
      return res.data || []
    },
  })

  // Add User dialog
  const [addDialogOpen, setAddDialogOpen] = useState(false)
  const [newUser, setNewUser] = useState<UserCreate>({
    username: "",
    email: "",
    password: "",
    full_name: "",
    phone: "",
    role: "viewer",
  })

  // Edit User dialog
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<User | null>(null)
  const [editForm, setEditForm] = useState({ email: "", full_name: "", phone: "", is_active: true })

  // Reset Password dialog
  const [resetPwDialogOpen, setResetPwDialogOpen] = useState(false)
  const [resetPwTarget, setResetPwTarget] = useState<User | null>(null)
  const [newPassword, setNewPassword] = useState("")

  // Delete dialog
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<User | null>(null)

  const addMutation = useMutationAction({
    mutationFn: (data: UserCreate) => userApi.create(data),
    successMessage: "User berhasil ditambahkan",
    invalidateKeys: ["users"],
    onSuccess: () => {
      setAddDialogOpen(false)
      setNewUser({ username: "", email: "", password: "", full_name: "", phone: "", role: "viewer" })
    },
  })

  const editMutation = useMutationAction({
    mutationFn: ({ id, data }: { id: number; data: typeof editForm }) => userApi.update(id, data),
    successMessage: "User berhasil diupdate",
    invalidateKeys: ["users"],
    onSuccess: () => setEditDialogOpen(false),
  })

  const resetPwMutation = useMutationAction({
    mutationFn: ({ id, password }: { id: number; password: string }) => userApi.resetPassword(id, password),
    successMessage: "Password berhasil direset",
    onSuccess: () => setResetPwDialogOpen(false),
  })

  const deleteMutation = useMutationAction({
    mutationFn: (id: number) => userApi.delete(id),
    successMessage: "User berhasil dihapus",
    invalidateKeys: ["users"],
    onSuccess: () => setDeleteDialogOpen(false),
  })

  const openEditDialog = (user: User) => {
    setEditTarget(user)
    setEditForm({ email: user.email, full_name: user.full_name, phone: user.phone || "", is_active: user.is_active })
    setEditDialogOpen(true)
  }

  const handleAddUser = () => {
    if (!newUser.username || !newUser.email || !newUser.password || !newUser.full_name) {
      toast.error("Harap isi semua field wajib")
      return
    }
    addMutation.mutate(newUser)
  }

  const handleResetPassword = () => {
    if (!resetPwTarget || !newPassword) {
      toast.error("Password baru tidak boleh kosong")
      return
    }
    resetPwMutation.mutate({ id: resetPwTarget.id, password: newPassword })
  }

  const getRoleBadge = (role: string) => {
    switch (role) {
      case "super_admin":
        return (
          <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium bg-primary/10 text-primary border border-primary/20">
            Super Admin
          </span>
        )
      case "admin":
        return (
          <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium bg-blue-500/10 text-blue-600 border border-blue-500/20">
            Admin
          </span>
        )
      default:
        return (
          <span className="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium bg-muted text-muted-foreground border border-border/50">
            Viewer
          </span>
        )
    }
  }

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Users</h1>
          <p className="text-[13px] text-muted-foreground mt-0.5">Kelola akun pengguna sistem</p>
        </div>
        <Button className="h-8 text-xs" onClick={() => setAddDialogOpen(true)}>
          <UserPlus className="size-3.5 mr-1.5" /> Tambah User
        </Button>
      </div>

      {/* Table */}
      <Card className="border-border/50">
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">User</TableHead>
                <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Email</TableHead>
                <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Role</TableHead>
                <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Status</TableHead>
                <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Login Terakhir</TableHead>
                <TableHead className="text-xs font-medium text-muted-foreground uppercase tracking-wider text-right">Aksi</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-16">
                    <Loader2 className="size-5 animate-spin mx-auto text-muted-foreground" />
                  </TableCell>
                </TableRow>
              ) : users.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-16">
                    <div className="flex flex-col items-center gap-2">
                      <div className="size-10 rounded-full bg-muted flex items-center justify-center">
                        <Users className="size-5 text-muted-foreground" />
                      </div>
                      <p className="text-sm font-medium text-foreground">Belum ada user</p>
                      <p className="text-xs text-muted-foreground">Tambahkan user baru untuk memulai</p>
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                users.map((user) => (
                  <TableRow key={user.id} className="group">
                    <TableCell className="py-3">
                      <div className="flex items-center gap-2.5">
                        <div className={`size-7 rounded-full flex items-center justify-center text-[11px] font-semibold ${getAvatarColor(user.full_name || user.username)}`}>
                          {getInitials(user.full_name || user.username)}
                        </div>
                        <div>
                          <p className="text-[13px] font-medium leading-tight">{user.full_name}</p>
                          <p className="text-xs text-muted-foreground leading-tight">@{user.username}</p>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="text-[13px] text-muted-foreground">{user.email}</TableCell>
                    <TableCell>{getRoleBadge(user.role)}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1.5">
                        <div className={`size-1.5 rounded-full ${user.is_active ? "bg-emerald-500" : "bg-muted-foreground/40"}`} />
                        <span className="text-[13px] text-muted-foreground">
                          {user.is_active ? "Active" : "Inactive"}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="text-[13px] text-muted-foreground">
                      {user.last_login_at ? formatRelativeLogin(user.last_login_at) : (
                        <span className="text-muted-foreground/50">Belum login</span>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon" className="h-7 w-7 opacity-0 group-hover:opacity-100 transition-opacity">
                            <MoreHorizontal className="size-3.5" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="w-40">
                          <DropdownMenuItem onClick={() => openEditDialog(user)} className="text-[13px]">
                            <Pencil className="size-3.5 mr-2" /> Edit
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => { setResetPwTarget(user); setNewPassword(""); setResetPwDialogOpen(true) }} className="text-[13px]">
                            <KeyRound className="size-3.5 mr-2" /> Reset Password
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            className="text-[13px] text-destructive focus:bg-destructive/10 focus:text-destructive"
                            onClick={() => { setDeleteTarget(user); setDeleteDialogOpen(true) }}
                          >
                            <Trash2 className="size-3.5 mr-2" /> Hapus
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Add User Dialog */}
      <Dialog open={addDialogOpen} onOpenChange={setAddDialogOpen}>
        <DialogContent className="sm:max-w-[440px]">
          <DialogHeader>
            <DialogTitle className="text-sm font-medium">Tambah User Baru</DialogTitle>
            <DialogDescription className="text-xs">Buat akun pengguna baru untuk sistem monitoring</DialogDescription>
          </DialogHeader>
          <div className="space-y-3 pt-1">
            <div className="space-y-1.5">
              <Label htmlFor="add-username" className="text-xs">Username <span className="text-destructive">*</span></Label>
              <Input id="add-username" className="h-9 text-[13px]" placeholder="username" value={newUser.username} onChange={(e) => setNewUser({ ...newUser, username: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="add-email" className="text-xs">Email <span className="text-destructive">*</span></Label>
              <Input id="add-email" className="h-9 text-[13px]" type="email" placeholder="email@example.com" value={newUser.email} onChange={(e) => setNewUser({ ...newUser, email: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="add-fullname" className="text-xs">Nama Lengkap <span className="text-destructive">*</span></Label>
              <Input id="add-fullname" className="h-9 text-[13px]" placeholder="Nama Lengkap" value={newUser.full_name} onChange={(e) => setNewUser({ ...newUser, full_name: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="add-phone" className="text-xs">Telepon</Label>
              <Input id="add-phone" className="h-9 text-[13px]" placeholder="+62..." value={newUser.phone} onChange={(e) => setNewUser({ ...newUser, phone: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="add-password" className="text-xs">Password <span className="text-destructive">*</span></Label>
              <Input id="add-password" className="h-9 text-[13px]" type="password" placeholder="Minimal 8 karakter" value={newUser.password} onChange={(e) => setNewUser({ ...newUser, password: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs">Role</Label>
              <Select value={newUser.role || "viewer"} onValueChange={(v) => setNewUser({ ...newUser, role: v })}>
                <SelectTrigger className="h-9 text-[13px]"><SelectValue placeholder="Pilih role" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="super_admin" className="text-[13px]">Super Admin</SelectItem>
                  <SelectItem value="admin" className="text-[13px]">Admin</SelectItem>
                  <SelectItem value="viewer" className="text-[13px]">Viewer</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter className="pt-2">
            <Button variant="outline" className="h-8 text-xs" onClick={() => setAddDialogOpen(false)}>Batal</Button>
            <Button className="h-8 text-xs" onClick={handleAddUser} disabled={addMutation.isPending}>
              {addMutation.isPending && <Loader2 className="size-3.5 animate-spin mr-1.5" />}
              Tambah User
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit User Dialog */}
      <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
        <DialogContent className="sm:max-w-[440px]">
          <DialogHeader>
            <DialogTitle className="text-sm font-medium">Edit User</DialogTitle>
            <DialogDescription className="text-xs">Edit data user: {editTarget?.username}</DialogDescription>
          </DialogHeader>
          <div className="space-y-3 pt-1">
            <div className="space-y-1.5">
              <Label htmlFor="edit-email" className="text-xs">Email</Label>
              <Input id="edit-email" className="h-9 text-[13px]" type="email" value={editForm.email} onChange={(e) => setEditForm({ ...editForm, email: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="edit-fullname" className="text-xs">Nama Lengkap</Label>
              <Input id="edit-fullname" className="h-9 text-[13px]" value={editForm.full_name} onChange={(e) => setEditForm({ ...editForm, full_name: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="edit-phone" className="text-xs">Telepon</Label>
              <Input id="edit-phone" className="h-9 text-[13px]" value={editForm.phone} onChange={(e) => setEditForm({ ...editForm, phone: e.target.value })} />
            </div>
            <div className="flex items-center justify-between rounded-lg border border-border/50 px-3 py-2.5">
              <div>
                <Label htmlFor="edit-active" className="text-[13px] font-medium cursor-pointer">Status Aktif</Label>
                <p className="text-xs text-muted-foreground mt-0.5">User dapat login ke sistem</p>
              </div>
              <Switch id="edit-active" checked={editForm.is_active} onCheckedChange={(checked) => setEditForm({ ...editForm, is_active: checked })} />
            </div>
          </div>
          <DialogFooter className="pt-2">
            <Button variant="outline" className="h-8 text-xs" onClick={() => setEditDialogOpen(false)}>Batal</Button>
            <Button className="h-8 text-xs" onClick={() => editTarget && editMutation.mutate({ id: editTarget.id, data: editForm })} disabled={editMutation.isPending}>
              {editMutation.isPending && <Loader2 className="size-3.5 animate-spin mr-1.5" />}
              Simpan
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Reset Password Dialog */}
      <Dialog open={resetPwDialogOpen} onOpenChange={setResetPwDialogOpen}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader>
            <DialogTitle className="text-sm font-medium">Reset Password</DialogTitle>
            <DialogDescription className="text-xs">Reset password untuk user: <span className="font-medium text-foreground">{resetPwTarget?.username}</span></DialogDescription>
          </DialogHeader>
          <div className="space-y-3 pt-1">
            <div className="space-y-1.5">
              <Label htmlFor="new-password" className="text-xs">Password Baru</Label>
              <Input id="new-password" className="h-9 text-[13px]" type="password" placeholder="Minimal 8 karakter" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
            </div>
          </div>
          <DialogFooter className="pt-2">
            <Button variant="outline" className="h-8 text-xs" onClick={() => setResetPwDialogOpen(false)}>Batal</Button>
            <Button className="h-8 text-xs" onClick={handleResetPassword} disabled={resetPwMutation.isPending}>
              {resetPwMutation.isPending && <Loader2 className="size-3.5 animate-spin mr-1.5" />}
              Reset Password
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader>
            <DialogTitle className="text-sm font-medium">Hapus User</DialogTitle>
            <DialogDescription className="text-xs">
              Apakah Anda yakin ingin menghapus user <strong className="text-foreground">{deleteTarget?.username}</strong>? Tindakan ini tidak dapat dibatalkan.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="pt-2">
            <Button variant="outline" className="h-8 text-xs" onClick={() => setDeleteDialogOpen(false)}>Batal</Button>
            <Button variant="destructive" className="h-8 text-xs" onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)} disabled={deleteMutation.isPending}>
              {deleteMutation.isPending && <Loader2 className="size-3.5 animate-spin mr-1.5" />}
              Hapus User
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
