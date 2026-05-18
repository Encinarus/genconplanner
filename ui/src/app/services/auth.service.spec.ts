import { TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import Cookies from 'js-cookie';
import { AuthService } from './auth.service';
import * as fireAuth from 'firebase/auth';

// Mock firebase modules
vi.mock('firebase/app', () => ({
  initializeApp: vi.fn(() => ({ name: 'mock-app' }))
}));

vi.mock('firebase/auth', () => ({
  getAuth: vi.fn(() => ({ name: 'mock-auth' })),
  GoogleAuthProvider: vi.fn().mockImplementation(function() { return { name: 'google-provider' }; }),
  signInWithPopup: vi.fn(),
  signOut: vi.fn(),
  onAuthStateChanged: vi.fn()
}));

vi.mock('@firebase/auth', () => ({
  getAuth: vi.fn(() => ({ name: 'mock-auth' })),
  GoogleAuthProvider: vi.fn().mockImplementation(function() { return { name: 'google-provider' }; }),
  signInWithPopup: vi.fn(),
  signOut: vi.fn(),
  onAuthStateChanged: vi.fn()
}));

vi.mock('js-cookie', () => ({
  default: {
    set: vi.fn(),
    remove: vi.fn()
  }
}));

describe('AuthService (Firebase Interaction)', () => {
  let service: AuthService;

  beforeEach(() => {
    vi.clearAllMocks();
    // Clear any window serverSideUser before test
    delete (window as any).serverSideUser;

    TestBed.configureTestingModule({
      providers: [AuthService]
    });
    service = TestBed.inject(AuthService);
  });

  it('should initialize Firebase app and auth on creation', () => {
    expect(fireAuth.getAuth).toHaveBeenCalled();
    expect(fireAuth.onAuthStateChanged).toHaveBeenCalled();
  });

  it('should call signInWithPopup and set cookie on signIn()', async () => {
    const mockUser = {
      displayName: 'Firebase User',
      getIdToken: vi.fn().mockResolvedValue('mock-jwt-token')
    };
    (fireAuth.signInWithPopup as any).mockResolvedValue({ user: mockUser });

    await service.signIn();

    expect(fireAuth.signInWithPopup).toHaveBeenCalled();
    expect(Cookies.set).toHaveBeenCalledWith(
      'signinToken',
      'mock-jwt-token',
      expect.objectContaining({ path: '/', sameSite: 'strict' })
    );
    expect(service.user()).toEqual(mockUser);
    expect(service.displayName()).toBe('Firebase User');
  });

  it('should call signOut and remove cookie on signOut()', async () => {
    (fireAuth.signOut as any).mockResolvedValue(undefined);

    await service.signOut();

    expect(fireAuth.signOut).toHaveBeenCalled();
    expect(Cookies.remove).toHaveBeenCalledWith('signinToken', { path: '/' });
    expect(service.user()).toBeNull();
    expect(service.displayName()).toBeNull();
  });
});
