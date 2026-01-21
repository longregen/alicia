import { useState } from 'react';
import { useLocation } from 'wouter';
import { MCPSettings } from './MCPSettings';
import { Card, CardHeader, CardTitle, CardContent } from './atoms/Card';
import { Switch } from './atoms/Switch';
import { Slider } from './atoms/Slider';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from './atoms/Select';
import { Kbd, KbdGroup } from './atoms/Kbd';
import Button from './atoms/Button';
import { Label } from './atoms/Label';
import Tooltip from './atoms/Tooltip';
import { cls } from '../utils/cls';
import { useTheme } from '../hooks/useTheme';

interface SettingsProps {
  defaultTab?: SettingsTab;
}

export type SettingsTab = 'mcp' | 'preferences';

// Default preference values
const DEFAULT_PREFERENCES = {
  audioOutputEnabled: false,
  voiceSpeed: 1.0,
  theme: 'system' as const,
  responseLength: 'balanced' as const,
};

export function Settings({ defaultTab = 'mcp' }: SettingsProps) {
  const [, navigate] = useLocation();
  // Use defaultTab from URL (passed as prop from router)
  const activeTab = defaultTab;
  const [audioOutputEnabled, setAudioOutputEnabled] = useState(DEFAULT_PREFERENCES.audioOutputEnabled);

  const setActiveTab = (tab: SettingsTab) => {
    navigate(`/settings/${tab}`);
  };

  const [voiceSpeed, setVoiceSpeed] = useState(DEFAULT_PREFERENCES.voiceSpeed);
  const { theme, setTheme } = useTheme();
  const [responseLength, setResponseLength] = useState<'concise' | 'balanced' | 'detailed'>(DEFAULT_PREFERENCES.responseLength);

  return (
    <div className="layout-stack h-full bg-background">
      {/* Header */}
      <div className="p-6 md:px-8 border-b border-border bg-card">
        <h1 className="m-0 text-3xl md:text-[28px] font-semibold text-foreground">Settings</h1>
      </div>

      {/* Tab navigation - vertical on mobile, horizontal on desktop */}
      <div className="bg-card border-b border-border overflow-x-auto">
        <div className="flex flex-col md:flex-row md:gap-1 md:px-8">
          <button
            className={`tab whitespace-nowrap ${activeTab === 'mcp' ? 'tab-active' : ''}`}
            onClick={() => setActiveTab('mcp')}
          >
            MCP
          </button>
          <button
            className={`tab whitespace-nowrap ${activeTab === 'preferences' ? 'tab-active' : ''}`}
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
          {activeTab === 'preferences' && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
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
                        className={cls(
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

              {/* Memory Settings Card */}
              {/* Note: These settings are UI placeholders - state management to be implemented */}
              <Card>
                <CardHeader>
                  <CardTitle>Memory</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="layout-between">
                    <Label htmlFor="auto-pin">Auto-pin important memories</Label>
                    <Switch id="auto-pin" defaultChecked />
                  </div>
                  <div className="layout-between">
                    <Label htmlFor="confirm-delete">Confirm before deleting</Label>
                    <Switch id="confirm-delete" defaultChecked />
                  </div>
                  <div className="layout-between">
                    <Label htmlFor="show-relevance">Show relevance scores</Label>
                    <Switch id="show-relevance" defaultChecked />
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
                  <div className="layout-between">
                    <Label htmlFor="compact-mode">Compact mode</Label>
                    <Switch id="compact-mode" />
                  </div>
                  <div className="layout-between">
                    <Label htmlFor="reduce-motion">Reduce motion</Label>
                    <Switch id="reduce-motion" />
                  </div>
                </CardContent>
              </Card>

              {/* Notifications Card */}
              <Card>
                <CardHeader>
                  <CardTitle>Notifications</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="layout-between">
                    <Label htmlFor="sound-notifications">Sound notifications</Label>
                    <Switch id="sound-notifications" defaultChecked />
                  </div>
                  <div className="layout-between">
                    <Label htmlFor="desktop-notifications">Desktop notifications</Label>
                    <Switch id="desktop-notifications" />
                  </div>
                  <div className="layout-between">
                    <Label htmlFor="message-preview">Message previews</Label>
                    <Switch id="message-preview" defaultChecked />
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

              {/* Privacy & Data Card */}
              <Card>
                <CardHeader>
                  <CardTitle>Privacy & Data</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  {/* TODO: Implement data export/clear functionality */}
                  <Tooltip content="Coming soon">
                    <Button
                      variant="outline"
                      className="w-full"
                      disabled={true}
                    >
                      Export Data
                    </Button>
                  </Tooltip>
                  <Tooltip content="Coming soon">
                    <Button
                      variant="destructive"
                      className="w-full"
                      disabled={true}
                    >
                      Clear All Data
                    </Button>
                  </Tooltip>
                </CardContent>
              </Card>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
