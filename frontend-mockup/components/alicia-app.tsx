"use client"

import { useState } from "react"
import { Sidebar } from "./sidebar"
import { ChatArea } from "./chat-area"
import { ServerInfoPanel } from "./server-info-panel"
import { MemoryPanel } from "./memory-panel"
import { SettingsPanel } from "./settings-panel"

export type Conversation = {
  id: string
  title: string
  lastMessage: string
  updatedAt: Date
  status: "active" | "archived"
}

export type Message = {
  id: string
  role: "user" | "assistant" | "system"
  content: string
  timestamp: Date
  isStreaming?: boolean
  toolCalls?: ToolCall[]
  memories?: MemoryReference[]
  reasoningSteps?: string[]
  audioId?: string
  vote?: "up" | "down" | null
  siblings?: string[] // IDs of alternative versions of this message
  siblingIndex?: number // Current position in siblings array
  parentId?: string // ID of the parent message this was derived from
}

export type ToolCall = {
  id: string
  name: string
  parameters: Record<string, unknown>
  result?: unknown
  status: "pending" | "success" | "error"
}

export type MemoryReference = {
  id: string
  content: string
  relevance: number
  type: string
}

export type Commentary = {
  id: string
  conversationId: string
  targetId: string
  authorRole: "user" | "assistant" | "moderator"
  content: string
  rating?: 1 | 2 | 3 | 4 | 5
  commentType: "feedback-positive" | "feedback-negative" | "explanation" | "note" | "correction"
  timestamp: number
}

export type ConnectionStatus = "connected" | "connecting" | "disconnected" | "reconnecting"

export type VoiceState = "idle" | "listening" | "processing" | "speaking"

export function AliciaApp() {
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [activePanel, setActivePanel] = useState<"chat" | "memory" | "server" | "settings">("chat")
  const [conversations, setConversations] = useState<Conversation[]>([
    {
      id: "conv_1",
      title: "Restaurant Recommendations",
      lastMessage: "I found several Italian restaurants in NYC...",
      updatedAt: new Date(Date.now() - 1000 * 60 * 5),
      status: "active",
    },
    {
      id: "conv_2",
      title: "Weather Analysis",
      lastMessage: "The weather in Tokyo is sunny with...",
      updatedAt: new Date(Date.now() - 1000 * 60 * 60),
      status: "active",
    },
    {
      id: "conv_3",
      title: "Code Review Help",
      lastMessage: "I've analyzed your function and found...",
      updatedAt: new Date(Date.now() - 1000 * 60 * 60 * 24),
      status: "archived",
    },
  ])
  const [activeConversationId, setActiveConversationId] = useState<string>("conv_1")
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>("connected")
  const [voiceState, setVoiceState] = useState<VoiceState>("idle")

  const handleNewConversation = () => {
    const newConv: Conversation = {
      id: `conv_${Date.now()}`,
      title: "New Conversation",
      lastMessage: "",
      updatedAt: new Date(),
      status: "active",
    }
    setConversations([newConv, ...conversations])
    setActiveConversationId(newConv.id)
  }

  return (
    <div className="flex h-screen bg-background overflow-hidden">
      {/* Sidebar */}
      <Sidebar
        open={sidebarOpen}
        onToggle={() => setSidebarOpen(!sidebarOpen)}
        conversations={conversations}
        activeConversationId={activeConversationId}
        onSelectConversation={setActiveConversationId}
        onNewConversation={handleNewConversation}
        connectionStatus={connectionStatus}
        activePanel={activePanel}
        onPanelChange={setActivePanel}
      />

      {/* Main Content */}
      <main className="flex-1 flex flex-col min-w-0 min-h-0">
        {activePanel === "chat" && (
          <ChatArea
            conversationId={activeConversationId}
            voiceState={voiceState}
            onVoiceStateChange={setVoiceState}
            connectionStatus={connectionStatus}
          />
        )}
        {activePanel === "memory" && <MemoryPanel />}
        {activePanel === "server" && <ServerInfoPanel connectionStatus={connectionStatus} />}
        {activePanel === "settings" && <SettingsPanel />}
      </main>
    </div>
  )
}
