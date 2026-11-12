import React, { createContext, useContext, useState, useCallback } from "react"
import axios from "axios"

export const AuthContext = createContext({
  user: null,
  isLoading: true,
  login: async (email: string, password: string) => {},
  logout: () => {},
  error: null,
})

export const useAuth = () => useContext(AuthContext)

// Initialize auth state from localStorage
const storedUser = typeof window !== "undefined" ? localStorage.getItem("pikyon_user") : null

export const AuthProvider = ({ children }: { children: React.ReactNode }) => {
  const [user, setUser] = useState(
    storedUser ? JSON.parse(storedUser) : null,
  )
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const login = async (email: string, password: string) => {
    setIsLoading(true)
    setError(null)
    try {
      const response = await axios.post("/api/v1/auth/login", { email, password })
      const { user, tokens } = response.data
      setUser(user)
      localStorage.setItem("pikyon_user", JSON.stringify(user))
      setIsLoading(false)
      // TODO: Store tokens securely (localStorage with rotation mitigations)
      return user
    } catch (err: any) {
      setError(err.response?.data?.message || "Login failed")
      setIsLoading(false)
      throw err
    }
  }

  const logout = () => {
    setUser(null)
    localStorage.removeItem("pikyon_user")
  }

  React.useEffect(() => {
    setIsLoading(false)
  }, [])

  return (
    <AuthContext.Provider value={{ user, isLoading, login, logout, error }}>
      {children}
    </AuthContext.Provider>
  )
}