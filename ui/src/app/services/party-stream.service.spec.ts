import { TestBed } from '@angular/core/testing';
import { PartyStreamService } from './party-stream.service';

class MockEventSource {
  url: string;
  onopen?: () => void;
  onerror?: (err: any) => void;
  listeners: { [type: string]: ((event: any) => void)[] } = {};

  static instances: MockEventSource[] = [];
  static closedCount = 0;

  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }

  addEventListener(type: string, listener: (event: any) => void) {
    if (!this.listeners[type]) {
      this.listeners[type] = [];
    }
    this.listeners[type].push(listener);
  }

  close() {
    MockEventSource.closedCount++;
  }

  static reset() {
    MockEventSource.instances = [];
    MockEventSource.closedCount = 0;
  }
}

describe('PartyStreamService', () => {
  let service: PartyStreamService;
  let originalEventSource: any;

  beforeEach(() => {
    originalEventSource = (window as any).EventSource;
    (window as any).EventSource = MockEventSource as any;
    MockEventSource.reset();

    TestBed.configureTestingModule({
      providers: [PartyStreamService]
    });
    service = TestBed.inject(PartyStreamService);
  });

  afterEach(() => {
    service.disconnect();
    (window as any).EventSource = originalEventSource;
  });

  it('should connect to EventSource with correct URL', () => {
    service.connect('CODE123');
    expect(MockEventSource.instances.length).toBe(1);
    expect(MockEventSource.instances[0].url).toBe('/api/v1/party/CODE123/stream');
  });

  it('should receive interest_update and update latestInterestUpdate signal', () => {
    service.connect('CODE123');
    const mockMsg = { event_id: 'BGM101', tier: 'must_have' };

    const listeners = MockEventSource.instances[0].listeners['interest_update'];
    expect(listeners).toBeDefined();
    expect(listeners.length).toBe(1);

    listeners[0]({ data: JSON.stringify(mockMsg) });

    expect(service.latestInterestUpdate()).toEqual(mockMsg);
  });

  it('should suspend stream on document visibility hidden and resume on visible', () => {
    service.connect('CODE123');
    expect(MockEventSource.instances.length).toBe(1);

    // Simulate page hidden
    Object.defineProperty(document, 'visibilityState', { value: 'hidden', configurable: true });
    document.dispatchEvent(new Event('visibilitychange'));

    expect(MockEventSource.closedCount).toBe(1);

    // Simulate page visible
    Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true });
    document.dispatchEvent(new Event('visibilitychange'));

    // Should create a new EventSource instance
    expect(MockEventSource.instances.length).toBe(2);
    expect(service.streamResumed()).toBe(1);
  });
});
