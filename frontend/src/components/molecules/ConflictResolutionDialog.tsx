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

export interface ConflictResolutionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  localContent: string;
  serverContent: string;
  conflict?: ConflictDetails;
  onKeepLocal: () => void;
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
            <div className="rounded-lg border border-default bg-surface p-3 text-sm">
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
            <div className="rounded-lg border border-default bg-surface p-3 text-sm">
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
