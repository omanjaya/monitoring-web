"use client"

import { useState } from "react"
import {
  ArrowUpCircle,
  RefreshCw,
  AlertCircle,
  Plus,
  Trash2,
  Shield,
  Clock,
  Loader2,
  ChevronDown,
  ChevronRight,
  Zap,
  Layers,
} from "lucide-react"
import { toast } from "sonner"
import { useQuery } from "@tanstack/react-query"

import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
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

import { escalationApi } from "@/lib/api"
import { formatDate } from "@/lib/utils"
import { useMutationAction } from "@/hooks/use-mutation-action"

interface EscalationRule {
  id: number
  policy_id: number
  level: number
  delay_minutes: number
  notify_channels: string
  notify_contacts: string
  created_at: string
}

interface EscalationPolicy {
  id: number
  name: string
  description?: string
  conditions: string
  actions: string
  is_active: boolean
  created_at: string
  rules?: EscalationRule[]
}

interface EscalationHistory {
  id: number
  policy_id: number
  policy_name?: string
  alert_id?: number
  alert_title?: string
  action_taken: string
  status: string
  created_at: string
}

const channelConfig: Record<string, { bg: string; text: string; label: string }> = {
  "notify:telegram": { bg: "bg-blue-500/10", text: "text-blue-700 dark:text-blue-400", label: "Telegram" },
  "notify:email": { bg: "bg-amber-500/10", text: "text-amber-700 dark:text-amber-400", label: "Email" },
  "notify:webhook": { bg: "bg-purple-500/10", text: "text-purple-700 dark:text-purple-400", label: "Webhook" },
  "notify:all": { bg: "bg-indigo-500/10", text: "text-indigo-700 dark:text-indigo-400", label: "All Channels" },
  telegram: { bg: "bg-blue-500/10", text: "text-blue-700 dark:text-blue-400", label: "Telegram" },
  email: { bg: "bg-amber-500/10", text: "text-amber-700 dark:text-amber-400", label: "Email" },
  webhook: { bg: "bg-purple-500/10", text: "text-purple-700 dark:text-purple-400", label: "Webhook" },
}

const conditionLabels: Record<string, string> = {
  "severity:critical": "Critical",
  "severity:warning": "Warning",
  "type:downtime": "Downtime",
  "type:ssl_expiry": "SSL Expiry",
  "type:content_change": "Content Change",
  "unresolved:30m": "> 30 min",
  "unresolved:1h": "> 1 hour",
}

function getChannel(action: string) {
  return channelConfig[action] || { bg: "bg-gray-500/10", text: "text-gray-600", label: action }
}

