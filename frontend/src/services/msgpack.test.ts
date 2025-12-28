import { describe, it, expect, beforeEach, vi } from 'vitest';
import { encode, decode, fetchMsgpack } from './msgpack';

describe('msgpack', () => {
  describe('encode', () => {
    it('should encode a simple object', () => {
      const obj = { hello: 'world', count: 42 };
      const encoded = encode(obj);
      // In Node.js, msgpackr returns Buffer which extends Uint8Array
      expect(encoded instanceof Uint8Array || Buffer.isBuffer(encoded)).toBe(true);
      expect(encoded.length).toBeGreaterThan(0);
    });

    it('should encode an array', () => {
      const arr = [1, 2, 3, 'test', { nested: true }];
      const encoded = encode(arr);
      expect(encoded instanceof Uint8Array || Buffer.isBuffer(encoded)).toBe(true);
      expect(encoded.length).toBeGreaterThan(0);
    });

    it('should encode null and undefined', () => {
      const nullEncoded = encode(null);
      expect(nullEncoded instanceof Uint8Array || Buffer.isBuffer(nullEncoded)).toBe(true);

      const undefinedEncoded = encode(undefined);
      expect(undefinedEncoded instanceof Uint8Array || Buffer.isBuffer(undefinedEncoded)).toBe(true);
    });

    it('should encode nested objects', () => {
      const obj = {
        user: {
          name: 'Alice',
          messages: [
            { id: 1, text: 'Hello' },
            { id: 2, text: 'World' },
          ],
        },
      };
      const encoded = encode(obj);
      expect(encoded instanceof Uint8Array || Buffer.isBuffer(encoded)).toBe(true);
    });
  });

  describe('decode', () => {
    it('should decode an encoded object', () => {
      const original = { hello: 'world', count: 42 };
      const encoded = encode(original);
      const decoded = decode(encoded);
      expect(decoded).toEqual(original);
    });

    it('should decode an encoded array', () => {
      const original = [1, 2, 3, 'test', { nested: true }];
      const encoded = encode(original);
      const decoded = decode(encoded);
      expect(decoded).toEqual(original);
    });

    it('should decode nested objects', () => {
      const original = {
        user: {
          name: 'Alice',
          messages: [
            { id: 1, text: 'Hello' },
            { id: 2, text: 'World' },
          ],
        },
      };
      const encoded = encode(original);
      const decoded = decode(encoded);
      expect(decoded).toEqual(original);
    });

    it('should decode null', () => {
      const encoded = encode(null);
      const decoded = decode(encoded);
      expect(decoded).toBeNull();
    });

    it('should preserve data types', () => {
      const original = {
        string: 'hello',
        number: 42,
        boolean: true,
        null: null,
        array: [1, 2, 3],
        object: { key: 'value' },
      };
      const encoded = encode(original);
      const decoded = decode<typeof original>(encoded);

      expect(decoded.string).toBe('hello');
      expect(decoded.number).toBe(42);
      expect(decoded.boolean).toBe(true);
      expect(decoded.null).toBeNull();
      expect(decoded.array).toEqual([1, 2, 3]);
      expect(decoded.object).toEqual({ key: 'value' });
    });
  });

  describe('fetchMsgpack', () => {
    const mockFetch = vi.fn();

    beforeEach(() => {
      (global as any).fetch = mockFetch;
      vi.clearAllMocks();
    });

    it('should make a GET request when no body is provided', async () => {
      const responseData = { result: 'success' };
      const encodedResponse = encode(responseData);

      mockFetch.mockResolvedValueOnce({
        ok: true,
        arrayBuffer: async () => {
          // Convert to proper ArrayBuffer
          const arr = new Uint8Array(encodedResponse);
          return arr.buffer.slice(arr.byteOffset, arr.byteOffset + arr.byteLength);
        },
      });

      const result = await fetchMsgpack('/api/test');

      expect(mockFetch).toHaveBeenCalledWith('/api/test', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/msgpack',
          'Accept': 'application/msgpack',
        },
        body: undefined,
      });

      expect(result).toEqual(responseData);
    });

    it('should make a POST request with msgpack body', async () => {
      const requestData = { message: 'hello' };
      const responseData = { result: 'success' };
      const encodedResponse = encode(responseData);

      mockFetch.mockResolvedValueOnce({
        ok: true,
        arrayBuffer: async () => {
          const arr = new Uint8Array(encodedResponse);
          return arr.buffer.slice(arr.byteOffset, arr.byteOffset + arr.byteLength);
        },
      });

      const result = await fetchMsgpack('/api/test', requestData);

      expect(mockFetch).toHaveBeenCalledWith('/api/test', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/msgpack',
          'Accept': 'application/msgpack',
        },
        body: expect.anything(),
      });

      // Verify the body was encoded correctly
      const sentBody = mockFetch.mock.calls[0][1].body;
      const decodedSentBody = decode(new Uint8Array(sentBody));
      expect(decodedSentBody).toEqual(requestData);

      expect(result).toEqual(responseData);
    });

    it('should handle HTTP errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
      });

      await expect(fetchMsgpack('/api/test')).rejects.toThrow('HTTP 404: Not Found');
    });

    it('should handle 500 errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
      });

      await expect(fetchMsgpack('/api/test')).rejects.toThrow(
        'HTTP 500: Internal Server Error'
      );
    });

    it('should decode complex response data', async () => {
      const responseData = {
        users: [
          { id: 1, name: 'Alice', active: true },
          { id: 2, name: 'Bob', active: false },
        ],
        meta: {
          total: 2,
          page: 1,
        },
      };
      const encodedResponse = encode(responseData);

      mockFetch.mockResolvedValueOnce({
        ok: true,
        arrayBuffer: async () => {
          const arr = new Uint8Array(encodedResponse);
          return arr.buffer.slice(arr.byteOffset, arr.byteOffset + arr.byteLength);
        },
      });

      const result = await fetchMsgpack<typeof responseData>('/api/users');

      expect(result).toEqual(responseData);
      expect(result.users).toHaveLength(2);
      expect(result.meta.total).toBe(2);
    });

    it('should send complex request data', async () => {
      const requestData = {
        messages: [
          { id: 'msg1', content: 'Hello', timestamp: 123456 },
          { id: 'msg2', content: 'World', timestamp: 123457 },
        ],
      };
      const responseData = { synced: true };
      const encodedResponse = encode(responseData);

      mockFetch.mockResolvedValueOnce({
        ok: true,
        arrayBuffer: async () => {
          const arr = new Uint8Array(encodedResponse);
          return arr.buffer.slice(arr.byteOffset, arr.byteOffset + arr.byteLength);
        },
      });

      await fetchMsgpack('/api/sync', requestData);

      const sentBody = mockFetch.mock.calls[0][1].body;
      const decodedSentBody = decode(new Uint8Array(sentBody));
      expect(decodedSentBody).toEqual(requestData);
    });
  });
});
