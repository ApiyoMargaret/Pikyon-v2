import React from "react"
import { useAuth } from "@/context/AuthContext"
import { useNavigate } from "react-router-dom"

export default function HomePage() {
  const { user, isLoading, logout } = useAuth()
  const navigate = useNavigate()

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-950 text-gray-100 flex items-center justify-center p-4">
        <div className="text-center">
          <div />
          <p>Loading...</p>
        </div>
      </div>
    )
  }

  if (user) {
    return (
      <div className="text-center">
        <h2 className="text-2xl font-bold text-white">Welcome, {user.name}</h2>
        <p className="text-gray-400">You're logged into Pikyon</p>
        <button
          onClick={() => {
            logout()
            navigate("/login")
          }}
          className="mt-4 py-2 px-4 border border-gray-500 rounded-md text-sm text-gray-300 hover:bg-gray-700"
        >
          Logout
        </button>
      </div>
    )
  }

  return (
    <div className="text-center">
      <h3 className="text-xl font-bold text-white">Pikyon</h3>
      <p className="text-gray-400">A memory-sharing community</p>
      <div className="mt-6">
        <a
          href="/login"
          className="inline-block py-2 px-4 border border-gray-500 rounded-md text-sm text-gray-300 hover:bg-gray-700"
        >
          Sign in
        </a>
        <a
          href="/register"
          className="inline-block py-2 px-4 border border-indigo-600 rounded-md text-sm text-indigo-600 hover:bg-indigo-100 ml-3"
        >
          Sign up
        </a>
      </div>
    </div>
  )
}
