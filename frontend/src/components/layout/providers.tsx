"use client"

import { useEffect } from "react"
import { ThemeProvider } from "next-themes"
import { QueryClientProvider } from "@tanstack/react-query"
import { Toaster } from "sonner"
import { useAuthStore } from "@/stores/auth"
import { queryClient } from "@/lib/query-client"

function AuthInitializer() {
  const { init, token, fetchUser } = useAuthStore()

  useEffect(() => {
    init()
  }, [init])

  useEffect(() => {
    if (token) {
      fetchUser()
    }
  }, [token, fetchUser])

  return null
}

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
        <AuthInitializer />
        {children}
        <Toaster richColors position="top-right" />
      </ThemeProvider>
    </QueryClientProvider>
  )
}
