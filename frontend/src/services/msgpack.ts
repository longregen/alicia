import { pack, unpack } from 'msgpackr';

export function encode<T>(value: T): Uint8Array {
  return pack(value);
}

export function decode<T>(data: Uint8Array): T {
  return unpack(data) as T;
}

export async function fetchMsgpack<T>(url: string, body?: unknown): Promise<T> {
  const packedBody = body ? pack(body) : undefined;
  const response = await fetch(url, {
    method: body ? 'POST' : 'GET',
    headers: {
      'Content-Type': 'application/msgpack',
      'Accept': 'application/msgpack',
    },
    body: packedBody as BodyInit | undefined,
  });

  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
  }

  const buffer = await response.arrayBuffer();
  return unpack(new Uint8Array(buffer)) as T;
}
