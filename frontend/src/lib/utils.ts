import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDate(date: string | Date | null | undefined): string {
  if (!date) return "-"
  const d = new Date(date)
  if (isNaN(d.getTime())) return "-"
  return d.toLocaleDateString("id-ID", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

export function formatRelativeTime(date: string | Date | null | undefined): string {
  if (!date) return "-"
  const d = new Date(date)
  if (isNaN(d.getTime())) return "-"
  const now = new Date()
  const diff = Math.floor((now.getTime() - d.getTime()) / 1000)
  if (diff < 60) return `${diff} detik lalu`
  if (diff < 3600) return `${Math.floor(diff / 60)} menit lalu`
  if (diff < 86400) return `${Math.floor(diff / 3600)} jam lalu`
  return `${Math.floor(diff / 86400)} hari lalu`
}

export function getStatusColor(status: string): string {
  switch (status) {
    case "up":
    case "operational":
      return "text-green-500"
    case "down":
    case "major_outage":
      return "text-red-500"
    case "degraded":
    case "partial_outage":
      return "text-yellow-500"
    default:
      return "text-gray-500"
  }
}

export function getStatusBgColor(status: string): string {
  switch (status) {
    case "up":
    case "operational":
      return "bg-green-500/10 text-green-500 border-green-500/20"
    case "down":
    case "major_outage":
      return "bg-red-500/10 text-red-500 border-red-500/20"
    case "degraded":
    case "partial_outage":
      return "bg-yellow-500/10 text-yellow-500 border-yellow-500/20"
    default:
      return "bg-gray-500/10 text-gray-500 border-gray-500/20"
  }
}

export function getSeverityColor(severity: string): string {
  switch (severity) {
    case "critical":
      return "bg-red-500/10 text-red-500 border-red-500/20"
    case "warning":
      return "bg-yellow-500/10 text-yellow-500 border-yellow-500/20"
    case "info":
      return "bg-blue-500/10 text-blue-500 border-blue-500/20"
    default:
      return "bg-gray-500/10 text-gray-500 border-gray-500/20"
  }
}
