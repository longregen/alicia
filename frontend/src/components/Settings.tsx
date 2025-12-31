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

interface SettingsProps {
  isOpen: boolean;
  onClose: () => void;
  conversationId?: string | null;
}

type SettingsTab = 'mcp' | 'server' | 'memories' | 'notes' | 'optimization';

export function Settings({ isOpen, onClose, conversationId }: SettingsProps) {
  const [activeTab, setActiveTab] = useState<SettingsTab>('mcp');

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
        </div>
      </div>
    </div>
  );
}
