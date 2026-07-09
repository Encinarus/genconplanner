import { TestBed, ComponentFixture } from '@angular/core/testing';
import { signal } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { PartyComponent } from './party.component';
import { ApiService, Party } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { PartyService } from '../../services/party.service';
import { Title } from '@angular/platform-browser';
import { PartyStreamService } from '../../services/party-stream.service';

describe('PartyComponent', () => {
  let component: PartyComponent;
  let fixture: ComponentFixture<PartyComponent>;
  let mockApiService: any;
  let mockAuthService: any;
  let mockPartyService: any;
  let mockRouter: any;
  let mockActivatedRoute: any;

  const mockParty: Party = {
    id: 1,
    name: 'Alpha Party',
    year: 2026,
    leaderEmail: 'leader@example.com',
    shortCode: 'CODE1',
    inviteLink: 'http://localhost/party/CODE1',
    members: [{ displayName: 'Leader', email: 'leader@example.com' }]
  };

  beforeEach(async () => {
    mockApiService = {
      getParty: () => of(mockParty),
      joinParty: () => of({ success: true }),
      leaveParty: () => of({ success: true }),
      deleteParty: () => of({ success: true }),
      renameParty: () => of({ success: true }),
      transferLeadership: () => of({ success: true }),
      getPartyInterests: () => of([])
    };

    mockAuthService = {
      user: signal<any>({ email: 'leader@example.com' })
    };

    mockPartyService = {
      fetchParties: () => {},
      removeParty: () => {}
    };

    mockRouter = {
      navigate: () => {}
    };

    mockActivatedRoute = {
      params: of({ id: '1', tab: 'members' }),
      snapshot: { params: { id: '1' } }
    };

    let mockPartyStreamService = {
      connect: () => {},
      disconnect: () => {},
      latestInterestUpdate: signal(null),
      streamResumed: signal(0)
    };

    await TestBed.configureTestingModule({
      imports: [PartyComponent],
      providers: [
        { provide: ApiService, useValue: mockApiService },
        { provide: AuthService, useValue: mockAuthService },
        { provide: PartyService, useValue: mockPartyService },
        { provide: Router, useValue: mockRouter },
        { provide: ActivatedRoute, useValue: mockActivatedRoute },
        { provide: PartyStreamService, useValue: mockPartyStreamService },
        Title
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(PartyComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should load party details and update roles on init', () => {
    expect(component.party()).toEqual(mockParty);
    expect(component.isLeader()).toBe(true);
    expect(component.isMember()).toBe(true);
    expect(component.activeTab()).toBe('members');
  });

  it('should switch tabs correctly', () => {
    vi.spyOn(mockRouter, 'navigate');
    component.setTab('settings');
    expect(mockRouter.navigate).toHaveBeenCalledWith(['/party', 2026, 'settings'], { fragment: undefined });
  });

  it('should rename party correctly', () => {
    vi.spyOn(mockApiService, 'renameParty').mockImplementation(() => of({ success: true }));
    component.onEditName();
    component.tempName.set('New Alpha Party');
    component.onSaveName();
    expect(mockApiService.renameParty).toHaveBeenCalledWith(1, 'New Alpha Party');
    expect(component.editingName()).toBe(false);
  });
});
