import { ComponentFixture, TestBed } from '@angular/core/testing';
import { NO_ERRORS_SCHEMA, signal } from '@angular/core';
import { SearchComponent } from './search.component';
import { ActivatedRoute, provideRouter } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { of } from 'rxjs';
import { describe, it, expect, beforeEach } from 'vitest';

describe('SearchComponent', () => {
  let component: SearchComponent;
  let fixture: ComponentFixture<SearchComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SearchComponent],
      schemas: [NO_ERRORS_SCHEMA],
      providers: [
        provideRouter([]),
        { provide: ApiService, useValue: { searchEvents: () => of([]) } },
        { provide: AuthService, useValue: { user: signal(null) } },
        {
          provide: ActivatedRoute,
          useValue: {
            params: of({ grouping: 'by_system' }),
            queryParams: of({ q: 'test' }),
            snapshot: { params: { grouping: 'by_system' }, queryParams: { q: 'test' } }
          }
        }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(SearchComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  }, 30000);

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
