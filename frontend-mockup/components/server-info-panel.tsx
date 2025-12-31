"use client"

import { useState } from "react"
import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Server, Wifi, WifiOff, Activity, Cpu, Zap, Brain, Mic, Volume2, RefreshCw, Wrench } from "lucide-react"
import type { ConnectionStatus } from "./alicia-app"

type ServerInfoPanelProps = {
  connectionStatus: ConnectionStatus
}

export function ServerInfoPanel({ connectionStatus }: ServerInfoPanelProps) {
  const [serverInfo] = useState({
    version: "0.3.0",
    uptime: "3d 14h 22m",
    activeConversations: 1,
    totalConversations: 156,
    totalMessages: 2847,
    memoryUsed: 2.1,
    memoryTotal: 8.0,
    cpuUsage: 23,
    llm: {
      model: "Qwen3-8B-AWQ",
      provider: "vLLM",
      status: "online",
      avgLatency: 245,
    },
    asr: {
      model: "whisper-large-v3",
      provider: "speaches",
      status: "online",
      avgLatency: 180,
    },
    tts: {
      model: "kokoro",
      voice: "af_sky",
      provider: "speaches",
      status: "online",
      avgLatency: 120,
    },
    livekit: {
      url: "ws://localhost:7880",
      status: connectionStatus === "connected" ? "online" : "offline",
      rooms: 1,
    },
    tools: [
      { name: "web_search", enabled: true, usageCount: 89 },
      { name: "calculator", enabled: true, usageCount: 34 },
      { name: "memory_query", enabled: true, usageCount: 156 },
    ],
    embedding: {
      model: "text-embedding-3-small",
      dimensions: 1536,
      status: "online",
    },
  })

  const getStatusColor = (status: string) => {
    switch (status) {
      case "online":
        return "text-success"
      case "offline":
        return "text-destructive"
      case "degraded":
        return "text-warning"
      default:
        return "text-muted-foreground"
    }
  }

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "online":
        return <Badge className="bg-success/20 text-success border-success/30">Online</Badge>
      case "offline":
        return <Badge className="bg-destructive/20 text-destructive border-destructive/30">Offline</Badge>
      case "degraded":
        return <Badge className="bg-warning/20 text-warning border-warning/30">Degraded</Badge>
      default:
        return <Badge variant="secondary">Unknown</Badge>
    }
  }

  return (
    <div className="flex-1 flex flex-col bg-background min-h-0">
      {/* Header */}
      <header className="h-14 border-b border-border flex items-center justify-between px-4 shrink-0">
        <div className="flex items-center gap-3">
          <Server className="h-5 w-5 text-primary" />
          <h2 className="font-medium">Server Information</h2>
          {getStatusBadge(connectionStatus === "connected" ? "online" : "offline")}
        </div>
        <Button variant="outline" size="sm">
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </header>

      <div className="flex-1 overflow-y-auto p-4 min-h-0">
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {/* Connection Status */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm flex items-center gap-2">
                {connectionStatus === "connected" ? (
                  <Wifi className="h-4 w-4 text-success" />
                ) : (
                  <WifiOff className="h-4 w-4 text-destructive" />
                )}
                Connection
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Status</span>
                  <span className="capitalize font-medium">{connectionStatus}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">LiveKit</span>
                  <span className="font-mono text-xs">{serverInfo.livekit.url}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Active Rooms</span>
                  <span>{serverInfo.livekit.rooms}</span>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Server Stats */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm flex items-center gap-2">
                <Activity className="h-4 w-4 text-chart-1" />
                Server Stats
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Version</span>
                  <span className="font-mono">v{serverInfo.version}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Uptime</span>
                  <span>{serverInfo.uptime}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Total Conversations</span>
                  <span>{serverInfo.totalConversations}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Total Messages</span>
                  <span>{serverInfo.totalMessages}</span>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* System Resources */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm flex items-center gap-2">
                <Cpu className="h-4 w-4 text-chart-2" />
                Resources
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div>
                  <div className="flex justify-between text-sm mb-1">
                    <span className="text-muted-foreground">CPU</span>
                    <span>{serverInfo.cpuUsage}%</span>
                  </div>
                  <Progress value={serverInfo.cpuUsage} className="h-2" />
                </div>
                <div>
                  <div className="flex justify-between text-sm mb-1">
                    <span className="text-muted-foreground">Memory</span>
                    <span>
                      {serverInfo.memoryUsed}/{serverInfo.memoryTotal} GB
                    </span>
                  </div>
                  <Progress value={(serverInfo.memoryUsed / serverInfo.memoryTotal) * 100} className="h-2" />
                </div>
              </div>
            </CardContent>
          </Card>

          {/* LLM Service */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm flex items-center gap-2">
                <Brain className="h-4 w-4 text-chart-3" />
                LLM Service
              </CardTitle>
              <CardDescription className="text-xs">Language Model</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Model</span>
                  <span className="font-mono text-xs">{serverInfo.llm.model}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Provider</span>
                  <span>{serverInfo.llm.provider}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Status</span>
                  {getStatusBadge(serverInfo.llm.status)}
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Avg Latency</span>
                  <span>{serverInfo.llm.avgLatency}ms</span>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* ASR Service */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm flex items-center gap-2">
                <Mic className="h-4 w-4 text-chart-4" />
                ASR Service
              </CardTitle>
              <CardDescription className="text-xs">Speech Recognition</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Model</span>
                  <span className="font-mono text-xs">{serverInfo.asr.model}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Provider</span>
                  <span>{serverInfo.asr.provider}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Status</span>
                  {getStatusBadge(serverInfo.asr.status)}
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Avg Latency</span>
                  <span>{serverInfo.asr.avgLatency}ms</span>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* TTS Service */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm flex items-center gap-2">
                <Volume2 className="h-4 w-4 text-chart-5" />
                TTS Service
              </CardTitle>
              <CardDescription className="text-xs">Voice Synthesis</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Model</span>
                  <span className="font-mono text-xs">{serverInfo.tts.model}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Voice</span>
                  <span>{serverInfo.tts.voice}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Status</span>
                  {getStatusBadge(serverInfo.tts.status)}
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Avg Latency</span>
                  <span>{serverInfo.tts.avgLatency}ms</span>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Embedding Service */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm flex items-center gap-2">
                <Zap className="h-4 w-4 text-accent" />
                Embedding Service
              </CardTitle>
              <CardDescription className="text-xs">Vector Search</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Model</span>
                  <span className="font-mono text-xs">{serverInfo.embedding.model}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Dimensions</span>
                  <span>{serverInfo.embedding.dimensions}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Status</span>
                  {getStatusBadge(serverInfo.embedding.status)}
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Tools */}
          <Card className="md:col-span-2">
            <CardHeader className="pb-2">
              <CardTitle className="text-sm flex items-center gap-2">
                <Wrench className="h-4 w-4 text-primary" />
                Available Tools
              </CardTitle>
              <CardDescription className="text-xs">Integrated tool capabilities</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-3 sm:grid-cols-3">
                {serverInfo.tools.map((tool) => (
                  <div key={tool.name} className="flex items-center justify-between p-3 rounded-lg bg-secondary/50">
                    <div>
                      <span className="font-medium text-sm">{tool.name}</span>
                      <p className="text-xs text-muted-foreground">{tool.usageCount} uses</p>
                    </div>
                    <Badge
                      variant={tool.enabled ? "default" : "secondary"}
                      className={cn("text-xs", tool.enabled && "bg-success/20 text-success border-success/30")}
                    >
                      {tool.enabled ? "Enabled" : "Disabled"}
                    </Badge>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
