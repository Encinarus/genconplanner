import { Component, OnInit, signal, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterModule, Router } from '@angular/router';
import { Title } from '@angular/platform-browser';
import { ApiService, AdminOrganizer } from '../../services/api.service';

@Component({
  selector: 'app-admin-orgs',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterModule],
  template: `
    <div class="admin-container">
      <header class="admin-header">
        <div class="header-left">
          <a routerLink="/" class="back-link">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
            Back to Planner
          </a>
          <h1>Organizer Administration</h1>
          <p class="subtitle">Search, inspect, and merge duplicate organizers in the system.</p>
        </div>
      </header>

      <!-- Main Controls -->
      <div class="controls-card">
        <div class="search-wrapper">
          <svg class="search-icon" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            type="text"
            placeholder="Search by organizer name or alias..."
            [ngModel]="searchQuery()"
            (ngModelChange)="searchQuery.set($event)"
            class="search-input"
          />
        </div>
      </div>

      <!-- Loading State -->
      <div *ngIf="loading()" class="loading-state">
        <div class="spinner"></div>
        <p>Loading organizers...</p>
      </div>

      <!-- Error State -->
      <div *ngIf="error()" class="error-card">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="none" viewBox="0 0 24 24" stroke="currentColor" class="error-icon">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <div class="error-details">
          <h3>Error Loading Organizers</h3>
          <p>{{ error() }}</p>
          <button (click)="loadOrganizers()" class="retry-btn">Retry</button>
        </div>
      </div>

      <!-- Content -->
      <div *ngIf="!loading() && !error()" class="org-content">
        <div class="org-stats">
          Total Organizers: <strong>{{ organizers().length }}</strong> | 
          Filtered: <strong>{{ filteredOrganizers().length }}</strong>
        </div>

        <div class="org-grid">
          <div 
            *ngFor="let org of filteredOrganizers()" 
            class="org-card" 
            [class.selected]="isSelected(org.id)"
            (click)="toggleSelection(org.id)"
          >
            <div class="card-selection">
              <input 
                type="checkbox" 
                [checked]="isSelected(org.id)"
                (click)="$event.stopPropagation(); toggleSelection(org.id)"
                class="checkbox-custom"
                id="checkbox-{{org.id}}"
              />
              <label for="checkbox-{{org.id}}" class="checkbox-label" (click)="$event.stopPropagation()"></label>
            </div>
            
            <div class="card-body">
              <div class="card-header">
                <span class="org-id">ID: #{{ org.id }}</span>
                <span class="event-badge" [class.no-events]="org.numEvents === 0">
                  {{ org.numEvents }} {{ org.numEvents === 1 ? 'event' : 'events' }}
                </span>
              </div>
              
              <ul class="alias-list">
                <li *ngFor="let alias of org.aliases" class="alias-item">
                  <span *ngIf="alias; else noAlias">{{ alias }}</span>
                  <ng-template #noAlias><span class="empty-alias">No organizer name provided</span></ng-template>
                </li>
              </ul>
            </div>
          </div>
        </div>

        <div *ngIf="filteredOrganizers().length === 0" class="empty-search">
          <p>No organizers found matching "{{ searchQuery() }}"</p>
        </div>
      </div>

      <!-- Action Panel (Bottom Floating Bar) -->
      <div class="action-panel" [class.active]="selectedIds().size >= 2">
        <div class="action-content">
          <div class="selection-info">
            <span class="count">{{ selectedIds().size }}</span>
            <span class="label">organizers selected for merge</span>
          </div>
          <div class="action-buttons">
            <button (click)="clearSelection()" class="cancel-btn">Cancel</button>
            <button 
              (click)="confirmMerge()" 
              [disabled]="merging()" 
              class="merge-btn"
            >
              <span *ngIf="!merging()">Merge Selected</span>
              <span *ngIf="merging()" class="spinner-sm"></span>
            </button>
          </div>
        </div>
      </div>

      <!-- Confirmation Modal -->
      <div class="modal-backdrop" *ngIf="showConfirmModal()">
        <div class="confirm-modal">
          <div class="modal-header">
            <h3>Confirm Merge</h3>
            <button (click)="showConfirmModal.set(false)" class="close-btn">&times;</button>
          </div>
          <div class="modal-body">
            <p>Are you sure you want to merge these <strong>{{ selectedIds().size }}</strong> organizers?</p>
            
            <div class="modal-warning">
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              <span>This operation is permanent. The organizer with the smallest ID will be kept, and all other IDs will be merged into it in the events database.</span>
            </div>

            <div class="selected-summary">
              <ul>
                <li *ngFor="let org of getSelectedOrganizers()">
                  <strong>ID #{{ org.id }}</strong> ({{ org.numEvents }} events)
                  <div class="summary-aliases">{{ org.aliases.join(', ') }}</div>
                </li>
              </ul>
            </div>
          </div>
          <div class="modal-footer">
            <button (click)="showConfirmModal.set(false)" class="cancel-btn" [disabled]="merging()">Cancel</button>
            <button (click)="executeMerge()" class="execute-btn" [disabled]="merging()">
              <span *ngIf="!merging()">Proceed with Merge</span>
              <span *ngIf="merging()" class="spinner-sm"></span>
            </button>
          </div>
        </div>
      </div>
    </div>
  `,
  styles: [`
    :host {
      display: block;
      background-color: #0f172a;
      color: #e2e8f0;
      min-height: 100vh;
      font-family: 'Inter', system-ui, sans-serif;
    }

    .admin-container {
      max-width: 1200px;
      margin: 0 auto;
      padding: 40px 20px 120px 20px;
    }

    .admin-header {
      margin-bottom: 30px;
      border-bottom: 1px solid #1e293b;
      padding-bottom: 20px;
    }

    .back-link {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      color: #6366f1;
      text-decoration: none;
      font-size: 14px;
      font-weight: 500;
      margin-bottom: 15px;
      transition: color 0.2s ease;
    }

    .back-link:hover {
      color: #818cf8;
    }

    h1 {
      font-size: 32px;
      font-weight: 800;
      letter-spacing: -0.025em;
      margin: 0 0 8px 0;
      background: linear-gradient(to right, #e2e8f0, #94a3b8);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
    }

    .subtitle {
      color: #94a3b8;
      font-size: 16px;
      margin: 0;
    }

    /* Controls */
    .controls-card {
      background: rgba(30, 41, 59, 0.4);
      backdrop-filter: blur(12px);
      border: 1px solid rgba(255, 255, 255, 0.05);
      border-radius: 16px;
      padding: 20px;
      margin-bottom: 30px;
    }

    .search-wrapper {
      position: relative;
      width: 100%;
    }

    .search-icon {
      position: absolute;
      left: 16px;
      top: 50%;
      transform: translateY(-50%);
      width: 20px;
      height: 20px;
      color: #64748b;
    }

    .search-input {
      width: 100%;
      box-sizing: border-box;
      padding: 14px 16px 14px 48px;
      background: #0f172a;
      border: 1px solid #334155;
      border-radius: 12px;
      color: #f1f5f9;
      font-size: 16px;
      transition: all 0.2s ease;
    }

    .search-input:focus {
      outline: none;
      border-color: #6366f1;
      box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.15);
    }

    /* Loading / Spinner */
    .loading-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 80px 20px;
      color: #94a3b8;
    }

    .spinner {
      border: 3px solid rgba(99, 102, 241, 0.1);
      border-top: 3px solid #6366f1;
      border-radius: 50%;
      width: 40px;
      height: 40px;
      animation: spin 1s linear infinite;
      margin-bottom: 16px;
    }

    .spinner-sm {
      display: inline-block;
      border: 2px solid rgba(255, 255, 255, 0.2);
      border-top: 2px solid #ffffff;
      border-radius: 50%;
      width: 16px;
      height: 16px;
      animation: spin 1s linear infinite;
    }

    @keyframes spin {
      0% { transform: rotate(0deg); }
      100% { transform: rotate(360deg); }
    }

    /* Error Card */
    .error-card {
      display: flex;
      gap: 16px;
      background: rgba(239, 68, 68, 0.05);
      border: 1px solid rgba(239, 68, 68, 0.2);
      border-radius: 16px;
      padding: 24px;
      color: #fca5a5;
    }

    .error-icon {
      color: #ef4444;
      flex-shrink: 0;
    }

    .error-details h3 {
      margin: 0 0 6px 0;
      font-size: 18px;
    }

    .error-details p {
      margin: 0 0 16px 0;
      color: #f87171;
    }

    .retry-btn {
      background: #ef4444;
      color: white;
      border: none;
      padding: 8px 16px;
      border-radius: 8px;
      cursor: pointer;
      font-weight: 500;
      transition: background 0.2s;
    }

    .retry-btn:hover {
      background: #dc2626;
    }

    /* Grid & Cards */
    .org-stats {
      font-size: 14px;
      color: #64748b;
      margin-bottom: 20px;
      padding-left: 4px;
    }

    .org-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
      gap: 20px;
    }

    .org-card {
      background: #1e293b;
      border: 1px solid #334155;
      border-radius: 16px;
      padding: 20px;
      display: flex;
      gap: 16px;
      cursor: pointer;
      transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
      position: relative;
    }

    .org-card:hover {
      transform: translateY(-2px);
      border-color: #475569;
      background: #243249;
    }

    .org-card.selected {
      border-color: #6366f1;
      background: rgba(99, 102, 241, 0.08);
      box-shadow: 0 0 0 1px #6366f1;
    }

    .card-selection {
      display: flex;
      align-items: flex-start;
      padding-top: 4px;
    }

    .checkbox-custom {
      display: none;
    }

    .checkbox-label {
      width: 20px;
      height: 20px;
      border: 2px solid #475569;
      border-radius: 6px;
      display: inline-block;
      position: relative;
      cursor: pointer;
      transition: all 0.2s ease;
    }

    .org-card.selected .checkbox-label {
      background: #6366f1;
      border-color: #6366f1;
    }

    .org-card.selected .checkbox-label::after {
      content: '';
      position: absolute;
      left: 6px;
      top: 2px;
      width: 5px;
      height: 10px;
      border: solid white;
      border-width: 0 2px 2px 0;
      transform: rotate(45deg);
    }

    .card-body {
      flex: 1;
    }

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 12px;
    }

    .org-id {
      font-size: 13px;
      font-weight: 600;
      color: #64748b;
    }

    .event-badge {
      font-size: 12px;
      font-weight: 600;
      background: rgba(16, 185, 129, 0.1);
      color: #10b981;
      padding: 4px 10px;
      border-radius: 9999px;
    }

    .event-badge.no-events {
      background: rgba(148, 163, 184, 0.1);
      color: #94a3b8;
    }

    .alias-list {
      list-style: none;
      padding: 0;
      margin: 0;
      display: flex;
      flex-direction: column;
      gap: 6px;
    }

    .alias-item {
      font-size: 15px;
      font-weight: 500;
      color: #f1f5f9;
    }

    .empty-alias {
      color: #64748b;
      font-style: italic;
    }

    .empty-search {
      text-align: center;
      padding: 60px 20px;
      color: #64748b;
      font-size: 16px;
    }

    /* Floating Action Panel */
    .action-panel {
      position: fixed;
      bottom: -100px;
      left: 0;
      right: 0;
      background: rgba(30, 41, 59, 0.85);
      backdrop-filter: blur(16px);
      border-top: 1px solid rgba(255, 255, 255, 0.08);
      padding: 20px 24px;
      box-shadow: 0 -10px 25px -5px rgba(0, 0, 0, 0.3);
      transition: bottom 0.35s cubic-bezier(0.4, 0, 0.2, 1);
      z-index: 100;
    }

    .action-panel.active {
      bottom: 0;
    }

    .action-content {
      max-width: 1200px;
      margin: 0 auto;
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 20px;
    }

    .selection-info {
      display: flex;
      align-items: center;
      gap: 12px;
    }

    .selection-info .count {
      background: #6366f1;
      color: white;
      font-weight: 700;
      font-size: 18px;
      width: 32px;
      height: 32px;
      display: flex;
      align-items: center;
      justify-content: center;
      border-radius: 50%;
    }

    .selection-info .label {
      font-size: 16px;
      font-weight: 500;
      color: #e2e8f0;
    }

    .action-buttons {
      display: flex;
      gap: 12px;
    }

    .cancel-btn {
      background: transparent;
      border: 1px solid #475569;
      color: #94a3b8;
      padding: 10px 20px;
      border-radius: 10px;
      font-size: 15px;
      font-weight: 600;
      cursor: pointer;
      transition: all 0.2s;
    }

    .cancel-btn:hover:not(:disabled) {
      color: #f1f5f9;
      border-color: #94a3b8;
    }

    .merge-btn {
      background: linear-gradient(to right, #6366f1, #4f46e5);
      border: none;
      color: white;
      padding: 10px 24px;
      border-radius: 10px;
      font-size: 15px;
      font-weight: 600;
      cursor: pointer;
      box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
      transition: all 0.2s;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
    }

    .merge-btn:hover:not(:disabled) {
      background: linear-gradient(to right, #818cf8, #6366f1);
      transform: translateY(-1px);
    }

    .merge-btn:disabled {
      opacity: 0.6;
      cursor: not-allowed;
      box-shadow: none;
    }

    /* Modal */
    .modal-backdrop {
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background: rgba(15, 23, 42, 0.75);
      backdrop-filter: blur(4px);
      z-index: 1000;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 20px;
    }

    .confirm-modal {
      background: #1e293b;
      border: 1px solid #334155;
      border-radius: 20px;
      max-width: 500px;
      width: 100%;
      box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
      overflow: hidden;
      animation: modalFadeIn 0.3s cubic-bezier(0.16, 1, 0.3, 1);
    }

    @keyframes modalFadeIn {
      from { opacity: 0; transform: scale(0.95); }
      to { opacity: 1; transform: scale(1); }
    }

    .modal-header {
      padding: 20px 24px;
      border-bottom: 1px solid #334155;
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .modal-header h3 {
      margin: 0;
      font-size: 20px;
      color: #f1f5f9;
    }

    .close-btn {
      background: transparent;
      border: none;
      color: #64748b;
      font-size: 24px;
      cursor: pointer;
    }

    .close-btn:hover {
      color: #94a3b8;
    }

    .modal-body {
      padding: 24px;
    }

    .modal-warning {
      background: rgba(245, 158, 11, 0.08);
      border: 1px solid rgba(245, 158, 11, 0.2);
      border-radius: 12px;
      padding: 14px;
      color: #fca5a5;
      display: flex;
      gap: 12px;
      font-size: 14px;
      line-height: 1.5;
      margin-bottom: 20px;
    }

    .modal-warning svg {
      color: #f59e0b;
      flex-shrink: 0;
    }

    .selected-summary {
      max-height: 200px;
      overflow-y: auto;
      background: #0f172a;
      border-radius: 12px;
      padding: 16px;
      border: 1px solid #334155;
    }

    .selected-summary ul {
      margin: 0;
      padding: 0;
      list-style: none;
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .selected-summary li {
      font-size: 14px;
      color: #e2e8f0;
    }

    .summary-aliases {
      color: #64748b;
      font-size: 12px;
      margin-top: 4px;
    }

    .modal-footer {
      padding: 16px 24px;
      background: #182235;
      border-top: 1px solid #334155;
      display: flex;
      justify-content: flex-end;
      gap: 12px;
    }

    .execute-btn {
      background: #6366f1;
      border: none;
      color: white;
      padding: 10px 20px;
      border-radius: 10px;
      font-size: 15px;
      font-weight: 600;
      cursor: pointer;
      transition: background 0.2s;
    }

    .execute-btn:hover:not(:disabled) {
      background: #4f46e5;
    }

    .execute-btn:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }
  `]
})
export class AdminOrgsComponent implements OnInit {
  private api = inject(ApiService);
  private titleService = inject(Title);

