import { Component, OnInit, signal, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { Title } from '@angular/platform-browser';
import { ApiService, OrganizerWithSuggestions, MergeSuggestion } from '../../services/api.service';

@Component({
  selector: 'app-admin-orgs',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterModule],
  template: `
    <div class="container-fluid px-4 py-4">
      <div class="mb-4">
        <a routerLink="/" class="text-decoration-none small text-primary d-inline-flex align-items-center gap-1">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          Back to Planner
        </a>
        <h1 class="mt-2 mb-1 h3 fw-bold">Organizer Duplicate Review</h1>
        <p class="text-muted mb-0 small">Review and merge duplicate event organizers. Prioritized by highest likelihood of duplication.</p>
      </div>

      <!-- Loading State -->
      <div *ngIf="loading() && organizers().length === 0" class="text-center py-5">
        <div class="spinner-border text-primary" role="status">
          <span class="visually-hidden">Loading...</span>
        </div>
        <p class="text-muted mt-2">Running similarity analysis...</p>
      </div>

      <!-- Error State -->
      <div *ngIf="error()" class="alert alert-danger d-flex align-items-center gap-3 p-4 mb-4">
        <svg xmlns="http://www.w3.org/2000/svg" width="28" height="28" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <div>
          <h5 class="alert-heading mb-1 h6 fw-bold">Error Loading Suggestions</h5>
          <p class="mb-2 small">{{ error() }}</p>
          <button (click)="loadOrganizers()" class="btn btn-danger btn-sm">Retry</button>
        </div>
      </div>

      <div *ngIf="!loading() && organizers().length === 0" class="alert alert-success text-center py-4 shadow-sm border">
         <h5 class="h6 fw-bold mb-1 text-success">All Clean!</h5>
         <p class="mb-0 text-muted small">No duplicate organizer matches were detected in the system.</p>
      </div>

      <!-- Split-Pane Layout -->
      <div class="row g-4" *ngIf="organizers().length > 0">
        
        <!-- Left Pane: Candidates List -->
        <div class="col-lg-5 col-xl-4">
          <div class="card shadow-sm border">
            <div class="card-header bg-light">
              <h5 class="card-title h6 mb-0 fw-bold">Review Queue ({{ filteredOrganizers().length }})</h5>
            </div>
            <div class="p-3 border-bottom">
              <div class="input-group">
                <span class="input-group-text bg-white border-end-0">
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" fill="none" viewBox="0 0 24 24" stroke="currentColor" class="text-muted">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                  </svg>
                </span>
                <input
                  type="text"
                  placeholder="Search candidates..."
                  [ngModel]="searchQuery()"
                  (ngModelChange)="searchQuery.set($event)"
                  class="form-control form-control-sm border-start-0"
                />
              </div>
            </div>
            
            <div class="list-group list-group-flush overflow-auto" style="max-height: 65vh;">
              <div
                *ngFor="let org of filteredOrganizers()"
                (click)="selectOrganizer(org)"
                [class.active]="selectedOrg()?.id === org.id"
                class="list-group-item list-group-item-action candidate-row p-3 border-bottom"
              >
                <div class="d-flex justify-content-between align-items-center mb-1">
                  <strong class="text-truncate fw-bold" style="font-size: 0.95rem;">
                    {{ org.aliases[0] || 'Un-named Organizer' }}
                  </strong>
                  <span class="badge bg-danger rounded-pill" style="font-size: 0.75rem;">
                    {{ org.suggestions.length }} {{ org.suggestions.length === 1 ? 'match' : 'matches' }}
                  </span>
                </div>
                <div class="text-muted small d-flex justify-content-between">
                  <span>ID: #{{ org.id }}</span>
                  <span>{{ org.numEvents }} events</span>
                </div>
              </div>
              <div *ngIf="filteredOrganizers().length === 0" class="text-center py-4 text-muted small">
                No duplicate candidates match your search.
              </div>
            </div>
          </div>
        </div>

        <!-- Right Pane: Detail Review Panel -->
        <div class="col-lg-7 col-xl-8">
          
          <!-- Placeholder Screen -->
          <div *ngIf="!selectedOrg()" class="card bg-light border text-center p-5 shadow-sm">
            <div class="my-4">
              <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" fill="none" viewBox="0 0 24 24" stroke="currentColor" class="text-muted">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
            </div>
            <h5 class="h6 fw-bold">Select Organizer to Inspect</h5>
            <p class="text-muted small mb-0">Choose an organizer from the queue on the left to verify and merge duplicate records.</p>
          </div>

          <!-- Selected Organizer Review Panel -->
          <div *ngIf="selectedOrg()">
            
            <!-- Selected Base Organizer details -->
            <div class="card mb-4 border shadow-sm">
              <div class="card-header bg-light">
                <div class="d-flex justify-content-between align-items-center">
                  <h5 class="mb-0 h6 fw-bold">Active Organizer Record</h5>
                  <span class="badge bg-secondary">ID #{{ selectedOrg()?.id }}</span>
                </div>
              </div>
              <div class="card-body">
                <div class="row align-items-center">
                  <div class="col-sm-8">
                    <h6 class="fw-bold mb-1">{{ selectedOrg()?.aliases?.[0] }}</h6>
                    <div class="d-flex flex-wrap gap-1 mt-2">
                      <span *ngFor="let alias of selectedOrg()?.aliases" class="badge bg-light text-dark border px-2 py-1 small">
                        {{ alias }}
                      </span>
                    </div>
                  </div>
                  <div class="col-sm-4 text-sm-end mt-3 mt-sm-0 border-start-sm">
                    <div class="text-muted small">Event Count</div>
                    <div class="h4 fw-bold mb-0 text-primary">{{ selectedOrg()?.numEvents }}</div>
                  </div>
                </div>
              </div>
            </div>

            <h5 class="h6 fw-bold text-secondary mb-3 mt-4">Suggested Duplicate Matches ({{ selectedOrg()?.suggestions?.length }})</h5>

            <!-- List of match candidates -->
            <div *ngFor="let suggestion of selectedOrg()?.suggestions" class="card mb-4 border-warning border-start border-4 shadow-sm">
              <div class="card-header bg-white d-flex justify-content-between align-items-center border-bottom py-3">
                <div>
                  <h6 class="mb-0 fw-bold text-dark">{{ suggestion.aliases[0] }}</h6>
                  <small class="text-muted">ID: #{{ suggestion.id }} &bull; {{ suggestion.numEvents }} events</small>
                </div>
                <button class="btn btn-warning btn-sm fw-bold d-flex align-items-center gap-1" (click)="initiateMerge(selectedOrg()!, suggestion)">
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
                  </svg>
                  Merge...
                </button>
              </div>
              <div class="card-body">
                
                <!-- Similarity indicators reasons -->
                <div class="mb-3">
                  <label class="text-muted small fw-bold mb-1">Similarity Indicators:</label>
                  <div class="d-flex flex-column gap-1">
                    <div *ngFor="let reason of suggestion.reasons" class="d-flex align-items-start gap-2 small">
                      <span class="badge bg-warning-subtle text-warning-emphasis border border-warning-subtle mt-0.5" style="font-size: 0.7rem; padding: 2px 6px;">MATCH</span>
                      <span class="text-dark">{{ reason }}</span>
                    </div>
                  </div>
                </div>

                <!-- Events Side-by-Side Comparison -->
                <div class="mt-3">
                  <label class="text-muted small fw-bold mb-2">Events Timeline History Comparison:</label>
                  <div class="border rounded bg-light p-3" style="font-size: 0.85rem;">
                    <div class="row">
                      
                      <!-- Selected Org Event titles -->
                      <div class="col-md-6 border-end pb-3 pb-md-0">
                        <div class="fw-bold border-bottom pb-1 mb-2 text-dark text-truncate">#{{ selectedOrg()?.id }}: {{ selectedOrg()?.aliases?.[0] }}</div>
                        <div *ngFor="let yearSample of selectedOrg()?.eventSamples" class="mb-3">
                          <span class="badge bg-secondary mb-1" style="font-size: 0.75rem;">{{ yearSample.year }}</span>
                          <ul class="list-unstyled ps-2 mb-0">
                            <li *ngFor="let title of yearSample.titles" class="text-truncate text-muted small py-0.5" [title]="title">
                              &bull; {{ title }}
                            </li>
                          </ul>
                        </div>
                        <div *ngIf="!selectedOrg()?.eventSamples?.length" class="text-muted italic small">No event titles found</div>
                      </div>

                      <!-- Suggestion Org Event titles -->
                      <div class="col-md-6 pt-3 pt-md-0">
                        <div class="fw-bold border-bottom pb-1 mb-2 text-dark text-truncate">#{{ suggestion.id }}: {{ suggestion.aliases[0] }}</div>
                        <div *ngFor="let yearSample of suggestion.eventSamples" class="mb-3">
                          <span class="badge bg-secondary mb-1" style="font-size: 0.75rem;">{{ yearSample.year }}</span>
                          <ul class="list-unstyled ps-2 mb-0">
                            <li *ngFor="let title of yearSample.titles" class="text-truncate text-muted small py-0.5" [title]="title">
                              &bull; {{ title }}
                            </li>
                          </ul>
                        </div>
                        <div *ngIf="!suggestion.eventSamples.length" class="text-muted italic small">No event titles found</div>
                      </div>

                    </div>
                  </div>
                </div>

              </div>
            </div>

          </div>
        </div>

      </div>

      <!-- Confirmation Modal Overlay -->
      <div class="modal fade show d-block" tabindex="-1" style="background: rgba(0, 0, 0, 0.5);" *ngIf="showConfirmModal()">
        <div class="modal-dialog modal-dialog-centered">
          <div class="modal-content">
            <div class="modal-header">
              <h5 class="modal-title h6 fw-bold">Confirm Duplicate Merge</h5>
              <button type="button" class="btn-close" (click)="showConfirmModal.set(false)" aria-label="Close"></button>
            </div>
            <div class="modal-body">
              <p class="small text-dark mb-3">Are you sure you want to merge these duplicate organizer records?</p>
              
              <!-- Merge Direction Summary -->
              <div class="alert alert-warning py-3 px-3 mb-3 border">
                <div class="d-flex align-items-start gap-2">
                  <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="currentColor" class="text-warning flex-shrink-0 mt-0.5">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                  </svg>
                  <div class="small">
                    <div class="fw-bold mb-1 text-warning-emphasis">This action is permanent and cannot be undone.</div>
                    All aliases, history, and events mapping of the duplicate will be combined under the keeping record ID.
                  </div>
                </div>
              </div>

              <!-- Winner / Loser Detail boxes -->
              <div class="vstack gap-2">
                <div class="p-3 border rounded bg-success-subtle text-success-emphasis d-flex justify-content-between align-items-center">
                  <div>
                    <div class="text-muted small fw-medium">KEEPING RECORD (Smallest ID)</div>
                    <strong class="h6 fw-bold text-success-emphasis mb-0 mt-1 d-block">{{ getMergeWinnerAliases()[0] }}</strong>
                  </div>
                  <span class="badge bg-success" style="font-size: 0.8rem;">ID #{{ getMergeWinnerId() }}</span>
                </div>
                
                <div class="text-center text-muted small my-1">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 13l-7 7-7-7m14-6l-7 7-7-7" />
                  </svg>
                </div>

                <div class="p-3 border rounded bg-danger-subtle text-danger-emphasis d-flex justify-content-between align-items-center">
                  <div>
                    <div class="text-muted small fw-medium">MERGING RECORD (To be deleted/merged)</div>
                    <strong class="h6 fw-bold text-danger-emphasis mb-0 mt-1 d-block">{{ getMergeLoserAliases()[0] }}</strong>
                  </div>
                  <span class="badge bg-danger" style="font-size: 0.8rem;">ID #{{ getMergeLoserId() }}</span>
                </div>
              </div>

            </div>
            <div class="modal-footer">
              <button type="button" class="btn btn-secondary btn-sm" (click)="showConfirmModal.set(false)" [disabled]="merging()">Cancel</button>
              <button type="button" class="btn btn-danger btn-sm fw-bold" (click)="executeMerge()" [disabled]="merging()">
                <span *ngIf="!merging()">Execute Merge</span>
                <span *ngIf="merging()" class="spinner-border spinner-border-sm" role="status"></span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }
    .text-mono {
      font-family: var(--bs-font-monospace), SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    }
    .last-border-none:last-child {
      border-bottom: none !important;
    }
    .candidate-row {
      cursor: pointer;
      transition: background-color 0.15s ease;
    }
    .candidate-row:hover {
      background-color: var(--bs-gray-100);
    }
    .candidate-row.active {
      background-color: #e9ecef;
      border-left: 4px solid var(--bs-primary);
    }
    @media (min-width: 576px) {
      .border-start-sm {
        border-left: 1px solid #dee2e6 !important;
      }
    }
  `]
})
export class AdminOrgsComponent implements OnInit {
  private api = inject(ApiService);
  private titleService = inject(Title);

