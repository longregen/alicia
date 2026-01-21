import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { useBranchStore, type SiblingMessage } from './branchStore';
import { createMessageId } from '../types/streaming';

// Mock fetch for API calls
const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

// Mock setTipMessageId for the dynamically imported conversationStore
const mockSetTipMessageId = vi.fn();
vi.mock('./conversationStore', () => ({
  useConversationStore: {
    getState: () => ({
      setTipMessageId: mockSetTipMessageId,
    }),
  },
}));

describe('BranchStore', () => {
  beforeEach(() => {
    // Reset the store state between tests
    useBranchStore.setState({
      branchStates: new Map(),
    });
    mockFetch.mockReset();
    mockSetTipMessageId.mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('initializeFromMessages', () => {
    it('initializes branches from messages with same previous_id', () => {
      const messages = [
        { id: 'msg-1', previous_id: null, contents: 'First message', created_at: '2025-01-15T10:00:00Z' },
        { id: 'msg-2', previous_id: 'msg-1', contents: 'Response A', created_at: '2025-01-15T10:01:00Z' },
        { id: 'msg-3', previous_id: 'msg-1', contents: 'Response B', created_at: '2025-01-15T10:02:00Z' },
      ];

      useBranchStore.getState().initializeFromMessages('conv-1', messages as any);

      // msg-2 and msg-3 are siblings (same previous_id: msg-1)
      const msg2State = useBranchStore.getState().branchStates.get(createMessageId('msg-2'));
      expect(msg2State?.siblings.length).toBe(2);

      const msg3State = useBranchStore.getState().branchStates.get(createMessageId('msg-3'));
      expect(msg3State?.siblings.length).toBe(2);
    });

    it('does not create branch state for messages without siblings', () => {
      const messages = [
        { id: 'msg-1', previous_id: null, contents: 'First', created_at: '2025-01-15T10:00:00Z' },
        { id: 'msg-2', previous_id: 'msg-1', contents: 'Second', created_at: '2025-01-15T10:01:00Z' },
      ];

      useBranchStore.getState().initializeFromMessages('conv-1', messages as any);

      // No siblings, so no branch state created
      const msg1State = useBranchStore.getState().branchStates.get(createMessageId('msg-1'));
      expect(msg1State).toBeUndefined();

      const msg2State = useBranchStore.getState().branchStates.get(createMessageId('msg-2'));
      expect(msg2State).toBeUndefined();
    });
  });

  describe('fetchSiblings', () => {
    it('fetches siblings from backend and updates state', async () => {
      const siblings: SiblingMessage[] = [
        { id: 'msg-1', content: 'Version 1', createdAt: '2025-01-15T10:00:00Z' },
        { id: 'msg-2', content: 'Version 2', createdAt: '2025-01-15T10:01:00Z' },
      ];

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({
          siblings: siblings.map(s => ({ id: s.id, contents: s.content, created_at: s.createdAt })),
        }),
      });

      const messageId = createMessageId('msg-1');
      await useBranchStore.getState().fetchSiblings('conv-1', messageId);

      const state = useBranchStore.getState().branchStates.get(messageId);
      expect(state?.siblings.length).toBe(2);
      expect(state?.currentIndex).toBe(0); // msg-1 is at index 0
      expect(state?.loading).toBe(false);
      expect(state?.error).toBeNull();
    });

    it('sets loading state while fetching', async () => {
      let resolvePromise: (value: any) => void;
      const fetchPromise = new Promise((resolve) => {
        resolvePromise = resolve;
      });

      mockFetch.mockReturnValueOnce(fetchPromise);

      const messageId = createMessageId('msg-1');
      const fetchPromiseResult = useBranchStore.getState().fetchSiblings('conv-1', messageId);

      // Should be loading
      const loadingState = useBranchStore.getState().branchStates.get(messageId);
      expect(loadingState?.loading).toBe(true);

      // Resolve the fetch
      resolvePromise!({
        ok: true,
        json: () => Promise.resolve({ siblings: [] }),
      });

      await fetchPromiseResult;

      // Should no longer be loading
      const finalState = useBranchStore.getState().branchStates.get(messageId);
      expect(finalState?.loading).toBe(false);
    });

    it('handles fetch errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        text: () => Promise.resolve('Internal Server Error'),
      });

      const messageId = createMessageId('msg-1');
      await useBranchStore.getState().fetchSiblings('conv-1', messageId);

      const state = useBranchStore.getState().branchStates.get(messageId);
      expect(state?.loading).toBe(false);
      expect(state?.error).toBe('Internal Server Error');
    });
  });

  describe('switchBranch', () => {
    beforeEach(() => {
      // Set up initial branch state with 3 siblings
      const messageId = createMessageId('msg-2');
      useBranchStore.setState({
        branchStates: new Map([
          [messageId, {
            siblings: [
              { id: 'msg-1', content: 'Version 1', createdAt: '2025-01-15T10:00:00Z' },
              { id: 'msg-2', content: 'Version 2', createdAt: '2025-01-15T10:01:00Z' },
              { id: 'msg-3', content: 'Version 3', createdAt: '2025-01-15T10:02:00Z' },
            ],
            currentIndex: 1, // Currently at msg-2
            loading: false,
            error: null,
          }],
        ]),
      });
    });

    it('switches to previous branch', async () => {
      mockFetch.mockResolvedValueOnce({ ok: true });

      const messageId = createMessageId('msg-2');
      const result = await useBranchStore.getState().switchBranch('conv-1', messageId, 'prev');

      expect(result?.id).toBe('msg-1');
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/switch-branch'),
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify({ tipMessageId: 'msg-1' }),
        })
      );

      const state = useBranchStore.getState().branchStates.get(messageId);
      expect(state?.currentIndex).toBe(0);
    });

    it('switches to next branch', async () => {
      mockFetch.mockResolvedValueOnce({ ok: true });

      const messageId = createMessageId('msg-2');
      const result = await useBranchStore.getState().switchBranch('conv-1', messageId, 'next');

      expect(result?.id).toBe('msg-3');

      const state = useBranchStore.getState().branchStates.get(messageId);
      expect(state?.currentIndex).toBe(2);
    });

    it('returns null when already at first branch and going prev', async () => {
      const messageId = createMessageId('msg-2');
      // Set to first index
      useBranchStore.setState((state) => {
        const branchState = state.branchStates.get(messageId);
        if (branchState) {
          branchState.currentIndex = 0;
        }
      });

      const result = await useBranchStore.getState().switchBranch('conv-1', messageId, 'prev');

      expect(result).toBeNull();
      expect(mockFetch).not.toHaveBeenCalled();
    });

    it('returns null when already at last branch and going next', async () => {
      const messageId = createMessageId('msg-2');
      // Set to last index
      useBranchStore.setState((state) => {
        const branchState = state.branchStates.get(messageId);
        if (branchState) {
          branchState.currentIndex = 2;
        }
      });

      const result = await useBranchStore.getState().switchBranch('conv-1', messageId, 'next');

      expect(result).toBeNull();
      expect(mockFetch).not.toHaveBeenCalled();
    });

    it('handles API errors gracefully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        text: () => Promise.resolve('Server Error'),
      });

      const messageId = createMessageId('msg-2');
      const result = await useBranchStore.getState().switchBranch('conv-1', messageId, 'prev');

      expect(result).toBeNull();

      const state = useBranchStore.getState().branchStates.get(messageId);
      expect(state?.error).toBe('Server Error');
      expect(state?.currentIndex).toBe(1); // Should not change on error
    });
  });

  describe('selectBranch', () => {
    beforeEach(() => {
      const messageId = createMessageId('msg-1');
      useBranchStore.setState({
        branchStates: new Map([
          [messageId, {
            siblings: [
              { id: 'msg-1', content: 'Version 1', createdAt: '2025-01-15T10:00:00Z' },
              { id: 'msg-2', content: 'Version 2', createdAt: '2025-01-15T10:01:00Z' },
              { id: 'msg-3', content: 'Version 3', createdAt: '2025-01-15T10:02:00Z' },
            ],
            currentIndex: 0,
            loading: false,
            error: null,
          }],
        ]),
      });
    });

    it('selects a specific branch by message ID', async () => {
      mockFetch.mockResolvedValueOnce({ ok: true });

      const messageId = createMessageId('msg-1');
      const result = await useBranchStore.getState().selectBranch('conv-1', messageId, 'msg-3');

      expect(result?.id).toBe('msg-3');

      const state = useBranchStore.getState().branchStates.get(messageId);
      expect(state?.currentIndex).toBe(2);
    });

    it('returns null for non-existent target message', async () => {
      const messageId = createMessageId('msg-1');
      const result = await useBranchStore.getState().selectBranch('conv-1', messageId, 'msg-nonexistent');

      expect(result).toBeNull();
      expect(mockFetch).not.toHaveBeenCalled();
    });
  });

  describe('getCurrentSibling', () => {
    it('returns null for non-existent message', () => {
      const messageId = createMessageId('msg-nonexistent');
      const sibling = useBranchStore.getState().getCurrentSibling(messageId);
      expect(sibling).toBeNull();
    });

    it('returns current sibling', () => {
      const messageId = createMessageId('msg-1');
      useBranchStore.setState({
        branchStates: new Map([
          [messageId, {
            siblings: [
              { id: 'msg-1', content: 'Version 1', createdAt: '2025-01-15T10:00:00Z' },
              { id: 'msg-2', content: 'Version 2', createdAt: '2025-01-15T10:01:00Z' },
            ],
            currentIndex: 1,
            loading: false,
            error: null,
          }],
        ]),
      });

      const sibling = useBranchStore.getState().getCurrentSibling(messageId);
      expect(sibling?.id).toBe('msg-2');
      expect(sibling?.content).toBe('Version 2');
    });
  });

  describe('getSiblingCount', () => {
    it('returns 0 for non-existent message', () => {
      const messageId = createMessageId('msg-nonexistent');
      const count = useBranchStore.getState().getSiblingCount(messageId);
      expect(count).toBe(0);
    });

    it('returns correct count', () => {
      const messageId = createMessageId('msg-1');
      useBranchStore.setState({
        branchStates: new Map([
          [messageId, {
            siblings: [
              { id: 'msg-1', content: 'V1', createdAt: '' },
              { id: 'msg-2', content: 'V2', createdAt: '' },
              { id: 'msg-3', content: 'V3', createdAt: '' },
            ],
            currentIndex: 0,
            loading: false,
            error: null,
          }],
        ]),
      });

      const count = useBranchStore.getState().getSiblingCount(messageId);
      expect(count).toBe(3);
    });
  });

  describe('getCurrentIndex', () => {
    it('returns 0 for non-existent message', () => {
      const messageId = createMessageId('msg-nonexistent');
      const index = useBranchStore.getState().getCurrentIndex(messageId);
      expect(index).toBe(0);
    });

    it('returns correct index', () => {
      const messageId = createMessageId('msg-1');
      useBranchStore.setState({
        branchStates: new Map([
          [messageId, {
            siblings: [
              { id: 'msg-1', content: 'V1', createdAt: '' },
              { id: 'msg-2', content: 'V2', createdAt: '' },
            ],
            currentIndex: 1,
            loading: false,
            error: null,
          }],
        ]),
      });

      const index = useBranchStore.getState().getCurrentIndex(messageId);
      expect(index).toBe(1);
    });
  });

  describe('isLoading', () => {
    it('returns false for non-existent message', () => {
      const messageId = createMessageId('msg-nonexistent');
      const loading = useBranchStore.getState().isLoading(messageId);
      expect(loading).toBe(false);
    });

    it('returns correct loading state', () => {
      const messageId = createMessageId('msg-1');
      useBranchStore.setState({
        branchStates: new Map([
          [messageId, {
            siblings: [],
            currentIndex: 0,
            loading: true,
            error: null,
          }],
        ]),
      });

      const loading = useBranchStore.getState().isLoading(messageId);
      expect(loading).toBe(true);
    });
  });

  describe('clearBranches', () => {
    it('clears all branch state', () => {
      const messageId = createMessageId('msg-1');
      useBranchStore.setState({
        branchStates: new Map([
          [messageId, {
            siblings: [{ id: 'msg-1', content: 'V1', createdAt: '' }],
            currentIndex: 0,
            loading: false,
            error: null,
          }],
        ]),
      });

      useBranchStore.getState().clearBranches();

      const state = useBranchStore.getState().branchStates;
      expect(state.size).toBe(0);
    });
  });

  describe('handleBranchUpdate', () => {
    it('updates branch state for all siblings when new sibling created', () => {
      const allSiblings: SiblingMessage[] = [
        { id: 'msg-1', content: 'Version 1', createdAt: '2025-01-15T10:00:00Z' },
        { id: 'msg-2', content: 'Version 2', createdAt: '2025-01-15T10:01:00Z' },
        { id: 'msg-3', content: 'Version 3 (new)', createdAt: '2025-01-15T10:02:00Z' },
      ];

      useBranchStore.getState().handleBranchUpdate({
        conversationId: 'conv-1',
        parentMessageId: 'parent-1',
        newSibling: allSiblings[2],
        allSiblings,
        totalCount: 3,
      });

      // All siblings should have branch state updated
      const msg1State = useBranchStore.getState().branchStates.get(createMessageId('msg-1'));
      expect(msg1State?.siblings.length).toBe(3);

      const msg2State = useBranchStore.getState().branchStates.get(createMessageId('msg-2'));
      expect(msg2State?.siblings.length).toBe(3);

      const msg3State = useBranchStore.getState().branchStates.get(createMessageId('msg-3'));
      expect(msg3State?.siblings.length).toBe(3);
      // New sibling should be current
      expect(msg3State?.currentIndex).toBe(2);
    });
  });
});
