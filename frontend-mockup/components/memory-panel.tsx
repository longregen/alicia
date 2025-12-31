"use client"

import { useState } from "react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Search, Plus, Edit2, Trash2, Pin, Archive, Tag, Brain, Star, MoreHorizontal } from "lucide-react"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Card, CardContent, CardHeader } from "@/components/ui/card"

type Memory = {
  id: string
  content: string
  type: "preference" | "fact" | "instruction" | "context"
  tags: string[]
  importance: number
  pinned: boolean
  archived: boolean
  createdAt: Date
  usageCount: number
}

export function MemoryPanel() {
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedType, setSelectedType] = useState<string | null>(null)
  const [memories, setMemories] = useState<Memory[]>([
    {
      id: "mem_1",
      content: "User prefers Italian cuisine and vegetarian options",
      type: "preference",
      tags: ["food", "dietary"],
      importance: 0.9,
      pinned: true,
      archived: false,
      createdAt: new Date(Date.now() - 1000 * 60 * 60 * 24 * 7),
      usageCount: 12,
    },
    {
      id: "mem_2",
      content: "User lives in New York City, Manhattan area",
      type: "fact",
      tags: ["location", "personal"],
      importance: 0.95,
      pinned: true,
      archived: false,
      createdAt: new Date(Date.now() - 1000 * 60 * 60 * 24 * 14),
      usageCount: 28,
    },
    {
      id: "mem_3",
      content: "Always provide code examples when explaining technical concepts",
      type: "instruction",
      tags: ["style", "technical"],
      importance: 0.8,
      pinned: false,
      archived: false,
      createdAt: new Date(Date.now() - 1000 * 60 * 60 * 24 * 3),
      usageCount: 8,
    },
    {
      id: "mem_4",
      content: "User is working on a restaurant recommendation app project",
      type: "context",
      tags: ["project", "work"],
      importance: 0.7,
      pinned: false,
      archived: false,
      createdAt: new Date(Date.now() - 1000 * 60 * 60 * 24),
      usageCount: 5,
    },
    {
      id: "mem_5",
      content: "User prefers metric system for measurements",
      type: "preference",
      tags: ["settings"],
      importance: 0.6,
      pinned: false,
      archived: true,
      createdAt: new Date(Date.now() - 1000 * 60 * 60 * 24 * 30),
      usageCount: 3,
    },
  ])

  const types = ["preference", "fact", "instruction", "context"]
  const allTags = [...new Set(memories.flatMap((m) => m.tags))]

  const filteredMemories = memories
    .filter((m) => !m.archived)
    .filter((m) => {
      if (searchQuery) {
        return (
          m.content.toLowerCase().includes(searchQuery.toLowerCase()) ||
          m.tags.some((t) => t.toLowerCase().includes(searchQuery.toLowerCase()))
        )
      }
      return true
    })
    .filter((m) => {
      if (selectedType) return m.type === selectedType
      return true
    })
    .sort((a, b) => {
      if (a.pinned !== b.pinned) return a.pinned ? -1 : 1
      return b.importance - a.importance
    })

  const getTypeColor = (type: string) => {
    switch (type) {
      case "preference":
        return "bg-chart-1/20 text-chart-1 border-chart-1/30"
      case "fact":
        return "bg-chart-2/20 text-chart-2 border-chart-2/30"
      case "instruction":
        return "bg-chart-3/20 text-chart-3 border-chart-3/30"
      case "context":
        return "bg-chart-4/20 text-chart-4 border-chart-4/30"
      default:
        return "bg-muted text-muted-foreground"
    }
  }

  const togglePin = (id: string) => {
    setMemories((prev) => prev.map((m) => (m.id === id ? { ...m, pinned: !m.pinned } : m)))
  }

  const archiveMemory = (id: string) => {
    setMemories((prev) => prev.map((m) => (m.id === id ? { ...m, archived: true } : m)))
  }

  return (
    <div className="flex-1 flex flex-col bg-background min-h-0">
      {/* Header */}
      <header className="h-14 border-b border-border flex items-center justify-between px-4 shrink-0">
        <div className="flex items-center gap-3">
          <Brain className="h-5 w-5 text-accent" />
          <h2 className="font-medium">Memory Management</h2>
          <Badge variant="secondary">{filteredMemories.length} memories</Badge>
        </div>
        <Button size="sm">
          <Plus className="h-4 w-4 mr-2" />
          Add Memory
        </Button>
      </header>

      {/* Search & Filters */}
      <div className="p-4 border-b border-border space-y-3 shrink-0">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search memories..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>

        <div className="flex gap-2 flex-wrap">
          {types.map((type) => (
            <Button
              key={type}
              variant={selectedType === type ? "default" : "outline"}
              size="sm"
              onClick={() => setSelectedType(selectedType === type ? null : type)}
              className="capitalize"
            >
              {type}
            </Button>
          ))}
        </div>
      </div>

      {/* Memory List */}
      <div className="flex-1 overflow-y-auto p-4 min-h-0">
        <div className="grid gap-4 md:grid-cols-2">
          {filteredMemories.map((memory) => (
            <Card
              key={memory.id}
              className={cn("group relative transition-all", memory.pinned && "ring-1 ring-primary/50")}
            >
              <CardHeader className="pb-2">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className={cn("text-xs", getTypeColor(memory.type))}>
                      {memory.type}
                    </Badge>
                    {memory.pinned && <Pin className="h-3 w-3 text-primary fill-primary" />}
                  </div>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7 opacity-0 group-hover:opacity-100 transition-opacity"
                      >
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={() => togglePin(memory.id)}>
                        <Pin className="h-3 w-3 mr-2" />
                        {memory.pinned ? "Unpin" : "Pin"}
                      </DropdownMenuItem>
                      <DropdownMenuItem>
                        <Edit2 className="h-3 w-3 mr-2" />
                        Edit
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem onClick={() => archiveMemory(memory.id)}>
                        <Archive className="h-3 w-3 mr-2" />
                        Archive
                      </DropdownMenuItem>
                      <DropdownMenuItem className="text-destructive">
                        <Trash2 className="h-3 w-3 mr-2" />
                        Delete
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </CardHeader>
              <CardContent>
                <p className="text-sm mb-3">{memory.content}</p>

                <div className="flex flex-wrap gap-1 mb-3">
                  {memory.tags.map((tag) => (
                    <Badge key={tag} variant="secondary" className="text-xs">
                      <Tag className="h-2.5 w-2.5 mr-1" />
                      {tag}
                    </Badge>
                  ))}
                </div>

                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <div className="flex items-center gap-1">
                    <Star className="h-3 w-3" />
                    <span>{Math.round(memory.importance * 100)}%</span>
                  </div>
                  <span>Used {memory.usageCount} times</span>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    </div>
  )
}
