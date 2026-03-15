"use client"

import { useEffect, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { settingsApi, authApi, adminApi, type SystemStatus, type DigestSettings, type AISettings } from "@/lib/api"
import { toast } from "sonner"
import {
  Bell,
  Bot,
  Key,
  Loader2,
  Mail,
  Monitor,
  Save,
  Send,
  Webhook,
  Server,
  Activity,
  Eye,
  EyeOff,
  Clock,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { useMutationAction } from "@/hooks/use-mutation-action"

export default function SettingsPage() {
  // Notifications state
  const [telegramEnabled, setTelegramEnabled] = useState(false)
  const [telegramBotToken, setTelegramBotToken] = useState("")
  const [telegramChatIds, setTelegramChatIds] = useState("")
  const [emailEnabled, setEmailEnabled] = useState(false)
  const [smtpHost, setSmtpHost] = useState("")
  const [smtpPort, setSmtpPort] = useState("")
  const [smtpUser, setSmtpUser] = useState("")
  const [smtpPassword, setSmtpPassword] = useState("")
  const [smtpFrom, setSmtpFrom] = useState("")
  const [emailRecipients, setEmailRecipients] = useState("")
  const [webhookEnabled, setWebhookEnabled] = useState(false)
  const [webhookUrl, setWebhookUrl] = useState("")
  const [webhookSecret, setWebhookSecret] = useState("")

  // Digest state
  const [digestEnabled, setDigestEnabled] = useState(false)
  const [digestInterval, setDigestInterval] = useState("15")
  const [quietHoursStart, setQuietHoursStart] = useState("")
  const [quietHoursEnd, setQuietHoursEnd] = useState("")

  // Monitoring state
  const [defaultInterval, setDefaultInterval] = useState("300")
  const [defaultTimeout, setDefaultTimeout] = useState("30")
  const [retryCount, setRetryCount] = useState("3")
  const [sslWarningDays, setSslWarningDays] = useState("30")
  const [responseTimeThreshold, setResponseTimeThreshold] = useState("5000")
  const [dnsInterval, setDnsInterval] = useState("720")

  // AI state
  const [aiEnabled, setAiEnabled] = useState(false)
  const [aiProvider, setAiProvider] = useState("groq")
  const [aiApiKey, setAiApiKey] = useState("")
  const [aiModel, setAiModel] = useState("")
  const [showAiKey, setShowAiKey] = useState(false)

  // Account state
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [showCurrentPw, setShowCurrentPw] = useState(false)
  const [showNewPw, setShowNewPw] = useState(false)

  // Diagnostic state
  const [testing, setTesting] = useState<string | null>(null)

  // Fetch notification settings
  const notifQuery = useQuery({
    queryKey: ["settings-notifications"],
    queryFn: () => settingsApi.notifications(),
  })

  // Fetch digest settings
  const digestQuery = useQuery({
    queryKey: ["settings-digest"],
    queryFn: () => settingsApi.digest(),
  })

  // Fetch monitoring settings
  const monitoringQuery = useQuery({
    queryKey: ["settings-monitoring"],
    queryFn: () => settingsApi.monitoring(),
  })

  // Fetch AI settings
  const aiQuery = useQuery({
    queryKey: ["settings-ai"],
    queryFn: () => settingsApi.ai(),
  })

  // Fetch system status
  const systemQuery = useQuery({
    queryKey: ["system-status"],
    queryFn: () => adminApi.status(),
  })

  const systemStatus = systemQuery.data?.data as SystemStatus | null

  // Populate notification form fields when data arrives
  useEffect(() => {
    const data = notifQuery.data?.data as Record<string, unknown> | null
    if (data) {
      const telegram = data.telegram as Record<string, unknown> | undefined
      if (telegram) {
        setTelegramEnabled(!!telegram.enabled)
        setTelegramBotToken((telegram.bot_token as string) || "")
        setTelegramChatIds(
          Array.isArray(telegram.chat_ids)
            ? (telegram.chat_ids as string[]).join(", ")
            : (telegram.chat_ids as string) || ""
        )
      }
      const email = data.email as Record<string, unknown> | undefined
      if (email) {
        setEmailEnabled(!!email.enabled)
        setSmtpHost((email.smtp_host as string) || "")
        setSmtpPort(String(email.smtp_port || ""))
        setSmtpUser((email.smtp_user as string) || "")
        setSmtpPassword((email.smtp_password as string) || "")
        setSmtpFrom((email.from_address as string) || "")
        setEmailRecipients(
          Array.isArray(email.recipients)
            ? (email.recipients as string[]).join(", ")
            : (email.recipients as string) || ""
        )
      }
      const webhook = data.webhook as Record<string, unknown> | undefined
      if (webhook) {
        setWebhookEnabled(!!webhook.enabled)
        setWebhookUrl((webhook.url as string) || "")
        setWebhookSecret((webhook.secret as string) || "")
      }
    }
  }, [notifQuery.data])

  // Populate digest form fields when data arrives
  useEffect(() => {
    const data = digestQuery.data?.data
    if (data) {
      setDigestEnabled(!!data.digest_enabled)
      setDigestInterval(String(data.digest_interval || 15))
      setQuietHoursStart(data.quiet_hours_start || "")
      setQuietHoursEnd(data.quiet_hours_end || "")
    }
  }, [digestQuery.data])

  // Populate monitoring form fields when data arrives
  useEffect(() => {
    const data = monitoringQuery.data?.data as Record<string, unknown> | null
    if (data) {
      setDefaultInterval(String(data.default_interval || "300"))
      setDefaultTimeout(String(data.default_timeout || "30"))
      setRetryCount(String(data.retry_count || "3"))
      setSslWarningDays(String(data.ssl_warning_days || "30"))
      setResponseTimeThreshold(String(data.response_time_threshold || "5000"))
      setDnsInterval(String(data.dns_interval ?? "720"))
    }
  }, [monitoringQuery.data])

  // Populate AI form fields when data arrives
  useEffect(() => {
    const data = aiQuery.data?.data as AISettings | null
    if (data) {
      setAiEnabled(data.enabled || false)
      setAiProvider(data.provider || "groq")
      setAiApiKey(data.api_key || "")
      setAiModel(data.model || "")
    }
  }, [aiQuery.data])

  // Mutations
  const saveTelegramMutation = useMutationAction({
    mutationFn: (data: { enabled: boolean; bot_token: string; chat_ids: string[] }) =>
      settingsApi.updateTelegram(data),
    successMessage: "Pengaturan Telegram berhasil disimpan",
    errorMessage: "Gagal menyimpan pengaturan Telegram",
    invalidateKeys: ["settings-notifications"],
  })

  const saveEmailMutation = useMutationAction({
    mutationFn: (data: {
      enabled: boolean
      smtp_host: string
      smtp_port: number
      smtp_user: string
      smtp_password: string
      from_address: string
      recipients: string[]
    }) => settingsApi.updateEmail(data),
    successMessage: "Pengaturan Email berhasil disimpan",
    errorMessage: "Gagal menyimpan pengaturan Email",
    invalidateKeys: ["settings-notifications"],
  })

  const saveWebhookMutation = useMutationAction({
    mutationFn: (data: { enabled: boolean; url: string; secret: string }) =>
      settingsApi.updateWebhook(data),
    successMessage: "Pengaturan Webhook berhasil disimpan",
    errorMessage: "Gagal menyimpan pengaturan Webhook",
    invalidateKeys: ["settings-notifications"],
  })

  const saveDigestMutation = useMutationAction({
    mutationFn: (data: Partial<DigestSettings>) =>
      settingsApi.updateDigest(data),
    successMessage: "Pengaturan Digest berhasil disimpan",
    errorMessage: "Gagal menyimpan pengaturan Digest",
    invalidateKeys: ["settings-digest", "settings-notifications"],
  })

  const saveMonitoringMutation = useMutationAction({
    mutationFn: (data: {
      default_interval: number
      default_timeout: number
      retry_count: number
      ssl_warning_days: number
      response_time_threshold: number
      dns_interval: number
    }) => settingsApi.updateMonitoring(data),
    successMessage: "Pengaturan Monitoring berhasil disimpan",
    errorMessage: "Gagal menyimpan pengaturan Monitoring",
    invalidateKeys: ["settings-monitoring"],
  })

  const saveAIMutation = useMutationAction({
    mutationFn: (data: Partial<AISettings>) => settingsApi.updateAI(data),
    successMessage: "Pengaturan AI berhasil disimpan",
    errorMessage: "Gagal menyimpan pengaturan AI",
    invalidateKeys: ["settings-ai"],
  })

  const changePasswordMutation = useMutationAction({
    mutationFn: ({ currentPassword, newPassword }: { currentPassword: string; newPassword: string }) =>
      authApi.changePassword(currentPassword, newPassword),
    successMessage: "Password berhasil diubah",
    errorMessage: "Gagal mengubah password",
    onSuccess: () => {
      setCurrentPassword("")
      setNewPassword("")
      setConfirmPassword("")
    },
  })

  const saveTelegram = () => {
    saveTelegramMutation.mutate({
      enabled: telegramEnabled,
      bot_token: telegramBotToken,
      chat_ids: telegramChatIds
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean),
    })
  }

  const saveEmail = () => {
    saveEmailMutation.mutate({
      enabled: emailEnabled,
      smtp_host: smtpHost,
      smtp_port: Number(smtpPort) || 587,
      smtp_user: smtpUser,
      smtp_password: smtpPassword,
      from_address: smtpFrom,
      recipients: emailRecipients
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean),
    })
  }

  const saveWebhook = () => {
    saveWebhookMutation.mutate({
      enabled: webhookEnabled,
      url: webhookUrl,
      secret: webhookSecret,
    })
  }

  const saveDigest = () => {
    saveDigestMutation.mutate({
      digest_enabled: digestEnabled,
      digest_interval: Math.max(5, Number(digestInterval) || 15),
      quiet_hours_start: quietHoursStart,
      quiet_hours_end: quietHoursEnd,
    })
  }

  const saveMonitoring = () => {
    saveMonitoringMutation.mutate({
      default_interval: Number(defaultInterval),
      default_timeout: Number(defaultTimeout),
      retry_count: Number(retryCount),
      ssl_warning_days: Number(sslWarningDays),
      response_time_threshold: Number(responseTimeThreshold),
      dns_interval: Number(dnsInterval),
    })
  }

  const handleChangePassword = () => {
    if (!currentPassword || !newPassword) {
      toast.error("Harap isi password saat ini dan password baru")
      return
    }
    if (newPassword !== confirmPassword) {
      toast.error("Password baru dan konfirmasi tidak cocok")
      return
    }
    if (newPassword.length < 8) {
      toast.error("Password baru minimal 8 karakter")
      return
    }
    changePasswordMutation.mutate({ currentPassword, newPassword })
  }

  const handleTest = async (type: string) => {
    try {
      setTesting(type)
      if (type === "telegram") await adminApi.testTelegram()
      if (type === "email") await adminApi.testEmail()
      if (type === "webhook") await adminApi.testWebhook()
      toast.success(`Test notifikasi ${type} berhasil dikirim`)
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : `Gagal mengirim test ${type}`)
    } finally {
      setTesting(null)
    }
  }

  return (
    <div className="space-y-5">
      {/* Header */}
      <div>
        <h1 className="text-xl font-semibold tracking-tight">Settings</h1>
        <p className="text-[13px] text-muted-foreground mt-0.5">Konfigurasi sistem monitoring</p>
      </div>

      <Tabs defaultValue="notifications" className="space-y-5">
        {/* Underline-style tab navigation */}
        <TabsList className="h-auto p-0 bg-transparent border-b border-border/50 rounded-none w-full justify-start gap-0">
          <TabsTrigger
            value="notifications"
            className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-4 pb-2.5 pt-1 text-[13px]"
          >
            <Bell className="size-3.5 mr-1.5" />
            Notifikasi
          </TabsTrigger>
          <TabsTrigger
            value="monitoring"
            className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-4 pb-2.5 pt-1 text-[13px]"
          >
            <Monitor className="size-3.5 mr-1.5" />
            Monitoring
          </TabsTrigger>
          <TabsTrigger
            value="ai"
            className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-4 pb-2.5 pt-1 text-[13px]"
          >
            <Bot className="size-3.5 mr-1.5" />
            AI Verification
          </TabsTrigger>
          <TabsTrigger
            value="system"
            className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-4 pb-2.5 pt-1 text-[13px]"
          >
            <Server className="size-3.5 mr-1.5" />
            System
          </TabsTrigger>
          <TabsTrigger
            value="account"
            className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-4 pb-2.5 pt-1 text-[13px]"
          >
            <Key className="size-3.5 mr-1.5" />
            Akun
          </TabsTrigger>
        </TabsList>

        {/* Notifications Tab */}
        <TabsContent value="notifications" className="space-y-4 mt-0">
          {notifQuery.isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="size-5 animate-spin text-muted-foreground" />
            </div>
          ) : (
            <>
              {/* Telegram */}
              <Card className="border-border/50">
                <CardContent className="px-5 py-4">
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex items-center gap-3">
                      <div className="size-8 rounded-lg bg-blue-500/10 flex items-center justify-center">
                        <Send className="size-4 text-blue-500" />
                      </div>
                      <div>
                        <h3 className="text-sm font-medium">Telegram</h3>
                        <p className="text-xs text-muted-foreground mt-0.5">Kirim notifikasi via Telegram Bot</p>
                      </div>
                    </div>
                    <Switch
                      checked={telegramEnabled}
                      onCheckedChange={setTelegramEnabled}
                    />
                  </div>
                  <div className="space-y-3">
                    <div className="space-y-1.5">
                      <Label htmlFor="tg-token" className="text-xs">Bot Token</Label>
                      <Input
                        id="tg-token"
                        className="h-9 text-[13px]"
                        type="password"
                        placeholder="123456:ABC-DEF..."
                        value={telegramBotToken}
                        onChange={(e) => setTelegramBotToken(e.target.value)}
                      />
                    </div>
                    <div className="space-y-1.5">
                      <Label htmlFor="tg-chats" className="text-xs">Chat IDs</Label>
                      <Textarea
                        id="tg-chats"
                        className="text-[13px] min-h-[60px]"
                        placeholder="Pisahkan dengan koma, misal: -100123456, -100789012"
                        value={telegramChatIds}
                        onChange={(e) => setTelegramChatIds(e.target.value)}
                        rows={2}
                      />
                      <p className="text-[11px] text-muted-foreground">Pisahkan beberapa Chat ID dengan tanda koma</p>
                    </div>
                    <div className="flex justify-end gap-2 pt-1">
                      <Button variant="outline" className="h-8 text-xs" onClick={() => handleTest("telegram")} disabled={testing === "telegram" || !telegramEnabled}>
                        {testing === "telegram" ? <Loader2 className="size-3.5 animate-spin mr-1.5" /> : <Activity className="size-3.5 mr-1.5" />}
                        Test
                      </Button>
                      <Button className="h-8 text-xs" onClick={saveTelegram} disabled={saveTelegramMutation.isPending}>
                        {saveTelegramMutation.isPending ? (
                          <Loader2 className="size-3.5 animate-spin mr-1.5" />
                        ) : (
                          <Save className="size-3.5 mr-1.5" />
                        )}
                        Simpan
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Email */}
              <Card className="border-border/50">
                <CardContent className="px-5 py-4">
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex items-center gap-3">
                      <div className="size-8 rounded-lg bg-emerald-500/10 flex items-center justify-center">
                        <Mail className="size-4 text-emerald-500" />
                      </div>
                      <div>
                        <h3 className="text-sm font-medium">Email (SMTP)</h3>
                        <p className="text-xs text-muted-foreground mt-0.5">Kirim notifikasi via Email</p>
                      </div>
                    </div>
                    <Switch
                      checked={emailEnabled}
                      onCheckedChange={setEmailEnabled}
                    />
                  </div>
                  <div className="space-y-3">
                    <div className="grid gap-3 md:grid-cols-2">
                      <div className="space-y-1.5">
                        <Label htmlFor="smtp-host" className="text-xs">SMTP Host</Label>
                        <Input
                          id="smtp-host"
                          className="h-9 text-[13px]"
                          placeholder="smtp.gmail.com"
                          value={smtpHost}
                          onChange={(e) => setSmtpHost(e.target.value)}
                        />
                      </div>
                      <div className="space-y-1.5">
                        <Label htmlFor="smtp-port" className="text-xs">SMTP Port</Label>
                        <Input
                          id="smtp-port"
                          className="h-9 text-[13px]"
                          placeholder="587"
                          value={smtpPort}
                          onChange={(e) => setSmtpPort(e.target.value)}
                        />
                      </div>
                      <div className="space-y-1.5">
                        <Label htmlFor="smtp-user" className="text-xs">SMTP Username</Label>
                        <Input
                          id="smtp-user"
                          className="h-9 text-[13px]"
                          placeholder="user@gmail.com"
                          value={smtpUser}
                          onChange={(e) => setSmtpUser(e.target.value)}
                        />
                      </div>
                      <div className="space-y-1.5">
                        <Label htmlFor="smtp-pass" className="text-xs">SMTP Password</Label>
                        <Input
                          id="smtp-pass"
                          className="h-9 text-[13px]"
                          type="password"
                          placeholder="App password"
                          value={smtpPassword}
                          onChange={(e) => setSmtpPassword(e.target.value)}
                        />
                      </div>
                    </div>
                    <div className="space-y-1.5">
                      <Label htmlFor="smtp-from" className="text-xs">From Address</Label>
                      <Input
                        id="smtp-from"
                        className="h-9 text-[13px]"
                        type="email"
                        placeholder="noreply@example.com"
                        value={smtpFrom}
                        onChange={(e) => setSmtpFrom(e.target.value)}
                      />
                      <p className="text-[11px] text-muted-foreground">Alamat email pengirim yang ditampilkan ke penerima</p>
                    </div>
                    <div className="space-y-1.5">
                      <Label htmlFor="email-recipients" className="text-xs">Recipients</Label>
                      <Textarea
                        id="email-recipients"
                        className="text-[13px] min-h-[60px]"
                        placeholder="Pisahkan dengan koma, misal: admin@example.com, ops@example.com"
                        value={emailRecipients}
                        onChange={(e) => setEmailRecipients(e.target.value)}
                        rows={2}
                      />
                    </div>
                    <div className="flex justify-end gap-2 pt-1">
                      <Button variant="outline" className="h-8 text-xs" onClick={() => handleTest("email")} disabled={testing === "email" || !emailEnabled}>
                        {testing === "email" ? <Loader2 className="size-3.5 animate-spin mr-1.5" /> : <Activity className="size-3.5 mr-1.5" />}
                        Test
                      </Button>
                      <Button className="h-8 text-xs" onClick={saveEmail} disabled={saveEmailMutation.isPending}>
                        {saveEmailMutation.isPending ? (
                          <Loader2 className="size-3.5 animate-spin mr-1.5" />
                        ) : (
                          <Save className="size-3.5 mr-1.5" />
                        )}
                        Simpan
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Webhook */}
              <Card className="border-border/50">
                <CardContent className="px-5 py-4">
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex items-center gap-3">
                      <div className="size-8 rounded-lg bg-orange-500/10 flex items-center justify-center">
                        <Webhook className="size-4 text-orange-500" />
                      </div>
                      <div>
                        <h3 className="text-sm font-medium">Webhook</h3>
                        <p className="text-xs text-muted-foreground mt-0.5">Kirim notifikasi ke webhook endpoint</p>
                      </div>
                    </div>
                    <Switch
                      checked={webhookEnabled}
                      onCheckedChange={setWebhookEnabled}
                    />
                  </div>
                  <div className="space-y-3">
                    <div className="space-y-1.5">
                      <Label htmlFor="webhook-url" className="text-xs">Webhook URL</Label>
                      <Input
                        id="webhook-url"
                        className="h-9 text-[13px]"
                        placeholder="https://example.com/webhook"
                        value={webhookUrl}
                        onChange={(e) => setWebhookUrl(e.target.value)}
                      />
                    </div>
                    <div className="space-y-1.5">
                      <Label htmlFor="webhook-secret" className="text-xs">Secret</Label>
                      <Input
                        id="webhook-secret"
                        className="h-9 text-[13px]"
                        type="password"
                        placeholder="Webhook signing secret"
                        value={webhookSecret}
                        onChange={(e) => setWebhookSecret(e.target.value)}
                      />
                      <p className="text-[11px] text-muted-foreground">Opsional. Digunakan untuk memverifikasi keaslian webhook</p>
                    </div>
                    <div className="flex justify-end gap-2 pt-1">
                      <Button variant="outline" className="h-8 text-xs" onClick={() => handleTest("webhook")} disabled={testing === "webhook" || !webhookEnabled}>
                        {testing === "webhook" ? <Loader2 className="size-3.5 animate-spin mr-1.5" /> : <Activity className="size-3.5 mr-1.5" />}
                        Test
                      </Button>
                      <Button className="h-8 text-xs" onClick={saveWebhook} disabled={saveWebhookMutation.isPending}>
                        {saveWebhookMutation.isPending ? (
                          <Loader2 className="size-3.5 animate-spin mr-1.5" />
                        ) : (
                          <Save className="size-3.5 mr-1.5" />
                        )}
                        Simpan
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Digest / Batching */}
              <Card className="border-border/50">
                <CardContent className="px-5 py-4">
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex items-center gap-3">
                      <div className="size-8 rounded-lg bg-violet-500/10 flex items-center justify-center">
                        <Clock className="size-4 text-violet-500" />
                      </div>
                      <div>
                        <h3 className="text-sm font-medium">Notification Digest</h3>
                        <p className="text-xs text-muted-foreground mt-0.5">
                          Kumpulkan beberapa alert menjadi satu notifikasi ringkasan
                        </p>
                      </div>
                    </div>
                    <Switch
                      checked={digestEnabled}
                      onCheckedChange={setDigestEnabled}
                    />
                  </div>
                  {digestEnabled && (
                    <div className="space-y-3 pt-3 border-t border-border/30">
                      <div className="space-y-1.5">
                        <Label htmlFor="digest-interval" className="text-xs">Interval Digest (menit)</Label>
                        <Input
                          id="digest-interval"
                          className="h-9 text-[13px] max-w-[200px]"
                          type="number"
                          min={5}
                          max={120}
                          placeholder="15"
                          value={digestInterval}
                          onChange={(e) => setDigestInterval(e.target.value)}
                        />
                        <p className="text-[11px] text-muted-foreground">
                          Alert dikumpulkan dan dikirim setiap X menit (minimal 5 menit)
                        </p>
                      </div>
                      <div className="grid gap-3 md:grid-cols-2">
                        <div className="space-y-1.5">
                          <Label htmlFor="quiet-start" className="text-xs">Quiet Hours Mulai</Label>
                          <Input
                            id="quiet-start"
                            className="h-9 text-[13px]"
                            type="time"
                            value={quietHoursStart}
                            onChange={(e) => setQuietHoursStart(e.target.value)}
                          />
                        </div>
                        <div className="space-y-1.5">
                          <Label htmlFor="quiet-end" className="text-xs">Quiet Hours Selesai</Label>
                          <Input
                            id="quiet-end"
                            className="h-9 text-[13px]"
                            type="time"
                            value={quietHoursEnd}
                            onChange={(e) => setQuietHoursEnd(e.target.value)}
                          />
                        </div>
                      </div>
                      <p className="text-[11px] text-muted-foreground">
                        Selama quiet hours, notifikasi non-critical ditunda hingga jam aktif.
                        Alert critical selalu dikirim langsung.
                      </p>
                    </div>
                  )}
                  <div className="flex justify-end gap-2 pt-3">
                    <Button className="h-8 text-xs" onClick={saveDigest} disabled={saveDigestMutation.isPending}>
                      {saveDigestMutation.isPending ? (
                        <Loader2 className="size-3.5 animate-spin mr-1.5" />
                      ) : (
                        <Save className="size-3.5 mr-1.5" />
                      )}
                      Simpan
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </>
          )}
        </TabsContent>

        {/* Monitoring Tab */}
        <TabsContent value="monitoring" className="space-y-4 mt-0">
          {monitoringQuery.isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="size-5 animate-spin text-muted-foreground" />
            </div>
          ) : (
            <Card className="border-border/50">
              <CardContent className="px-5 py-4">
                <div className="mb-4">
                  <h3 className="text-sm font-medium">Konfigurasi Monitoring</h3>
                  <p className="text-xs text-muted-foreground mt-0.5">Pengaturan interval, timeout, dan threshold monitoring</p>
                </div>
                <div className="space-y-4">
                  <div className="grid gap-4 md:grid-cols-2">
                    <div className="space-y-1.5">
                      <Label htmlFor="mon-interval" className="text-xs">Default Interval (detik)</Label>
                      <Input
                        id="mon-interval"
                        className="h-9 text-[13px]"
                        type="number"
                        placeholder="300"
                        value={defaultInterval}
                        onChange={(e) => setDefaultInterval(e.target.value)}
                      />
                      <p className="text-[11px] text-muted-foreground">
                        Interval pengecekan default untuk website baru
                      </p>
                    </div>
                    <div className="space-y-1.5">
                      <Label htmlFor="mon-timeout" className="text-xs">Default Timeout (detik)</Label>
                      <Input
                        id="mon-timeout"
                        className="h-9 text-[13px]"
                        type="number"
                        placeholder="30"
                        value={defaultTimeout}
                        onChange={(e) => setDefaultTimeout(e.target.value)}
                      />
                      <p className="text-[11px] text-muted-foreground">
                        Timeout request untuk setiap pengecekan
                      </p>
                    </div>
                    <div className="space-y-1.5">
                      <Label htmlFor="mon-retry" className="text-xs">Retry Count</Label>
                      <Input
                        id="mon-retry"
                        className="h-9 text-[13px]"
                        type="number"
                        placeholder="3"
                        value={retryCount}
                        onChange={(e) => setRetryCount(e.target.value)}
                      />
                      <p className="text-[11px] text-muted-foreground">
                        Jumlah retry sebelum dianggap down
                      </p>
                    </div>
                    <div className="space-y-1.5">
                      <Label htmlFor="mon-ssl" className="text-xs">SSL Warning Days</Label>
                      <Input
                        id="mon-ssl"
                        className="h-9 text-[13px]"
                        type="number"
                        placeholder="30"
                        value={sslWarningDays}
                        onChange={(e) => setSslWarningDays(e.target.value)}
                      />
                      <p className="text-[11px] text-muted-foreground">
                        Peringatan SSL expiry dalam hari
                      </p>
                    </div>
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="mon-response" className="text-xs">Response Time Threshold (ms)</Label>
                    <Input
                      id="mon-response"
                      className="h-9 text-[13px] max-w-md"
                      type="number"
                      placeholder="5000"
                      value={responseTimeThreshold}
                      onChange={(e) => setResponseTimeThreshold(e.target.value)}
                    />
                    <p className="text-[11px] text-muted-foreground">
                      Threshold response time sebelum dianggap degraded (milliseconds)
                    </p>
                  </div>

                  {/* DNS Scan Settings */}
                  <div className="pt-3 border-t border-border/30">
                    <h4 className="text-xs font-medium mb-3">DNS Monitoring</h4>
                    <div className="space-y-1.5 max-w-md">
                      <Label htmlFor="mon-dns" className="text-xs">DNS Scan Interval (menit)</Label>
                      <Input
                        id="mon-dns"
                        className="h-9 text-[13px]"
                        type="number"
                        min={0}
                        placeholder="720"
                        value={dnsInterval}
                        onChange={(e) => setDnsInterval(e.target.value)}
                      />
                      <p className="text-[11px] text-muted-foreground">
                        Interval scan DNS (SPF, DMARC, subdomain). Set <strong>0</strong> untuk menonaktifkan.
                        Minimal 60 menit jika aktif. Default: 720 menit (12 jam).
                      </p>
                      {Number(dnsInterval) === 0 && (
                        <p className="text-[11px] text-amber-600 dark:text-amber-400 font-medium">
                          DNS scan dinonaktifkan
                        </p>
                      )}
                    </div>
                  </div>

                  <div className="flex justify-end pt-2 border-t border-border/50">
                    <Button className="h-8 text-xs" onClick={saveMonitoring} disabled={saveMonitoringMutation.isPending}>
                      {saveMonitoringMutation.isPending ? (
                        <Loader2 className="size-3.5 animate-spin mr-1.5" />
                      ) : (
                        <Save className="size-3.5 mr-1.5" />
                      )}
                      Simpan Monitoring
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* AI Verification Tab */}
        <TabsContent value="ai" className="space-y-4 mt-0">
          {aiQuery.isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="size-5 animate-spin text-muted-foreground" />
            </div>
          ) : (
            <Card className="border-border/50">
              <CardContent className="px-5 py-4">
                <div className="flex items-center justify-between mb-5">
                  <div className="flex items-center gap-3">
                    <div className="size-9 rounded-lg bg-[#8b5cf6]/10 flex items-center justify-center">
                      <Bot className="size-4 text-[#8b5cf6]" />
                    </div>
                    <div>
                      <h3 className="text-[13px] font-medium">AI False Positive Verification</h3>
                      <p className="text-[11px] text-muted-foreground">Gunakan AI untuk memverifikasi deteksi dork dan mengurangi false positive</p>
                    </div>
                  </div>
                  <Switch checked={aiEnabled} onCheckedChange={setAiEnabled} />
                </div>

                {aiEnabled && (
                  <div className="space-y-4">
                    <div className="space-y-1.5">
                      <Label className="text-[12px]">Provider</Label>
                      <Select value={aiProvider} onValueChange={(v) => { setAiProvider(v); setAiModel(""); }}>
                        <SelectTrigger className="h-8 text-[13px]">
                          <SelectValue placeholder="Pilih provider" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="groq">Groq (Gratis - Tercepat)</SelectItem>
                          <SelectItem value="mistral">Mistral (Gratis)</SelectItem>
                          <SelectItem value="anthropic">Anthropic Claude (Berbayar)</SelectItem>
                        </SelectContent>
                      </Select>
                      <p className="text-[11px] text-muted-foreground">
                        {aiProvider === "groq" && "Groq menyediakan inference super cepat dengan free tier 14.400 request/hari"}
                        {aiProvider === "mistral" && "Mistral menyediakan free tier untuk model kecil"}
                        {aiProvider === "anthropic" && "Anthropic Claude - model paling akurat, memerlukan pembayaran"}
                      </p>
                    </div>

                    <div className="space-y-1.5">
                      <Label className="text-[12px]">API Key</Label>
                      <div className="relative">
                        <Input
                          type={showAiKey ? "text" : "password"}
                          value={aiApiKey}
                          onChange={(e) => setAiApiKey(e.target.value)}
                          placeholder={
                            aiProvider === "groq" ? "gsk_xxxxxxxxxxxx" :
                            aiProvider === "mistral" ? "xxxxxxxxxxxxxxxx" :
                            "sk-ant-api03-xxxxxxxxxxxx"
                          }
                          className="h-8 text-[13px] pr-9"
                        />
                        <button
                          type="button"
                          onClick={() => setShowAiKey(!showAiKey)}
                          className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                        >
                          {showAiKey ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
                        </button>
                      </div>
                      <p className="text-[11px] text-muted-foreground">
                        {aiProvider === "groq" && (
                          <>Dapatkan API key gratis di <a href="https://console.groq.com/keys" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">console.groq.com/keys</a></>
                        )}
                        {aiProvider === "mistral" && (
                          <>Dapatkan API key di <a href="https://console.mistral.ai/api-keys" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">console.mistral.ai/api-keys</a></>
                        )}
                        {aiProvider === "anthropic" && (
                          <>Dapatkan API key di <a href="https://console.anthropic.com/settings/keys" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">console.anthropic.com</a></>
                        )}
                      </p>
                    </div>

                    <div className="space-y-1.5">
                      <Label className="text-[12px]">Model</Label>
                      <Select value={aiModel || "_default"} onValueChange={(v) => setAiModel(v === "_default" ? "" : v)}>
                        <SelectTrigger className="h-8 text-[13px]">
                          <SelectValue placeholder="Pilih model" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="_default">Default</SelectItem>
                          {aiProvider === "groq" && (
                            <>
                              <SelectItem value="llama-3.3-70b-versatile">Llama 3.3 70B Versatile</SelectItem>
                              <SelectItem value="llama-3.1-8b-instant">Llama 3.1 8B Instant</SelectItem>
                              <SelectItem value="meta-llama/llama-4-scout-17b-16e-instruct">Llama 4 Scout 17B</SelectItem>
                              <SelectItem value="qwen/qwen3-32b">Qwen 3 32B</SelectItem>
                            </>
                          )}
                          {aiProvider === "mistral" && (
                            <>
                              <SelectItem value="mistral-small-latest">Mistral Small</SelectItem>
                              <SelectItem value="mistral-medium-latest">Mistral Medium</SelectItem>
                              <SelectItem value="mistral-large-latest">Mistral Large</SelectItem>
                              <SelectItem value="open-mistral-nemo">Mistral Nemo (Open)</SelectItem>
                            </>
                          )}
                          {aiProvider === "anthropic" && (
                            <>
                              <SelectItem value="claude-haiku-4-5-20251001">Claude Haiku 4.5</SelectItem>
                              <SelectItem value="claude-sonnet-4-6">Claude Sonnet 4.6</SelectItem>
                              <SelectItem value="claude-opus-4-6">Claude Opus 4.6</SelectItem>
                            </>
                          )}
                        </SelectContent>
                      </Select>
                      <p className="text-[11px] text-muted-foreground">Pilih model atau gunakan default untuk provider ini</p>
                    </div>

                    <div className="rounded-lg bg-muted/50 p-3 text-[11px] text-muted-foreground space-y-1">
                      <p className="font-medium text-foreground text-[12px]">Cara kerja:</p>
                      <p>1. Scanner mendeteksi konten mencurigakan menggunakan pattern matching</p>
                      <p>2. AI menganalisis konteks deteksi untuk memverifikasi apakah ancaman nyata atau false positive</p>
                      <p>3. Deteksi yang dikonfirmasi false positive oleh AI akan otomatis dibuang</p>
                      <p className="mt-2">Contoh: kata &quot;war&quot; dalam CSS gradient &quot;warm-spectrum&quot; akan dikenali sebagai false positive oleh AI</p>
                    </div>
                  </div>
                )}

                <div className="flex justify-end mt-4">
                  <Button
                    size="sm"
                    className="h-8 text-xs"
                    onClick={() => saveAIMutation.mutate({
                      enabled: aiEnabled,
                      provider: aiProvider,
                      api_key: aiApiKey,
                      model: aiModel,
                    })}
                    disabled={saveAIMutation.isPending}
                  >
                    {saveAIMutation.isPending ? (
                      <Loader2 className="size-3.5 animate-spin mr-1.5" />
                    ) : (
                      <Save className="size-3.5 mr-1.5" />
                    )}
                    Simpan
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* System Tab */}
        <TabsContent value="system" className="space-y-4 mt-0">
          <Card className="border-border/50">
            <CardContent className="px-5 py-4">
              <div className="mb-5">
                <h3 className="text-sm font-medium">Status Sistem</h3>
                <p className="text-xs text-muted-foreground mt-0.5">Informasi status komponen inti sistem monitoring</p>
              </div>
              {systemQuery.isLoading ? (
                <div className="flex justify-center py-12">
                  <Loader2 className="size-5 animate-spin text-muted-foreground" />
                </div>
              ) : systemStatus ? (
                <div className="space-y-5">
                  {/* Status indicators */}
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                    <div className="rounded-lg border border-border/50 p-4">
                      <div className="flex items-center gap-3">
                        <div className={`size-8 rounded-lg flex items-center justify-center ${systemStatus.status === 'running' ? 'bg-emerald-500/10' : 'bg-red-500/10'}`}>
                          <Activity className={`size-4 ${systemStatus.status === 'running' ? 'text-emerald-500' : 'text-red-500'}`} />
                        </div>
                        <div>
                          <p className="text-xs text-muted-foreground">Server</p>
                          <div className="flex items-center gap-1.5 mt-0.5">
                            <div className={`size-1.5 rounded-full ${systemStatus.status === 'running' ? 'bg-emerald-500' : 'bg-red-500'}`} />
                            <p className="text-sm font-medium capitalize">{systemStatus.status}</p>
                          </div>
                          <p className="text-[11px] text-muted-foreground mt-0.5">v{systemStatus.version} &middot; {systemStatus.total_websites} websites</p>
                        </div>
                      </div>
                    </div>

                    <div className="rounded-lg border border-border/50 p-4">
                      <div className="flex items-center gap-3">
                        <div className={`size-8 rounded-lg flex items-center justify-center ${systemStatus.monitor_running ? 'bg-emerald-500/10' : 'bg-red-500/10'}`}>
                          <Monitor className={`size-4 ${systemStatus.monitor_running ? 'text-emerald-500' : 'text-red-500'}`} />
                        </div>
                        <div>
                          <p className="text-xs text-muted-foreground">Monitor</p>
                          <div className="flex items-center gap-1.5 mt-0.5">
                            <div className={`size-1.5 rounded-full ${systemStatus.monitor_running ? 'bg-emerald-500' : 'bg-red-500'}`} />
                            <p className="text-sm font-medium">{systemStatus.monitor_running ? "Berjalan" : "Berhenti"}</p>
                          </div>
                        </div>
                      </div>
                    </div>

                    <div className="rounded-lg border border-border/50 p-4">
                      <div className="flex items-center gap-3">
                        <div className={`size-8 rounded-lg flex items-center justify-center ${systemStatus.scheduler === 'active' ? 'bg-emerald-500/10' : 'bg-amber-500/10'}`}>
                          <Server className={`size-4 ${systemStatus.scheduler === 'active' ? 'text-emerald-500' : 'text-amber-500'}`} />
                        </div>
                        <div>
                          <p className="text-xs text-muted-foreground">Scheduler</p>
                          <div className="flex items-center gap-1.5 mt-0.5">
                            <div className={`size-1.5 rounded-full ${systemStatus.scheduler === 'active' ? 'bg-emerald-500' : 'bg-amber-500'}`} />
                            <p className="text-sm font-medium">
                              {systemStatus.scheduler === 'active' ? "Berjalan" : "Berhenti"}
                            </p>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Active jobs */}
                  {systemStatus.active_jobs && systemStatus.active_jobs.length > 0 && (
                    <div>
                      <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">Scheduler Jobs</h4>
                      <div className="space-y-1">
                        {systemStatus.active_jobs.map((job: string) => (
                          <div key={job} className="flex items-center gap-2 rounded-md border border-border/50 px-3 py-2">
                            <div className="size-1.5 rounded-full bg-emerald-500" />
                            <span className="text-[13px] font-mono text-muted-foreground">{job}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  <div className="flex justify-end pt-3 border-t border-border/50">
                    <Button variant="outline" className="h-8 text-xs" onClick={() => systemQuery.refetch()} disabled={systemQuery.isFetching}>
                      {systemQuery.isFetching ? (
                        <Loader2 className="size-3.5 animate-spin mr-1.5" />
                      ) : (
                        <Activity className="size-3.5 mr-1.5" />
                      )}
                      Refresh
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="flex flex-col items-center gap-2 py-12">
                  <div className="size-10 rounded-full bg-muted flex items-center justify-center">
                    <Server className="size-5 text-muted-foreground" />
                  </div>
                  <p className="text-sm font-medium">Gagal memuat status sistem</p>
                  <p className="text-xs text-muted-foreground">API kemungkinan tidak merespon</p>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Account Tab */}
        <TabsContent value="account" className="space-y-4 mt-0">
          <Card className="border-border/50">
            <CardContent className="px-5 py-4">
              <div className="mb-4">
                <h3 className="text-sm font-medium">Ubah Password</h3>
                <p className="text-xs text-muted-foreground mt-0.5">Ubah password akun Anda untuk keamanan</p>
              </div>
              <div className="space-y-3 max-w-sm">
                <div className="space-y-1.5">
                  <Label htmlFor="current-pw" className="text-xs">Password Saat Ini</Label>
                  <div className="relative">
                    <Input
                      id="current-pw"
                      className="h-9 text-[13px] pr-9"
                      type={showCurrentPw ? "text" : "password"}
                      placeholder="Password saat ini"
                      value={currentPassword}
                      onChange={(e) => setCurrentPassword(e.target.value)}
                    />
                    <button
                      type="button"
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                      onClick={() => setShowCurrentPw(!showCurrentPw)}
                    >
                      {showCurrentPw ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
                    </button>
                  </div>
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="new-pw" className="text-xs">Password Baru</Label>
                  <div className="relative">
                    <Input
                      id="new-pw"
                      className="h-9 text-[13px] pr-9"
                      type={showNewPw ? "text" : "password"}
                      placeholder="Minimal 8 karakter"
                      value={newPassword}
                      onChange={(e) => setNewPassword(e.target.value)}
                    />
                    <button
                      type="button"
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                      onClick={() => setShowNewPw(!showNewPw)}
                    >
                      {showNewPw ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
                    </button>
                  </div>
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="confirm-pw" className="text-xs">Konfirmasi Password Baru</Label>
                  <Input
                    id="confirm-pw"
                    className="h-9 text-[13px]"
                    type="password"
                    placeholder="Ulangi password baru"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                  />
                  {confirmPassword && newPassword !== confirmPassword && (
                    <p className="text-[11px] text-destructive">Password tidak cocok</p>
                  )}
                </div>
                <div className="pt-2">
                  <Button className="h-8 text-xs" onClick={handleChangePassword} disabled={changePasswordMutation.isPending}>
                    {changePasswordMutation.isPending ? (
                      <Loader2 className="size-3.5 animate-spin mr-1.5" />
                    ) : (
                      <Key className="size-3.5 mr-1.5" />
                    )}
                    Ubah Password
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
