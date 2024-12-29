<!-- src/App.vue -->
<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Loading overlay -->
    <div v-if="loading" class="fixed inset-0 bg-gray-900 bg-opacity-50 flex items-center justify-center z-50">
      <div class="animate-spin rounded-full h-32 w-32 border-t-2 border-b-2 border-indigo-500"></div>
    </div>

    <!-- Main content -->
    <router-view v-else></router-view>

    <!-- Toast notifications -->
    <div class="fixed bottom-4 right-4 z-50">
      <div v-for="toast in toasts" :key="toast.id"
           class="mb-2 p-4 rounded-lg shadow-lg text-white"
           :class="{
             'bg-green-500': toast.type === 'success',
             'bg-red-500': toast.type === 'error',
             'bg-blue-500': toast.type === 'info'
           }">
        {{ toast.message }}
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useAuthStore } from './stores/auth'

const loading = ref(true)
const toasts = ref([])
const authStore = useAuthStore()

// Initialize app
onMounted(async () => {
  try {
    // Initialize any app-wide resources here
    await authStore.initializeAuth()
  } catch (error) {
    console.error('App initialization failed:', error)
  } finally {
    loading.value = false
  }
})

// Toast notification system
function showToast(message, type = 'info') {
  const id = Date.now()
  toasts.value.push({ id, message, type })
  setTimeout(() => {
    toasts.value = toasts.value.filter(toast => toast.id !== id)
  }, 3000)
}

// Make toast function available globally
window.showToast = showToast
</script>

<style>
/* Base styles */
html {
  scroll-behavior: smooth;
}

body {
  @apply antialiased text-gray-900;
}

/* Form elements */
input, select, textarea {
  @apply block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500;
}

/* Buttons */
button {
  @apply inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500;
}

button:disabled {
  @apply opacity-50 cursor-not-allowed;
}

/* Transitions */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
