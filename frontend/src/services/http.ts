import axios from 'axios'

export const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 15_000,
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('shikan_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (response) => response,
  (error) => {
    const message = error.response?.data?.error?.message
    if (typeof message === 'string' && message) {
      error.message = message
    }
    if (error.response?.status === 401) {
      localStorage.removeItem('shikan_token')
      localStorage.removeItem('shikan_user')
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  },
)
