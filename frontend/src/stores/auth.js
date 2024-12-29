import { defineStore } from 'pinia'
import axios from 'axios'

// Create axios instance with default config
const api = axios.create({
  baseURL: '/api'
})

// Add request interceptor to add token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('token') || null,
    user: null
  }),

  actions: {
    async initializeAuth() {
      const token = localStorage.getItem('token')
      if (!token) return

      try {
        // Validate token with backend
        const response = await api.get('/validate-token')
        this.user = response.data.user
        this.token = token
      } catch (error) {
        // If token is invalid, clear auth state
        console.error('Token validation failed:', error)
        this.logout()
      }
    },

    async login(username, password) {
      try {
        const response = await api.post('/login', {
          username,
          password
        })

        const token =response.data.token
        this.token = token
        this.user = response.data.user
        localStorage.setItem('token', token)

        return true
      } catch (error) {
        console.error('Login failed:', error)
        throw error
      }
    },

    async register(username, password) {
      try {
        await api.post('/register', {
          username,
          password
        })
        return true
      } catch (error) {
        console.error('Registration failed:', error)
        throw error
      }
    },

    logout() {
      this.token = null
      this.user = null
      localStorage.removeItem('token')
    }
  }
})
