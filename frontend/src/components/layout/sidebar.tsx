"use client"

import React from "react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import {
  Activity,
  LayoutDashboard,
  Globe,
  Bell,
  Building2,
  Shield,
  Wrench,
  FileText,
  TrendingUp,
  Search,
  Bug,
  Tag,
  Radar,
  Users,
  ScrollText,
  Terminal,
  Settings,
  LogOut,
  PanelLeftClose,
  PanelLeft,
  Skull,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { useAuthStore } from "@/stores/auth"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"

interface NavItem {
  label: string
  href: string
  icon: React.ComponentType<{ className?: string }>
  roles: string[]
}

interface NavGroup {
  title: string
  items: NavItem[]
}

const navGroups: NavGroup[] = [
  {
    title: "Overview",
    items: [
      { label: "Dashboard", href: "/", icon: LayoutDashboard, roles: ["super_admin", "admin_opd", "viewer"] },
      { label: "Websites", href: "/websites", icon: Globe, roles: ["super_admin", "admin_opd", "viewer"] },
      { label: "Alerts", href: "/alerts", icon: Bell, roles: ["super_admin", "admin_opd", "viewer"] },
    ],
  },
  {
    title: "Security",
    items: [
      { label: "Security Headers", href: "/security", icon: Shield, roles: ["super_admin", "admin_opd"] },
      { label: "Vulnerability", href: "/vulnerability", icon: Bug, roles: ["super_admin", "admin_opd"] },
      { label: "DNS Monitor", href: "/dns", icon: Radar, roles: ["super_admin", "admin_opd"] },
      { label: "Dork Detection", href: "/dork", icon: Search, roles: ["super_admin", "admin_opd"] },
      { label: "Defacement", href: "/defacement", icon: Skull, roles: ["super_admin", "admin_opd"] },
      { label: "Keywords", href: "/keywords", icon: Tag, roles: ["super_admin"] },
    ],
  },
  {
    title: "Operations",
    items: [
      { label: "Maintenance", href: "/maintenance", icon: Wrench, roles: ["super_admin", "admin_opd"] },
      { label: "Escalation", href: "/escalation", icon: TrendingUp, roles: ["super_admin"] },
      { label: "Reports", href: "/reports", icon: FileText, roles: ["super_admin", "admin_opd"] },
    ],
  },
  {
    title: "Administration",
    items: [
      { label: "OPD", href: "/opd", icon: Building2, roles: ["super_admin"] },
      { label: "Users", href: "/users", icon: Users, roles: ["super_admin"] },
      { label: "Audit Logs", href: "/audit-logs", icon: ScrollText, roles: ["super_admin"] },
      { label: "Admin Panel", href: "/admin", icon: Terminal, roles: ["super_admin"] },
      { label: "Settings", href: "/settings", icon: Settings, roles: ["super_admin"] },
    ],
  },
]

interface SidebarProps {
  collapsed?: boolean
  onToggle?: () => void
  className?: string
}

