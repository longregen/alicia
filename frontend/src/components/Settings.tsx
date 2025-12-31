import { useState, useCallback } from 'react';
import { MCPSettings } from './MCPSettings';
import ServerInfoPanel from './organisms/ServerPanel/ServerInfoPanel';
import { MemoryManager } from './organisms/MemoryManager';
import UserNotesPanel from './organisms/NotesPanel/UserNotesPanel';
import { PivotModeSelector } from './molecules/PivotModeSelector/PivotModeSelector';
import { EliteSolutionSelector } from './molecules/EliteSolutionSelector/EliteSolutionSelector';
import { PIVOT_PRESETS, type PresetId } from '../stores/dimensionStore';
import { sendDimensionPreference, sendEliteSelect } from '../adapters/protocolAdapter';
import type { DimensionWeights } from '../types/protocol';
import { Card, CardHeader, CardTitle, CardContent } from './atoms/Card';
import { Switch } from './atoms/Switch';
import { Slider } from './atoms/Slider';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from './atoms/Select';
import { Kbd, KbdGroup } from './atoms/Kbd';
import Button from './atoms/Button';
import { Label } from './atoms/Label';
import { cn } from '../lib/utils';

interface SettingsProps {
  isOpen: boolean;
  onClose: () => void;
  conversationId?: string | null;
}

type SettingsTab = 'mcp' | 'server' | 'memories' | 'notes' | 'optimization' | 'preferences';

