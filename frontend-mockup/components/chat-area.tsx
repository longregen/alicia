"use client"

import type React from "react"

import { useState, useRef, useEffect } from "react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import {
  Mic,
  MicOff,
  Send,
  RefreshCw,
  ThumbsUp,
  ThumbsDown,
  Wrench,
  Brain,
  Volume2,
  Copy,
  Check,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Pencil,
  Paperclip,
  X,
} from "lucide-react"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { VoiceVisualizer } from "./voice-visualizer"
import type { Message, ToolCall, MemoryReference, VoiceState, ConnectionStatus, Commentary } from "./alicia-app"

type ChatAreaProps = {
  conversationId: string
  voiceState: VoiceState
  onVoiceStateChange: (state: VoiceState) => void
  connectionStatus: ConnectionStatus
}

export function ChatArea({ conversationId, voiceState, onVoiceStateChange, connectionStatus }: ChatAreaProps) {
  const [inputValue, setInputValue] = useState("")
  const [isRecording, setIsRecording] = useState(false)
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const [feedbackOpenFor, setFeedbackOpenFor] = useState<{ id: string; type: "up" | "down" } | null>(null)
  const [feedbackContent, setFeedbackContent] = useState("")
  const [feedbackType, setFeedbackType] = useState<Commentary["commentType"]>("feedback-positive")
  const [editingMessageId, setEditingMessageId] = useState<string | null>(null)
  const [editContent, setEditContent] = useState("")
  const scrollRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Mock messages for demonstration - now with siblings for branching
  const [messages, setMessages] = useState<Message[]>([
    {
      id: "msg_1",
      role: "user",
      content: "What are some good Italian restaurants in New York?",
      timestamp: new Date(Date.now() - 1000 * 60 * 10),
      siblings: ["msg_1", "msg_1_alt"],
      siblingIndex: 0,
    },
    {
      id: "msg_2",
      role: "assistant",
      content:
        "I found several highly-rated Italian restaurants in New York City for you. Let me share the top recommendations based on recent reviews and ratings.",
      timestamp: new Date(Date.now() - 1000 * 60 * 9),
      toolCalls: [
        {
          id: "tool_1",
          name: "web_search",
          parameters: { query: "best Italian restaurants NYC 2024", limit: 5 },
          result: { count: 5, source: "Google" },
          status: "success",
        },
      ],
      memories: [
        {
          id: "mem_1",
          content: "User prefers authentic Italian cuisine",
          relevance: 0.92,
          type: "preference",
        },
      ],
    },
    {
      id: "msg_3",
      role: "assistant",
      content:
        "**Luigi's Trattoria** stands out with a 4.5-star rating. They're known for their handmade pasta and warm atmosphere. Located in the West Village.\n\n**Pasta Palace** is another excellent choice with 4.3 stars. Their seafood linguine is particularly popular. They have locations in both Midtown and SoHo.\n\nWould you like more details about either of these, or would you prefer recommendations for a specific neighborhood?",
      timestamp: new Date(Date.now() - 1000 * 60 * 8),
      vote: "up",
      siblings: ["msg_3", "msg_3_v2"],
      siblingIndex: 0,
    },
    {
      id: "msg_4",
      role: "user",
      content: "Tell me more about Luigi's Trattoria",
      timestamp: new Date(Date.now() - 1000 * 60 * 5),
    },
    {
      id: "msg_5",
      role: "assistant",
      content: "",
      timestamp: new Date(),
      isStreaming: true,
      reasoningSteps: [
        "User wants more details about Luigi's Trattoria",
        "Looking up restaurant details including menu, hours, and reviews...",
      ],
    },
  ])

  const [messageVersions, setMessageVersions] = useState<Record<string, Message>>({
    msg_1_alt: {
      id: "msg_1_alt",
      role: "user",
      content: "Can you recommend some Italian places in NYC for a date night?",
      timestamp: new Date(Date.now() - 1000 * 60 * 10),
      siblings: ["msg_1", "msg_1_alt"],
      siblingIndex: 1,
    },
    msg_3_v2: {
      id: "msg_3_v2",
      role: "assistant",
      content:
        "Here are my top Italian restaurant picks for NYC:\n\n1. **Carbone** - Upscale Italian-American in Greenwich Village\n2. **L'Artusi** - Modern Italian with great wine selection\n3. **Via Carota** - Charming West Village spot\n\nAll three have excellent reviews and unique atmospheres!",
      timestamp: new Date(Date.now() - 1000 * 60 * 8),
      siblings: ["msg_3", "msg_3_v2"],
      siblingIndex: 1,
    },
  })

  const handleSend = () => {
    if (!inputValue.trim()) return

    const newMessage: Message = {
      id: `msg_${Date.now()}`,
      role: "user",
      content: inputValue,
      timestamp: new Date(),
    }
    setMessages([...messages, newMessage])
    setInputValue("")

    setTimeout(() => {
      setMessages((prev) => [
        ...prev,
        {
          id: `msg_${Date.now()}`,
          role: "assistant",
          content: "I'll help you with that...",
          timestamp: new Date(),
          isStreaming: true,
        },
      ])
    }, 500)
  }

  const toggleRecording = () => {
    if (isRecording) {
      setIsRecording(false)
      onVoiceStateChange("processing")
      setTimeout(() => onVoiceStateChange("idle"), 2000)
    } else {
      setIsRecording(true)
      onVoiceStateChange("listening")
    }
  }

  const handleVote = (messageId: string, vote: "up" | "down" | null) => {
    if (vote) {
      setFeedbackOpenFor({ id: messageId, type: vote })
      setFeedbackType(vote === "up" ? "feedback-positive" : "feedback-negative")
      setFeedbackContent("")
    } else {
      setMessages((prev) => prev.map((msg) => (msg.id === messageId ? { ...msg, vote: null } : msg)))
    }
  }

  const submitFeedback = (messageId: string, type: "up" | "down") => {
    const commentary: Commentary = {
      id: `comm_${Date.now()}`,
      conversationId,
      targetId: messageId,
      authorRole: "user",
      content: feedbackContent,
      rating: type === "up" ? 5 : 1,
      commentType: feedbackType,
      timestamp: Date.now(),
    }

    // In real app, send this via LiveKit data channel
    console.log("Submitting commentary:", commentary)

    setMessages((prev) => prev.map((msg) => (msg.id === messageId ? { ...msg, vote: type } : msg)))
    setFeedbackOpenFor(null)
    setFeedbackContent("")
  }

  const startEditing = (message: Message) => {
    setEditingMessageId(message.id)
    setEditContent(message.content)
  }

  const cancelEditing = () => {
    setEditingMessageId(null)
    setEditContent("")
  }

  const saveEdit = (message: Message) => {
    if (!editContent.trim()) return

    const newVersionId = `${message.id}_v${Date.now()}`

    if (message.role === "user") {
      // User edit: create alternative branch
      const currentSiblings = message.siblings || [message.id]
      const newSiblings = [...currentSiblings, newVersionId]

      // Create new version
      const newVersion: Message = {
        ...message,
        id: newVersionId,
        content: editContent,
        siblings: newSiblings,
        siblingIndex: newSiblings.length - 1,
        timestamp: new Date(),
      }

      // Update original message with new siblings
      setMessages((prev) =>
        prev.map((msg) => (msg.id === message.id ? { ...msg, siblings: newSiblings, siblingIndex: 0 } : msg)),
      )

      setMessageVersions((prev) => ({
        ...prev,
        [newVersionId]: newVersion,
      }))
    } else {
      // Assistant edit: treated as correction feedback
      const correction: Commentary = {
        id: `comm_${Date.now()}`,
        conversationId,
        targetId: message.id,
        authorRole: "user",
        content: editContent,
        rating: 2,
        commentType: "correction",
        timestamp: Date.now(),
      }

      console.log("Submitting correction:", correction)

      // Create corrected sibling
      const currentSiblings = message.siblings || [message.id]
      const newSiblings = [...currentSiblings, newVersionId]

      const correctedVersion: Message = {
        ...message,
        id: newVersionId,
        content: editContent,
        siblings: newSiblings,
        siblingIndex: newSiblings.length - 1,
        timestamp: new Date(),
      }

      setMessages((prev) =>
        prev.map((msg) => (msg.id === message.id ? { ...msg, siblings: newSiblings, siblingIndex: 0 } : msg)),
      )

      setMessageVersions((prev) => ({
        ...prev,
        [newVersionId]: correctedVersion,
      }))
    }

    setEditingMessageId(null)
    setEditContent("")
  }

  const navigateVersion = (message: Message, direction: "prev" | "next") => {
    if (!message.siblings || message.siblings.length <= 1) return

    const currentIndex = message.siblingIndex || 0
    const newIndex = direction === "prev" ? currentIndex - 1 : currentIndex + 1

    if (newIndex < 0 || newIndex >= message.siblings.length) return

    const targetVersionId = message.siblings[newIndex]

    if (targetVersionId === message.id) {
      // Already on this version
      setMessages((prev) => prev.map((msg) => (msg.id === message.id ? { ...msg, siblingIndex: newIndex } : msg)))
    } else {
      // Swap to different version
      const targetVersion = messageVersions[targetVersionId]
      if (targetVersion) {
        setMessages((prev) =>
          prev.map((msg) =>
            msg.id === message.id
              ? {
                  ...targetVersion,
                  siblingIndex: newIndex,
                }
              : msg,
          ),
        )
      }
    }
  }

  const handleFileSelect = () => {
    fileInputRef.current?.click()
  }

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files
    if (files && files.length > 0) {
      // In real app, upload and attach to message
      console.log(
        "Files selected:",
        Array.from(files).map((f) => f.name),
      )
    }
  }

  const handleCopy = (content: string, id: string) => {
    navigator.clipboard.writeText(content)
    setCopiedId(id)
    setTimeout(() => setCopiedId(null), 2000)
  }

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messages])

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div className="flex-1 flex flex-col bg-background min-h-0">
      {/* Chat Header */}
      <header className="h-14 border-b border-border flex items-center justify-between px-4 flex-shrink-0">
        <div className="flex items-center gap-3">
          <h2 className="font-medium">Restaurant Recommendations</h2>
          <span className="text-xs text-muted-foreground">conv_1</span>
        </div>
        <div className="flex items-center gap-2">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon">
                  <Volume2 className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Toggle audio output</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
      </header>

      {/* Messages */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto p-4 min-h-0">
        <div className="max-w-3xl mx-auto space-y-6">
          {messages.map((message) => (
            <MessageBubble
              key={message.id}
              message={message}
              onVote={handleVote}
              onCopy={handleCopy}
              copiedId={copiedId}
              onEdit={startEditing}
              onNavigateVersion={navigateVersion}
              isEditing={editingMessageId === message.id}
              editContent={editContent}
              onEditContentChange={setEditContent}
              onSaveEdit={() => saveEdit(message)}
              onCancelEdit={cancelEditing}
              feedbackOpenFor={feedbackOpenFor}
              onFeedbackOpenChange={(open) => !open && setFeedbackOpenFor(null)}
              feedbackContent={feedbackContent}
              onFeedbackContentChange={setFeedbackContent}
              feedbackType={feedbackType}
              onFeedbackTypeChange={setFeedbackType}
              onSubmitFeedback={submitFeedback}
            />
          ))}
        </div>
      </div>

      {/* Voice State Indicator */}
      {voiceState !== "idle" && (
        <div className="border-t border-border p-4 flex-shrink-0">
          <VoiceVisualizer state={voiceState} />
        </div>
      )}

      {/* Input Area */}
      <div className="border-t border-border p-4 flex-shrink-0">
        <div className="max-w-3xl mx-auto">
          <div className="flex items-center gap-3">
            {/* Voice Button */}
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant={isRecording ? "destructive" : "secondary"}
                    size="icon"
                    onClick={toggleRecording}
                    className={cn("flex-shrink-0 h-10 w-10 transition-all", isRecording && "animate-pulse")}
                  >
                    {isRecording ? <MicOff className="h-4 w-4" /> : <Mic className="h-4 w-4" />}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{isRecording ? "Stop recording" : "Start voice input"}</TooltipContent>
              </Tooltip>
            </TooltipProvider>

            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="secondary"
                    size="icon"
                    onClick={handleFileSelect}
                    className="flex-shrink-0 h-10 w-10"
                  >
                    <Paperclip className="h-4 w-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Attach file</TooltipContent>
              </Tooltip>
            </TooltipProvider>
            <input ref={fileInputRef} type="file" className="hidden" onChange={handleFileChange} multiple />

            {/* Text Input */}
            <div className="flex-1 relative">
              <Textarea
                ref={inputRef}
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Type a message or press the mic to speak..."
                className="min-h-[44px] max-h-32 resize-none pr-12 bg-input border-border py-2.5"
                rows={1}
              />
              <Button
                variant="ghost"
                size="icon"
                onClick={handleSend}
                disabled={!inputValue.trim()}
                className="absolute right-2 top-1/2 -translate-y-1/2 h-8 w-8"
              >
                <Send className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>
      </div>

      {/* Removed Dialog, feedback is now handled inline */}
    </div>
  )
}

