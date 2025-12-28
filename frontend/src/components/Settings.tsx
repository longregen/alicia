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
import './Settings.css';

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
    <div className="settings-modal-overlay" onClick={onClose}>
      <div className="settings-modal" onClick={(e) => e.stopPropagation()}>
        <div className="settings-header">
          <h1>Settings & Info</h1>
          <button className="settings-close-btn" onClick={onClose} title="Close settings">
            ×
          </button>
        </div>

        {/* Tab navigation */}
        <div className="settings-tabs">
          <button
            className={`settings-tab ${activeTab === 'mcp' ? 'active' : ''}`}
            onClick={() => setActiveTab('mcp')}
          >
            MCP Settings
          </button>
          <button
            className={`settings-tab ${activeTab === 'server' ? 'active' : ''}`}
            onClick={() => setActiveTab('server')}
          >
            Server Info
          </button>
          <button
            className={`settings-tab ${activeTab === 'memories' ? 'active' : ''}`}
            onClick={() => setActiveTab('memories')}
          >
            Memories
          </button>
          <button
            className={`settings-tab ${activeTab === 'notes' ? 'active' : ''}`}
            onClick={() => setActiveTab('notes')}
            disabled={!conversationId}
          >
            Notes
          </button>
          <button
            className={`settings-tab ${activeTab === 'optimization' ? 'active' : ''}`}
            onClick={() => setActiveTab('optimization')}
          >
            Optimization
          </button>
        </div>

        <div className="settings-content">
          <div className="settings-section">
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
              <div style={{ textAlign: 'center', padding: '40px', color: '#888' }}>
                Please select a conversation to view notes
              </div>
            )}
            {activeTab === 'optimization' && (
              <div className="optimization-settings">
                <PivotModeSelector
                  onPresetChange={handlePresetChange}
                  onWeightsChange={handleWeightsChange}
                  disabled={!conversationId}
                />
                <div style={{ marginTop: '24px' }}>
                  <EliteSolutionSelector
                    onSelectElite={handleSelectElite}
                    disabled={!conversationId}
                  />
                </div>
                {!conversationId && (
                  <div style={{ textAlign: 'center', padding: '20px', color: '#888', marginTop: '16px' }}>
                    Select a conversation to sync optimization preferences with the server
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
