"use client"

import type React from "react"

import { useState, useRef, useEffect } from "react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  MessageSquare,
  Plus,
  Search,
  Settings,
  Brain,
  Server,
  ChevronLeft,
  ChevronRight,
  Archive,
  MoreHorizontal,
  Trash2,
  Pencil,
  GripVertical,
  Check,
  X,
  ArchiveRestore,
} from "lucide-react"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import type { Conversation, ConnectionStatus } from "./alicia-app"

type SidebarProps = {
  open: boolean
  onToggle: () => void
  conversations: Conversation[]
  activeConversationId: string
  onSelectConversation: (id: string) => void
  onNewConversation: () => void
  connectionStatus: ConnectionStatus
  activePanel: "chat" | "memory" | "server" | "settings"
  onPanelChange: (panel: "chat" | "memory" | "server" | "settings") => void
  onRenameConversation?: (id: string, newTitle: string) => void
  onArchiveConversation?: (id: string) => void
  onUnarchiveConversation?: (id: string) => void
  onDeleteConversation?: (id: string) => void
}

export function Sidebar({
  open,
  onToggle,
  conversations,
  activeConversationId,
  onSelectConversation,
  onNewConversation,
  connectionStatus,
  activePanel,
  onPanelChange,
  onRenameConversation,
  onArchiveConversation,
  onUnarchiveConversation,
  onDeleteConversation,
}: SidebarProps) {
  const [searchQuery, setSearchQuery] = useState("")
  const [sidebarWidth, setSidebarWidth] = useState(288)
  const [isResizing, setIsResizing] = useState(false)
  const sidebarRef = useRef<HTMLDivElement>(null)
  const searchInputRef = useRef<HTMLInputElement>(null)

  const filteredConversations = conversations.filter((conv) =>
    conv.title.toLowerCase().includes(searchQuery.toLowerCase()),
  )

  const activeConvs = filteredConversations.filter((c) => c.status === "active")
  const archivedConvs = filteredConversations.filter((c) => c.status === "archived")

  const formatTime = (date: Date) => {
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const minutes = Math.floor(diff / 60000)
    const hours = Math.floor(diff / 3600000)
    const days = Math.floor(diff / 86400000)

    if (minutes < 60) return `${minutes}m ago`
    if (hours < 24) return `${hours}h ago`
    return `${days}d ago`
  }

  const handleMouseDown = (e: React.MouseEvent) => {
    e.preventDefault()
    setIsResizing(true)
  }

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault()
        if (searchInputRef.current) {
          searchInputRef.current.focus()
        }
      }
    }

    document.addEventListener("keydown", handleKeyDown)
    return () => document.removeEventListener("keydown", handleKeyDown)
  }, [])

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing) return
      const newWidth = Math.max(200, Math.min(480, e.clientX))
      setSidebarWidth(newWidth)
    }

    const handleMouseUp = () => {
      setIsResizing(false)
    }

    if (isResizing) {
      document.addEventListener("mousemove", handleMouseMove)
      document.addEventListener("mouseup", handleMouseUp)
      document.body.style.cursor = "col-resize"
      document.body.style.userSelect = "none"
    }

    return () => {
      document.removeEventListener("mousemove", handleMouseMove)
      document.removeEventListener("mouseup", handleMouseUp)
      document.body.style.cursor = ""
      document.body.style.userSelect = ""
    }
  }, [isResizing])

  return (
    <aside
      ref={sidebarRef}
      style={{ width: open ? sidebarWidth : 64 }}
      className={cn(
        "bg-sidebar border-r border-sidebar-border flex flex-col transition-all duration-300 relative",
        open && "min-w-[200px]",
        !open && "!w-16",
      )}
    >
      {/* Header */}
      <div className="p-4 border-b border-sidebar-border flex items-center justify-between shrink-0 min-w-0">
        {open && (
          <div className="flex items-center gap-2 min-w-0 flex-1">
            <div className="w-8 h-8 rounded-lg bg-emerald-600 flex items-center justify-center shrink-0">
              <span className="text-white font-bold text-sm">A</span>
            </div>
            <span className="font-semibold text-sidebar-foreground truncate">Alicia</span>
          </div>
        )}
        <Button
          variant="ghost"
          size="icon"
          onClick={onToggle}
          className={cn("text-sidebar-foreground hover:bg-sidebar-accent shrink-0", !open && "mx-auto")}
        >
          {open ? <ChevronLeft className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
        </Button>
      </div>

      {/* New Chat Button */}
      <div className="p-3 shrink-0">
        <Button
          onClick={onNewConversation}
          className={cn("w-full bg-emerald-600 hover:bg-emerald-700 text-white", !open && "px-0")}
        >
          <Plus className="h-4 w-4" />
          {open && <span className="ml-2">New Chat</span>}
        </Button>
      </div>

      {/* Search */}
      {open && (
        <div className="px-3 pb-3 shrink-0">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              ref={searchInputRef}
              placeholder="Search"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9 pr-14 bg-sidebar-accent border-sidebar-border text-sidebar-foreground placeholder:text-muted-foreground"
            />
            <kbd className="absolute right-3 top-1/2 -translate-y-1/2 px-1.5 py-0.5 bg-sidebar-border/50 rounded text-[10px] font-mono text-muted-foreground">
              ⌘K
            </kbd>
          </div>
        </div>
      )}

      {/* Conversation List */}
      <ScrollArea className="flex-1 px-3 min-h-0">
        {open ? (
          <div className="space-y-4">
            {/* Active Conversations */}
            <div className="min-w-0">
              <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">Active</h3>
              <div className="space-y-1">
                {activeConvs.map((conv) => (
                  <ConversationItem
                    key={conv.id}
                    conversation={conv}
                    isActive={conv.id === activeConversationId && activePanel === "chat"}
                    onClick={() => {
                      onSelectConversation(conv.id)
                      onPanelChange("chat")
                    }}
                    formatTime={formatTime}
                    onRename={onRenameConversation}
                    onArchive={onArchiveConversation}
                    onUnarchive={onUnarchiveConversation}
                    onDelete={onDeleteConversation}
                  />
                ))}
              </div>
            </div>

            {/* Archived */}
            {archivedConvs.length > 0 && (
              <div className="min-w-0">
                <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2 flex items-center gap-1">
                  <Archive className="h-3 w-3" />
                  Archived
                </h3>
                <div className="space-y-1">
                  {archivedConvs.map((conv) => (
                    <ConversationItem
                      key={conv.id}
                      conversation={conv}
                      isActive={conv.id === activeConversationId && activePanel === "chat"}
                      onClick={() => {
                        onSelectConversation(conv.id)
                        onPanelChange("chat")
                      }}
                      formatTime={formatTime}
                      onRename={onRenameConversation}
                      onArchive={onArchiveConversation}
                      onUnarchive={onUnarchiveConversation}
                      onDelete={onDeleteConversation}
                    />
                  ))}
                </div>
              </div>
            )}
          </div>
        ) : (
          <div className="space-y-1">
            {activeConvs.slice(0, 5).map((conv) => (
              <Button
                key={conv.id}
                variant={conv.id === activeConversationId && activePanel === "chat" ? "secondary" : "ghost"}
                size="icon"
                onClick={() => {
                  onSelectConversation(conv.id)
                  onPanelChange("chat")
                }}
                className="w-full"
                title={conv.title}
              >
                <MessageSquare className="h-4 w-4" />
              </Button>
            ))}
          </div>
        )}
      </ScrollArea>

      {/* Footer */}
      <div className="border-t border-sidebar-border shrink-0">
        {/* Navigation Items */}
        <div className={cn("p-2", !open && "flex flex-col items-center")}>
          <TooltipProvider delayDuration={0}>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant={activePanel === "memory" ? "secondary" : "ghost"}
                  size={open ? "default" : "icon"}
                  onClick={() => onPanelChange("memory")}
                  className={cn(
                    "w-full justify-start text-sidebar-foreground",
                    activePanel === "memory" && "bg-sidebar-accent",
                    !open && "justify-center",
                  )}
                >
                  <Brain className="h-4 w-4" />
                  {open && <span className="ml-2">Memory</span>}
                </Button>
              </TooltipTrigger>
              {!open && <TooltipContent side="right">Memory</TooltipContent>}
            </Tooltip>
          </TooltipProvider>

          <TooltipProvider delayDuration={0}>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant={activePanel === "server" ? "secondary" : "ghost"}
                  size={open ? "default" : "icon"}
                  onClick={() => onPanelChange("server")}
                  className={cn(
                    "w-full justify-start text-sidebar-foreground mt-1",
                    activePanel === "server" && "bg-sidebar-accent",
                    !open && "justify-center",
                  )}
                >
                  <Server className="h-4 w-4" />
                  {open && <span className="ml-2">Server</span>}
                </Button>
              </TooltipTrigger>
              {!open && <TooltipContent side="right">Server</TooltipContent>}
            </Tooltip>
          </TooltipProvider>

          <TooltipProvider delayDuration={0}>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant={activePanel === "settings" ? "secondary" : "ghost"}
                  size={open ? "default" : "icon"}
                  onClick={() => onPanelChange("settings")}
                  className={cn(
                    "w-full justify-start text-sidebar-foreground mt-1",
                    activePanel === "settings" && "bg-sidebar-accent",
                    !open && "justify-center",
                  )}
                >
                  <Settings className="h-4 w-4" />
                  {open && <span className="ml-2">Settings</span>}
                </Button>
              </TooltipTrigger>
              {!open && <TooltipContent side="right">Settings</TooltipContent>}
            </Tooltip>
          </TooltipProvider>
        </div>

        <div className={cn("p-3 border-t border-sidebar-border", !open && "flex justify-center")}>
          <div className="flex items-center gap-2">
            <div
              className={cn(
                "w-2 h-2 rounded-full",
                connectionStatus === "connected" && "bg-success",
                connectionStatus === "connecting" && "bg-warning animate-pulse",
                connectionStatus === "disconnected" && "bg-destructive",
                connectionStatus === "reconnecting" && "bg-warning animate-pulse",
              )}
            />
            {open && <span className="text-xs text-muted-foreground capitalize">{connectionStatus}</span>}
          </div>
        </div>
      </div>

      {open && (
        <div
          onMouseDown={handleMouseDown}
          className={cn(
            "absolute top-0 right-0 w-1 h-full cursor-col-resize hover:bg-primary/20 transition-colors group",
            isResizing && "bg-primary/30",
          )}
        >
          <div className="absolute top-1/2 right-0 -translate-y-1/2 opacity-0 group-hover:opacity-100 transition-opacity">
            <GripVertical className="h-4 w-4 text-muted-foreground" />
          </div>
        </div>
      )}
    </aside>
  )
}

