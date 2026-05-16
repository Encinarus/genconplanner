import { Component, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { ApiService, Party } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { PartyService } from '../../services/party.service';
import { Title } from '@angular/platform-browser';
import { AgendaComponent } from '../agenda/agenda.component';

@Component({
  selector: 'app-user',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterModule, AgendaComponent],
  templateUrl: './user.component.html',
  styleUrl: './user.component.css'
})
export class UserComponent implements OnInit {
  private api = inject(ApiService);
  private auth = inject(AuthService);
  public partyService = inject(PartyService);
  private titleService = inject(Title);

  user = this.auth.user;
  displayName = this.auth.displayName;
  parties = this.partyService.parties;
  loading = this.partyService.loading;
  creatingParty = signal<boolean>(false);
  editingName = signal<boolean>(false);
  tempDisplayName = signal<string>('');
  selectedYear = signal<number>(2026);

  // Form fields
  newPartyName = '';
  newPartyYear = 2026;

  years = [2026, 2025, 2024, 2023, 2022, 2021, 2019];

  constructor() {
    this.titleService.setTitle('My Profile');
  }

  ngOnInit(): void {
    if (this.parties().length === 0) {
      this.partyService.fetchParties();
    }
  }

  loadParties(): void {
    this.partyService.fetchParties();
  }

  isAlreadyInPartyForYear(year: number): boolean {
    return this.parties().some(p => p.year === Number(year));
  }

  onCreateParty(): void {
    if (!this.newPartyName.trim() || this.isAlreadyInPartyForYear(this.newPartyYear)) return;

    this.creatingParty.set(true);
    this.api.createParty(this.newPartyName, this.newPartyYear).subscribe({
      next: (newParty) => {
        this.partyService.addParty(newParty);
        this.newPartyName = '';
        this.creatingParty.set(false);
      },
      error: (err) => {
        console.error('Error creating party:', err);
        alert('Failed to create party: ' + (err.error?.error || err.message));
        this.creatingParty.set(false);
      }
    });
  }

  onEditName(): void {
    this.tempDisplayName.set(this.displayName() || '');
    this.editingName.set(true);
  }

  onCancelEdit(): void {
    this.editingName.set(false);
  }

  onSaveName(): void {
    const newName = this.tempDisplayName().trim();
    if (!newName || newName === this.displayName()) {
      this.editingName.set(false);
      return;
    }

    this.api.renameUser(newName).subscribe({
      next: () => {
        this.auth.updateUserDisplayName(newName);
        this.editingName.set(false);
      },
      error: (err) => {
        console.error('Error renaming user:', err);
        alert('Failed to rename user: ' + (err.error?.error || err.message));
      }
    });
  }
}
