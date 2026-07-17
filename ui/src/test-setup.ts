import { vi } from 'vitest';

const mockAuth = {
  getAuth: () => {
    if ((globalThis as any).__mockGetAuth) {
      return (globalThis as any).__mockGetAuth();
    }
    return { name: 'mock-auth' };
  },
  GoogleAuthProvider: class {
    constructor() {
      if ((globalThis as any).__mockGoogleAuthProvider) {
        return (globalThis as any).__mockGoogleAuthProvider();
      }
      return { name: 'google-provider' };
    }
  },
  signInWithPopup: (...args: any[]) => {
    if ((globalThis as any).__mockSignInWithPopup) {
      return (globalThis as any).__mockSignInWithPopup(...args);
    }
    return Promise.resolve({ user: null });
  },
  signOut: (...args: any[]) => {
    if ((globalThis as any).__mockSignOut) {
      return (globalThis as any).__mockSignOut(...args);
    }
    return Promise.resolve();
  },
  onAuthStateChanged: (...args: any[]) => {
    if ((globalThis as any).__mockOnAuthStateChanged) {
      return (globalThis as any).__mockOnAuthStateChanged(...args);
    }
    return () => {};
  }
};

vi.mock('firebase/app', () => ({
  initializeApp: vi.fn(() => ({ name: 'mock-app' }))
}));

vi.mock('firebase/auth', () => mockAuth);
vi.mock('@firebase/auth', () => mockAuth);
