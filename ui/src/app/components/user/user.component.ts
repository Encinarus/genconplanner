import { Component, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { ApiService } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { PartyService } from '../../services/party.service';
import { Title } from '@angular/platform-browser';
import { AgendaComponent } from '../agenda/agenda.component';

import { GenconIdentityFormComponent } from '../gencon-identity-form/gencon-identity-form.component';

@Component({
  selector: 'app-user',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterModule, AgendaComponent, GenconIdentityFormComponent],
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
  genconName = this.auth.genconName;
  genconId = this.auth.genconId;
  genconEmail = this.auth.genconEmail;
  parties = this.partyService.parties;
  loading = this.partyService.loading;
  creatingParty = signal<boolean>(false);
  editingName = signal<boolean>(false);
  tempDisplayName = signal<string>('');
  tempGenconName = signal<string>('');
  tempGenconId = signal<string>('');
  tempGenconEmail = signal<string>('');
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
    this.tempGenconName.set(this.genconName() || '');
    this.tempGenconId.set(this.genconId() || '');
    this.tempGenconEmail.set(this.genconEmail() || '');
    this.editingName.set(true);
  }

  onCancelEdit(): void {
    this.editingName.set(false);
  }

  onSaveName(): void {
    const newName = this.tempDisplayName().trim();
    const newGenconName = this.tempGenconName().trim();
    const newGenconId = this.tempGenconId().trim();
    const newGenconEmail = this.tempGenconEmail().trim();

    if (!newName) {
      alert('Display name cannot be empty');
      return;
    }

    if (newName === this.displayName() && newGenconName === (this.genconName() || '') && newGenconId === (this.genconId() || '') && newGenconEmail === (this.genconEmail() || '')) {
      this.editingName.set(false);
      return;
    }

    this.api.renameUser(newName, newGenconName, newGenconId, newGenconEmail).subscribe({
      next: () => {
        this.auth.updateUserProfile(newName, newGenconName, newGenconId, newGenconEmail);
        this.editingName.set(false);
      },
      error: (err) => {
        console.error('Error updating user profile:', err);
        alert('Failed to update user profile: ' + (err.error?.error || err.message));
      }
    });
  }
}
