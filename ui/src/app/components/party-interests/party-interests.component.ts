import { Component, OnInit, OnDestroy, signal, inject, Input, effect, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { ApiService, SharedInterestGroup, Event } from '../../services/api.service';
import { PartyStreamService } from '../../services/party-stream.service';
import { AuthService } from '../../services/auth.service';
import { LinkService } from '../../services/link.service';

@Component({
  selector: 'app-party-interests',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterModule],
  templateUrl: './party-interests.component.html',
  styleUrl: './party-interests.component.css'
})
export class PartyInterestsComponent implements OnInit, OnDestroy {
  @Input() partyId!: number;
  @Input() year!: number;

  private api = inject(ApiService);
  private partyStream = inject(PartyStreamService);
  public auth = inject(AuthService);
  public linkService = inject(LinkService);

  groups = signal<SharedInterestGroup[]>([]);
  loading = signal<boolean>(true);
  error = signal<string | null>(null);

  selectedGroup = signal<SharedInterestGroup | null>(null);
  selectedEventDetails = signal<Event | null>(null);
  loadingDetails = signal<boolean>(false);

  // Search and Filter
  searchQuery = signal<string>('');
  selectedCategory = signal<string>('ALL');
  sortBy = signal<'score' | 'title'>('title');
  myInterestFilter = signal<string>('all');
  collapsedCategories = signal<Set<string>>(new Set<string>());

  toggleCategoryCollapse(categoryName: string, event: MouseEvent) {
    event.stopPropagation();
    const current = new Set(this.collapsedCategories());
    if (current.has(categoryName)) {
      current.delete(categoryName);
    } else {
      current.add(categoryName);
    }
    this.collapsedCategories.set(current);
  }

  isCategoryCollapsed(categoryName: string): boolean {
    return this.collapsedCategories().has(categoryName);
  }

  constructor() {
    // Listen for real-time SSE updates
    effect(() => {
      const update = this.partyStream.latestInterestUpdate();
      if (update && update.party_id === this.partyId) {
        this.handleRealtimeUpdate(update);
      }
    });

    // Listen for SSE stream resumption when tab becomes visible
    effect(() => {
      const count = this.partyStream.streamResumed();
      if (count > 0) {
        console.log('SSE stream resumed from background, reloading party interests to ensure fresh state.');
        this.loadInterests();
      }
    });
  }

  ngOnInit() {
    this.loadInterests();
    this.partyStream.connect(this.partyId);
  }

  ngOnDestroy() {
    this.partyStream.disconnect();
  }

  loadInterests() {
    this.loading.set(true);
    this.api.getPartyInterests(this.partyId, this.year).subscribe({
      next: (groups) => {
        this.groups.set(groups || []);
        if (groups && groups.length > 0) {
          this.selectGroup(groups[0]);
        }
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Error loading party interests:', err);
        this.error.set('Failed to load shared interests.');
        this.loading.set(false);
      }
    });
  }

  selectGroup(group: SharedInterestGroup) {
    this.selectedGroup.set(group);
    this.loadingDetails.set(true);
    this.api.getEvent(group.repEventId).subscribe({
      next: (event) => {
        this.selectedEventDetails.set(event);
        this.loadingDetails.set(false);
      },
      error: (err) => {
        console.error('Error loading event details:', err);
        this.selectedEventDetails.set(null);
        this.loadingDetails.set(false);
      }
    });
  }

  onStar(group: SharedInterestGroup, tier: string) {
    const user = this.auth.user();
    if (!user || !user.email) return;
    const userEmail = user.email;
    const userDisplayName = user.displayName || userEmail.split('@')[0];

    // Optimistic UI update
    let memberInterests = [...(group.memberInterests || [])];
    const existingIndex = memberInterests.findIndex(m => m.email === userEmail);

    if (existingIndex >= 0) {
      if (memberInterests[existingIndex].tier === tier) {
        // Toggle off
        memberInterests.splice(existingIndex, 1);
        tier = ''; // Send empty tier to unstar
      } else {
        // Update tier
        memberInterests[existingIndex] = { ...memberInterests[existingIndex], tier };
      }
    } else {
      // Add new
      memberInterests.push({ email: userEmail, displayName: userDisplayName, tier });
    }

    // Recalculate score
    let newScore = 0;
    memberInterests.forEach(m => {
      if (m.tier === 'purchased') newScore += 500;
      else if (m.tier === 'must_have') newScore += 100;
      else if (m.tier === 'very_interested') newScore += 50;
      else if (m.tier === 'somewhat_interested') newScore += 10;
    });

    const updatedGroup = { ...group, memberInterests, groupScore: newScore };
    this.groups.update(list => list.map(g => g.clusterId === group.clusterId ? updatedGroup : g));
    if (this.selectedGroup()?.clusterId === group.clusterId) {
      this.selectedGroup.set(updatedGroup);
    }

    this.api.starEvent(group.repEventId, tier !== '', true, tier).subscribe({
      error: (err) => {
        console.error('Failed to star event:', err);
        this.loadInterests(); // Revert on failure
      }
    });
  }

