import React from "react"
import { useNavigate } from "react-router-dom"
import { useAuth } from "@/context/AuthContext"

export default function RegisterPage() {
  const navigate = useNavigate()
  const { isLoading } = useAuth()

  return (
    <div className="min-h-screen bg-gray-950 text-gray-100 flex items-center justify-center p-4">
      <div className="w-full max-w-md space-y-6">
        <h2 className="text-2xl font-bold tracking-tight text-center text-white">Sign up</h2>
        <p className="text-sm text-gray-400 text-center">Create your Pikyon account</p>

        <div className="bg-gray-800 p-6 rounded-md max-w-md">
          <h3 className="text-xl font-bold text-white mb-4">Register with Email</h3>
          <form className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Email</label>
              <input
                type="email"
                name="email"
                placeholder="you@email.com"
                className="w-full bg-gray-700 border border-gray-500 rounded-md px-3 py-2 focus:outline-none focus:border-indigo-500"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Full Name</label>
              <input
                type="text"
                name="fullName"
                placeholder="John Doe"
                className="w-full bg-gray-700 border border-gray-500 rounded-md px-3 py-2 focus:outline-none focus:border-indigo-500"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">Password</label>
              <input
                type="password"
                name="password"
                placeholder="••••••••"
                className="w-full bg-gray-700 border border-gray-500 rounded-md px-3 py-2 focus:outline-none focus:border-indigo-500"
                required
                minlength={8}
              />
            </div>
            <button
              type="submit"
              disabled={isLoading}
              className="w-full py-2.5 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 animate-pulse"
            >
              Sign up
            </button>
          </form>
          <p className="text-center text-sm text-gray-400 mt-4">
            Already have an account? <a href="/login" className="font-medium text-indigo-600 hover:text-indigo-500">Sign in</a>
          </p>
        </div>
      </div>
    </div>
  )
}
