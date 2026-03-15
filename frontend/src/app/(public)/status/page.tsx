"use client"

import { useEffect, useState, useCallback } from "react"
import { Activity, CheckCircle2, XCircle, AlertTriangle, MinusCircle, RefreshCw } from "lucide-react"
import { statusApi } from "@/lib/api"
import { cn } from "@/lib/utils"

interface ServiceStatus {
  id: number
  name: string
  url: string
  status: string
  last_checked_at?: string
  uptime_percentage?: number
}

interface StatusOverview {
  overall_status: string
  total_services: number
  services_up: number
  services_down: number
  services_degraded: number
  last_updated?: string
}

function getStatusIcon(status: string) {
  switch (status) {
    case "up":
    case "operational":
      return <CheckCircle2 className="h-5 w-5 text-green-500" />
    case "down":
    case "major_outage":
      return <XCircle className="h-5 w-5 text-red-500" />
    case "degraded":
    case "partial_outage":
      return <AlertTriangle className="h-5 w-5 text-yellow-500" />
    default:
      return <MinusCircle className="h-5 w-5 text-gray-400" />
  }
}

function getOverallBanner(status: string) {
  switch (status) {
    case "operational":
    case "up":
      return {
        bg: "bg-green-500",
        text: "All Systems Operational",
        icon: <CheckCircle2 className="h-6 w-6" />,
      }
    case "major_outage":
    case "down":
      return {
        bg: "bg-red-500",
        text: "Major System Outage",
        icon: <XCircle className="h-6 w-6" />,
      }
    case "partial_outage":
    case "degraded":
      return {
        bg: "bg-yellow-500",
        text: "Partial System Outage",
        icon: <AlertTriangle className="h-6 w-6" />,
      }
    default:
      return {
        bg: "bg-gray-500",
        text: "Status Unknown",
        icon: <MinusCircle className="h-6 w-6" />,
      }
  }
}

function getStatusLabel(status: string) {
  switch (status) {
    case "up":
    case "operational":
      return "Operational"
    case "down":
    case "major_outage":
      return "Down"
    case "degraded":
    case "partial_outage":
      return "Degraded"
    default:
      return "Unknown"
  }
}

export default function StatusPage() {
  const [overview, setOverview] = useState<StatusOverview | null>(null)
  const [services, setServices] = useState<ServiceStatus[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date())

  const fetchData = useCallback(async () => {
    try {
      const [overviewRes, servicesRes] = await Promise.all([
        statusApi.overview(),
        statusApi.services(),
      ])
      setOverview(overviewRes.data ?? overviewRes)
      setServices(servicesRes.data ?? servicesRes)
      setError("")
      setLastRefresh(new Date())
    } catch {
      setError("Failed to load status data")
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
    const interval = setInterval(fetchData, 30_000)
    return () => clearInterval(interval)
  }, [fetchData])

  const banner = overview
    ? getOverallBanner(overview.overall_status)
    : { bg: "bg-gray-500", text: "Loading...", icon: <MinusCircle className="h-6 w-6" /> }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
          <p className="text-sm text-muted-foreground">Loading status...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      {/* Header */}
      <div className="mb-8 flex items-center gap-3">
        <Activity className="h-7 w-7 text-primary" />
        <h1 className="text-2xl font-bold">System Status</h1>
      </div>

      {/* Status banner */}
      <div
        className={cn(
          "mb-8 flex items-center gap-3 rounded-xl p-4 text-white",
          banner.bg
        )}
      >
        {banner.icon}
        <span className="text-lg font-semibold">{banner.text}</span>
      </div>

      {error && (
        <div className="mb-6 rounded-lg border border-destructive/20 bg-destructive/10 p-3 text-sm text-destructive">
          {error}
        </div>
      )}

      {/* Services */}
      <div className="space-y-2">
        <h2 className="mb-4 text-lg font-semibold">Services</h2>
        {services.length === 0 && !error && (
          <p className="text-sm text-muted-foreground">No services found.</p>
        )}
        {services.map((service) => (
          <div
            key={service.id}
            className="flex items-center justify-between rounded-lg border bg-card p-4"
          >
            <div className="flex items-center gap-3">
              {getStatusIcon(service.status)}
              <div>
                <p className="font-medium">{service.name}</p>
                {service.uptime_percentage !== undefined && (
                  <p className="text-xs text-muted-foreground">
                    {service.uptime_percentage.toFixed(2)}% uptime
                  </p>
                )}
              </div>
            </div>
            <span
              className={cn(
                "rounded-full px-2.5 py-0.5 text-xs font-medium",
                service.status === "up" || service.status === "operational"
                  ? "bg-green-500/10 text-green-600 dark:text-green-400"
                  : service.status === "down" || service.status === "major_outage"
                  ? "bg-red-500/10 text-red-600 dark:text-red-400"
                  : service.status === "degraded" || service.status === "partial_outage"
                  ? "bg-yellow-500/10 text-yellow-600 dark:text-yellow-400"
                  : "bg-gray-500/10 text-gray-600 dark:text-gray-400"
              )}
            >
              {getStatusLabel(service.status)}
            </span>
          </div>
        ))}
      </div>

      {/* Footer with refresh info */}
      <div className="mt-8 flex items-center justify-between text-xs text-muted-foreground">
        <span>
          Last updated: {lastRefresh.toLocaleTimeString("id-ID")}
        </span>
        <button
          onClick={fetchData}
          className="flex items-center gap-1 hover:text-foreground transition-colors"
        >
          <RefreshCw className="h-3 w-3" />
          Refresh
        </button>
      </div>
      <p className="mt-2 text-center text-xs text-muted-foreground">
        Auto-refreshes every 30 seconds
      </p>
    </div>
  )
}
