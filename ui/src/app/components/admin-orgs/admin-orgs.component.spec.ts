import { ComponentFixture, TestBed } from '@angular/core/testing';
import { NO_ERRORS_SCHEMA, signal } from '@angular/core';
import { AdminOrgsComponent } from './admin-orgs.component';
import { ApiService, AdminOrganizer } from '../../services/api.service';
import { of } from 'rxjs';
import { provideRouter } from '@angular/router';
import { describe, it, expect, beforeEach, vi } from 'vitest';

describe('AdminOrgsComponent', () => {
  let component: AdminOrgsComponent;
  let fixture: ComponentFixture<AdminOrgsComponent>;
  let mockApiService: any;

  const dummyOrgs: AdminOrganizer[] = [
    { id: 1, aliases: ['Organizer A', 'Org A'], numEvents: 10 },
    { id: 2, aliases: ['Organizer B'], numEvents: 5 },
    { id: 3, aliases: ['Org C'], numEvents: 0 }
  ];

  beforeEach(async () => {
    mockApiService = {
      getAdminOrganizers: vi.fn().mockReturnValue(of(dummyOrgs)),
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

  it('should create and load organizers on init', () => {
    expect(component).toBeTruthy();
    expect(mockApiService.getAdminOrganizers).toHaveBeenCalled();
    expect(component.organizers().length).toBe(3);
    // Verified sorting by numEvents descending
    expect(component.organizers()[0].id).toBe(1);
    expect(component.organizers()[1].id).toBe(2);
  });

  it('should filter organizers by search query', () => {
    component.searchQuery.set('Org A');
    fixture.detectChanges();
    expect(component.filteredOrganizers().length).toBe(1);
    expect(component.filteredOrganizers()[0].id).toBe(1);

    component.searchQuery.set('1');
    fixture.detectChanges();
    expect(component.filteredOrganizers().length).toBe(1);
    expect(component.filteredOrganizers()[0].id).toBe(1);
  });

  it('should toggle selection of organizer IDs', () => {
    component.toggleSelection(1);
    expect(component.isSelected(1)).toBe(true);

    component.toggleSelection(1);
    expect(component.isSelected(1)).toBe(false);
  });

  it('should clear selection', () => {
    component.toggleSelection(1);
    component.toggleSelection(2);
    component.clearSelection();
    expect(component.selectedIds().size).toBe(0);
  });

  it('should trigger merge and reload list on success', () => {
    // Stub global alert to prevent visual modal prompts in test log
    const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {});

    component.toggleSelection(1);
    component.toggleSelection(2);
    component.executeMerge();

    expect(mockApiService.mergeAdminOrganizers).toHaveBeenCalledWith([1, 2]);
    expect(component.merging()).toBe(false);
    expect(component.selectedIds().size).toBe(0);
    // Reload organizers should be called
    expect(mockApiService.getAdminOrganizers).toHaveBeenCalledTimes(2);

    alertSpy.mockRestore();
  });
});