  // States
  organizers = signal<AdminOrganizer[]>([]);
  loading = signal<boolean>(true);
  error = signal<string | null>(null);
  searchQuery = signal<string>('');
  selectedIds = signal<Set<number>>(new Set<number>());
  merging = signal<boolean>(false);
  showConfirmModal = signal<boolean>(false);

  // Computed filtered list
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
    this.api.getAdminOrganizers().subscribe({
      next: (data) => {
        // Sort by number of events descending
        this.organizers.set(data.sort((a, b) => b.numEvents - a.numEvents));
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Error fetching organizers', err);
        this.error.set('Failed to retrieve organizers list from server. ' + (err.error?.error || err.message));
        this.loading.set(false);
      }
    });
  }

  isSelected(id: number): boolean {
    return this.selectedIds().has(id);
  }

  toggleSelection(id: number) {
    const current = new Set(this.selectedIds());
    if (current.has(id)) {
      current.delete(id);
    } else {
      current.add(id);
    }
    this.selectedIds.set(current);
  }

  clearSelection() {
    this.selectedIds.set(new Set<number>());
  }

  getSelectedOrganizers(): AdminOrganizer[] {
    const ids = this.selectedIds();
    return this.organizers().filter(org => ids.has(org.id));
  }

  confirmMerge() {
    if (this.selectedIds().size < 2) return;
    this.showConfirmModal.set(true);
  }

  executeMerge() {
    if (this.selectedIds().size < 2) return;
    this.merging.set(true);
    const ids = Array.from(this.selectedIds());
    
    this.api.mergeAdminOrganizers(ids).subscribe({
      next: () => {
        this.merging.set(false);
        this.showConfirmModal.set(false);
        this.clearSelection();
        this.loadOrganizers(); // Reload list to reflect merged state
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
