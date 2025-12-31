"use client"

import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Slider } from "@/components/ui/slider"
import { Settings, Volume2, Mic, Brain, Palette, Bell, Shield, Keyboard, RotateCcw } from "lucide-react"

export function SettingsPanel() {
  const [settings, setSettings] = useState({
    voiceInput: true,
    audioOutput: true,
    autoSaveMemory: true,
    useMemories: true,
    compactMode: false,
    showTimestamps: true,
    theme: "dark",
    speechRate: 1.0,
    autoPlayAudio: false,
    notifications: true,
    soundEffects: false,
  })

  const updateSetting = (key: string, value: boolean | string | number) => {
    setSettings((prev) => ({ ...prev, [key]: value }))
  }

  return (
    <div className="flex-1 flex flex-col bg-background min-h-0">
      {/* Header - matching MemoryPanel style */}
      <header className="h-14 border-b border-border flex items-center justify-between px-4 shrink-0">
        <div className="flex items-center gap-3">
          <Settings className="h-5 w-5 text-accent" />
          <h2 className="font-medium">Settings</h2>
          <Badge variant="secondary">6 categories</Badge>
        </div>
        <Button size="sm" variant="outline">
          <RotateCcw className="h-4 w-4 mr-2" />
          Reset All
        </Button>
      </header>

      {/* Scrollable Content - full width grid like MemoryPanel */}
      <div className="flex-1 overflow-y-auto p-4 min-h-0">
        <div className="grid gap-4 md:grid-cols-2">
          {/* Voice & Audio */}
          <Card className="group">
            <CardHeader className="pb-2">
              <div className="flex items-center gap-2">
                <Volume2 className="h-4 w-4 text-chart-1" />
                <span className="font-medium">Voice & Audio</span>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <Label htmlFor="voice-input" className="flex items-center gap-2 text-sm">
                  <Mic className="h-3.5 w-3.5 text-muted-foreground" />
                  Voice input
                </Label>
                <Switch
                  id="voice-input"
                  checked={settings.voiceInput}
                  onCheckedChange={(v) => updateSetting("voiceInput", v)}
                />
              </div>
              <div className="flex items-center justify-between">
                <Label htmlFor="audio-output" className="text-sm">
                  Audio responses
                </Label>
                <Switch
                  id="audio-output"
                  checked={settings.audioOutput}
                  onCheckedChange={(v) => updateSetting("audioOutput", v)}
                />
              </div>
              <div className="flex items-center justify-between">
                <Label htmlFor="auto-play" className="text-sm">
                  Auto-play audio
                </Label>
                <Switch
                  id="auto-play"
                  checked={settings.autoPlayAudio}
                  onCheckedChange={(v) => updateSetting("autoPlayAudio", v)}
                />
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label className="text-sm">Speech rate</Label>
                  <span className="text-xs text-muted-foreground">{settings.speechRate.toFixed(1)}x</span>
                </div>
                <Slider
                  value={[settings.speechRate]}
                  onValueChange={([v]) => updateSetting("speechRate", v)}
                  min={0.5}
                  max={2.0}
                  step={0.1}
                  className="w-full"
                />
              </div>
            </CardContent>
          </Card>

          {/* Memory */}
          <Card className="group">
            <CardHeader className="pb-2">
              <div className="flex items-center gap-2">
                <Brain className="h-4 w-4 text-chart-2" />
                <span className="font-medium">Memory</span>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <Label htmlFor="auto-memory" className="text-sm">
                  Auto-save memories
                </Label>
                <Switch
                  id="auto-memory"
                  checked={settings.autoSaveMemory}
                  onCheckedChange={(v) => updateSetting("autoSaveMemory", v)}
                />
              </div>
              <div className="flex items-center justify-between">
                <Label htmlFor="use-memories" className="text-sm">
                  Use in responses
                </Label>
                <Switch
                  id="use-memories"
                  checked={settings.useMemories}
                  onCheckedChange={(v) => updateSetting("useMemories", v)}
                />
              </div>
            </CardContent>
          </Card>

          {/* Appearance */}
          <Card className="group">
            <CardHeader className="pb-2">
              <div className="flex items-center gap-2">
                <Palette className="h-4 w-4 text-chart-3" />
                <span className="font-medium">Appearance</span>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <Label htmlFor="theme" className="text-sm">
                  Theme
                </Label>
                <Select value={settings.theme} onValueChange={(v) => updateSetting("theme", v)}>
                  <SelectTrigger className="w-28 h-8">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="light">Light</SelectItem>
                    <SelectItem value="dark">Dark</SelectItem>
                    <SelectItem value="system">System</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="flex items-center justify-between">
                <Label htmlFor="compact-mode" className="text-sm">
                  Compact mode
                </Label>
                <Switch
                  id="compact-mode"
                  checked={settings.compactMode}
                  onCheckedChange={(v) => updateSetting("compactMode", v)}
                />
              </div>
              <div className="flex items-center justify-between">
                <Label htmlFor="show-timestamps" className="text-sm">
                  Show timestamps
                </Label>
                <Switch
                  id="show-timestamps"
                  checked={settings.showTimestamps}
                  onCheckedChange={(v) => updateSetting("showTimestamps", v)}
                />
              </div>
            </CardContent>
          </Card>

          {/* Notifications */}
          <Card className="group">
            <CardHeader className="pb-2">
              <div className="flex items-center gap-2">
                <Bell className="h-4 w-4 text-chart-4" />
                <span className="font-medium">Notifications</span>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between">
                <Label htmlFor="notifications" className="text-sm">
                  Enable notifications
                </Label>
                <Switch
                  id="notifications"
                  checked={settings.notifications}
                  onCheckedChange={(v) => updateSetting("notifications", v)}
                />
              </div>
              <div className="flex items-center justify-between">
                <Label htmlFor="sound-effects" className="text-sm">
                  Sound effects
                </Label>
                <Switch
                  id="sound-effects"
                  checked={settings.soundEffects}
                  onCheckedChange={(v) => updateSetting("soundEffects", v)}
                />
              </div>
            </CardContent>
          </Card>

          {/* Keyboard Shortcuts */}
          <Card className="group">
            <CardHeader className="pb-2">
              <div className="flex items-center gap-2">
                <Keyboard className="h-4 w-4 text-chart-5" />
                <span className="font-medium">Keyboard Shortcuts</span>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-2 text-sm">
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Search</span>
                  <kbd className="px-1.5 py-0.5 bg-secondary rounded text-xs font-mono">⌘K</kbd>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">New chat</span>
                  <kbd className="px-1.5 py-0.5 bg-secondary rounded text-xs font-mono">⌘N</kbd>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Toggle sidebar</span>
                  <kbd className="px-1.5 py-0.5 bg-secondary rounded text-xs font-mono">⌘B</kbd>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Voice input</span>
                  <kbd className="px-1.5 py-0.5 bg-secondary rounded text-xs font-mono">Space</kbd>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Privacy & Data */}
          <Card className="group">
            <CardHeader className="pb-2">
              <div className="flex items-center gap-2">
                <Shield className="h-4 w-4 text-accent" />
                <span className="font-medium">Privacy & Data</span>
              </div>
            </CardHeader>
            <CardContent className="space-y-2">
              <Button variant="outline" size="sm" className="w-full justify-start bg-transparent">
                Export all data
              </Button>
              <Button
                variant="outline"
                size="sm"
                className="w-full justify-start text-destructive hover:text-destructive bg-transparent"
              >
                Clear all memories
              </Button>
              <Button
                variant="outline"
                size="sm"
                className="w-full justify-start text-destructive hover:text-destructive bg-transparent"
              >
                Delete all conversations
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
