import React from "react"
import { useNavigate } from "react-router-dom"
import { useAuth } from "@/context/AuthContext"
import LoginForm from "@/features/auth/LoginForm"

export default function LoginPage() {
  const navigate = useNavigate()
  const { login, error, isLoading } = useAuth()

  return (
    <div className="min-h-screen bg-gray-950 text-gray-100 flex items-center justify-center p-4">
      <div className="w-full max-w-md space-y-6">
        <h2 className="text-2xl font-bold tracking-tight text-center text-white">Sign in</h2>
        <p className="text-sm text-gray-400 text-center">Log in to your Pikyon account</p>

        {error && (
          <div className="bg-red-500 text-white p-4 rounded-md mb-4 text-center">
            {error}
          </div>
        )}

        {isLoading && (
          <div className="bg-blue-500 text-white p-4 rounded-md text-center">
            Logging in...
          </div>
        )}

        <LoginForm onLogin={() => navigate("/")} />
      </div>
    </div>
  )
}
