import React from 'react';
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogFooter,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogAction,
  AlertDialogCancel,
} from '../atoms/AlertDialog';
import { Card, CardContent, CardHeader, CardTitle } from '../atoms/Card';
import Button from '../atoms/Button';
import type { ConflictDetails } from '../../types/sync';

/**
 * ConflictResolutionDialog component for resolving offline sync conflicts.
 *
 * Displays both local and server versions of a message side-by-side,
 * allowing the user to choose which version to keep.
 *
 * @example
 * ```tsx
 * const [conflictOpen, setConflictOpen] = useState(false);
 *
 * // In ChatBubble or message component:
 * <ConflictResolutionDialog
 *   open={conflictOpen}
 *   onOpenChange={setConflictOpen}
 *   localContent="Local message content"
 *   serverContent="Server message content"
 *   conflict={{
 *     reason: "Content mismatch with existing message",
 *     resolution: "manual"
 *   }}
 *   onKeepLocal={() => {
 *     // Call API to resolve conflict with local version
 *     syncApi.resolveConflict(messageId, 'local');
 *   }}
 *   onKeepServer={() => {
 *     // Call API to resolve conflict with server version
 *     syncApi.resolveConflict(messageId, 'server');
 *   }}
 * />
 * ```
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
  const handleKeepLocal = () => {
    onKeepLocal();
    onOpenChange(false);
  };

  const handleKeepServer = () => {
    onKeepServer();
    onOpenChange(false);
  };

  const handleCancel = () => {
    onOpenChange(false);
  };

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent className="max-w-4xl">
        <AlertDialogHeader>
          <AlertDialogTitle>Message Conflict</AlertDialogTitle>
          <AlertDialogDescription>
            This message has a conflict with the server version.
            {conflict?.reason && (
              <span className="block mt-1 text-sm">
                Reason: {conflict.reason}
              </span>
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 my-4">
          {/* Local Version */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Local Version</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-sm whitespace-pre-wrap max-h-64 overflow-y-auto">
                {localContent}
              </div>
            </CardContent>
          </Card>

          {/* Server Version */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Server Version</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-sm whitespace-pre-wrap max-h-64 overflow-y-auto">
                {serverContent}
              </div>
            </CardContent>
          </Card>
        </div>

        <AlertDialogFooter>
          <AlertDialogCancel onClick={handleCancel}>
            Cancel
          </AlertDialogCancel>
          <Button
            variant="outline"
            onClick={handleKeepServer}
          >
            Keep Server
          </Button>
          <AlertDialogAction onClick={handleKeepLocal}>
            Keep Local
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
};

export default ConflictResolutionDialog;
