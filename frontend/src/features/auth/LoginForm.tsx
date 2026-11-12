import React, { useState } from "react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useNavigate } from "react-router-dom"
import { GoogleOAuthButton } from "./GoogleOAuthButton"
import { useAuth } from "@/context/AuthContext"

const loginSchema = z.object({
  email: z.string().email("Please enter a valid email"),
  password: z.string().min(8, "Password must be at least 8 characters"),
})

type LoginFormValues = z.infer<typeof loginSchema>

export default function LoginForm() {
  const { login, isLoading, error } = useAuth()
  const navigate = useNavigate()
  const [showGoogle, setShowGoogle] = useState(false)

  const { register, handleSubmit, watch } = useForm<loginSchema>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: watch("email"),
      password: "",
    },
  })

  const onSubmit = async (data: loginSchema) => {
    try {
      await login(data.email, data.password)
      navigate("/")
    } catch (err: any) {
      setShowGoogle(true)
    }
  }

  return (
    <div className="min-h-screen bg-gray-950 text-gray-100 flex items-center justify-center p-4">
      <div className="w-full max-w-md space-y-6">
        <div>
          <h2 className="text-2xl font-bold tracking-tight text-center text-white">Sign in</h2>
          <p className="text-sm text-gray-400">Log in to your Pikyon account</p>
        </div>

        {/* Email/Password Form */}
        <form onSubmit={handleSubmit(onSubmit)} className="mt-8 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2" htmlFor="email">
              Email
            </label>
            <Input
              id="email"
              type="email"
              placeholder="you@email.com"
              {...register("email")}
              required
              className="bg-gray-800"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2" htmlFor="password">
              Password
            </label>
            <Input
              id="password"
              type="password"
              placeholder="••••••••"
              {...register("password")}
              required
              className="bg-gray-800"
            />
          </div>

          <button
            type="submit"
            disabled={isLoading}
            className="w-full flex items-center justify-center py-2.5 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 animate-pulse"
          >
            <span className="hidden sm:inline">Sign in</span>
            <svg className="-ml-1 mr-2 h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M5 12h14M12 5l7 7-7 7"/>
            </svg>
            Sign in
          </button>
        </form>

        {/* Google OAuth Divider */}
        <div className="flex items-center text-sm text-gray-400">
          <span>or continue with</span>
        </div>

        {/* Google Button */}
        <GoogleOAuthButton onGoogleClick={() => setShowGoogle(false)} />

        {/* Forgot password link */}
        <div className="text-center text-sm text-gray-400 mt-4">
          <a href="javascript:void(0)" className="underline">
            Forgot password?
          </a>
        </div>
      </div>
    </div>
  )
}