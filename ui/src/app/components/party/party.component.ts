import { Component, OnInit, signal, computed, inject, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { ApiService, Party, PartyMember, PartyTicket, TicketTransfer } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { PartyService } from '../../services/party.service';
import { Title } from '@angular/platform-browser';
import { PartyInterestsComponent } from '../party-interests/party-interests.component';

export interface PurchaserGroupView {
  purchaserDisplayName: string;
  purchaserEmail: string;
  tickets: PartyTicket[];
}

export interface EventInstanceView {
  eventId: string;
  location: string;
  startTime: Date | null;
  startTimeFormatted: string;
  dayOfWeekShort: string;
  dayOfWeekLong: string;
  purchaserGroups: PurchaserGroupView[];
  totalTickets: number;
  ticketHoldersTooltip: string;
}

export interface EventGroupView {
  eventTitle: string;
  instances: EventInstanceView[];
  otherInstances: EventInstanceView[];
}

export interface GroupContainer {
  groupKey: string;
  events: EventGroupView[];
}

@Component({
  selector: 'app-party',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterModule, PartyInterestsComponent],
  templateUrl: './party.component.html',
  styleUrl: './party.component.css'
})
export class PartyComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private api = inject(ApiService);
  public auth = inject(AuthService);
  private partyService = inject(PartyService);
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
  
  activeTab = signal<'events' | 'members' | 'calendar' | 'tickets' | 'settings'>('events');

  // Member editing state
  editingMemberEmail = signal<string | null>(null);
  tempMemberDisplayName = signal<string>('');
  tempMemberGenconName = signal<string>('');
  tempMemberGenconId = signal<string>('');
  tempMemberGenconEmail = signal<string>('');

  // Tickets state
  tickets = signal<PartyTicket[]>([]);
  loadingTickets = signal<boolean>(false);
  
  addingTicket = signal<boolean>(false);
  newTicketEventId = '';
  newTicketPurchaserEmail = '';
  newTicketRecipientName = '';
  newTicketHolderEmail = '';
  newTicketType = 'physical';

  transferringTicket = signal<PartyTicket | null>(null);
  transferToEmail = '';
  transferType = 'name_only';
  transferNotes = '';

  ticketGroupBy = signal<'day' | 'category'>('day');
  selectedTicketActions = signal<PartyTicket | null>(null);

  groupedTickets = computed<GroupContainer[]>(() => {
    const rawTickets = this.tickets();
    if (!rawTickets || rawTickets.length === 0) return [];
    const groupBy = this.ticketGroupBy();

    const party = this.party();
    const memberMap = new Map<string, PartyMember>();
    if (party && party.members) {
      for (const m of party.members) {
        if (m.email) {
          memberMap.set(m.email.toLowerCase(), m);
        }
      }
    }

    const allTickets = rawTickets.map(t => {
      const m = t.holderEmail ? memberMap.get(t.holderEmail.toLowerCase()) : null;
      const purchM = t.purchaserEmail ? memberMap.get(t.purchaserEmail.toLowerCase()) : null;
      return {
        ...t,
        holderDisplayName: m?.displayName || t.holderDisplayName || t.holderEmail,
        genconPurchaserName: purchM?.displayName || t.genconPurchaserName || t.purchaserEmail
      };
    });

    const globalEventMap = new Map<string, PartyTicket[]>();
    for (const t of allTickets) {
      const eTitle = t.eventTitle || t.eventId || 'Unknown Event';
      if (!globalEventMap.has(eTitle)) {
        globalEventMap.set(eTitle, []);
      }
      globalEventMap.get(eTitle)!.push(t);
    }

    const createInstanceView = (instTickets: PartyTicket[]): EventInstanceView => {
      const instId = instTickets[0].eventId || 'Unknown';
      const location = instTickets[0].eventLocation || 'No location specified';
      let startTime: Date | null = null;
      let startTimeFormatted = 'Time TBD';
      let dayOfWeekShort = '';
      let dayOfWeekLong = 'Other / Unscheduled';

      if (instTickets[0].eventStartTime) {
        const dt = new Date(instTickets[0].eventStartTime);
        if (!isNaN(dt.getTime())) {
          startTime = dt;
          startTimeFormatted = dt.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
          dayOfWeekShort = dt.toLocaleDateString('en-US', { weekday: 'short' });
          dayOfWeekLong = dt.toLocaleDateString('en-US', { weekday: 'long' });
          if (groupBy !== 'day') {
            startTimeFormatted = dayOfWeekShort + ' ' + startTimeFormatted;
          }
        }
      }

      const purchaserMap = new Map<string, PartyTicket[]>();
      for (const t of instTickets) {
        const purchKey = t.genconPurchaserName || t.purchaserEmail || 'Unknown Purchaser';
        if (!purchaserMap.has(purchKey)) {
          purchaserMap.set(purchKey, []);
        }
        purchaserMap.get(purchKey)!.push(t);
      }

      const purchaserGroups: PurchaserGroupView[] = [];
      const sortedPurchasers = Array.from(purchaserMap.keys()).sort((a, b) => a.localeCompare(b));
      const holderNames: string[] = [];

      for (const purchKey of sortedPurchasers) {
        const ticketsForPurchaser = purchaserMap.get(purchKey)!;
        purchaserGroups.push({
          purchaserDisplayName: purchKey,
          purchaserEmail: ticketsForPurchaser[0].purchaserEmail,
          tickets: ticketsForPurchaser
        });
        for (const t of ticketsForPurchaser) {
          holderNames.push(t.holderDisplayName || t.holderEmail);
        }
      }

      return {
        eventId: instId,
        location: location,
        startTime: startTime,
        startTimeFormatted: startTimeFormatted,
        dayOfWeekShort: dayOfWeekShort,
        dayOfWeekLong: dayOfWeekLong,
        purchaserGroups: purchaserGroups,
        totalTickets: instTickets.length,
        ticketHoldersTooltip: holderNames.join(', ')
      };
    };

    const globalInstanceMap = new Map<string, EventInstanceView[]>();
    for (const [eTitle, ticketsForEvent] of globalEventMap.entries()) {
      const instMap = new Map<string, PartyTicket[]>();
      for (const t of ticketsForEvent) {
        const instId = t.eventId || 'Unknown';
        if (!instMap.has(instId)) {
          instMap.set(instId, []);
        }
        instMap.get(instId)!.push(t);
      }
      const instViews: EventInstanceView[] = [];
      for (const instTickets of instMap.values()) {
        instViews.push(createInstanceView(instTickets));
      }
      instViews.sort((a, b) => {
        if (a.startTime && b.startTime) return a.startTime.getTime() - b.startTime.getTime();
        return 0;
      });
      globalInstanceMap.set(eTitle, instViews);
    }

    const topLevelMap = new Map<string, PartyTicket[]>();
    for (const t of allTickets) {
      let key = 'Other / Unscheduled';
      if (groupBy === 'day') {
        if (t.eventStartTime) {
          const dt = new Date(t.eventStartTime);
          if (!isNaN(dt.getTime())) {
            key = dt.toLocaleDateString('en-US', { weekday: 'long' });
          }
        }
      } else {
        if (t.eventCategory) {
          key = t.eventCategory;
        } else {
          key = 'Uncategorized';
        }
      }
      if (!topLevelMap.has(key)) {
        topLevelMap.set(key, []);
      }
      topLevelMap.get(key)!.push(t);
    }

    const containers: GroupContainer[] = [];
    const dayOrder = ['Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday', 'Other / Unscheduled'];

    const sortedKeys = Array.from(topLevelMap.keys()).sort((a, b) => {
      if (groupBy === 'day') {
        const indexA = dayOrder.indexOf(a);
        const indexB = dayOrder.indexOf(b);
        if (indexA !== -1 && indexB !== -1) return indexA - indexB;
        if (indexA !== -1) return -1;
        if (indexB !== -1) return 1;
      }
      return a.localeCompare(b);
    });

    for (const key of sortedKeys) {
      const ticketsInContainer = topLevelMap.get(key)!;

      const eventMap = new Map<string, PartyTicket[]>();
      for (const t of ticketsInContainer) {
        const eTitle = t.eventTitle || t.eventId || 'Unknown Event';
        if (!eventMap.has(eTitle)) {
          eventMap.set(eTitle, []);
        }
        eventMap.get(eTitle)!.push(t);
      }

      const eventGroupViews: EventGroupView[] = [];
      const sortedEventTitles = Array.from(eventMap.keys()).sort((a, b) => a.localeCompare(b));

      for (const eTitle of sortedEventTitles) {
        const ticketsInEvent = eventMap.get(eTitle)!;

        const currentInstanceMap = new Map<string, PartyTicket[]>();
        for (const t of ticketsInEvent) {
          const instId = t.eventId || 'Unknown';
          if (!currentInstanceMap.has(instId)) {
            currentInstanceMap.set(instId, []);
          }
          currentInstanceMap.get(instId)!.push(t);
        }

        const currentInstanceViews: EventInstanceView[] = [];
        for (const instTickets of currentInstanceMap.values()) {
          currentInstanceViews.push(createInstanceView(instTickets));
        }
        currentInstanceViews.sort((a, b) => {
          if (a.startTime && b.startTime) return a.startTime.getTime() - b.startTime.getTime();
          return 0;
        });

        const allInstViewsForEvent = globalInstanceMap.get(eTitle) || [];
        const currentInstIds = new Set(currentInstanceViews.map(iv => iv.eventId));
        const otherInstanceViews = allInstViewsForEvent.filter(iv => !currentInstIds.has(iv.eventId));

        eventGroupViews.push({
          eventTitle: eTitle,
          instances: currentInstanceViews,
          otherInstances: otherInstanceViews
        });
      }

      containers.push({
        groupKey: key,
        events: eventGroupViews
      });
    }

    return containers;
  });

  @ViewChild(PartyInterestsComponent) partyInterests?: PartyInterestsComponent;

  ngOnInit() {
    this.route.params.subscribe(params => {
      const id = params['id'];
      if (id && (!this.party() || (this.party()!.id.toString() !== id.toString() && this.party()!.year.toString() !== id.toString() && this.party()!.shortCode !== id.toString()))) {
        this.loadParty(id);
      }
      const tab = params['tab'];
      if (tab && ['events', 'members', 'calendar', 'tickets', 'settings'].includes(tab)) {
        const oldTab = this.activeTab();
        this.activeTab.set(tab as any);

        if (oldTab !== tab && this.party()) {
          if (tab === 'events' && this.partyInterests) {
            this.partyInterests.loadInterests(true);
          } else if (tab === 'tickets') {
            this.fetchTickets();
          } else if (tab === 'members' || tab === 'settings') {
            this.loadParty(this.party()!.id, true);
          }
        }
      }
    });
  }

  setTab(tab: 'events' | 'members' | 'calendar' | 'tickets' | 'settings') {
    const p = this.party();
    if (p) {
      const fragment = tab === 'events' ? this.route.snapshot.fragment : undefined;
      this.router.navigate(['/party', p.year, tab], { fragment: fragment || undefined });
    } else {
      const id = this.route.snapshot.params['id'];
      if (id) {
        this.router.navigate(['/party', id, tab]);
      }
    }
  }

  loadParty(id: string | number, background = false) {
    if (!background) this.loading.set(true);
    this.api.getParty(id).subscribe({
      next: (party) => {
        this.party.set(party);
        this.titleService.setTitle(`Party: ${party.name}`);
        this.updateRoles();
        if (!background) this.loading.set(false);

        if (this.activeTab() === 'tickets') {
          this.fetchTickets();
        }

        const currentParam = this.route.snapshot.params['id'];
        if (this.isMember() && currentParam !== party.year.toString()) {
          this.router.navigate(['/party', party.year, this.activeTab()], { replaceUrl: true });
        }
      },
      error: (err) => {
        console.error('Error loading party:', err);
        this.error.set('Failed to load party details.');
        if (!background) this.loading.set(false);
      }
    });
  }

  updateRoles() {
    const p = this.party();
    const user = this.auth.user();
	if (!p || !user || !user.email) {
      this.isLeader.set(false);
      this.isMember.set(false);
      return;
    }
    this.isLeader.set(p.leaderEmail.toLowerCase() === user.email.toLowerCase());
    this.isMember.set(p.members.some(m => m.email.toLowerCase() === user.email!.toLowerCase()));
  }

  onJoin() {
    const p = this.party();
    if (!p) return;
    this.api.joinParty(p.shortCode).subscribe({
      next: () => {
        this.partyService.fetchParties();
        this.router.navigate(['/party', p.year]);
      },
      error: (err) => alert('Failed to join party: ' + (err.error?.error || err.message))
    });
  }

  onLeave() {
    const p = this.party();
    if (!p) return;
    if (confirm('Are you sure you want to leave this party?')) {
      this.api.leaveParty(p.id).subscribe({
        next: () => {
          this.partyService.removeParty(p.id);
          this.router.navigate(['/user']);
        },
        error: (err) => alert('Failed to leave party: ' + (err.error?.error || err.message))
      });
    }
  }

  onDelete() {
    const p = this.party();
    if (!p) return;
    if (confirm('Are you sure you want to delete this party? This action cannot be undone.')) {
      this.api.deleteParty(p.id).subscribe({
        next: () => {
          this.partyService.removeParty(p.id);
          this.router.navigate(['/user']);
        },
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
        this.partyService.fetchParties();
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
          this.partyService.fetchParties();
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

  canEditMember(member: PartyMember): boolean {
    if (this.isLeader()) return true;
    const user = this.auth.user();
    return !!user && !!user.email && member.email.toLowerCase() === user.email.toLowerCase();
  }

  onEditMember(member: PartyMember) {
    if (!this.canEditMember(member)) return;
    this.editingMemberEmail.set(member.email);
    this.tempMemberDisplayName.set(member.displayName || '');
    this.tempMemberGenconName.set(member.genconName || '');
    this.tempMemberGenconId.set(member.genconId || '');
    this.tempMemberGenconEmail.set(member.genconEmail || '');
  }

  onCancelEditMember() {
    this.editingMemberEmail.set(null);
  }

  onSaveMember(member: PartyMember) {
    const p = this.party();
    if (!p) return;
    const newDisplayName = this.tempMemberDisplayName().trim();
    const newGenconName = this.tempMemberGenconName().trim();
    const newGenconId = this.tempMemberGenconId().trim();
    const newGenconEmail = this.tempMemberGenconEmail().trim();

    if (!newDisplayName) {
      alert('Display name cannot be empty');
      return;
    }

    this.api.updatePartyMember(p.id, member.email, newDisplayName, newGenconName, newGenconId, newGenconEmail).subscribe({
      next: () => {
        this.loadParty(p.id);
        this.editingMemberEmail.set(null);
      },
      error: (err) => alert('Failed to update member info: ' + (err.error?.error || err.message))
    });
  }

  trackByMemberEmail(index: number, member: PartyMember): string {
    return member.email;
  }

  // --- Tickets Methods ---

  fetchTickets() {
    const p = this.party();
    if (!p) return;
    this.loadingTickets.set(true);
    this.api.getPartyTickets(p.year).subscribe({
      next: (res) => {
        this.tickets.set(res.tickets || []);
        this.loadingTickets.set(false);
      },
      error: (err) => {
        console.error('Error fetching tickets:', err);
        this.loadingTickets.set(false);
      }
    });
  }

  showAddTicket() {
    const p = this.party();
    const user = this.auth.user();
    if (!p || !user || !user.email) return;
    this.newTicketEventId = '';
    this.newTicketPurchaserEmail = user.email;
    this.newTicketRecipientName = user.displayName || user.email;
    this.newTicketHolderEmail = user.email;
    this.newTicketType = 'physical';
    this.addingTicket.set(true);
  }

  onSaveNewTicket() {
    const p = this.party();
    if (!p) return;
    if (!this.newTicketEventId || !this.newTicketPurchaserEmail || !this.newTicketRecipientName || !this.newTicketHolderEmail) {
      alert('Please fill in all required fields');
      return;
    }
    this.api.addPartyTicket(p.year, {
      eventId: this.newTicketEventId.trim(),
      purchaserEmail: this.newTicketPurchaserEmail.trim(),
      genconRecipientName: this.newTicketRecipientName.trim(),
      holderEmail: this.newTicketHolderEmail.trim(),
      ticketType: this.newTicketType
    }).subscribe({
      next: () => {
        this.addingTicket.set(false);
        this.fetchTickets();
      },
      error: (err) => alert('Failed to add ticket: ' + (err.error?.error || err.message))
    });
  }

  onDeleteTicket(t: PartyTicket) {
    const p = this.party();
    if (!p) return;
    if (confirm(`Are you sure you want to delete ticket ${t.genconTicketId || t.ticketId}?`)) {
      this.api.deletePartyTicket(p.year, t.ticketId).subscribe({
        next: () => this.fetchTickets(),
        error: (err) => alert('Failed to delete ticket: ' + (err.error?.error || err.message))
      });
    }
  }

  startTransfer(t: PartyTicket) {
    this.transferringTicket.set(t);
    this.transferToEmail = '';
    this.transferType = 'name_only';
    this.transferNotes = '';
  }

  onConfirmTransfer() {
    const p = this.party();
    const t = this.transferringTicket();
    if (!p || !t || !this.transferToEmail) return;

    this.api.transferPartyTicket(p.year, t.ticketId, {
      toEmail: this.transferToEmail,
      transferType: this.transferType,
      notes: this.transferNotes
    }).subscribe({
      next: () => {
        this.transferringTicket.set(null);
        this.fetchTickets();
      },
      error: (err) => alert('Failed to initiate transfer: ' + (err.error?.error || err.message))
    });
  }

  onRespondTransfer(t: PartyTicket, action: string) {
    const p = this.party();
    if (!p) return;
    this.api.respondTicketTransfer(p.year, t.ticketId, action).subscribe({
      next: () => this.fetchTickets(),
      error: ((err: any) => alert('Failed to complete transfer: ' + (err.error?.error || err.message)))
    });
  }

  openTicketActions(t: PartyTicket) {
    this.selectedTicketActions.set(t);
    this.transferToEmail = '';
    this.transferType = 'name_only';
    this.transferNotes = '';
  }

  onConfirmTransferActions() {
    const p = this.party();
    const t = this.selectedTicketActions();
    if (!p || !t || !this.transferToEmail) return;

    this.api.transferPartyTicket(p.year, t.ticketId, {
      toEmail: this.transferToEmail,
      transferType: this.transferType,
      notes: this.transferNotes
    }).subscribe({
      next: () => {
        this.selectedTicketActions.set(null);
        this.fetchTickets();
      },
      error: (err) => alert('Failed to initiate transfer: ' + (err.error?.error || err.message))
    });
  }

  onToggleTicketReturn(t: PartyTicket) {
    const p = this.party();
    if (!p) return;
    this.api.toggleTicketReturn(p.year, t.ticketId).subscribe({
      next: (res) => {
        if (this.selectedTicketActions()?.ticketId === t.ticketId) {
          this.selectedTicketActions.set(res.ticket);
        }
        this.fetchTickets();
      },
      error: (err) => alert('Failed to update ticket status: ' + (err.error?.error || err.message))
    });
  }
}