  // States
  organizers = signal<OrganizerWithSuggestions[]>([]);
  selectedOrg = signal<OrganizerWithSuggestions | null>(null);
  loading = signal<boolean>(true);
  error = signal<string | null>(null);
  searchQuery = signal<string>('');
  merging = signal<boolean>(false);
  showConfirmModal = signal<boolean>(false);
  mergeSource = signal<OrganizerWithSuggestions | null>(null);
  mergeTarget = signal<MergeSuggestion | null>(null);

  // Computed filtered list of organizers with suggestions
  filteredOrganizers = computed(() => {
    const query = this.searchQuery().toLowerCase().trim();
    if (!query) {
      return this.organizers();
    }
    return this.organizers().filter(org => 
      org.id.toString().includes(query) ||
      org.aliases.some(alias => alias.toLowerCase().includes(query))
    );
  });

  constructor() {
    this.titleService.setTitle('Admin: Organizers Merge');
  }

  ngOnInit() {
    this.loadOrganizers();
  }

  loadOrganizers() {
    this.loading.set(true);
    this.error.set(null);
    this.api.getMergeSuggestions().subscribe({
      next: (data) => {
        this.organizers.set(data);
        this.loading.set(false);
        const currentSelected = this.selectedOrg();
        if (currentSelected) {
          const fresh = data.find(o => o.id === currentSelected.id);
          this.selectedOrg.set(fresh || null);
        } else if (data.length > 0) {
          this.selectedOrg.set(data[0]);
        }
      },
      error: (err) => {
        console.error('Error fetching merge suggestions', err);
        this.error.set('Failed to retrieve duplicate suggestions from server. ' + (err.error?.error || err.message));
        this.loading.set(false);
      }
    });
  }

