import { ComponentFixture, TestBed } from '@angular/core/testing';
import { NO_ERRORS_SCHEMA } from '@angular/core';
import { AdminOrgsComponent } from './admin-orgs.component';
import { ApiService, OrganizerWithSuggestions } from '../../services/api.service';
import { of } from 'rxjs';
import { provideRouter } from '@angular/router';
import { describe, it, expect, beforeEach, vi } from 'vitest';

describe('AdminOrgsComponent', () => {
  let component: AdminOrgsComponent;
  let fixture: ComponentFixture<AdminOrgsComponent>;
  let mockApiService: any;

  const dummySuggestions: OrganizerWithSuggestions[] = [
    {
      id: 1,
      aliases: ['Organizer A', 'Org A'],
      numEvents: 10,
      eventSamples: [{ year: 2026, titles: ['Event 1'] }],
      suggestions: [
        {
          id: 2,
          aliases: ['Organizer B'],
          numEvents: 5,
          reasons: ['Similar name'],
          eventSamples: [{ year: 2026, titles: ['Event 2'] }]
        }
      ]
    }
  ];

  beforeEach(async () => {
    mockApiService = {
      getMergeSuggestions: vi.fn().mockReturnValue(of(dummySuggestions)),
      mergeAdminOrganizers: vi.fn().mockReturnValue(of({ status: 'success' }))
    };

    await TestBed.configureTestingModule({
      imports: [AdminOrgsComponent],
      schemas: [NO_ERRORS_SCHEMA],
      providers: [
        { provide: ApiService, useValue: mockApiService },
        provideRouter([])
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(AdminOrgsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create and load merge suggestions on init', () => {
    expect(component).toBeTruthy();
    expect(mockApiService.getMergeSuggestions).toHaveBeenCalled();
    expect(component.organizers().length).toBe(1);
    expect(component.selectedOrg()).toEqual(dummySuggestions[0]);
  });

  it('should filter organizers by search query', () => {
    component.searchQuery.set('Org A');
    fixture.detectChanges();
    expect(component.filteredOrganizers().length).toBe(1);

    component.searchQuery.set('Nonexistent');
    fixture.detectChanges();
    expect(component.filteredOrganizers().length).toBe(0);
  });

  it('should select another organizer candidate', () => {
    const extraOrg: OrganizerWithSuggestions = {
      id: 3,
      aliases: ['Extra Org'],
      numEvents: 1,
      eventSamples: [],
      suggestions: []
    };
    component.selectOrganizer(extraOrg);
    expect(component.selectedOrg()).toEqual(extraOrg);
  });

  it('should compute winner and loser IDs correctly based on smallest ID rule', () => {
    const src = dummySuggestions[0];
    const tgt = dummySuggestions[0].suggestions[0];
    component.initiateMerge(src, tgt);

    expect(component.getMergeWinnerId()).toBe(1); // Smallest ID
    expect(component.getMergeLoserId()).toBe(2);  // Larger ID
    expect(component.getMergeWinnerAliases()).toEqual(['Organizer A', 'Org A']);
    expect(component.getMergeLoserAliases()).toEqual(['Organizer B']);
  });

  it('should trigger merge and reload list on success', () => {
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {});

    const src = dummySuggestions[0];
    const tgt = dummySuggestions[0].suggestions[0];
    component.initiateMerge(src, tgt);
    component.executeMerge();

    expect(mockApiService.mergeAdminOrganizers).toHaveBeenCalledWith([1, 2]);
    expect(component.merging()).toBe(false);
    expect(mockApiService.getMergeSuggestions).toHaveBeenCalledTimes(2);

    alertSpy.mockRestore();
  });
});
