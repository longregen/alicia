"use client"

import { useEffect, useState } from "react"
import { cn } from "@/lib/utils"
import type { VoiceState } from "./alicia-app"

type VoiceVisualizerProps = {
  state: VoiceState
}

export function VoiceVisualizer({ state }: VoiceVisualizerProps) {
  const [bars, setBars] = useState<number[]>(new Array(20).fill(8))

  useEffect(() => {
    if (state === "listening" || state === "speaking") {
      const interval = setInterval(() => {
        setBars(
          Array.from({ length: 20 }, () => (state === "listening" ? Math.random() * 32 + 8 : Math.random() * 24 + 4)),
        )
      }, 100)
      return () => clearInterval(interval)
    } else {
      setBars(new Array(20).fill(8))
    }
  }, [state])

  const getStateLabel = () => {
    switch (state) {
      case "listening":
        return "Listening..."
      case "processing":
        return "Processing..."
      case "speaking":
        return "Speaking..."
      default:
        return ""
    }
  }

  const getStateColor = () => {
    switch (state) {
      case "listening":
        return "bg-voice-listening"
      case "processing":
        return "bg-voice-processing"
      case "speaking":
        return "bg-voice-active"
      default:
        return "bg-muted"
    }
  }

  return (
    <div className="flex flex-col items-center gap-4">
      {/* Central orb with pulse effect */}
      <div className="relative">
        {state !== "idle" && (
          <>
            <div
              className={cn("absolute inset-0 rounded-full animate-pulse-ring", getStateColor(), "opacity-30")}
              style={{ width: 80, height: 80, left: -20, top: -20 }}
            />
            <div
              className={cn("absolute inset-0 rounded-full animate-pulse-ring", getStateColor(), "opacity-20")}
              style={{
                width: 100,
                height: 100,
                left: -30,
                top: -30,
                animationDelay: "0.5s",
              }}
            />
          </>
        )}
        <div
          className={cn(
            "w-10 h-10 rounded-full flex items-center justify-center transition-all",
            getStateColor(),
            state === "processing" && "animate-pulse",
          )}
        >
          {state === "processing" ? (
            <div className="w-5 h-5 border-2 border-background border-t-transparent rounded-full animate-spin" />
          ) : (
            <span className="text-background text-xs font-bold">A</span>
          )}
        </div>
      </div>

      {/* Waveform visualization */}
      <div className="flex items-center justify-center gap-1 h-10">
        {bars.map((height, i) => (
          <div
            key={i}
            className={cn("w-1 rounded-full transition-all duration-100", getStateColor())}
            style={{ height: `${height}px` }}
          />
        ))}
      </div>

      {/* State label */}
      <span className="text-sm text-muted-foreground">{getStateLabel()}</span>
    </div>
  )
}