  handleRealtimeUpdate(update: any) {
    this.groups.update(list => {
      return list.map(g => {
        if (g.clusterId !== update.cluster_id) return g;

        let memberInterests = [...(g.memberInterests || [])];
        const existingIndex = memberInterests.findIndex(m => m.email === update.email);

        if (!update.tier || update.tier === '') {
          if (existingIndex >= 0) memberInterests.splice(existingIndex, 1);
        } else {
          if (existingIndex >= 0) {
            memberInterests[existingIndex] = { ...memberInterests[existingIndex], tier: update.tier };
          } else {
            memberInterests.push({ email: update.email, displayName: update.email.split('@')[0], tier: update.tier });
          }
        }

        let newScore = 0;
        memberInterests.forEach(m => {
          if (m.tier === 'purchased') newScore += 500;
          else if (m.tier === 'must_have') newScore += 100;
          else if (m.tier === 'very_interested') newScore += 50;
          else if (m.tier === 'somewhat_interested') newScore += 10;
        });

        const updated = { ...g, memberInterests, groupScore: newScore };
        if (this.selectedGroup()?.clusterId === g.clusterId) {
          this.selectedGroup.set(updated);
        }
        return updated;
      });
    });
  }

  showScrollIndicator = signal<boolean>(true);
  showTopScrollIndicator = signal<boolean>(false);

  onListScroll(event: any) {
    const target = event.target as HTMLElement;
    if (!target) return;
    const { scrollTop, scrollHeight, clientHeight } = target;
    this.showTopScrollIndicator.set(scrollTop > 20);
    this.showScrollIndicator.set(scrollTop + clientHeight < scrollHeight - 20);
  }

  checkScroll() {
    setTimeout(() => {
      const el = document.querySelector('.event-cards-list') as HTMLElement;
      if (el) {
        const { scrollTop, scrollHeight, clientHeight } = el;
        this.showTopScrollIndicator.set(scrollTop > 20);
        this.showScrollIndicator.set(scrollHeight > clientHeight && scrollTop + clientHeight < scrollHeight - 20);
      }
    }, 100);
  }

  getUserTier(group: SharedInterestGroup): string {
    const user = this.auth.user();
    if (!user || !user.email || !group.memberInterests) return '';
    const m = group.memberInterests.find(item => item.email === user.email);
    return m ? m.tier : '';
  }

  getOtherMembers(group: SharedInterestGroup) {
    const user = this.auth.user();
    if (!user || !user.email || !group.memberInterests) return [];
    return group.memberInterests.filter(m => m.email !== user.email);
  }

  getOtherMembersByTier(group: SharedInterestGroup, tier: string) {
    return this.getOtherMembers(group).filter(m => m.tier === tier);
  }

  getOtherMemberCountByTier(group: SharedInterestGroup, tier: string): number {
    return this.getOtherMembersByTier(group, tier).length;
  }

  getMembersByTier(group: SharedInterestGroup, tier: string) {
    if (!group.memberInterests) return [];
    return group.memberInterests.filter(m => m.tier === tier);
  }

  private categoryMap: { [key: string]: string } = {
    "ANI": "Anime Activities",
    "BGM": "Board Games",
    "CGM": "Non-Collectable/Tradable Card Games",
    "EGM": "Electronic Games",
    "ENT": "Entertainment Events",
    "ESC": "Escape Rooms",
    "FLM": "Film Fest",
    "HMN": "Historical Miniatures",
    "KID": "Kids Activities",
    "LRP": "Larps",
    "MHE": "Miniature Hobby Events",
    "NMN": "Non-Historical Miniatures",
    "RPG": "Role Playing Games",
    "RPGA": "Role Playing Game Association",
    "SEM": "Seminiars",
    "SPA": "Supplemental Activities",
    "TCG": "Tradeable Card Game",
    "TDA": "True Dungeon",
    "TRD": "Trade Day Events",
    "WKS": "Workshop",
    "ZED": "Isle of Misfit Events"
  };

  availableCategories = computed(() => {
    const map = this.categoryMap;
    return Object.keys(map).sort((a, b) => map[a].localeCompare(map[b])).map(code => ({
      code,
      name: map[code]
    }));
  });

  filteredGroupsByCategory() {
    let list = this.groups();
    const q = this.searchQuery().toLowerCase().trim();
    const cat = this.selectedCategory();
    const filter = this.myInterestFilter();

    if (q) {
      list = list.filter(g => g.title.toLowerCase().includes(q) || g.gameSystem.toLowerCase().includes(q));
    }
    if (cat !== 'ALL') {
      list = list.filter(g => g.shortCategory === cat);
    }
    if (filter !== 'all') {
      list = list.filter(g => {
        const myTier = this.getUserTier(g);
        if (filter === 'unrated') return !myTier || myTier === '';
        return myTier === filter;
      });
    }

    list.sort((a, b) => {
      if (this.sortBy() === 'score') return b.groupScore - a.groupScore;
      return a.title.localeCompare(b.title);
    });

    // Group by Category Name
    const groupsMap = new Map<string, SharedInterestGroup[]>();
    list.forEach(g => {
      const catName = this.categoryMap[g.shortCategory] || g.shortCategory || 'Unspecified Category';
      if (!groupsMap.has(catName)) {
        groupsMap.set(catName, []);
      }
      groupsMap.get(catName)!.push(g);
    });

    // Sort category keys alphabetically
    const sortedKeys = Array.from(groupsMap.keys()).sort((a, b) => a.localeCompare(b));

    const result = sortedKeys.map(key => ({
      categoryName: key,
      groups: groupsMap.get(key)!
    }));

    this.checkScroll();
    return result;
  }

  getTotalFilteredCount(): number {
    return this.filteredGroupsByCategory().reduce((sum, cat) => sum + cat.groups.length, 0);
  }
}
