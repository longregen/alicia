import { useState } from 'react';
import { cls } from '../../../utils/cls';
import { OptimizationRunList } from './OptimizationRunList';
import { OptimizationRunDetails } from './OptimizationRunDetails';

export function OptimizationManager() {
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);

  return (
    <div className={cls('optimization-manager')}>
      {selectedRunId ? (
        <OptimizationRunDetails
          runId={selectedRunId}
          onBack={() => setSelectedRunId(null)}
        />
      ) : (
        <OptimizationRunList onSelectRun={setSelectedRunId} />
      )}
    </div>
  );
}
