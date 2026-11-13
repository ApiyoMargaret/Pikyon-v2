import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from "axios"

// Request queue for refresh token operations
// Ensures only one refresh token request runs at a time to prevent race conditions
class RequestQueue {
  private pending: Array<{
    resolve: (value: any) => void
    reject: (reason: any) => void
    request: Promise<any>
  }> = []
  private active: boolean = false

  enqueue<T>(promiseFactory: () => Promise<T>): Promise<T> {
    const promise = promiseFactory()

    if (this.active) {
      // Wait for the ongoing request to complete
      return new Promise<T>((resolve, reject) => {
        this.pending.push({ resolve, reject, request: promise })
      })
    }

    this.active = true

    return promise
      .then((result) => {
        this.active = false
        this.processNext()
        return result
      })
      .catch((error) => {
        this.active = false
        this.processNext()
        throw error
      })
  }

  private processNext() {
    if (this.pending.length === 0) return

    const next = this.pending.shift()
    this.active = true

    next.request
      .then((result) => {
        next.resolve(result)
        this.active = false
        this.processNext()
      })
      .catch((error) => {
        next.reject(error)
        this.active = false
        this.processNext()
      })
  }
}

// Singleton instance
export const apiRequestQueue = new RequestQueue()

// API client configuration
const api: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || "http://localhost:8080/api/v1",
  timeout: 15000,
  withCredentials: true, // Required for cookies (access/refresh tokens)
})

// Request/response interceptors
api.interceptors.request.use(
  (config) => config,
  (error) => Promise.reject(error)
)

api.interceptors.response.use(
  (response: AxiosResponse) => response,
  (error) => {
    // Handle 401 - token might be expired
    if (error.response?.status === 401) {
      // Try to refresh the token through the queue
      return apiRequestQueue.enqueue(() => refreshToken()).then(() => {
        // Retry the original request with new token
        return api.request(error.config!)
      })
    }
    return Promise.reject(error)
  }
)

// Helper to refresh token
async function refreshToken(): Promise<void> {
  try {
    const response = await api.post("/auth/refresh", {})
    // Store new tokens - handled by auth interceptor / auth store
    // The response should contain new access token
  } catch (err) {
    // If refresh fails, logout
    // TODO: Dispatch logout action
    throw err
  }
}

// Type-safe API request wrapper with queue protection
export const apiGet = <T>(url: string, config?: AxiosRequestConfig): Promise<AxiosResponse<T>> => {
  return apiRequestQueue.enqueue(() => api.get<T>(url, config)).then(
    (res) => res as AxiosResponse<T>,
    (err) => Promise.reject(err)
  )
}

export const apiPost = <T, D = any>(url: string, data?: D, config?: AxiosRequestConfig): Promise<AxiosResponse<T>> => {
  return apiRequestQueue.enqueue(() => api.post<T>(url, data, config)).then(
    (res) => res as AxiosResponse<T>,
    (err) => Promise.reject(err)
  )
}

export const apiPut = <T, D = any>(url: string, data?: D, config?: AxiosRequestConfig): Promise<AxiosResponse<T>> => {
  return apiRequestQueue.enqueue(() => api.put<T>(url, data, config)).then(
    (res) => res as AxiosResponse<T>,
    (err) => Promise.reject(err)
  )
}

export const apiDelete = <T>(url: string, config?: AxiosRequestConfig): Promise<AxiosResponse<T>> => {
  return apiRequestQueue.enqueue(() => api.delete<T>(url, config)).then(
    (res) => res as AxiosResponse<T>,
    (err) => Promise.reject(err)
  )
}