export default function EscalationPage() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [ruleDialogOpen, setRuleDialogOpen] = useState(false)
  const [triggerDialogOpen, setTriggerDialogOpen] = useState(false)
  const [activeTab, setActiveTab] = useState<"policies" | "history">("policies")
  const [expandedPolicy, setExpandedPolicy] = useState<number | null>(null)
  const [rulePolicyId, setRulePolicyId] = useState<number | null>(null)
  const [triggerPolicyId, setTriggerPolicyId] = useState<number | null>(null)

  const [form, setForm] = useState({
    name: "",
    description: "",
    conditions: "severity:critical",
    actions: "notify:telegram",
    is_active: true,
  })

  const [ruleForm, setRuleForm] = useState({
    level: 1,
    delay_minutes: 5,
    notify_channels: "telegram",
    notify_contacts: "",
  })

  const { data: policiesData, isLoading: loadingPolicies, error, refetch } = useQuery({
    queryKey: ["escalation-policies"],
    queryFn: async () => {
      const res = await escalationApi.policies()
      return (res.data as EscalationPolicy[]) || []
    },
  })

  const { data: historyData, isLoading: loadingHistory } = useQuery({
    queryKey: ["escalation-history"],
    queryFn: async () => {
      const res = await escalationApi.history()
      return (res.data as EscalationHistory[]) || []
    },
  })

  // Fetch policy detail (with rules) when expanded
  const { data: policyDetail } = useQuery({
    queryKey: ["escalation-policy-detail", expandedPolicy],
    queryFn: async () => {
      if (!expandedPolicy) return null
      const res = await escalationApi.getPolicy(expandedPolicy)
      return res.data as EscalationPolicy
    },
    enabled: !!expandedPolicy,
  })

  const policies = policiesData || []
  const history = historyData || []
  const loading = loadingPolicies || loadingHistory

  const createMutation = useMutationAction({
    mutationFn: (payload: { name: string; description?: string; conditions: string; actions: string; is_active: boolean }) =>
      escalationApi.createPolicy(payload),
    successMessage: "Policy berhasil dibuat",
    invalidateKeys: ["escalation-policies"],
    onSuccess: () => setDialogOpen(false),
  })

  const deleteMutation = useMutationAction({
    mutationFn: (id: number) => escalationApi.deletePolicy(id),
    successMessage: "Policy dihapus",
    invalidateKeys: ["escalation-policies"],
  })

  const createRuleMutation = useMutationAction({
    mutationFn: (payload: { policy_id: number; level: number; delay_minutes: number; notify_channels: string; notify_contacts: string }) =>
      escalationApi.createRule(payload),
    successMessage: "Rule berhasil ditambahkan",
    invalidateKeys: ["escalation-policies", "escalation-policy-detail"],
    onSuccess: () => setRuleDialogOpen(false),
  })

  const deleteRuleMutation = useMutationAction({
    mutationFn: (id: number) => escalationApi.deleteRule(id),
    successMessage: "Rule dihapus",
    invalidateKeys: ["escalation-policies", "escalation-policy-detail"],
  })

  const triggerMutation = useMutationAction({
    mutationFn: (payload: { policy_id: number; alert_id?: number }) =>
      escalationApi.trigger(payload),
    successMessage: "Escalation berhasil di-trigger",
    invalidateKeys: ["escalation-history"],
    onSuccess: () => setTriggerDialogOpen(false),
  })

  const openCreate = () => {
    setForm({
      name: "",
      description: "",
      conditions: "severity:critical",
      actions: "notify:telegram",
      is_active: true,
    })
    setDialogOpen(true)
  }

  const handleSubmit = () => {
    if (!form.name) {
      toast.error("Nama policy harus diisi")
      return
    }
    createMutation.mutate({
      name: form.name,
      description: form.description || undefined,
      conditions: form.conditions,
      actions: form.actions,
      is_active: form.is_active,
    })
  }

  const handleDelete = (id: number) => {
    deleteMutation.mutate(id)
  }

  const openAddRule = (policyId: number) => {
    setRulePolicyId(policyId)
    setRuleForm({
      level: 1,
      delay_minutes: 5,
      notify_channels: "telegram",
      notify_contacts: "",
    })
    setRuleDialogOpen(true)
  }

  const handleSubmitRule = () => {
    if (!rulePolicyId) return
    if (!ruleForm.notify_contacts) {
      toast.error("Kontak notifikasi harus diisi")
      return
    }
    createRuleMutation.mutate({
      policy_id: rulePolicyId,
      level: ruleForm.level,
      delay_minutes: ruleForm.delay_minutes,
      notify_channels: ruleForm.notify_channels,
      notify_contacts: ruleForm.notify_contacts,
    })
  }

  const openTrigger = (policyId: number) => {
    setTriggerPolicyId(policyId)
    setTriggerDialogOpen(true)
  }

  const handleTrigger = () => {
    if (!triggerPolicyId) return
    triggerMutation.mutate({ policy_id: triggerPolicyId })
  }

  const toggleExpand = (policyId: number) => {
    setExpandedPolicy(expandedPolicy === policyId ? null : policyId)
  }

  if (loading) return <EscalationSkeleton />

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-20">
        <div className="rounded-full bg-red-500/10 p-3">
          <AlertCircle className="h-5 w-5 text-red-500" />
        </div>
        <h2 className="text-sm font-medium">Gagal Memuat Data Escalation</h2>
        <p className="text-xs text-muted-foreground">{error instanceof Error ? error.message : "Gagal memuat data escalation"}</p>
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
          <h1 className="text-xl font-semibold tracking-tight">Escalation</h1>
          <p className="text-xs text-muted-foreground mt-0.5">
            Kelola kebijakan eskalasi alert
          </p>
        </div>
        <Button onClick={openCreate} className="h-8 text-xs">
          <Plus className="mr-1.5 h-3.5 w-3.5" />
          Buat Policy
        </Button>
      </div>

      {/* Tabs */}
      <div className="border-b border-border/50">
        <div className="flex gap-6">
          <button
            onClick={() => setActiveTab("policies")}
            className={`pb-2.5 text-[13px] font-medium border-b-2 transition-colors ${
              activeTab === "policies"
                ? "border-primary text-foreground"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            Policies
            {policies.length > 0 && (
              <span className="ml-1.5 inline-flex items-center justify-center rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-medium">
                {policies.length}
              </span>
            )}
          </button>
          <button
            onClick={() => setActiveTab("history")}
            className={`pb-2.5 text-[13px] font-medium border-b-2 transition-colors ${
              activeTab === "history"
                ? "border-primary text-foreground"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            History
            {history.length > 0 && (
              <span className="ml-1.5 inline-flex items-center justify-center rounded-full bg-muted px-1.5 py-0.5 text-[10px] font-medium">
                {history.length}
              </span>
            )}
          </button>
        </div>
      </div>

      {/* Policies Tab */}
      {activeTab === "policies" && (
        <>
          {policies.length > 0 ? (
            <div className="space-y-2">
              {policies.map((policy) => {
                const channel = getChannel(policy.actions)
                const isExpanded = expandedPolicy === policy.id
                const rules = (isExpanded && policyDetail?.rules) || policy.rules || []
                return (
                  <Card key={policy.id} className="border-border/50">
                    <CardContent className="px-5 py-4">
                      <div className="flex items-start justify-between gap-4">
                        <div className="flex-1 min-w-0 space-y-2">
                          <div className="flex items-center gap-2">
                            <button
                              onClick={() => toggleExpand(policy.id)}
                              className="shrink-0 text-muted-foreground hover:text-foreground transition-colors"
                            >
                              {isExpanded ? (
                                <ChevronDown className="h-3.5 w-3.5" />
                              ) : (
                                <ChevronRight className="h-3.5 w-3.5" />
                              )}
                            </button>
                            <Shield className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                            <span className="text-[13px] font-medium truncate">{policy.name}</span>
                            <span
                              className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ${
                                policy.is_active
                                  ? "bg-emerald-500/10 text-emerald-700 dark:text-emerald-400"
                                  : "bg-gray-500/10 text-gray-600 dark:text-gray-400"
                              }`}
                            >
                              {policy.is_active ? "Active" : "Inactive"}
                            </span>
                          </div>
                          {policy.description && (
                            <p className="text-xs text-muted-foreground line-clamp-1 pl-5">
                              {policy.description}
                            </p>
                          )}
                          <div className="flex items-center gap-2 pl-5">
                            <span className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-[11px] font-medium text-muted-foreground">
                              {conditionLabels[policy.conditions] || policy.conditions}
                            </span>
                            <span className="text-muted-foreground/40 text-[11px]">&rarr;</span>
                            <span
                              className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ${channel.bg} ${channel.text}`}
                            >
                              {channel.label}
                            </span>
                          </div>
                        </div>
                        <div className="flex items-center gap-1 shrink-0">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => openTrigger(policy.id)}
                            className="h-7 px-2 text-amber-600 hover:text-amber-700 hover:bg-amber-500/10"
                            title="Test Trigger"
                          >
                            <Zap className="h-3.5 w-3.5" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDelete(policy.id)}
                            className="h-7 px-2 text-red-500 hover:text-red-600 hover:bg-red-500/10"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </div>
                      </div>

                      {/* Expanded: Rules Section */}
                      {isExpanded && (
                        <div className="mt-4 pt-3 border-t border-border/50">
                          <div className="flex items-center justify-between mb-3">
                            <div className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
                              <Layers className="h-3.5 w-3.5" />
                              Escalation Rules
                            </div>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => openAddRule(policy.id)}
                              className="h-6 text-[11px] px-2"
                            >
                              <Plus className="mr-1 h-3 w-3" />
                              Tambah Rule
                            </Button>
                          </div>

                          {rules.length > 0 ? (
                            <div className="space-y-1.5">
                              {rules
                                .sort((a, b) => a.level - b.level)
                                .map((rule) => {
                                  const ruleChannel = getChannel(rule.notify_channels)
                                  return (
                                    <div
                                      key={rule.id}
                                      className="flex items-center justify-between gap-3 rounded-md bg-muted/50 px-3 py-2"
                                    >
                                      <div className="flex items-center gap-3 min-w-0 flex-1">
                                        <span className="inline-flex items-center justify-center rounded-full bg-primary/10 text-primary text-[11px] font-bold w-6 h-6 shrink-0">
                                          L{rule.level}
                                        </span>
                                        <div className="min-w-0">
                                          <div className="flex items-center gap-2 text-xs">
                                            <span className="text-muted-foreground">
                                              Delay: <span className="font-medium text-foreground">{rule.delay_minutes} menit</span>
                                            </span>
                                            <span className="text-muted-foreground/40">|</span>
                                            <span
                                              className={`inline-flex items-center rounded-full px-1.5 py-0.5 text-[10px] font-medium ${ruleChannel.bg} ${ruleChannel.text}`}
                                            >
                                              {ruleChannel.label}
                                            </span>
                                          </div>
                                          {rule.notify_contacts && (
                                            <p className="text-[11px] text-muted-foreground truncate mt-0.5">
                                              {rule.notify_contacts}
                                            </p>
                                          )}
                                        </div>
                                      </div>
                                      <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => deleteRuleMutation.mutate(rule.id)}
                                        className="h-6 w-6 p-0 text-red-500 hover:text-red-600 hover:bg-red-500/10 shrink-0"
                                      >
                                        <Trash2 className="h-3 w-3" />
                                      </Button>
                                    </div>
                                  )
                                })}
                            </div>
                          ) : (
                            <div className="text-center py-4">
                              <p className="text-xs text-muted-foreground">Belum ada rule untuk policy ini</p>
                            </div>
                          )}
                        </div>
                      )}
                    </CardContent>
                  </Card>
                )
              })}
            </div>
          ) : (
            <Card className="border-border/50">
              <CardContent className="py-16">
                <div className="flex flex-col items-center justify-center gap-2">
                  <div className="rounded-full bg-muted p-3">
                    <ArrowUpCircle className="h-5 w-5 text-muted-foreground" />
                  </div>
                  <p className="text-sm font-medium">Belum ada escalation policy</p>
                  <p className="text-xs text-muted-foreground">Buat policy baru untuk mengatur eskalasi alert</p>
                </div>
              </CardContent>
            </Card>
          )}
        </>
      )}

      {/* History Tab */}
      {activeTab === "history" && (
        <>
          {history.length > 0 ? (
            <div className="space-y-0">
              {history.map((item, idx) => {
                const isLast = idx === history.length - 1
                const statusColor =
                  item.status === "success"
                    ? "bg-emerald-500"
                    : item.status === "failed"
                    ? "bg-red-500"
                    : "bg-amber-500"
                return (
                  <div key={item.id} className="flex gap-3">
                    {/* Timeline line */}
                    <div className="flex flex-col items-center">
                      <div className={`w-2 h-2 rounded-full mt-1.5 shrink-0 ${statusColor}`} />
                      {!isLast && <div className="w-px flex-1 bg-border/50 my-1" />}
                    </div>
                    {/* Content */}
                    <div className={`flex-1 ${!isLast ? "pb-4" : "pb-0"}`}>
                      <div className="flex items-start justify-between gap-3">
                        <div className="space-y-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <span className="text-[13px] font-medium truncate">
                              {item.policy_name || `Policy #${item.policy_id}`}
                            </span>
                            <span
                              className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ${
                                item.status === "success"
                                  ? "bg-emerald-500/10 text-emerald-700 dark:text-emerald-400"
                                  : item.status === "failed"
                                  ? "bg-red-500/10 text-red-700 dark:text-red-400"
                                  : "bg-amber-500/10 text-amber-700 dark:text-amber-400"
                              }`}
                            >
                              {item.status}
                            </span>
                          </div>
                          <p className="text-xs text-muted-foreground">
                            {item.alert_title || (item.alert_id ? `Alert #${item.alert_id}` : "No alert")}
                            {" \u2022 "}
                            <span className="font-mono text-[11px]">{item.action_taken}</span>
                          </p>
                        </div>
                        <div className="flex items-center gap-1 text-xs text-muted-foreground shrink-0">
                          <Clock className="h-3 w-3" />
                          <span>{formatDate(item.created_at)}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          ) : (
            <Card className="border-border/50">
              <CardContent className="py-16">
                <div className="flex flex-col items-center justify-center gap-2">
                  <div className="rounded-full bg-muted p-3">
                    <Clock className="h-5 w-5 text-muted-foreground" />
                  </div>
                  <p className="text-sm font-medium">Belum ada riwayat eskalasi</p>
                  <p className="text-xs text-muted-foreground">Riwayat akan muncul saat eskalasi terjadi</p>
                </div>
              </CardContent>
            </Card>
          )}
        </>
      )}

      {/* Create Policy Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-[480px]">
          <DialogHeader>
            <DialogTitle className="text-sm font-medium">Buat Escalation Policy</DialogTitle>
            <DialogDescription className="text-xs">
              Buat kebijakan eskalasi baru untuk alert
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="name" className="text-xs">Nama Policy</Label>
              <Input
                id="name"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                placeholder="Nama policy"
                className="h-8 text-[13px]"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="description" className="text-xs">Deskripsi</Label>
              <Textarea
                id="description"
                value={form.description}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                placeholder="Deskripsi policy (opsional)"
                rows={2}
                className="text-[13px] resize-none"
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label htmlFor="conditions" className="text-xs">Kondisi</Label>
                <Select
                  value={form.conditions}
                  onValueChange={(value) => setForm((f) => ({ ...f, conditions: value }))}
                >
                  <SelectTrigger className="h-8 text-[13px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="severity:critical">Severity: Critical</SelectItem>
                    <SelectItem value="severity:warning">Severity: Warning</SelectItem>
                    <SelectItem value="type:downtime">Type: Downtime</SelectItem>
                    <SelectItem value="type:ssl_expiry">Type: SSL Expiry</SelectItem>
                    <SelectItem value="type:content_change">Type: Content Change</SelectItem>
                    <SelectItem value="unresolved:30m">{"Unresolved > 30 min"}</SelectItem>
                    <SelectItem value="unresolved:1h">{"Unresolved > 1 hour"}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="actions" className="text-xs">Notifikasi</Label>
                <Select
                  value={form.actions}
                  onValueChange={(value) => setForm((f) => ({ ...f, actions: value }))}
                >
                  <SelectTrigger className="h-8 text-[13px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="notify:telegram">Telegram</SelectItem>
                    <SelectItem value="notify:email">Email</SelectItem>
                    <SelectItem value="notify:webhook">Webhook</SelectItem>
                    <SelectItem value="notify:all">All Channels</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)} className="h-8 text-xs">
              Batal
            </Button>
            <Button onClick={handleSubmit} disabled={createMutation.isPending} className="h-8 text-xs">
              {createMutation.isPending && <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />}
              Buat Policy
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Add Rule Dialog */}
      <Dialog open={ruleDialogOpen} onOpenChange={setRuleDialogOpen}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader>
            <DialogTitle className="text-sm font-medium">Tambah Escalation Rule</DialogTitle>
            <DialogDescription className="text-xs">
              Tambahkan rule eskalasi baru ke policy
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label htmlFor="rule-level" className="text-xs">Level</Label>
                <Input
                  id="rule-level"
                  type="number"
                  min={1}
                  max={10}
                  value={ruleForm.level}
                  onChange={(e) => setRuleForm((f) => ({ ...f, level: parseInt(e.target.value) || 1 }))}
                  className="h-8 text-[13px]"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="rule-delay" className="text-xs">Delay (menit)</Label>
                <Input
                  id="rule-delay"
                  type="number"
                  min={1}
                  value={ruleForm.delay_minutes}
                  onChange={(e) => setRuleForm((f) => ({ ...f, delay_minutes: parseInt(e.target.value) || 5 }))}
                  className="h-8 text-[13px]"
                />
              </div>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="rule-channels" className="text-xs">Channel Notifikasi</Label>
              <Select
                value={ruleForm.notify_channels}
                onValueChange={(value) => setRuleForm((f) => ({ ...f, notify_channels: value }))}
              >
                <SelectTrigger className="h-8 text-[13px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="telegram">Telegram</SelectItem>
                  <SelectItem value="email">Email</SelectItem>
                  <SelectItem value="webhook">Webhook</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="rule-contacts" className="text-xs">Kontak Notifikasi</Label>
              <Input
                id="rule-contacts"
                value={ruleForm.notify_contacts}
                onChange={(e) => setRuleForm((f) => ({ ...f, notify_contacts: e.target.value }))}
                placeholder="Email, chat ID, atau URL webhook"
                className="h-8 text-[13px]"
              />
              <p className="text-[11px] text-muted-foreground">
                Pisahkan beberapa kontak dengan koma
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRuleDialogOpen(false)} className="h-8 text-xs">
              Batal
            </Button>
            <Button onClick={handleSubmitRule} disabled={createRuleMutation.isPending} className="h-8 text-xs">
              {createRuleMutation.isPending && <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />}
              Tambah Rule
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Test Trigger Dialog */}
      <Dialog open={triggerDialogOpen} onOpenChange={setTriggerDialogOpen}>
        <DialogContent className="sm:max-w-[380px]">
          <DialogHeader>
            <DialogTitle className="text-sm font-medium">Test Trigger Escalation</DialogTitle>
            <DialogDescription className="text-xs">
              Trigger eskalasi secara manual untuk menguji policy ini
            </DialogDescription>
          </DialogHeader>
          <div className="rounded-md bg-amber-500/10 border border-amber-500/20 px-3 py-2.5">
            <p className="text-xs text-amber-700 dark:text-amber-400">
              Ini akan menjalankan eskalasi secara manual. Notifikasi akan terkirim sesuai konfigurasi rule.
            </p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setTriggerDialogOpen(false)} className="h-8 text-xs">
              Batal
            </Button>
            <Button
              onClick={handleTrigger}
              disabled={triggerMutation.isPending}
              className="h-8 text-xs bg-amber-600 hover:bg-amber-700 text-white"
            >
              {triggerMutation.isPending && <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />}
              <Zap className="mr-1.5 h-3 w-3" />
              Trigger Sekarang
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function EscalationSkeleton() {
  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <Skeleton className="h-6 w-32 mb-1" />
          <Skeleton className="h-3 w-52" />
        </div>
        <Skeleton className="h-8 w-32" />
      </div>
      <div className="border-b border-border/50 pb-0">
        <div className="flex gap-6">
          <Skeleton className="h-4 w-16 mb-2.5" />
          <Skeleton className="h-4 w-16 mb-2.5" />
        </div>
      </div>
      <div className="space-y-2">
        {Array.from({ length: 4 }).map((_, i) => (
          <Card key={i} className="border-border/50">
            <CardContent className="px-5 py-4">
              <div className="flex items-start justify-between">
                <div className="space-y-2 flex-1">
                  <div className="flex items-center gap-2">
                    <Skeleton className="h-3.5 w-3.5 rounded" />
                    <Skeleton className="h-4 w-36" />
                    <Skeleton className="h-5 w-16 rounded-full" />
                  </div>
                  <div className="flex items-center gap-2 pl-5">
                    <Skeleton className="h-5 w-20 rounded-full" />
                    <Skeleton className="h-5 w-20 rounded-full" />
                  </div>
                </div>
                <Skeleton className="h-7 w-7" />
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