export function Sidebar({ collapsed = false, onToggle, className }: SidebarProps) {
  const pathname = usePathname()
  const { user, logout } = useAuthStore()

  const isActive = (href: string) => {
    if (href === "/") return pathname === "/"
    return pathname.startsWith(href)
  }

  const filteredGroups = navGroups
    .map((group) => ({
      ...group,
      items: user ? group.items.filter((item) => item.roles.includes(user.role)) : [],
    }))
    .filter((group) => group.items.length > 0)

  return (
    <TooltipProvider delayDuration={0}>
      <aside
        className={cn(
          "flex h-screen flex-col bg-sidebar text-sidebar-foreground transition-all duration-200 ease-out",
          collapsed ? "w-[60px]" : "w-[240px]",
          className
        )}
      >
        {/* Logo */}
        <div className={cn(
          "flex h-14 items-center shrink-0",
          collapsed ? "justify-center px-2" : "gap-2.5 px-4"
        )}>
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <Activity className="h-4 w-4" />
          </div>
          {!collapsed && (
            <div className="flex flex-col leading-none">
              <span className="text-sm font-semibold tracking-tight">Monitoring</span>
              <span className="text-[10px] text-muted-foreground font-medium">Pemprov Bali</span>
            </div>
          )}
        </div>

        {/* Navigation */}
        <ScrollArea className="flex-1 px-2">
          <nav className="flex flex-col gap-4 py-2">
            {filteredGroups.map((group) => (
              <div key={group.title}>
                {!collapsed && (
                  <p className="mb-1 px-3 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                    {group.title}
                  </p>
                )}
                {collapsed && group !== filteredGroups[0] && (
                  <div className="mx-auto mb-2 h-px w-6 bg-sidebar-border" />
                )}
                <div className="flex flex-col gap-0.5">
                  {group.items.map((item) => {
                    const Icon = item.icon
                    const active = isActive(item.href)

                    const linkContent = (
                      <Link
                        href={item.href}
                        className={cn(
                          "group flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-[13px] font-medium transition-colors",
                          active
                            ? "bg-primary/10 text-primary"
                            : "text-sidebar-foreground/65 hover:bg-sidebar-accent hover:text-sidebar-foreground",
                          collapsed && "justify-center px-0 py-2"
                        )}
                      >
                        <Icon className={cn(
                          "h-4 w-4 shrink-0",
                          active ? "text-primary" : "text-sidebar-foreground/50 group-hover:text-sidebar-foreground/75"
                        )} />
                        {!collapsed && <span>{item.label}</span>}
                      </Link>
                    )

                    if (collapsed) {
                      return (
                        <Tooltip key={item.href}>
                          <TooltipTrigger asChild>{linkContent}</TooltipTrigger>
                          <TooltipContent side="right" sideOffset={8} className="text-xs font-medium">
                            {item.label}
                          </TooltipContent>
                        </Tooltip>
                      )
                    }

                    return <React.Fragment key={item.href}>{linkContent}</React.Fragment>
                  })}
                </div>
              </div>
            ))}
          </nav>
        </ScrollArea>

        {/* Bottom section */}
        <div className="shrink-0 border-t border-sidebar-border p-2">
          {/* Collapse toggle */}
          {onToggle && (
            <button
              onClick={onToggle}
              className={cn(
                "flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-[13px] font-medium text-sidebar-foreground/50 transition-colors hover:bg-sidebar-accent hover:text-sidebar-foreground",
                collapsed && "justify-center px-0"
              )}
            >
              {collapsed ? (
                <PanelLeft className="h-4 w-4" />
              ) : (
                <>
                  <PanelLeftClose className="h-4 w-4" />
                  <span>Collapse</span>
                </>
              )}
            </button>
          )}

          {/* User + Logout */}
          <div className={cn(
            "mt-1 flex items-center rounded-md",
            collapsed ? "justify-center py-1.5" : "gap-2.5 px-2.5 py-1.5"
          )}>
            {!collapsed && user && (
              <div className="flex-1 min-w-0">
                <p className="truncate text-[13px] font-medium text-sidebar-foreground/85">{user.full_name}</p>
                <p className="truncate text-[10px] text-sidebar-foreground/45 capitalize">
                  {user.role.replace("_", " ")}
                </p>
              </div>
            )}
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={logout}
                  className="h-7 w-7 shrink-0 text-sidebar-foreground/45 hover:text-destructive hover:bg-destructive/10"
                >
                  <LogOut className="h-3.5 w-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side={collapsed ? "right" : "top"} className="text-xs">
                Logout
              </TooltipContent>
            </Tooltip>
          </div>
        </div>
      </aside>
    </TooltipProvider>
  )
}

/** Sidebar content for mobile Sheet */
export function SidebarContent() {
  const pathname = usePathname()
  const { user, logout } = useAuthStore()

  const isActive = (href: string) => {
    if (href === "/") return pathname === "/"
    return pathname.startsWith(href)
  }

  const filteredGroups = navGroups
    .map((group) => ({
      ...group,
      items: user ? group.items.filter((item) => item.roles.includes(user.role)) : [],
    }))
    .filter((group) => group.items.length > 0)

  return (
    <div className="flex h-full flex-col">
      <div className="flex h-14 items-center gap-2.5 px-4 shrink-0">
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
          <Activity className="h-4 w-4" />
        </div>
        <div className="flex flex-col leading-none">
          <span className="text-sm font-semibold tracking-tight">Monitoring</span>
          <span className="text-[10px] text-muted-foreground font-medium">Pemprov Bali</span>
        </div>
      </div>

      <ScrollArea className="flex-1 px-2">
        <nav className="flex flex-col gap-4 py-2">
          {filteredGroups.map((group) => (
            <div key={group.title}>
              <p className="mb-1 px-3 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">
                {group.title}
              </p>
              <div className="flex flex-col gap-0.5">
                {group.items.map((item) => {
                  const Icon = item.icon
                  const active = isActive(item.href)

                  return (
                    <Link
                      key={item.href}
                      href={item.href}
                      className={cn(
                        "flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-[13px] font-medium transition-colors",
                        active
                          ? "bg-primary/10 text-primary"
                          : "text-muted-foreground hover:bg-accent hover:text-foreground"
                      )}
                    >
                      <Icon className="h-4 w-4 shrink-0" />
                      <span>{item.label}</span>
                    </Link>
                  )
                })}
              </div>
            </div>
          ))}
        </nav>
      </ScrollArea>

      <div className="shrink-0 border-t p-3">
        <div className="flex items-center gap-2.5">
          {user && (
            <div className="flex-1 min-w-0">
              <p className="truncate text-[13px] font-medium">{user.full_name}</p>
              <p className="truncate text-[10px] text-muted-foreground capitalize">
                {user.role.replace("_", " ")}
              </p>
            </div>
          )}
          <Button
            variant="ghost"
            size="icon"
            onClick={logout}
            className="h-7 w-7 shrink-0 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
          >
            <LogOut className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>
    </div>
  )
}
