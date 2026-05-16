import { Component, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { ApiService, Party, PartyMember } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { Title } from '@angular/platform-browser';

@Component({
  selector: 'app-party',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterModule],
  templateUrl: './party.component.html',
  styleUrl: './party.component.css'
})
export class PartyComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private api = inject(ApiService);
  public auth = inject(AuthService);
  private titleService = inject(Title);

  party = signal<Party | null>(null);
  loading = signal<boolean>(true);
  error = signal<string | null>(null);
  
  isLeader = signal<boolean>(false);
  isMember = signal<boolean>(false);
  
  editingName = signal<boolean>(false);
  tempName = signal<string>('');
  
  transferringLeadership = signal<boolean>(false);
  newLeaderEmail = signal<string>('');

  ngOnInit() {
    this.route.params.subscribe(params => {
      const id = params['id'];
      if (id) {
        this.loadParty(id);
      }
    });
  }

  loadParty(id: string | number) {
    this.loading.set(true);
    this.api.getParty(id).subscribe({
      next: (party) => {
        this.party.set(party);
        this.titleService.setTitle(`Party: ${party.name}`);
        this.updateRoles();
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Error loading party:', err);
        this.error.set('Failed to load party details.');
        this.loading.set(false);
      }
    });
  }

  updateRoles() {
    const p = this.party();
    const user = this.auth.user();
    if (!p || !user) {
      this.isLeader.set(false);
      this.isMember.set(false);
      return;
    }
    this.isLeader.set(p.leaderEmail === user.email);
    this.isMember.set(p.members.some(m => m.email === user.email));
  }

  onJoin() {
    const p = this.party();
    if (!p) return;
    this.api.joinParty(p.shortCode).subscribe({
      next: () => this.loadParty(p.id),
      error: (err) => alert('Failed to join party: ' + (err.error?.error || err.message))
    });
  }

  onLeave() {
    const p = this.party();
    if (!p) return;
    if (confirm('Are you sure you want to leave this party?')) {
      this.api.leaveParty(p.id).subscribe({
        next: () => this.router.navigate(['/user']),
        error: (err) => alert('Failed to leave party: ' + (err.error?.error || err.message))
      });
    }
  }

  onDelete() {
    const p = this.party();
    if (!p) return;
    if (confirm('Are you sure you want to delete this party? This action cannot be undone.')) {
      this.api.deleteParty(p.id).subscribe({
        next: () => this.router.navigate(['/user']),
        error: (err) => alert('Failed to delete party: ' + (err.error?.error || err.message))
      });
    }
  }

  onEditName() {
    this.tempName.set(this.party()?.name || '');
    this.editingName.set(true);
  }

  onSaveName() {
    const p = this.party();
    if (!p) return;
    const newName = this.tempName().trim();
    if (!newName || newName === p.name) {
      this.editingName.set(false);
      return;
    }
    this.api.renameParty(p.id, newName).subscribe({
      next: () => {
        this.loadParty(p.id);
        this.editingName.set(false);
      },
      error: (err) => alert('Failed to rename party: ' + (err.error?.error || err.message))
    });
  }

  onTransferLeadership() {
    const p = this.party();
    if (!p) return;
    const targetEmail = this.newLeaderEmail();
    if (!targetEmail) return;
    
    if (confirm(`Are you sure you want to transfer leadership to ${targetEmail}? You will lose leader privileges.`)) {
      this.api.transferLeadership(p.id, targetEmail).subscribe({
        next: () => {
          this.loadParty(p.id);
          this.transferringLeadership.set(false);
        },
        error: (err) => alert('Failed to transfer leadership: ' + (err.error?.error || err.message))
      });
    }
  }

  copyToClipboard(text: string) {
    navigator.clipboard.writeText(text).then(() => {
      alert('Link copied to clipboard!');
    }).catch(err => {
      console.error('Failed to copy:', err);
    });
  }
}
