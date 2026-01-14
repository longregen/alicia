import React from 'react';
import Button from '../atoms/Button';
import { Conversation } from '../../types/models';
import { cls } from '../../utils/cls';

export interface WelcomeScreenProps {
  conversations: Conversation[];
  onNewConversation: () => void;
  onSelectConversation: (id: string) => void;
  loading?: boolean;
}

function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays === 1) return 'Yesterday';
  if (diffDays < 7) return `${diffDays}d ago`;
  return date.toLocaleDateString();
}

const WelcomeScreen: React.FC<WelcomeScreenProps> = ({
  conversations,
  onNewConversation,
  onSelectConversation,
  loading = false,
}) => {
  // Show up to 5 most recent active conversations
  const recentConversations = conversations
    .filter(c => c.status === 'active')
    .sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())
    .slice(0, 5);

  return (
    <div className="flex-1 flex flex-col items-center justify-center p-8 bg-background">
      <div className="max-w-md w-full text-center space-y-8">
        {/* Logo/Icon */}
        <div className="flex justify-center">
          <div className="w-16 h-16 rounded-2xl bg-primary/10 flex items-center justify-center">
            <svg
              className="w-8 h-8 text-primary"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
              />
            </svg>
          </div>
        </div>

        {/* Welcome message */}
        <div className="space-y-2">
          <h1 className="text-2xl font-semibold text-foreground">Welcome to Alicia</h1>
          <p className="text-muted-foreground">Your AI-powered assistant</p>
        </div>

        {/* New chat button */}
        <Button
          size="lg"
          onClick={onNewConversation}
          loading={loading}
          className="w-full"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Start New Chat
        </Button>

        {/* Recent conversations */}
        {recentConversations.length > 0 && (
          <div className="space-y-3">
            <h2 className="text-sm font-medium text-muted-foreground">Recent Conversations</h2>
            <div className="space-y-1">
              {recentConversations.map((conv) => (
                <button
                  key={conv.id}
                  onClick={() => onSelectConversation(conv.id)}
                  className={cls(
                    'w-full text-left px-4 py-3 rounded-lg',
                    'bg-secondary/50 hover:bg-secondary',
                    'transition-colors cursor-pointer',
                    'group'
                  )}
                >
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-foreground truncate flex-1">
                      {conv.title || 'Untitled'}
                    </span>
                    <span className="text-xs text-muted-foreground ml-2 shrink-0">
                      {formatRelativeTime(conv.updated_at)}
                    </span>
                  </div>
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Tip */}
        <p className="text-xs text-muted-foreground">
          Tip: Use the sidebar to access your conversations
        </p>
      </div>
    </div>
  );
};

export default WelcomeScreen;
