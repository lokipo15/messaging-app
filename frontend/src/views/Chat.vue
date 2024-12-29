<!-- src/views/Chat.vue -->
<template>
  <div class="h-screen flex">
    <!-- Sidebar -->
    <div class="w-64 bg-gray-100 border-r">
      <div class="p-4 border-b bg-white flex justify-between items-center">
        <h2 class="text-xl font-bold">🦍 Ape Chat</h2>
        <button
          @click="logout"
          class="text-gray-100"
        >
          Logout
        </button>
      </div>

      <!-- User list -->
      <div class="overflow-y-auto h-full">
        <div
          v-for="user in chatStore.users"
          :key="user.ID"
          @click="chatStore.selectUser(user)"
          class="p-4 hover:bg-gray-200 cursor-pointer"
          :class="{ 'bg-gray-200': chatStore.selectedUser.ID === user.ID }"
        >
          <div class="font-medium">{{ user.username }}</div>
        </div>
      </div>
    </div>

    <!-- Chat area -->
    <div class="flex-1 flex flex-col">
      <!-- Chat header -->
      <div class="p-4 border-b bg-white">
        <h3 class="text-lg font-medium">
          {{ chatStore.selectedUser ? `Chat with ${chatStore.selectedUser.username}` : 'Select a user to start chatting' }}
        </h3>
      </div>

      <!-- Messages -->
      <div
        ref="messagesContainer"
        class="flex-1 overflow-y-auto p-4 space-y-4"
      >
        <template v-if="chatStore.selectedUser">
          <div
            v-for="message in chatStore.messages[chatStore.selectedUser.ID]"
            :key="message.id"
            class="flex"
            :class="{ 'justify-end': message.sender_id === currentUserId }"
          >
            <div
              class="max-w-sm rounded-lg px-4 py-2"
              :class="message.sender_id === currentUserId ? 'bg-indigo-500 text-white' : 'bg-gray-200'"
            >
              {{ message.content }}
            </div>
          </div>
        </template>
        <div v-else class="text-center text-gray-500">
          Select a user to start chatting
        </div>
      </div>

      <!-- Message input -->
      <div class="p-4 border-t bg-white">
        <form @submit.prevent="sendMessage" class="flex space-x-4">
          <input
            v-model="newMessage"
            type="text"
            placeholder="Type a message..."
            :disabled="!chatStore.selectedUser"
            class="flex-1 rounded-lg border-gray-300 focus:ring-indigo-500 focus:border-indigo-500"
          >
          <button
            type="submit"
            :disabled="!chatStore.selectedUser || !newMessage.trim()"
            class="bg-indigo-500 text-white px-4 py-2 rounded-lg hover:bg-indigo-600 disabled:opacity-50"
          >
            Send
          </button>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import {ref, computed, onMounted, nextTick, watchEffect, onBeforeMount} from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'

const router = useRouter()
const authStore = useAuthStore()
const chatStore = useChatStore()

const messagesContainer = ref(null)
const newMessage = ref('')
const currentUserId = computed(() => authStore.user?.ID)
// const userMessages = chatStore.getUserMessages(currentUserId)

// const sendMessageClient = () => chatStore.sendMessage(newMessage.value)


// Scroll to bottom when new messages arrive
watchEffect(async () => {
  if (chatStore.userMessages.length) {
    await nextTick()
    messagesContainer.value?.scrollTo({
      top: messagesContainer.value.scrollHeight,
      behavior: 'smooth'
    })
  }
})

onBeforeMount(async () => {
  // Initialize WebSocket with token
  const token = localStorage.getItem('token')
  chatStore.initWebSocket(token)
  await chatStore.fetchUsers()
  await chatStore.selectUser(chatStore.users[0])
})

// onMounted(async () => {
//   // Initialize WebSocket with token
//   const token = localStorage.getItem('token')
//   chatStore.initWebSocket(token)
//   await chatStore.fetchUsers()
//   await chatStore.selectUser(chatStore.users[0])
//
// })

async function sendMessage() {
  if (!newMessage.value.trim()) return

  try {
    await chatStore.sendMessage(newMessage.value)
    newMessage.value = ''
  } catch (error) {
    console.error('Failed to send message:', error)
  }
  // await chatStore.sendMessage(newMessage.value)
  // newMessage.value = ''
}

function logout() {
  authStore.logout()
  router.push('/login')
}
</script>