  selectOrganizer(org: OrganizerWithSuggestions) {
    this.selectedOrg.set(org);
  }

  initiateMerge(org: OrganizerWithSuggestions, suggestion: MergeSuggestion) {
    this.mergeSource.set(org);
    this.mergeTarget.set(suggestion);
    this.showConfirmModal.set(true);
  }

  getMergeWinnerId(): number {
    const src = this.mergeSource();
    const tgt = this.mergeTarget();
    if (!src || !tgt) return 0;
    return Math.min(src.id, tgt.id);
  }

  getMergeLoserId(): number {
    const src = this.mergeSource();
    const tgt = this.mergeTarget();
    if (!src || !tgt) return 0;
    return Math.max(src.id, tgt.id);
  }

  getMergeWinnerAliases(): string[] {
    const src = this.mergeSource();
    const tgt = this.mergeTarget();
    if (!src || !tgt) return [];
    return src.id < tgt.id ? src.aliases : tgt.aliases;
  }

  getMergeLoserAliases(): string[] {
    const src = this.mergeSource();
    const tgt = this.mergeTarget();
    if (!src || !tgt) return [];
    return src.id > tgt.id ? src.aliases : tgt.aliases;
  }

  executeMerge() {
    const src = this.mergeSource();
    const tgt = this.mergeTarget();
    if (!src || !tgt) return;

    this.merging.set(true);
    const ids = [src.id, tgt.id];
    
    this.api.mergeAdminOrganizers(ids).subscribe({
      next: () => {
        this.merging.set(false);
        this.showConfirmModal.set(false);
        this.mergeSource.set(null);
        this.mergeTarget.set(null);
        this.loadOrganizers();
        alert('Organizers merged successfully!');
      },
      error: (err) => {
        console.error('Error merging organizers', err);
        alert('Failed to merge organizers: ' + (err.error?.error || err.message));
        this.merging.set(false);
      }
    });
  }
}
