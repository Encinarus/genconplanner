import { ComponentFixture, TestBed } from '@angular/core/testing';
import { NO_ERRORS_SCHEMA, signal } from '@angular/core';
import { CategoryDetailComponent } from './category-detail.component';
import { ActivatedRoute, provideRouter } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { of } from 'rxjs';
import { describe, it, expect, beforeEach } from 'vitest';

describe('CategoryDetailComponent', () => {
  let component: CategoryDetailComponent;
  let fixture: ComponentFixture<CategoryDetailComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CategoryDetailComponent],
      schemas: [NO_ERRORS_SCHEMA],
      providers: [
        provideRouter([]),
        { provide: ApiService, useValue: { searchEvents: () => of([]) } },
        { provide: AuthService, useValue: { user: signal(null) } },
        {
          provide: ActivatedRoute,
          useValue: {
            params: of({ year: '2026', cat: 'BGM', grouping: 'by_system' }),
            queryParams: of({}),
            snapshot: { params: { year: '2026', cat: 'BGM', grouping: 'by_system' } }
          }
        }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(CategoryDetailComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  }, 30000);

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
