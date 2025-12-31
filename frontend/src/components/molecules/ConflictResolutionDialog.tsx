import React from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../atoms/Dialog';
import Button from '../atoms/Button';
import type { ConflictDetails } from '../../types/sync';

/**
 * ConflictResolutionDialog component for resolving offline sync conflicts.
 *
 * Displays both local and server versions of a message side-by-side,
 * allowing the user to choose which version to keep.
 */
export interface ConflictResolutionDialogProps {
  /** Whether the dialog is open */
  open: boolean;
  /** Callback when dialog open state changes */
  onOpenChange: (open: boolean) => void;
  /** Local version of the message content */
  localContent: string;
  /** Server version of the message content */
  serverContent: string;
  /** Conflict details including reason */
  conflict?: ConflictDetails;
  /** Callback when user chooses to keep local version */
  onKeepLocal: () => void;
  /** Callback when user chooses to keep server version */
  onKeepServer: () => void;
}

const ConflictResolutionDialog: React.FC<ConflictResolutionDialogProps> = ({
  open,
  onOpenChange,
  localContent,
  serverContent,
  conflict,
  onKeepLocal,
  onKeepServer,
}) => {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Sync Conflict Detected</DialogTitle>
          <DialogDescription>
            This message was modified both locally and on the server. Choose which version to keep.
            {conflict?.reason && (
              <span className="block mt-1 text-sm">
                Reason: {conflict.reason}
              </span>
            )}
          </DialogDescription>
        </DialogHeader>

        <div className="layout-stack-gap-4 py-4">
          {/* Local version */}
          <div className="layout-stack-gap">
            <div className="layout-center-gap">
              <span className="font-medium text-default">Your Version</span>
              {conflict?.localModifiedAt && (
                <span className="text-xs text-muted-foreground">
                  Modified: {new Date(conflict.localModifiedAt).toLocaleString()}
                </span>
              )}
            </div>
            <div className="rounded-lg border border-default bg-surface p-3 text-sm max-h-64 overflow-y-auto">
              <pre className="whitespace-pre-wrap font-sans">{localContent}</pre>
            </div>
          </div>

          {/* Server version */}
          <div className="layout-stack-gap">
            <div className="layout-center-gap">
              <span className="font-medium text-default">Server Version</span>
              {conflict?.serverModifiedAt && (
                <span className="text-xs text-muted-foreground">
                  Modified: {new Date(conflict.serverModifiedAt).toLocaleString()}
                </span>
              )}
            </div>
            <div className="rounded-lg border border-default bg-surface p-3 text-sm max-h-64 overflow-y-auto">
              <pre className="whitespace-pre-wrap font-sans">{serverContent}</pre>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onKeepServer}>
            Keep Server Version
          </Button>
          <Button onClick={onKeepLocal}>
            Keep Your Version
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default ConflictResolutionDialog;
