import { MCPSettings } from './MCPSettings';
import './Settings.css';

interface SettingsProps {
  isOpen: boolean;
  onClose: () => void;
}

export function Settings({ isOpen, onClose }: SettingsProps) {
  if (!isOpen) return null;

  return (
    <div className="settings-modal-overlay" onClick={onClose}>
      <div className="settings-modal" onClick={(e) => e.stopPropagation()}>
        <div className="settings-header">
          <h1>Settings</h1>
          <button className="settings-close-btn" onClick={onClose} title="Close settings">
            ×
          </button>
        </div>

        <div className="settings-content">
          <div className="settings-section">
            <MCPSettings />
          </div>
        </div>
      </div>
    </div>
  );
}
