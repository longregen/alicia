import { describe, it, expect, beforeEach } from 'vitest';
import { useBranchStore } from './branchStore';
import { createMessageId } from '../types/streaming';

describe('BranchStore', () => {
  beforeEach(() => {
    // Reset the store state between tests
    useBranchStore.setState({
      branches: new Map(),
      currentVersionIndex: new Map(),
    });
  });

  describe('initializeBranch', () => {
    it('creates initial branch for a message', () => {
      const messageId = createMessageId('msg-1');
      const content = 'Hello world';

      useBranchStore.getState().initializeBranch(messageId, content);

      const branch = useBranchStore.getState().getCurrentBranch(messageId);
      expect(branch).toBeTruthy();
      expect(branch?.content).toBe(content);
    });

    it('does not overwrite existing branches', () => {
      const messageId = createMessageId('msg-1');
      const originalContent = 'Original content';
      const newContent = 'New content';

      useBranchStore.getState().initializeBranch(messageId, originalContent);
      useBranchStore.getState().initializeBranch(messageId, newContent);

      const branch = useBranchStore.getState().getCurrentBranch(messageId);
      expect(branch?.content).toBe(originalContent);
    });

    it('sets initial branch count to 1', () => {
      const messageId = createMessageId('msg-1');
      useBranchStore.getState().initializeBranch(messageId, 'content');

      const count = useBranchStore.getState().getBranchCount(messageId);
      expect(count).toBe(1);
    });

    it('sets initial index to 0', () => {
      const messageId = createMessageId('msg-1');
      useBranchStore.getState().initializeBranch(messageId, 'content');

      const index = useBranchStore.getState().getCurrentIndex(messageId);
      expect(index).toBe(0);
    });
  });

  describe('createBranch', () => {
    it('creates a new branch for a message', () => {
      const messageId = createMessageId('msg-1');
      const firstContent = 'First version';
      const secondContent = 'Second version';

      useBranchStore.getState().initializeBranch(messageId, firstContent);
      useBranchStore.getState().createBranch(messageId, secondContent);

      const count = useBranchStore.getState().getBranchCount(messageId);
      expect(count).toBe(2);
    });

    it('sets current index to the new branch', () => {
      const messageId = createMessageId('msg-1');
      useBranchStore.getState().initializeBranch(messageId, 'First');
      useBranchStore.getState().createBranch(messageId, 'Second');

      const index = useBranchStore.getState().getCurrentIndex(messageId);
      expect(index).toBe(1);

      const branch = useBranchStore.getState().getCurrentBranch(messageId);
      expect(branch?.content).toBe('Second');
    });

    it('creates multiple branches correctly', () => {
      const messageId = createMessageId('msg-1');
      useBranchStore.getState().initializeBranch(messageId, 'V1');
      useBranchStore.getState().createBranch(messageId, 'V2');
      useBranchStore.getState().createBranch(messageId, 'V3');

      const count = useBranchStore.getState().getBranchCount(messageId);
      expect(count).toBe(3);

      const branch = useBranchStore.getState().getCurrentBranch(messageId);
      expect(branch?.content).toBe('V3');
    });
  });

  describe('navigateToBranch', () => {
    beforeEach(() => {
      const messageId = createMessageId('msg-1');
      useBranchStore.getState().initializeBranch(messageId, 'V1');
      useBranchStore.getState().createBranch(messageId, 'V2');
      useBranchStore.getState().createBranch(messageId, 'V3');
    });

    it('navigates to previous branch', () => {
      const messageId = createMessageId('msg-1');
      useBranchStore.getState().navigateToBranch(messageId, 'prev');

      const index = useBranchStore.getState().getCurrentIndex(messageId);
      expect(index).toBe(1);

      const branch = useBranchStore.getState().getCurrentBranch(messageId);
      expect(branch?.content).toBe('V2');
    });

    it('navigates to next branch', () => {
      const messageId = createMessageId('msg-1');
      useBranchStore.getState().navigateToBranch(messageId, 'prev');
      useBranchStore.getState().navigateToBranch(messageId, 'prev');
      useBranchStore.getState().navigateToBranch(messageId, 'next');

      const index = useBranchStore.getState().getCurrentIndex(messageId);
      expect(index).toBe(1);

      const branch = useBranchStore.getState().getCurrentBranch(messageId);
      expect(branch?.content).toBe('V2');
    });

    it('does not go below index 0', () => {
      const messageId = createMessageId('msg-1');
      useBranchStore.getState().navigateToBranch(messageId, 'prev');
      useBranchStore.getState().navigateToBranch(messageId, 'prev');
      useBranchStore.getState().navigateToBranch(messageId, 'prev');

      const index = useBranchStore.getState().getCurrentIndex(messageId);
      expect(index).toBe(0);
    });

    it('does not go above max index', () => {
      const messageId = createMessageId('msg-1');
      useBranchStore.getState().navigateToBranch(messageId, 'next');
      useBranchStore.getState().navigateToBranch(messageId, 'next');
      useBranchStore.getState().navigateToBranch(messageId, 'next');

      const index = useBranchStore.getState().getCurrentIndex(messageId);
      expect(index).toBe(2);
    });

    it('does nothing when message has no branches', () => {
      const messageId = createMessageId('msg-nonexistent');
      useBranchStore.getState().navigateToBranch(messageId, 'prev');

      const count = useBranchStore.getState().getBranchCount(messageId);
      expect(count).toBe(0);
    });

    it('does nothing when message has only one branch', () => {
      const messageId = createMessageId('msg-single');
      useBranchStore.getState().initializeBranch(messageId, 'Only one');

      useBranchStore.getState().navigateToBranch(messageId, 'prev');
      useBranchStore.getState().navigateToBranch(messageId, 'next');

      const index = useBranchStore.getState().getCurrentIndex(messageId);
      expect(index).toBe(0);
    });
  });

  describe('getCurrentBranch', () => {
    it('returns null for non-existent message', () => {
      const messageId = createMessageId('msg-nonexistent');
      const branch = useBranchStore.getState().getCurrentBranch(messageId);
      expect(branch).toBeNull();
    });

    it('returns current branch content', () => {
      const messageId = createMessageId('msg-1');
      useBranchStore.getState().initializeBranch(messageId, 'Content');

      const branch = useBranchStore.getState().getCurrentBranch(messageId);
      expect(branch?.content).toBe('Content');
    });
  });

  describe('getBranchCount', () => {
    it('returns 0 for non-existent message', () => {
      const messageId = createMessageId('msg-nonexistent');
      const count = useBranchStore.getState().getBranchCount(messageId);
      expect(count).toBe(0);
    });

    it('returns correct count for message with branches', () => {
      const messageId = createMessageId('msg-1');
      useBranchStore.getState().initializeBranch(messageId, 'V1');
      useBranchStore.getState().createBranch(messageId, 'V2');

      const count = useBranchStore.getState().getBranchCount(messageId);
      expect(count).toBe(2);
    });
  });

  describe('getCurrentIndex', () => {
    it('returns 0 for non-existent message', () => {
      const messageId = createMessageId('msg-nonexistent');
      const index = useBranchStore.getState().getCurrentIndex(messageId);
      expect(index).toBe(0);
    });

    it('returns correct index after navigation', () => {
      const messageId = createMessageId('msg-1');
      useBranchStore.getState().initializeBranch(messageId, 'V1');
      useBranchStore.getState().createBranch(messageId, 'V2');
      useBranchStore.getState().createBranch(messageId, 'V3');
      useBranchStore.getState().navigateToBranch(messageId, 'prev');

      const index = useBranchStore.getState().getCurrentIndex(messageId);
      expect(index).toBe(1);
    });
  });
});
