import { TestBed, ComponentFixture } from '@angular/core/testing';
import { GenconCalendarComponent } from './gencon-calendar.component';

describe('GenconCalendarComponent', () => {
  let component: GenconCalendarComponent;
  let fixture: ComponentFixture<GenconCalendarComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [GenconCalendarComponent]
    }).compileComponents();

    fixture = TestBed.createComponent(GenconCalendarComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create and calculate correct Wednesday week start date for Gen Con', () => {
    expect(component).toBeTruthy();
    // Gen Con 2026 runs July 30 - Aug 2 -> Wednesday is July 29 (2026-07-29)
    const weekStart2026 = component.getDesiredWeekStart(2026);
    expect(weekStart2026).toBe('2026-07-29');
  });

  it('should update calendar events with category colors and event ordering', () => {
    component.year = 2026;
    component.events = [
      {
        id: 'BGM26ND1001',
        title: 'Catan Championship',
        start: '2026-08-05T10:00:00Z',
        end: '2026-08-05T12:00:00Z',
        categoryCode: 'BGM',
        location: 'ICC : Hall B',
        isMine: true
      },
      {
        id: 'RPG26ND2002',
        title: 'D&D Adventure',
        start: '2026-08-05T10:00:00Z',
        end: '2026-08-05T14:00:00Z',
        categoryCode: 'RPG',
        location: 'ICC : Room 101',
        isMine: false
      }
    ];

    component.ngOnChanges({
      events: {
        currentValue: component.events,
        previousValue: [],
        firstChange: true,
        isFirstChange: () => true
      }
    });

    const opts = component.calendarOptions();
    expect(opts.events).toBeTruthy();
    const evList = opts.events as any[];
    expect(evList.length).toBe(2);
    expect(evList[0].backgroundColor).toBe('#0073AA'); // BGM color
    expect(evList[1].backgroundColor).toBe('#448A80'); // RPG color
  });
});
