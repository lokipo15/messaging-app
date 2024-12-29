import { defineStore } from 'pinia'
import {computed, ref} from 'vue'
import axios from "axios";
import {useAuthStore} from "@/stores/auth.js";

export const useChatStore = defineStore('chat', () => {
  const messages = ref({})
  const selectedUser = ref(null)
  const selectedConversation = ref(null)
  const users = ref([])
  let ws = null

  const getUserMessages = (currentUserId) => computed((currentUserId) => {
    console.log(`ChatStore.selectedUser.ID=${selectedUser.value.ID}`)
    return messages.value.filter(msg => {
        console.log(`Message.fromId=${msg.fromId}`)
        console.log(`Message.from_id=${msg.from_id}`)
        return (msg.from_id === currentUserId && msg.to_id === selectedUser.value.ID) ||
          (msg.from_id === selectedUser.value.ID && msg.to_id === currentUserId)
      }
    )
  })

  function initWebSocket(token) {
    try {
      ws = new WebSocket(`ws://localhost:8080/ws?token=${encodeURIComponent(token)}`)
    } catch (e) {
      console.error(`Couldn't connect to WS server. Error=${e.messages[0]}`);
    }


    ws.onopen = () => {
      console.log("Connected to WS")
    }

    ws.onmessage = (event) => {
      const message = JSON.parse(event.data)
      messages.value[message.sender_id].push(message)
    }

    ws.onclose = () => {
      console.error("Connection to WS server lost. Retrying in 3 seconds...");
      setTimeout(() => initWebSocket(token), 3000)
    }
  }


  function sendMessage(content) {
    const authStore = useAuthStore()
    if (!selectedUser.value || !ws) return

    const message = {
      content,
      sender_id: authStore.user.ID,
      conversation_id: selectedConversation.value
    }

    console.log(`message sender_id: ${message.sender_id}`)
    messages.value[selectedUser.value.ID].push(message)
    ws.send(JSON.stringify(message))
  }

  async function fetchUsers() {
    try {
      const response = await fetch('/api/users', {
        headers: {
          'Authorization': `${localStorage.getItem('token')}`
        }
      })
      const responseObj = await response.json()
      users.value = responseObj.users;
      console.log(`Users value: ${users.value[0].username} ${users.value[0].ID}`)

    } catch (error) {
      console.error('Failed to fetch users:', error)
    }
  }

  async function selectUser(user) {
    selectedUser.value = user
    await fetchMessages()
  }

  async function fetchMessages() {
    const authStore = useAuthStore()
    if (!authStore.user) return null
    if (messages.value[selectedUser.value.ID]) return

    try {
      const response = await axios.get(`/api/conversation/${authStore.user.ID}/${selectedUser.value.ID}`)
      messages.value[selectedUser.value.ID] = response.data.conversation.messages;
      selectedConversation.value = response.data.conversation.ID
    } catch (error) {
      console.error('Failed to fetch messages:', error)
    }
  }

  return {
    messages,
    selectedUser,
    users,
    selectedConversation,
    getUserMessages,
    initWebSocket,
    sendMessage,
    fetchUsers,
    selectUser
  }
})
