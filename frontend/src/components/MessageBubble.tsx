import { Message } from '../types/models';
import { ToolUsage } from '../contexts/MessageContext';
import { ToolUsageDisplay } from './ToolUsageDisplay';

interface MessageBubbleProps {
  message: Message;
  toolUsages?: ToolUsage[];
  isLatestMessage?: boolean;
}

export function MessageBubble({ message, toolUsages = [], isLatestMessage = false }: MessageBubbleProps) {
  return (
    <div className={`message-bubble ${message.role}`}>
      <div className="message-role">
        {message.role === 'user' ? 'You' : 'Alicia'}
      </div>
      <div className="message-content">
        {message.contents.trim()}
      </div>
      {toolUsages.length > 0 && (
        <ToolUsageDisplay toolUsages={toolUsages} isLatestMessage={isLatestMessage} />
      )}
      <div className="message-time">
        {new Date(message.created_at).toLocaleTimeString()}
      </div>
    </div>
  );
}
