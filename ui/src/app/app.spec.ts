import { TestBed } from '@angular/core/testing';
import { App } from './app';
import { provideRouter } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { AuthService } from './services/auth.service';
import { signal } from '@angular/core';

describe('App', () => {
  beforeEach(async () => {
    const mockAuthService = {
      user: signal<any>(null),
      displayName: signal<string | null>(null),
      isAdmin: signal<boolean>(false),
      authLoaded: signal<boolean>(true),
      signOut: () => Promise.resolve(),
      signIn: () => Promise.resolve()
    };

    await TestBed.configureTestingModule({
      imports: [App],
      providers: [
        provideRouter([]),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: AuthService, useValue: mockAuthService }
      ]
    }).compileComponents();
  }, 30000);

  it('should create the app', () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    expect(app).toBeTruthy();
  });

  it('should render title', async () => {
    const fixture = TestBed.createComponent(App);
    await fixture.whenStable();
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('.navbar-brand')?.textContent).toContain('Gen Con Planner');
  });
});
