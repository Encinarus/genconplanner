import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { of } from 'rxjs';
import { PartyService } from './party.service';
import { ApiService, Party } from './api.service';
import { AuthService } from './auth.service';

describe('PartyService', () => {
  let service: PartyService;
  let mockApiService: any;
  let mockAuthService: any;

  const mockParties: Party[] = [
    {
      id: 1,
      name: 'Alpha Party',
      year: 2026,
      leaderEmail: 'leader@example.com',
      shortCode: 'CODE1',
      inviteLink: 'http://localhost/party/CODE1',
      members: []
    }
  ];

  beforeEach(() => {
    mockApiService = {
      getParties: () => of(mockParties)
    };

    mockAuthService = {
      user: signal<any>({ email: 'leader@example.com' })
    };

    TestBed.configureTestingModule({
      providers: [
        PartyService,
        { provide: ApiService, useValue: mockApiService },
        { provide: AuthService, useValue: mockAuthService }
      ]
    });
    service = TestBed.inject(PartyService);
  });

  it('should fetch parties on initialization if user is present', () => {
    // Because effect() runs asynchronously in Angular testing, we can also call fetchParties directly
    service.fetchParties();
    expect(service.parties()).toEqual(mockParties);
    expect(service.loading()).toBe(false);
  });

  it('should add party to state', () => {
    service.fetchParties();
    const newParty: Party = {
      id: 2,
      name: 'Beta Party',
      year: 2026,
      leaderEmail: 'leader@example.com',
      shortCode: 'CODE2',
      inviteLink: 'http://localhost/party/CODE2',
      members: []
    };

    service.addParty(newParty);
    expect(service.parties().length).toBe(2);
    expect(service.parties()[0]).toEqual(newParty);
  });

  it('should update party in state', () => {
    service.fetchParties();
    const updatedParty: Party = {
      ...mockParties[0],
      name: 'Renamed Alpha Party'
    };

    service.updateParty(updatedParty);
    expect(service.parties()[0].name).toBe('Renamed Alpha Party');
  });

  it('should remove party from state', () => {
    service.fetchParties();
    expect(service.parties().length).toBe(1);

    service.removeParty(1);
    expect(service.parties().length).toBe(0);
  });
});
