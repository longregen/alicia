import { useState, useRef, useEffect } from 'react';
import { useLocation } from 'wouter';
import { QRCodeSVG } from 'qrcode.react';
import { MCPSettings } from './MCPSettings';
import { Card, CardHeader, CardTitle, CardContent } from './atoms/Card';
import { Switch } from './atoms/Switch';
import { Slider } from './atoms/Slider';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from './atoms/Select';
import { Kbd, KbdGroup } from './atoms/Kbd';
import Button from './atoms/Button';
import { Input } from './atoms/Input';
import { Label } from './atoms/Label';
import StarRating from './atoms/StarRating';
import { usePreferences } from '../hooks/usePreferences';
import { useSidebarStore } from '../stores/sidebarStore';
import { getCustomUserId, setUserId } from '../utils/deviceId';
import { useWhatsAppStore, WhatsAppConnectionStatus, WhatsAppRole, WhatsAppEvent } from '../stores/whatsappStore';
import { useWebSocket } from '../contexts/WebSocketContext';

interface SettingsProps {
  defaultTab?: SettingsTab;
}

export type SettingsTab = 'mcp' | 'preferences' | 'whatsapp';

export function Settings({ defaultTab = 'mcp' }: SettingsProps) {
  const [, navigate] = useLocation();
  const openSidebar = useSidebarStore((state) => state.setOpen);

  const setActiveTab = (tab: SettingsTab) => {
    navigate(`/settings/${tab}`);
  };

  const {
    theme,
    audio_output_enabled,
    voice_speed,
    memory_min_importance,
    memory_min_historical,
    memory_min_personal,
    memory_min_factual,
    memory_retrieval_count,
    max_tokens,
    pareto_target_score,
    pareto_max_generations,
    pareto_branches_per_gen,
    pareto_archive_size,
    pareto_enable_crossover,
    confirm_delete_memory,
    show_relevance_scores,
    updatePreference,
  } = usePreferences();

  const [userIdInput, setUserIdInput] = useState(getCustomUserId() || '');

  const handleSaveUserId = () => {
    setUserId(userIdInput.trim() || null);
    window.location.reload();
  };

  return (
    <div className="layout-stack h-full bg-background">
      {/* Header */}
      <div className="p-6 md:px-8 border-b border-border bg-card flex items-center gap-3">
        <button
          onClick={() => openSidebar(true)}
          className="lg:hidden p-2 -ml-2 hover:bg-elevated rounded-md transition-colors"
          aria-label="Open sidebar"
        >
          <svg className="w-6 h-6 text-default" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
          </svg>
        </button>
        <h1 className="m-0 text-3xl md:text-[28px] font-semibold text-foreground">Settings</h1>
      </div>

      {/* Tab navigation */}
      <div className="bg-card border-b border-border overflow-x-auto">
        <div className="flex flex-col md:flex-row md:gap-1 md:px-8">
          <button
            className={`tab whitespace-nowrap ${defaultTab === 'mcp' ? 'tab-active' : ''}`}
            onClick={() => setActiveTab('mcp')}
          >
            MCP
          </button>
          <button
            className={`tab whitespace-nowrap ${defaultTab === 'preferences' ? 'tab-active' : ''}`}
            onClick={() => setActiveTab('preferences')}
          >
            Preferences
          </button>
          <button
            className={`tab whitespace-nowrap ${defaultTab === 'whatsapp' ? 'tab-active' : ''}`}
            onClick={() => setActiveTab('whatsapp')}
          >
            WhatsApp
          </button>
        </div>
      </div>

      {/* Content area */}
      <div className="flex-1 overflow-y-auto p-4 md:p-8">
        <div className="mb-8 last:mb-0">
          {defaultTab === 'mcp' && <MCPSettings />}
          {defaultTab === 'whatsapp' && <WhatsAppSettings />}
          {defaultTab === 'preferences' && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* Account Card - Cross-device sync */}
              <Card>
                <CardHeader>
                  <CardTitle>Account</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="user-id">User ID</Label>
                    <p className="text-xs text-muted">
                      Set a custom user ID to sync conversations across devices.
                      Use the same ID on all your devices.
                    </p>
                    <Input
                      id="user-id"
                      type="text"
                      value={userIdInput}
                      onChange={(e) => setUserIdInput(e.target.value)}
                      placeholder="Enter user ID (e.g., your email)"
                    />
                  </div>
                  <Button onClick={handleSaveUserId} className="w-full">
                    Save & Reload
                  </Button>
                </CardContent>
              </Card>

              {/* Voice & Audio Card */}
              <Card>
                <CardHeader>
                  <CardTitle>Voice & Audio</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="layout-between">
                    <Label htmlFor="audio-output">Audio Output</Label>
                    <Switch
                      id="audio-output"
                      checked={audio_output_enabled}
                      onCheckedChange={(v) => updatePreference('audio_output_enabled', v)}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="voice-speed">Voice Speed: {voice_speed.toFixed(1)}x</Label>
                    <Slider
                      id="voice-speed"
                      min={0.5}
                      max={2.0}
                      step={0.1}
                      value={[voice_speed]}
                      onValueChange={(values) => updatePreference('voice_speed', values[0])}
                    />
                  </div>
                </CardContent>
              </Card>

              {/* Memory Thresholds Card */}
              <Card>
                <CardHeader>
                  <CardTitle>Memory Thresholds</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <p className="text-xs text-muted -mt-2">
                    Minimum rating required for the agent to create memories of each type.
                  </p>
                  <div className="space-y-3">
                    <div className="layout-between">
                      <Label>Importance</Label>
                      <StarRating
                        rating={memory_min_importance}
                        onRate={(v) => updatePreference('memory_min_importance', v === 0 ? null : v)}
                        compact
                      />
                    </div>
                    <div className="layout-between">
                      <Label>Historical</Label>
                      <StarRating
                        rating={memory_min_historical}
                        onRate={(v) => updatePreference('memory_min_historical', v === 0 ? null : v)}
                        compact
                      />
                    </div>
                    <div className="layout-between">
                      <Label>Personal</Label>
                      <StarRating
                        rating={memory_min_personal}
                        onRate={(v) => updatePreference('memory_min_personal', v === 0 ? null : v)}
                        compact
                      />
                    </div>
                    <div className="layout-between">
                      <Label>Factual</Label>
                      <StarRating
                        rating={memory_min_factual}
                        onRate={(v) => updatePreference('memory_min_factual', v === 0 ? null : v)}
                        compact
                      />
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Agent Settings Card */}
              <Card>
                <CardHeader>
                  <CardTitle>Agent</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="memory-count">Memories to Retrieve</Label>
                    <p className="text-xs text-muted">
                      Number of relevant memories to include in each response.
                    </p>
                    <Input
                      id="memory-count"
                      type="number"
                      min={1}
                      max={50}
                      value={memory_retrieval_count}
                      onChange={(e) => updatePreference('memory_retrieval_count', Math.max(1, Math.min(50, parseInt(e.target.value) || 1)))}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="max-tokens">Max Response Tokens</Label>
                    <p className="text-xs text-muted">
                      Maximum length of agent responses.
                    </p>
                    <Input
                      id="max-tokens"
                      type="number"
                      min={256}
                      max={16384}
                      step={256}
                      value={max_tokens}
                      onChange={(e) => updatePreference('max_tokens', Math.max(256, Math.min(16384, parseInt(e.target.value) || 256)))}
                    />
                  </div>
                </CardContent>
              </Card>

              {/* Pareto Exploration Card */}
              <Card>
                <CardHeader>
                  <CardTitle>Pareto Exploration</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <p className="text-xs text-muted -mt-2">
                    Multi-objective optimization for response quality.
                  </p>
                  <div className="space-y-2">
                    <Label htmlFor="pareto-target">Target Score: {pareto_target_score.toFixed(1)}</Label>
                    <Slider
                      id="pareto-target"
                      min={0.5}
                      max={5.0}
                      step={0.1}
                      value={[pareto_target_score]}
                      onValueChange={(values) => updatePreference('pareto_target_score', values[0])}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="pareto-generations">Max Generations</Label>
                    <Input
                      id="pareto-generations"
                      type="number"
                      min={1}
                      max={20}
                      value={pareto_max_generations}
                      onChange={(e) => updatePreference('pareto_max_generations', Math.max(1, Math.min(20, parseInt(e.target.value) || 1)))}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="pareto-branches">Branches Per Generation</Label>
                    <Input
                      id="pareto-branches"
                      type="number"
                      min={1}
                      max={10}
                      value={pareto_branches_per_gen}
                      onChange={(e) => updatePreference('pareto_branches_per_gen', Math.max(1, Math.min(10, parseInt(e.target.value) || 1)))}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="pareto-archive">Archive Size</Label>
                    <Input
                      id="pareto-archive"
                      type="number"
                      min={10}
                      max={200}
                      value={pareto_archive_size}
                      onChange={(e) => updatePreference('pareto_archive_size', Math.max(10, Math.min(200, parseInt(e.target.value) || 10)))}
                    />
                  </div>
                  <div className="layout-between">
                    <Label htmlFor="pareto-crossover">Enable Crossover</Label>
                    <Switch
                      id="pareto-crossover"
                      checked={pareto_enable_crossover}
                      onCheckedChange={(v) => updatePreference('pareto_enable_crossover', v)}
                    />
                  </div>
                </CardContent>
              </Card>

              {/* Memory UI Card */}
              <Card>
                <CardHeader>
                  <CardTitle>Memory UI</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="layout-between">
                    <Label htmlFor="confirm-delete">Confirm before deleting</Label>
                    <Switch
                      id="confirm-delete"
                      checked={confirm_delete_memory}
                      onCheckedChange={(v) => updatePreference('confirm_delete_memory', v)}
                    />
                  </div>
                  <div className="layout-between">
                    <Label htmlFor="show-relevance">Show relevance scores</Label>
                    <Switch
                      id="show-relevance"
                      checked={show_relevance_scores}
                      onCheckedChange={(v) => updatePreference('show_relevance_scores', v)}
                    />
                  </div>
                </CardContent>
              </Card>

              {/* Appearance Card */}
              <Card>
                <CardHeader>
                  <CardTitle>Appearance</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="theme-select">Theme</Label>
                    <Select value={theme} onValueChange={(v) => updatePreference('theme', v as 'light' | 'dark' | 'system')}>
                      <SelectTrigger id="theme-select">
                        <SelectValue placeholder="Select theme" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="light">Light</SelectItem>
                        <SelectItem value="dark">Dark</SelectItem>
                        <SelectItem value="system">System</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </CardContent>
              </Card>

              {/* Keyboard Shortcuts Card */}
              <Card>
                <CardHeader>
                  <CardTitle>Keyboard Shortcuts</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="layout-between">
                    <span className="text-sm text-muted">Toggle sidebar</span>
                    <KbdGroup>
                      <Kbd>⌘</Kbd>
                      <Kbd>B</Kbd>
                    </KbdGroup>
                  </div>
                  <div className="layout-between">
                    <span className="text-sm text-muted">Search conversations</span>
                    <KbdGroup>
                      <Kbd>⌘</Kbd>
                      <Kbd>K</Kbd>
                    </KbdGroup>
                  </div>
                  <div className="layout-between">
                    <span className="text-sm text-muted">Send message</span>
                    <KbdGroup>
                      <Kbd>⌘</Kbd>
                      <Kbd>Enter</Kbd>
                    </KbdGroup>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function WhatsAppConnectionCard({ role, title, description, showDebug }: { role: WhatsAppRole; title: string; description: string; showDebug: boolean }) {
  const { sendWhatsAppPairRequest, isConnected } = useWebSocket();
  const status = useWhatsAppStore((s) => s[role].status);
  const qrCode = useWhatsAppStore((s) => s[role].qrCode);
  const phone = useWhatsAppStore((s) => s[role].phone);
  const error = useWhatsAppStore((s) => s[role].error);

  const isWaConnected = status === WhatsAppConnectionStatus.Connected;
  const isPairing = status === WhatsAppConnectionStatus.Pairing;

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted">{description}</p>

        <div className="layout-between">
          <Label>Status</Label>
          <span className={`text-sm font-medium ${isWaConnected ? 'text-green-600 dark:text-green-400' : 'text-muted'}`}>
            {isWaConnected ? 'Connected' : isPairing ? 'Pairing...' : status === WhatsAppConnectionStatus.Error ? 'Error' : 'Disconnected'}
          </span>
        </div>

        {isWaConnected && phone && (
          <div className="layout-between">
            <Label>Phone</Label>
            <span className="text-sm text-muted">+{phone}</span>
          </div>
        )}

        {error && (
          <p className="text-sm text-red-500">{error}</p>
        )}

        {qrCode && (
          <div className="flex flex-col items-center gap-3 py-2">
            <p className="text-sm text-muted">Scan this QR code with WhatsApp on your phone.</p>
            <div className="bg-white p-4 rounded-lg">
              <QRCodeSVG value={qrCode} size={256} />
            </div>
          </div>
        )}

        {!isWaConnected && !isPairing && (
          <Button
            onClick={() => sendWhatsAppPairRequest(role)}
            disabled={!isConnected}
            className="w-full"
          >
            Start Pairing
          </Button>
        )}

        {isPairing && !qrCode && (
          <p className="text-sm text-muted text-center">Waiting for QR code...</p>
        )}

        {showDebug && (
          <div className="mt-3 pt-3 border-t border-border">
            <p className="text-xs font-mono text-muted mb-1">Raw state:</p>
            <pre className="text-xs font-mono bg-elevated p-2 rounded overflow-x-auto">
{JSON.stringify({ status, qrCode: qrCode ? qrCode.slice(0, 30) + '...' : null, phone, error }, null, 2)}
            </pre>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function WhatsAppEventLog({ events }: { events: WhatsAppEvent[] }) {
  const clearEvents = useWhatsAppStore((s) => s.clearEvents);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ block: 'nearest' });
  }, [events.length]);

  if (events.length === 0) {
    return <p className="text-xs text-muted italic">No events yet. Pair a device to see activity.</p>;
  }

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between mb-1">
        <span className="text-xs text-muted">{events.length} events</span>
        <button onClick={clearEvents} className="text-xs text-muted hover:text-foreground transition-colors">
          Clear
        </button>
      </div>
      <div className="max-h-48 overflow-y-auto font-mono text-xs bg-elevated rounded p-2 space-y-0.5">
        {events.map((evt, i) => (
          <div key={i} className="flex gap-2">
            <span className="text-muted shrink-0">{evt.time}</span>
            <span className={`shrink-0 ${evt.role === 'reader' ? 'text-blue-500' : 'text-purple-500'}`}>
              [{evt.role}]
            </span>
            <span className="text-yellow-600 dark:text-yellow-400 shrink-0">{evt.type}</span>
            <span className="text-foreground truncate">{evt.detail}</span>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>
    </div>
  );
}

function WhatsAppSettings() {
  const [showDebug, setShowDebug] = useState(false);
  const { isConnected } = useWebSocket();
  const events = useWhatsAppStore((s) => s.events);

  return (
    <div className="max-w-lg mx-auto space-y-4">
      <WhatsAppConnectionCard
        role="reader"
        title="Your WhatsApp (Reader)"
        description="Archives all your messages. Never sends replies."
        showDebug={showDebug}
      />

      <WhatsAppConnectionCard
        role="alicia"
        title="Alicia's WhatsApp"
        description="Uses a separate WhatsApp number. Allowlisted contacts can chat with Alicia."
        showDebug={showDebug}
      />

      <Card>
        <CardHeader>
          <CardTitle>How it works</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <p className="text-sm text-muted">
            Two WhatsApp connections work together:
          </p>
          <ul className="text-sm text-muted list-disc list-inside space-y-1">
            <li><strong>Reader</strong> links to your personal WhatsApp and passively archives all messages for search and context.</li>
            <li><strong>Alicia</strong> links to a separate WhatsApp number. Contacts on the allowlist can message this number to chat with Alicia.</li>
          </ul>
          <p className="text-sm text-muted mt-2">
            Each connection is paired independently:
          </p>
          <ol className="text-sm text-muted list-decimal list-inside space-y-1">
            <li>Click "Start Pairing" on the card you want to connect</li>
            <li>Open WhatsApp on the corresponding phone</li>
            <li>Go to Settings &gt; Linked Devices &gt; Link a Device</li>
            <li>Scan the QR code shown here</li>
          </ol>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Debug</CardTitle>
            <Switch checked={showDebug} onCheckedChange={setShowDebug} />
          </div>
        </CardHeader>
        {showDebug && (
          <CardContent className="space-y-3">
            <div className="layout-between">
              <Label>Hub WebSocket</Label>
              <span className={`text-sm font-medium ${isConnected ? 'text-green-600 dark:text-green-400' : 'text-red-500'}`}>
                {isConnected ? 'Connected' : 'Disconnected'}
              </span>
            </div>
            <div>
              <Label className="mb-1 block">Event Log</Label>
              <WhatsAppEventLog events={events} />
            </div>
          </CardContent>
        )}
      </Card>
    </div>
  );
}
