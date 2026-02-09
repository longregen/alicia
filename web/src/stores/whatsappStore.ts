import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';

export type WhatsAppRole = 'reader' | 'alicia';

export enum WhatsAppConnectionStatus {
  Unknown = 'unknown',
  Disconnected = 'disconnected',
  Pairing = 'pairing',
  Connected = 'connected',
  Error = 'error',
}

export interface WhatsAppEvent {
  time: string;
  role: WhatsAppRole;
  type: string;
  detail: string;
}

interface WhatsAppConnectionState {
  status: WhatsAppConnectionStatus;
  qrCode: string | null;
  phone: string | null;
  error: string | null;
}

const MAX_EVENTS = 200;

interface WhatsAppState {
  reader: WhatsAppConnectionState;
  alicia: WhatsAppConnectionState;
  events: WhatsAppEvent[];
}

interface WhatsAppActions {
  setQR: (role: WhatsAppRole, code: string, event: string) => void;
  setStatus: (role: WhatsAppRole, connected: boolean, phone?: string, error?: string) => void;
  setPairing: (role: WhatsAppRole) => void;
  reset: (role?: WhatsAppRole) => void;
  addDebugEvent: (role: WhatsAppRole, event: string, detail: string) => void;
  clearEvents: () => void;
}

type WhatsAppStore = WhatsAppState & WhatsAppActions;

const initialConnectionState: WhatsAppConnectionState = {
  status: WhatsAppConnectionStatus.Unknown,
  qrCode: null,
  phone: null,
  error: null,
};

function pushEvent(state: WhatsAppState, role: WhatsAppRole, eventType: string, detail: string) {
  state.events.push({
    time: new Date().toISOString().slice(11, 23),
    role,
    type: eventType,
    detail,
  });
  if (state.events.length > MAX_EVENTS) {
    state.events.splice(0, state.events.length - MAX_EVENTS);
  }
}

export const useWhatsAppStore = create<WhatsAppStore>()(
  immer((set) => ({
    reader: { ...initialConnectionState },
    alicia: { ...initialConnectionState },
    events: [],

    setQR: (role: WhatsAppRole, code: string, event: string) =>
      set((state) => {
        pushEvent(state, role, 'qr', `event=${event} code=${code ? code.slice(0, 20) + '...' : '(empty)'}`);
        const conn = state[role];
        switch (event) {
          case 'code':
            conn.status = WhatsAppConnectionStatus.Pairing;
            conn.qrCode = code;
            conn.error = null;
            break;
          case 'login':
            conn.status = WhatsAppConnectionStatus.Connected;
            conn.qrCode = null;
            conn.error = null;
            break;
          case 'timeout':
            conn.status = WhatsAppConnectionStatus.Disconnected;
            conn.qrCode = null;
            conn.error = 'QR code timed out';
            break;
          case 'error':
            conn.status = WhatsAppConnectionStatus.Error;
            conn.qrCode = null;
            conn.error = 'Pairing error';
            break;
        }
      }),

    setStatus: (role: WhatsAppRole, connected: boolean, phone?: string, error?: string) =>
      set((state) => {
        pushEvent(state, role, 'status', `connected=${connected} phone=${phone || '-'} error=${error || '-'}`);
        const conn = state[role];
        if (connected) {
          conn.status = WhatsAppConnectionStatus.Connected;
          conn.phone = phone || null;
          conn.error = null;
          conn.qrCode = null;
        } else if (conn.status !== WhatsAppConnectionStatus.Pairing) {
          conn.status = error
            ? WhatsAppConnectionStatus.Error
            : WhatsAppConnectionStatus.Disconnected;
          conn.phone = null;
          conn.error = error || null;
        }
      }),

    setPairing: (role: WhatsAppRole) =>
      set((state) => {
        pushEvent(state, role, 'action', 'pair request sent');
        const conn = state[role];
        conn.status = WhatsAppConnectionStatus.Pairing;
        conn.qrCode = null;
        conn.error = null;
      }),

    reset: (role?: WhatsAppRole) =>
      set((state) => {
        if (role) {
          pushEvent(state, role, 'action', 'reset');
          Object.assign(state[role], initialConnectionState);
        } else {
          pushEvent(state, 'reader', 'action', 'reset all');
          Object.assign(state.reader, initialConnectionState);
          Object.assign(state.alicia, initialConnectionState);
        }
      }),

    addDebugEvent: (role: WhatsAppRole, event: string, detail: string) =>
      set((state) => {
        pushEvent(state, role, event, detail);
      }),

    clearEvents: () =>
      set((state) => {
        state.events = [];
      }),
  }))
);
