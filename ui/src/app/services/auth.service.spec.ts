import { TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { AuthService } from './auth.service';

const mockGetAuth = vi.fn(() => ({ name: 'mock-auth' }));
const mockGoogleAuthProvider = vi.fn().mockImplementation(function() { return { name: 'google-provider' }; });
const mockSignInWithPopup = vi.fn().mockResolvedValue({
  user: {
    email: 'test@example.com',
    displayName: 'Test User',
    getIdToken: vi.fn().mockResolvedValue('mock-token')
  }
});
const mockSignOut = vi.fn().mockResolvedValue(undefined);
const mockOnAuthStateChanged = vi.fn().mockImplementation((auth, callback) => {
  callback(null);
  return () => {};
});

(globalThis as any).__mockGetAuth = mockGetAuth;
(globalThis as any).__mockGoogleAuthProvider = mockGoogleAuthProvider;
(globalThis as any).__mockSignInWithPopup = mockSignInWithPopup;
(globalThis as any).__mockSignOut = mockSignOut;
(globalThis as any).__mockOnAuthStateChanged = mockOnAuthStateChanged;



describe('AuthService (Firebase Interaction)', () => {
  let service: AuthService;

  beforeEach(() => {
    vi.clearAllMocks();
    mockGetAuth.mockClear();
    mockGoogleAuthProvider.mockClear();
    mockSignInWithPopup.mockClear();
    mockSignOut.mockClear();
    mockOnAuthStateChanged.mockClear();
    if ((globalThis as any).__mockCookies) {
      (globalThis as any).__mockCookies.set.mockClear();
      (globalThis as any).__mockCookies.remove.mockClear();
    }
    
    mockSignInWithPopup.mockResolvedValue({ user: { email: 'test@example.com', displayName: 'Test User' } });
    mockOnAuthStateChanged.mockImplementation((auth, callback) => {
      callback(null);
      return () => {};
    });

    delete (window as any).serverSideUser;

    TestBed.resetTestingModule();
    TestBed.configureTestingModule({
      providers: [AuthService]
    });
  });

  afterAll(() => {
    delete (globalThis as any).__mockGetAuth;
    delete (globalThis as any).__mockGoogleAuthProvider;
    delete (globalThis as any).__mockSignInWithPopup;
    delete (globalThis as any).__mockSignOut;
    delete (globalThis as any).__mockOnAuthStateChanged;
  });

  it('should initialize Firebase app and auth on creation', () => {
    service = TestBed.inject(AuthService);
    expect(mockGetAuth).toHaveBeenCalled();
    expect(mockOnAuthStateChanged).toHaveBeenCalled();
  });

  it('should call signInWithPopup and set cookie on signIn()', async () => {
    service = TestBed.inject(AuthService);
    
    const mockUser = {
      email: 'user@example.com',
      displayName: 'Jane',
      getIdToken: vi.fn().mockResolvedValue('mock-token')
    };
    mockSignInWithPopup.mockResolvedValue({ user: mockUser });

    await service.signIn();

    expect(mockSignInWithPopup).toHaveBeenCalled();
    expect((globalThis as any).__mockCookies.set).toHaveBeenCalledWith(
      'signinToken',
      expect.any(String),
      expect.objectContaining({ path: '/' })
    );
    expect(service.user()).toEqual(mockUser);
  });

  it('should call signOut and remove cookie on signOut()', async () => {
    service = TestBed.inject(AuthService);
    service.user.set({ email: 'user@example.com', displayName: 'Jane' } as any);

    await service.signOut();

    expect(mockSignOut).toHaveBeenCalled();
    expect((globalThis as any).__mockCookies.remove).toHaveBeenCalledWith('signinToken', { path: '/' });
    expect(service.user()).toBeNull();
  });
});