function ConversationItem({
  conversation,
  isActive,
  onClick,
  formatTime,
  onRename,
  onArchive,
  onUnarchive,
  onDelete,
}: {
  conversation: Conversation
  isActive: boolean
  onClick: () => void
  formatTime: (date: Date) => string
  onRename?: (id: string, newTitle: string) => void
  onArchive?: (id: string) => void
  onUnarchive?: (id: string) => void
  onDelete?: (id: string) => void
}) {
  const [isRenaming, setIsRenaming] = useState(false)
  const [renameValue, setRenameValue] = useState(conversation.title)
  const inputRef = useRef<HTMLInputElement>(null)

  const handleRename = () => {
    if (renameValue.trim() && renameValue !== conversation.title) {
      onRename?.(conversation.id, renameValue.trim())
    }
    setIsRenaming(false)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleRename()
    } else if (e.key === "Escape") {
      setRenameValue(conversation.title)
      setIsRenaming(false)
    }
  }

  useEffect(() => {
    if (isRenaming && inputRef.current) {
      inputRef.current.focus()
      inputRef.current.select()
    }
  }, [isRenaming])

  return (
    <div
      className={cn(
        "group relative flex items-start gap-3 p-2 pr-8 rounded-lg cursor-pointer transition-colors min-w-0",
        isActive ? "bg-sidebar-accent" : "hover:bg-sidebar-accent/50",
      )}
      onClick={isRenaming ? undefined : onClick}
    >
      <div className="w-8 h-8 rounded-lg bg-secondary flex items-center justify-center flex-shrink-0">
        <MessageSquare className="h-4 w-4 text-muted-foreground" />
      </div>
      <div className="flex-1 min-w-0 overflow-hidden">
        <div className="flex items-center gap-1">
          {isRenaming ? (
            <div className="flex items-center gap-1 flex-1">
              <Input
                ref={inputRef}
                value={renameValue}
                onChange={(e) => setRenameValue(e.target.value)}
                onKeyDown={handleKeyDown}
                onBlur={handleRename}
                className="h-6 text-sm py-0 px-1"
                onClick={(e) => e.stopPropagation()}
              />
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={(e) => {
                  e.stopPropagation()
                  handleRename()
                }}
              >
                <Check className="h-3 w-3" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={(e) => {
                  e.stopPropagation()
                  setRenameValue(conversation.title)
                  setIsRenaming(false)
                }}
              >
                <X className="h-3 w-3" />
              </Button>
            </div>
          ) : (
            <h4 className="text-sm font-medium text-sidebar-foreground truncate">{conversation.title}</h4>
          )}
        </div>
        <p className="text-xs text-muted-foreground truncate">{conversation.lastMessage || "No messages yet"}</p>
        <span className="text-xs text-muted-foreground">{formatTime(conversation.updatedAt)}</span>
      </div>
      {!isRenaming && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="absolute right-1 top-2 h-6 w-6 opacity-0 group-hover:opacity-100 focus:opacity-100 transition-opacity"
              onClick={(e) => e.stopPropagation()}
            >
              <MoreHorizontal className="h-3 w-3" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent side="right" sideOffset={8} align="start" className="w-44">
            <DropdownMenuItem
              onClick={(e) => {
                e.stopPropagation()
                setIsRenaming(true)
              }}
            >
              <Pencil className="h-3 w-3 mr-2" />
              Rename
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            {conversation.status === "active" ? (
              <DropdownMenuItem
                onClick={(e) => {
                  e.stopPropagation()
                  onArchive?.(conversation.id)
                }}
              >
                <Archive className="h-3 w-3 mr-2" />
                Archive conversation
              </DropdownMenuItem>
            ) : (
              <DropdownMenuItem
                onClick={(e) => {
                  e.stopPropagation()
                  onUnarchive?.(conversation.id)
                }}
              >
                <ArchiveRestore className="h-3 w-3 mr-2" />
                Unarchive conversation
              </DropdownMenuItem>
            )}
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={(e) => {
                e.stopPropagation()
                onDelete?.(conversation.id)
              }}
              className="text-destructive focus:text-destructive"
            >
              <Trash2 className="h-3 w-3 mr-2" />
              Delete permanently
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  )
}
