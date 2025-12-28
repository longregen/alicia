import { renderHook } from '@testing-library/react';
import { describe, it, expect, beforeEach } from 'vitest';
import { useServerInfo, useConnectionStatus, useSessionStats } from './useServerInfo';
import { useServerInfoStore } from '../stores/serverInfoStore';

describe('useServerInfo', () => {
  beforeEach(() => {
    // Reset store to initial state
    useServerInfoStore.setState({
      connectionStatus: 'disconnected',
      latency: 0,
      modelInfo: null,
      mcpServers: [],
      sessionStats: {
        messageCount: 0,
        toolCallCount: 0,
        memoriesUsed: 0,
        sessionDuration: 0,
      },
    });
  });

  describe('useServerInfo', () => {
    it('should initialize with default values', () => {
      const { result } = renderHook(() => useServerInfo());

      expect(result.current.connectionStatus).toBe('disconnected');
      expect(result.current.latency).toBe(0);
      expect(result.current.isConnected).toBe(false);
      expect(result.current.isConnecting).toBe(false);
      expect(result.current.modelInfo).toBeNull();
      expect(result.current.mcpServers).toEqual([]);
      expect(result.current.sessionStats).toEqual({
        messageCount: 0,
        toolCallCount: 0,
        memoriesUsed: 0,
        sessionDuration: 0,
      });
    });

    it('should return isConnected true when status is connected', () => {
      useServerInfoStore.setState({ connectionStatus: 'connected' });

      const { result } = renderHook(() => useServerInfo());

      expect(result.current.isConnected).toBe(true);
      expect(result.current.isConnecting).toBe(false);
    });

    it('should return isConnecting true when status is connecting', () => {
      useServerInfoStore.setState({ connectionStatus: 'connecting' });

      const { result } = renderHook(() => useServerInfo());

      expect(result.current.isConnected).toBe(false);
      expect(result.current.isConnecting).toBe(true);
    });

    it('should return isConnecting true when status is reconnecting', () => {
      useServerInfoStore.setState({ connectionStatus: 'reconnecting' });

      const { result } = renderHook(() => useServerInfo());

      expect(result.current.isConnected).toBe(false);
      expect(result.current.isConnecting).toBe(true);
    });

    it('should calculate excellent connection quality for low latency', () => {
      useServerInfoStore.setState({ connectionStatus: 'connected', latency: 30 });

      const { result } = renderHook(() => useServerInfo());

      expect(result.current.connectionQuality).toBe('excellent');
    });

    it('should calculate good connection quality for moderate latency', () => {
      useServerInfoStore.setState({ connectionStatus: 'connected', latency: 75 });

      const { result } = renderHook(() => useServerInfo());

      expect(result.current.connectionQuality).toBe('good');
    });

    it('should calculate fair connection quality for higher latency', () => {
      useServerInfoStore.setState({ connectionStatus: 'connected', latency: 150 });

      const { result } = renderHook(() => useServerInfo());

      expect(result.current.connectionQuality).toBe('fair');
    });

    it('should calculate poor connection quality for high latency', () => {
      useServerInfoStore.setState({ connectionStatus: 'connected', latency: 250 });

      const { result } = renderHook(() => useServerInfo());

      expect(result.current.connectionQuality).toBe('poor');
    });

    it('should return poor connection quality when disconnected', () => {
      useServerInfoStore.setState({ connectionStatus: 'disconnected', latency: 30 });

      const { result } = renderHook(() => useServerInfo());

      expect(result.current.connectionQuality).toBe('poor');
    });

    it('should provide MCP server summary', () => {
      const mockServers = [
        { name: 'server1', status: 'connected', url: 'http://server1.com' },
        { name: 'server2', status: 'connected', url: 'http://server2.com' },
        { name: 'server3', status: 'disconnected', url: 'http://server3.com' },
      ];

      useServerInfoStore.setState({ mcpServers: mockServers as any });

      const { result } = renderHook(() => useServerInfo());

      expect(result.current.mcpServerSummary.total).toBe(3);
      expect(result.current.mcpServerSummary.connected).toBe(2);
      expect(result.current.mcpServerSummary.disconnected).toBe(1);
    });

    it('should format session duration in seconds', () => {
      useServerInfoStore.setState({
        sessionStats: {
          messageCount: 5,
          toolCallCount: 3,
          memoriesUsed: 0,
          sessionDuration: 45,
        },
      });

      const { result } = renderHook(() => useServerInfo());

      expect(result.current.formattedSessionDuration).toBe('45s');
    });

    it('should format session duration in minutes and seconds', () => {
      useServerInfoStore.setState({
        sessionStats: {
          messageCount: 10,
          toolCallCount: 5,
          memoriesUsed: 0,
          sessionDuration: 125, // 2m 5s
        },
      });

      const { result } = renderHook(() => useServerInfo());

      expect(result.current.formattedSessionDuration).toBe('2m 5s');
    });

    it('should format session duration in hours and minutes', () => {
      useServerInfoStore.setState({
        sessionStats: {
          messageCount: 50,
          toolCallCount: 25,
          memoriesUsed: 0,
          sessionDuration: 3725, // 1h 2m
        },
      });

      const { result } = renderHook(() => useServerInfo());

      expect(result.current.formattedSessionDuration).toBe('1h 2m');
    });

    it('should provide model info when available', () => {
      const mockModelInfo = {
        name: 'GPT-4',
        version: '1.0',
        provider: 'OpenAI',
      };

      useServerInfoStore.setState({ modelInfo: mockModelInfo as any });

      const { result } = renderHook(() => useServerInfo());

      expect(result.current.modelInfo).toEqual(mockModelInfo);
    });

    it('should filter connected MCP servers', () => {
      const mockServers = [
        { name: 'server1', status: 'connected', url: 'http://server1.com' },
        { name: 'server2', status: 'disconnected', url: 'http://server2.com' },
        { name: 'server3', status: 'connected', url: 'http://server3.com' },
      ];

      useServerInfoStore.setState({ mcpServers: mockServers as any });

      const { result } = renderHook(() => useServerInfo());

      expect(result.current.connectedMCPServers).toHaveLength(2);
      expect(result.current.connectedMCPServers.map((s) => s.name)).toEqual(['server1', 'server3']);
    });

    it('should filter disconnected MCP servers', () => {
      const mockServers = [
        { name: 'server1', status: 'connected', url: 'http://server1.com' },
        { name: 'server2', status: 'disconnected', url: 'http://server2.com' },
        { name: 'server3', status: 'disconnected', url: 'http://server3.com' },
      ];

      useServerInfoStore.setState({ mcpServers: mockServers as any });

      const { result } = renderHook(() => useServerInfo());

      expect(result.current.disconnectedMCPServers).toHaveLength(2);
      expect(result.current.disconnectedMCPServers.map((s) => s.name)).toEqual(['server2', 'server3']);
    });

    it('should update when store state changes', () => {
      useServerInfoStore.setState({ latency: 50 });

      const { result, rerender } = renderHook(() => useServerInfo());

      expect(result.current.latency).toBe(50);

      useServerInfoStore.setState({ latency: 100 });
      rerender();

      expect(result.current.latency).toBe(100);
    });
  });

  describe('useConnectionStatus', () => {
    it('should return connection status and quality', () => {
      useServerInfoStore.setState({ connectionStatus: 'connected', latency: 40 });

      const { result } = renderHook(() => useConnectionStatus());

      expect(result.current.connectionStatus).toBe('connected');
      expect(result.current.latency).toBe(40);
      expect(result.current.isConnected).toBe(true);
      expect(result.current.isConnecting).toBe(false);
      expect(result.current.connectionQuality).toBe('excellent');
    });

    it('should return isConnecting true when reconnecting', () => {
      useServerInfoStore.setState({ connectionStatus: 'reconnecting' });

      const { result } = renderHook(() => useConnectionStatus());

      expect(result.current.isConnected).toBe(false);
      expect(result.current.isConnecting).toBe(true);
    });

    it('should calculate connection quality correctly', () => {
      useServerInfoStore.setState({ connectionStatus: 'connected', latency: 180 });

      const { result } = renderHook(() => useConnectionStatus());

      expect(result.current.connectionQuality).toBe('fair');
    });

    it('should return poor quality when disconnected regardless of latency', () => {
      useServerInfoStore.setState({ connectionStatus: 'disconnected', latency: 20 });

      const { result } = renderHook(() => useConnectionStatus());

      expect(result.current.connectionQuality).toBe('poor');
      expect(result.current.isConnected).toBe(false);
    });
  });

  describe('useSessionStats', () => {
    it('should return session statistics', () => {
      useServerInfoStore.setState({
        sessionStats: {
          messageCount: 15,
          toolCallCount: 8,
          memoriesUsed: 0,
          sessionDuration: 300,
        },
      });

      const { result } = renderHook(() => useSessionStats());

      expect(result.current.messageCount).toBe(15);
      expect(result.current.toolCallCount).toBe(8);
      expect(result.current.sessionDuration).toBe(300);
    });

    it('should format duration in seconds', () => {
      useServerInfoStore.setState({
        sessionStats: {
          messageCount: 5,
          toolCallCount: 2,
          memoriesUsed: 0,
          sessionDuration: 30,
        },
      });

      const { result } = renderHook(() => useSessionStats());

      expect(result.current.formattedDuration).toBe('30s');
    });

    it('should format duration in minutes and seconds', () => {
      useServerInfoStore.setState({
        sessionStats: {
          messageCount: 10,
          toolCallCount: 5,
          memoriesUsed: 0,
          sessionDuration: 90, // 1m 30s
        },
      });

      const { result } = renderHook(() => useSessionStats());

      expect(result.current.formattedDuration).toBe('1m 30s');
    });

    it('should format duration in hours and minutes', () => {
      useServerInfoStore.setState({
        sessionStats: {
          messageCount: 100,
          toolCallCount: 50,
          memoriesUsed: 0,
          sessionDuration: 7200, // 2h 0m
        },
      });

      const { result } = renderHook(() => useSessionStats());

      expect(result.current.formattedDuration).toBe('2h 0m');
    });

    it('should update when stats change', () => {
      useServerInfoStore.setState({
        sessionStats: {
          messageCount: 5,
          toolCallCount: 2,
          memoriesUsed: 0,
          sessionDuration: 60,
        },
      });

      const { result, rerender } = renderHook(() => useSessionStats());

      expect(result.current.messageCount).toBe(5);

      useServerInfoStore.setState({
        sessionStats: {
          messageCount: 10,
          toolCallCount: 5,
          memoriesUsed: 0,
          sessionDuration: 120,
        },
      });

      rerender();

      expect(result.current.messageCount).toBe(10);
      expect(result.current.toolCallCount).toBe(5);
    });

    it('should handle zero duration', () => {
      useServerInfoStore.setState({
        sessionStats: {
          messageCount: 0,
          toolCallCount: 0,
          memoriesUsed: 0,
          sessionDuration: 0,
        },
      });

      const { result } = renderHook(() => useSessionStats());

      expect(result.current.formattedDuration).toBe('0s');
    });
  });
});