export function Settings({ isOpen, onClose, conversationId }: SettingsProps) {
  const [activeTab, setActiveTab] = useState<SettingsTab>('mcp');
  const [audioOutputEnabled, setAudioOutputEnabled] = useState(false);
  const [voiceSpeed, setVoiceSpeed] = useState(1.0);
  const [theme, setTheme] = useState('system');
  const [responseLength, setResponseLength] = useState<'concise' | 'balanced' | 'detailed'>('balanced');

  // Handle dimension preference changes - send to server via WebSocket
  const handlePresetChange = useCallback((presetId: PresetId) => {
    if (!conversationId) return;
    const preset = PIVOT_PRESETS.find(p => p.id === presetId);
    if (!preset) return;

    sendDimensionPreference({
      conversationId,
      weights: preset.weights,
      preset: presetId,
      timestamp: Date.now(),
    });
  }, [conversationId]);

  const handleWeightsChange = useCallback((weights: DimensionWeights) => {
    if (!conversationId) return;

    sendDimensionPreference({
      conversationId,
      weights,
      timestamp: Date.now(),
    });
  }, [conversationId]);

  // Handle elite selection - send to server via WebSocket
  const handleSelectElite = useCallback((eliteId: string) => {
    if (!conversationId) return;

    sendEliteSelect({
      conversationId,
      eliteId,
      timestamp: Date.now(),
    });
  }, [conversationId]);

  if (!isOpen) return null;

  return (
    <div className="flex flex-col h-full bg-app">
      {/* Header */}
      <div className="flex justify-between items-center p-6 md:px-8 border-b border-default bg-surface">
        <h1 className="m-0 text-3xl md:text-[28px] font-semibold text-default">Settings & Info</h1>
        <button className="btn-ghost text-3xl w-10 h-10" onClick={onClose} title="Close settings">
          ×
        </button>
      </div>

      {/* Tab navigation - vertical on mobile, horizontal on desktop */}
      <div className="bg-surface border-b border-default overflow-x-auto">
        <div className="flex flex-col md:flex-row md:px-8">
          <button
            className={`tab whitespace-nowrap ${activeTab === 'mcp' ? 'active' : ''}`}
            onClick={() => setActiveTab('mcp')}
          >
            MCP Settings
          </button>
          <button
            className={`tab whitespace-nowrap ${activeTab === 'server' ? 'active' : ''}`}
            onClick={() => setActiveTab('server')}
          >
            Server Info
          </button>
          <button
            className={`tab whitespace-nowrap ${activeTab === 'memories' ? 'active' : ''}`}
            onClick={() => setActiveTab('memories')}
          >
            Memories
          </button>
          <button
            className={`tab whitespace-nowrap ${activeTab === 'notes' ? 'active' : ''}`}
            onClick={() => setActiveTab('notes')}
            disabled={!conversationId}
          >
            Notes
          </button>
          <button
            className={`tab whitespace-nowrap ${activeTab === 'optimization' ? 'active' : ''}`}
            onClick={() => setActiveTab('optimization')}
          >
            Optimization
          </button>
          <button
            className={`tab whitespace-nowrap ${activeTab === 'preferences' ? 'active' : ''}`}
            onClick={() => setActiveTab('preferences')}
          >
            Preferences
          </button>
        </div>
      </div>

      {/* Content area */}
      <div className="flex-1 overflow-y-auto p-4 md:p-8">
        <div className="mb-8 last:mb-0">
          {activeTab === 'mcp' && <MCPSettings />}
          {activeTab === 'server' && <ServerInfoPanel />}
          {activeTab === 'memories' && <MemoryManager />}
          {activeTab === 'notes' && conversationId && (
            <UserNotesPanel
              targetType="message"
              targetId={conversationId}
            />
          )}
          {activeTab === 'notes' && !conversationId && (
            <div className="text-center p-10 text-muted">
              Please select a conversation to view notes
            </div>
          )}
          {activeTab === 'optimization' && (
            <div>
              <PivotModeSelector
                onPresetChange={handlePresetChange}
                onWeightsChange={handleWeightsChange}
                disabled={!conversationId}
              />
              <div className="mt-6">
                <EliteSolutionSelector
                  onSelectElite={handleSelectElite}
                  disabled={!conversationId}
                />
              </div>
              {!conversationId && (
                <div className="text-center p-5 text-muted mt-4">
                  Select a conversation to sync optimization preferences with the server
                </div>
              )}
            </div>
          )}
          {activeTab === 'preferences' && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* Voice & Audio Card */}
              <Card>
                <CardHeader>
                  <CardTitle>Voice & Audio</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex items-center justify-between">
                    <Label htmlFor="audio-output">Audio Output</Label>
                    <Switch
                      id="audio-output"
                      checked={audioOutputEnabled}
                      onCheckedChange={setAudioOutputEnabled}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="voice-speed">Voice Speed: {voiceSpeed.toFixed(1)}x</Label>
                    <Slider
                      id="voice-speed"
                      min={0.5}
                      max={2.0}
                      step={0.1}
                      value={[voiceSpeed]}
                      onValueChange={(values) => setVoiceSpeed(values[0])}
                    />
                  </div>
                </CardContent>
              </Card>

              {/* Response Length Card */}
              <Card>
                <CardHeader>
                  <CardTitle>Response Length</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="flex justify-between text-xs text-muted">
                    <span>Concise</span>
                    <span>Balanced</span>
                    <span>Detailed</span>
                  </div>
                  <div className="flex gap-2">
                    {(['concise', 'balanced', 'detailed'] as const).map((length) => (
                      <button
                        key={length}
                        onClick={() => setResponseLength(length)}
                        className={cn(
                          'flex-1 py-2 px-3 rounded-lg text-sm font-medium transition-colors',
                          responseLength === length
                            ? 'bg-accent text-accent-foreground'
                            : 'bg-muted text-muted-foreground hover:bg-muted/80'
                        )}
                      >
                        {length.charAt(0).toUpperCase() + length.slice(1)}
                      </button>
                    ))}
                  </div>
                  <p className="text-xs text-muted">
                    {responseLength === 'concise' && 'Short, direct answers without elaboration.'}
                    {responseLength === 'balanced' && 'Clear explanations with relevant details.'}
                    {responseLength === 'detailed' && 'Comprehensive responses with examples and context.'}
                  </p>
                </CardContent>
              </Card>

              {/* Appearance Card */}
              <Card>
                <CardHeader>
                  <CardTitle>Appearance</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <Label htmlFor="theme-select">Theme</Label>
                    <Select value={theme} onValueChange={setTheme}>
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
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-muted">Toggle sidebar</span>
                    <KbdGroup>
                      <Kbd>⌘</Kbd>
                      <Kbd>B</Kbd>
                    </KbdGroup>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-muted">Search conversations</span>
                    <KbdGroup>
                      <Kbd>⌘</Kbd>
                      <Kbd>K</Kbd>
                    </KbdGroup>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-muted">Send message</span>
                    <KbdGroup>
                      <Kbd>⌘</Kbd>
                      <Kbd>Enter</Kbd>
                    </KbdGroup>
                  </div>
                </CardContent>
              </Card>

              {/* Privacy & Data Card */}
              <Card>
                <CardHeader>
                  <CardTitle>Privacy & Data</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <Button
                    variant="outline"
                    className="w-full"
                    onClick={() => console.log('Export data clicked')}
                  >
                    Export Data
                  </Button>
                  <Button
                    variant="destructive"
                    className="w-full"
                    onClick={() => console.log('Clear all data clicked')}
                  >
                    Clear All Data
                  </Button>
                </CardContent>
              </Card>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