function MessageBubble({
  message,
  onVote,
  onCopy,
  copiedId,
  onEdit,
  onNavigateVersion,
  isEditing,
  editContent,
  onEditContentChange,
  onSaveEdit,
  onCancelEdit,
  feedbackOpenFor,
  onFeedbackOpenChange,
  feedbackContent,
  onFeedbackContentChange,
  feedbackType,
  onFeedbackTypeChange,
  onSubmitFeedback,
}: {
  message: Message
  onVote: (id: string, vote: "up" | "down" | null) => void
  onCopy: (content: string, id: string) => void
  copiedId: string | null
  onEdit: (message: Message) => void
  onNavigateVersion: (message: Message, direction: "prev" | "next") => void
  isEditing: boolean
  editContent: string
  onEditContentChange: (content: string) => void
  onSaveEdit: () => void
  onCancelEdit: () => void
  feedbackOpenFor: { id: string; type: "up" | "down" } | null
  onFeedbackOpenChange: (open: boolean) => void
  feedbackContent: string
  onFeedbackContentChange: (content: string) => void
  feedbackType: Commentary["commentType"]
  onFeedbackTypeChange: (type: Commentary["commentType"]) => void
  onSubmitFeedback: (messageId: string, type: "up" | "down") => void
}) {
  const [showReasoning, setShowReasoning] = useState(false)

  const isUser = message.role === "user"
  const hasSiblings = message.siblings && message.siblings.length > 1
  const currentSiblingIndex = message.siblingIndex || 0
  const totalSiblings = message.siblings?.length || 1

  const isFeedbackOpen = feedbackOpenFor?.id === message.id

  return (
    <div className={cn("flex gap-3", isUser && "flex-row-reverse")}>
      {/* Avatar */}
      <div
        className={cn(
          "w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0",
          isUser ? "bg-blue-500" : "bg-emerald-600",
        )}
      >
        <span className="text-xs font-medium text-white">{isUser ? "U" : "A"}</span>
      </div>

      {/* Content */}
      <div className={cn("flex-1 max-w-[80%]", isUser && "flex flex-col items-end")}>
        {/* Reasoning Steps */}
        {message.reasoningSteps && message.reasoningSteps.length > 0 && (
          <div className="mb-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowReasoning(!showReasoning)}
              className="text-xs text-muted-foreground"
            >
              <ChevronDown className={cn("h-3 w-3 mr-1 transition-transform", showReasoning && "rotate-180")} />
              {showReasoning ? "Hide" : "Show"} reasoning ({message.reasoningSteps.length} steps)
            </Button>
            {showReasoning && (
              <div className="mt-2 p-3 rounded-lg bg-muted/50 border border-border text-sm space-y-2">
                {message.reasoningSteps.map((step, i) => (
                  <div key={i} className="flex gap-2">
                    <span className="text-muted-foreground">{i + 1}.</span>
                    <span className="text-muted-foreground">{step}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Tool Calls */}
        {message.toolCalls && message.toolCalls.length > 0 && (
          <div className="mb-2 space-y-2">
            {message.toolCalls.map((tool) => (
              <ToolCallBadge key={tool.id} tool={tool} />
            ))}
          </div>
        )}

        {/* Memory References */}
        {message.memories && message.memories.length > 0 && (
          <div className="mb-2 flex flex-wrap gap-2">
            {message.memories.map((memory) => (
              <MemoryBadge key={memory.id} memory={memory} />
            ))}
          </div>
        )}

        {/* Message Content */}
        <div
          className={cn(
            "rounded-2xl px-4 py-2.5",
            isUser ? "bg-blue-500 text-white rounded-tr-sm" : "bg-card border border-border rounded-tl-sm",
          )}
        >
          {isEditing ? (
            <div className="space-y-2">
              <Textarea
                value={editContent}
                onChange={(e) => onEditContentChange(e.target.value)}
                className={cn(
                  "min-h-[60px] resize-none text-sm",
                  isUser ? "bg-blue-600 border-blue-400 text-white placeholder:text-blue-200" : "bg-card",
                )}
                autoFocus
              />
              <div className="flex gap-2 justify-end">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={onCancelEdit}
                  className={isUser ? "text-white hover:bg-blue-600" : ""}
                >
                  <X className="h-3 w-3 mr-1" />
                  Cancel
                </Button>
                <Button size="sm" onClick={onSaveEdit} variant={isUser ? "secondary" : "default"}>
                  <Check className="h-3 w-3 mr-1" />
                  Save
                </Button>
              </div>
            </div>
          ) : message.isStreaming && !message.content ? (
            <div className="flex items-center gap-1">
              <div className="w-2 h-2 rounded-full bg-current animate-bounce" />
              <div className="w-2 h-2 rounded-full bg-current animate-bounce" style={{ animationDelay: "0.1s" }} />
              <div className="w-2 h-2 rounded-full bg-current animate-bounce" style={{ animationDelay: "0.2s" }} />
            </div>
          ) : (
            <p className="text-sm whitespace-pre-wrap">{message.content}</p>
          )}
        </div>

        {!isEditing && (
          <div className={cn("flex items-center gap-2 mt-1.5", isUser && "flex-row-reverse")}>
            {/* Timestamp */}
            <span className="text-xs text-muted-foreground">
              {message.timestamp.toLocaleTimeString([], {
                hour: "2-digit",
                minute: "2-digit",
              })}
            </span>

            {/* Branch navigation */}
            {hasSiblings && (
              <div className="flex items-center gap-0.5">
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-5 w-5"
                  onClick={() => onNavigateVersion(message, "prev")}
                  disabled={currentSiblingIndex === 0}
                >
                  <ChevronLeft className="h-3 w-3" />
                </Button>
                <span className="text-xs text-muted-foreground min-w-[32px] text-center">
                  {currentSiblingIndex + 1}/{totalSiblings}
                </span>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-5 w-5"
                  onClick={() => onNavigateVersion(message, "next")}
                  disabled={currentSiblingIndex === totalSiblings - 1}
                >
                  <ChevronRight className="h-3 w-3" />
                </Button>
              </div>
            )}

            {/* Edit button */}
            {message.content && (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="ghost" size="icon" className="h-5 w-5" onClick={() => onEdit(message)}>
                      <Pencil className="h-3 w-3" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>{isUser ? "Edit message" : "Suggest correction"}</TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}

            {/* Assistant-only actions */}
            {!isUser && message.content && (
              <TooltipProvider>
                <Popover open={isFeedbackOpen && feedbackOpenFor?.type === "up"} onOpenChange={onFeedbackOpenChange}>
                  <PopoverTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      className={cn("h-5 w-5", message.vote === "up" && "text-success bg-success/10")}
                      onClick={() => onVote(message.id, message.vote === "up" ? null : "up")}
                    >
                      <ThumbsUp className="h-3 w-3" />
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent
                    side="top"
                    align="start"
                    className="w-72 p-3"
                    onEscapeKeyDown={() => onFeedbackOpenChange(false)}
                    onPointerDownOutside={() => onFeedbackOpenChange(false)}
                  >
                    <div className="space-y-3">
                      <p className="text-sm font-medium">What did you like?</p>
                      <RadioGroup
                        value={feedbackType}
                        onValueChange={(v) => onFeedbackTypeChange(v as Commentary["commentType"])}
                        className="space-y-1"
                      >
                        <div className="flex items-center space-x-2">
                          <RadioGroupItem value="feedback-positive" id={`positive-${message.id}`} />
                          <Label htmlFor={`positive-${message.id}`} className="text-sm cursor-pointer">
                            Helpful response
                          </Label>
                        </div>
                        <div className="flex items-center space-x-2">
                          <RadioGroupItem value="note" id={`note-${message.id}`} />
                          <Label htmlFor={`note-${message.id}`} className="text-sm cursor-pointer">
                            Good explanation
                          </Label>
                        </div>
                      </RadioGroup>
                      <Textarea
                        value={feedbackContent}
                        onChange={(e) => onFeedbackContentChange(e.target.value)}
                        placeholder="Additional comments (optional)"
                        className="min-h-[60px] text-sm"
                      />
                      <div className="flex gap-2 justify-end">
                        <Button variant="ghost" size="sm" onClick={() => onFeedbackOpenChange(false)}>
                          Cancel
                        </Button>
                        <Button size="sm" onClick={() => onSubmitFeedback(message.id, "up")}>
                          Submit
                        </Button>
                      </div>
                    </div>
                  </PopoverContent>
                </Popover>

                <Popover open={isFeedbackOpen && feedbackOpenFor?.type === "down"} onOpenChange={onFeedbackOpenChange}>
                  <PopoverTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      className={cn("h-5 w-5", message.vote === "down" && "text-destructive bg-destructive/10")}
                      onClick={() => onVote(message.id, message.vote === "down" ? null : "down")}
                    >
                      <ThumbsDown className="h-3 w-3" />
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent
                    side="top"
                    align="start"
                    className="w-72 p-3"
                    onEscapeKeyDown={() => onFeedbackOpenChange(false)}
                    onPointerDownOutside={() => onFeedbackOpenChange(false)}
                  >
                    <div className="space-y-3">
                      <p className="text-sm font-medium">What went wrong?</p>
                      <RadioGroup
                        value={feedbackType}
                        onValueChange={(v) => onFeedbackTypeChange(v as Commentary["commentType"])}
                        className="space-y-1"
                      >
                        <div className="flex items-center space-x-2">
                          <RadioGroupItem value="feedback-negative" id={`negative-${message.id}`} />
                          <Label htmlFor={`negative-${message.id}`} className="text-sm cursor-pointer">
                            Unhelpful
                          </Label>
                        </div>
                        <div className="flex items-center space-x-2">
                          <RadioGroupItem value="correction" id={`correction-${message.id}`} />
                          <Label htmlFor={`correction-${message.id}`} className="text-sm cursor-pointer">
                            Incorrect info
                          </Label>
                        </div>
                      </RadioGroup>
                      <Textarea
                        value={feedbackContent}
                        onChange={(e) => onFeedbackContentChange(e.target.value)}
                        placeholder="What should have been different?"
                        className="min-h-[60px] text-sm"
                      />
                      <div className="flex gap-2 justify-end">
                        <Button variant="ghost" size="sm" onClick={() => onFeedbackOpenChange(false)}>
                          Cancel
                        </Button>
                        <Button size="sm" onClick={() => onSubmitFeedback(message.id, "down")}>
                          Submit
                        </Button>
                      </div>
                    </div>
                  </PopoverContent>
                </Popover>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-5 w-5"
                      onClick={() => onCopy(message.content, message.id)}
                    >
                      {copiedId === message.id ? (
                        <Check className="h-3 w-3 text-success" />
                      ) : (
                        <Copy className="h-3 w-3" />
                      )}
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Copy to clipboard</TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="ghost" size="icon" className="h-5 w-5">
                      <RefreshCw className="h-3 w-3" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Regenerate response</TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

function ToolCallBadge({ tool }: { tool: ToolCall }) {
  return (
    <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-secondary text-secondary-foreground text-xs">
      <Wrench className="h-3 w-3" />
      <span className="font-medium">{tool.name}</span>
      <span
        className={cn(
          "w-1.5 h-1.5 rounded-full",
          tool.status === "success" && "bg-success",
          tool.status === "pending" && "bg-warning animate-pulse",
          tool.status === "error" && "bg-destructive",
        )}
      />
    </div>
  )
}

function MemoryBadge({ memory }: { memory: MemoryReference }) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-accent/20 text-accent-foreground text-xs">
            <Brain className="h-3 w-3" />
            <span className="font-medium">{memory.type}</span>
            <span className="text-muted-foreground">{Math.round(memory.relevance * 100)}%</span>
          </div>
        </TooltipTrigger>
        <TooltipContent className="max-w-xs">
          <p>{memory.content}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
