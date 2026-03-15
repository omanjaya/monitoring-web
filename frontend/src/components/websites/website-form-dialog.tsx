"use client"

import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useQuery } from "@tanstack/react-query"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { websiteApi, opdApi } from "@/lib/api"
import type { Website, OPD } from "@/lib/api"
import { useMutationAction } from "@/hooks/use-mutation-action"
import { Loader2 } from "lucide-react"

const websiteSchema = z.object({
  url: z.string().min(1, "URL wajib diisi").url("URL tidak valid"),
  name: z.string().min(1, "Nama wajib diisi"),
  description: z.string().optional(),
  opd_id: z.string().optional(),
  check_interval: z.number().min(30, "Minimal 30 detik"),
  timeout: z.number().min(5, "Minimal 5 detik"),
  is_active: z.boolean(),
})

type WebsiteFormValues = z.infer<typeof websiteSchema>

interface WebsiteFormDialogProps {
  website?: Website | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess: () => void
}

export function WebsiteFormDialog({ website, open, onOpenChange, onSuccess }: WebsiteFormDialogProps) {
  const isEditing = !!website

  const { data: opdList = [] } = useQuery<OPD[]>({
    queryKey: ["opd-list"],
    queryFn: async () => {
      const res = await opdApi.list()
      return res.data || []
    },
    enabled: open,
  })

  const form = useForm<WebsiteFormValues>({
    resolver: zodResolver(websiteSchema),
    defaultValues: {
      url: "",
      name: "",
      description: "",
      opd_id: "",
      check_interval: 300,
      timeout: 30,
      is_active: true,
    },
  })

  useEffect(() => {
    if (open) {
      if (website) {
        form.reset({
          url: website.url,
          name: website.name,
          description: website.description || "",
          opd_id: website.opd_id ? String(website.opd_id) : "",
          check_interval: website.check_interval,
          timeout: website.timeout,
          is_active: website.is_active,
        })
      } else {
        form.reset({
          url: "",
          name: "",
          description: "",
          opd_id: "",
          check_interval: 300,
          timeout: 30,
          is_active: true,
        })
      }
    }
  }, [website, open, form])

  const mutation = useMutationAction({
    mutationFn: async (values: WebsiteFormValues) => {
      const payload = {
        url: values.url.trim(),
        name: values.name.trim(),
        description: values.description?.trim() || undefined,
        opd_id: values.opd_id ? parseInt(values.opd_id) : undefined,
        check_interval: values.check_interval,
        timeout: values.timeout,
        is_active: values.is_active,
      }
      if (isEditing && website) {
        return websiteApi.update(website.id, payload)
      }
      return websiteApi.create(payload)
    },
    successMessage: isEditing ? "Website berhasil diperbarui" : "Website berhasil ditambahkan",
    invalidateKeys: ["websites"],
    onSuccess: () => {
      onOpenChange(false)
      onSuccess()
    },
  })

  const onSubmit = form.handleSubmit((values) => mutation.mutate(values))

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg p-0">
        <DialogHeader className="px-6 pt-6 pb-0">
          <DialogTitle className="text-base">{isEditing ? "Edit Website" : "Tambah Website"}</DialogTitle>
          <DialogDescription className="text-[13px]">
            {isEditing ? "Perbarui informasi website monitoring." : "Tambahkan website baru untuk dimonitor."}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={onSubmit} className="px-6 pb-6">
          <div className="space-y-4 mt-4">
            <div className="space-y-1.5">
              <Label htmlFor="url" className="text-xs font-medium">URL *</Label>
              <Input
                id="url"
                placeholder="https://example.com"
                className="h-9 text-[13px]"
                {...form.register("url")}
                aria-invalid={!!form.formState.errors.url}
              />
              {form.formState.errors.url && (
                <p className="text-[11px] text-destructive">{form.formState.errors.url.message}</p>
              )}
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="name" className="text-xs font-medium">Nama *</Label>
              <Input
                id="name"
                placeholder="Nama website"
                className="h-9 text-[13px]"
                {...form.register("name")}
                aria-invalid={!!form.formState.errors.name}
              />
              {form.formState.errors.name && (
                <p className="text-[11px] text-destructive">{form.formState.errors.name.message}</p>
              )}
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="description" className="text-xs font-medium">Deskripsi</Label>
              <Input
                id="description"
                placeholder="Deskripsi singkat (opsional)"
                className="h-9 text-[13px]"
                {...form.register("description")}
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-xs font-medium">OPD</Label>
              <Select value={form.watch("opd_id")} onValueChange={(val) => form.setValue("opd_id", val)}>
                <SelectTrigger className="h-9 text-[13px]">
                  <SelectValue placeholder="Pilih OPD (opsional)" />
                </SelectTrigger>
                <SelectContent>
                  {opdList.map((opd) => (
                    <SelectItem key={opd.id} value={String(opd.id)} className="text-[13px]">
                      {opd.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label htmlFor="check_interval" className="text-xs font-medium">Interval Cek (detik)</Label>
                <Input
                  id="check_interval"
                  type="number"
                  min={30}
                  className="h-9 text-[13px]"
                  {...form.register("check_interval", { valueAsNumber: true })}
                  aria-invalid={!!form.formState.errors.check_interval}
                />
                {form.formState.errors.check_interval && (
                  <p className="text-[11px] text-destructive">{form.formState.errors.check_interval.message}</p>
                )}
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="timeout" className="text-xs font-medium">Timeout (detik)</Label>
                <Input
                  id="timeout"
                  type="number"
                  min={5}
                  className="h-9 text-[13px]"
                  {...form.register("timeout", { valueAsNumber: true })}
                  aria-invalid={!!form.formState.errors.timeout}
                />
                {form.formState.errors.timeout && (
                  <p className="text-[11px] text-destructive">{form.formState.errors.timeout.message}</p>
                )}
              </div>
            </div>

            <div className="flex items-center justify-between rounded-lg border border-border/50 px-4 py-3">
              <div className="space-y-0.5">
                <Label className="text-xs font-medium">Aktif</Label>
                <p className="text-[11px] text-muted-foreground">Aktifkan monitoring untuk website ini</p>
              </div>
              <Switch
                checked={form.watch("is_active")}
                onCheckedChange={(checked) => form.setValue("is_active", checked)}
              />
            </div>
          </div>

          <DialogFooter className="mt-6 gap-2">
            <Button
              type="button"
              variant="outline"
              className="h-8 text-xs"
              onClick={() => onOpenChange(false)}
              disabled={mutation.isPending}
            >
              Batal
            </Button>
            <Button type="submit" className="h-8 text-xs" disabled={mutation.isPending}>
              {mutation.isPending && <Loader2 className="size-3.5 animate-spin mr-1" />}
              {isEditing ? "Simpan" : "Tambah"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
