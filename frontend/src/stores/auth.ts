import { create } from "zustand"
import { authApi, type User } from "@/lib/api"

interface AuthState {
  user: User | null
  token: string | null
  isLoading: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => void
  fetchUser: () => Promise<void>
  init: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  token: null,
  isLoading: true,

  init: () => {
    const token = localStorage.getItem("token")
    if (token) {
      set({ token })
    } else {
      set({ isLoading: false })
    }
  },

  login: async (username: string, password: string) => {
    const res = await authApi.login(username, password)
    localStorage.setItem("token", res.token)
    set({ token: res.token })
    const userRes = await authApi.me()
    set({ user: userRes.data, isLoading: false })
  },

  logout: () => {
    localStorage.removeItem("token")
    set({ user: null, token: null })
    window.location.href = "/login"
  },

  fetchUser: async () => {
    try {
      const res = await authApi.me()
      set({ user: res.data, isLoading: false })
    } catch {
      localStorage.removeItem("token")
      set({ user: null, token: null, isLoading: false })
    }
  },
}))
