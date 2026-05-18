-- Seed Data for Gen Con Planner Integration Tests

-- 1. Users
INSERT INTO public.users (email, display_name, gencon_name, gencon_id, gencon_email) VALUES
('leader@example.com', 'Party Leader', 'GCLeader', 'GC1001', 'leader@gencon.com'),
('member1@example.com', 'Active Member', 'GCMember1', 'GC1002', 'member1@gencon.com'),
('member2@example.com', 'Second Member', 'GCMember2', 'GC1003', 'member2@gencon.com'),
('solo@example.com', 'Solo User', 'GCSolo', 'GC1004', 'solo@gencon.com')
ON CONFLICT (email) DO NOTHING;

-- 2. Parties & Party Members
INSERT INTO public.parties (party_id, name, year, leader_email, short_code) VALUES
(101, '2026 Strategy Group', 2026, 'leader@example.com', 'CODE2026')
ON CONFLICT (party_id) DO NOTHING;

INSERT INTO public.party_members (party_id, email, display_name, gencon_name, gencon_id, gencon_email) VALUES
(101, 'leader@example.com', 'Party Leader', 'GCLeader', 'GC1001', 'leader@gencon.com'),
(101, 'member1@example.com', 'Active Member', 'GCMember1', 'GC1002', 'member1@gencon.com'),
(101, 'member2@example.com', 'Second Member', 'GCMember2', 'GC1003', 'member2@gencon.com')
ON CONFLICT (party_id, email) DO NOTHING;

-- 2.5 Orgs
INSERT INTO public.orgs (id, alias) VALUES
(1, 'Indie Games'),
(2, 'D&D Adventurers')
ON CONFLICT DO NOTHING;

-- 3. Events
INSERT INTO public.events (
    event_id, active, org_group, title, short_description, long_description,
    event_type, game_system, rules_edition, min_players, max_players, age_required,
    experience_required, materials_provided, start_time, duration, end_time, gm_names,
    website, email, tournament, round_number, total_rounds, min_play_time, attendee_registration,
    cost, location, room_name, table_number, special_category, tickets_available, year, short_category, last_modified
) VALUES
('BGM26ND100001', true, 'Indie Games', 'Catan Championship', 'Compete in Catan', 'Long description for Catan',
 'BGM - Board Game', 'Catan', '5th Edition', 3, 4, '12+', 'None', true, '2026-07-30 10:00:00-04', 120, '2026-07-30 12:00:00-04', 'GM Alice',
 'http://catan.com', 'alice@catan.com', true, 1, 3, 120, 'No', 10, 'ICC', 'Hall A', 'Table 1', '', 15, 2026, 'BGM', '2026-07-30 00:00:00-04'),

('BGM26ND100002', true, 'Indie Games', 'Catan Championship', 'Compete in Catan', 'Long description for Catan',
 'BGM - Board Game', 'Catan', '5th Edition', 3, 4, '12+', 'None', true, '2026-07-30 14:00:00-04', 120, '2026-07-30 16:00:00-04', 'GM Alice',
 'http://catan.com', 'alice@catan.com', true, 2, 3, 120, 'No', 10, 'ICC', 'Hall A', 'Table 2', '', 0, 2026, 'BGM', '2026-07-30 00:00:00-04'),

('BGM26ND100003', false, 'Indie Games', 'Catan Championship', 'Compete in Catan', 'Long description for Catan',
 'BGM - Board Game', 'Catan', '5th Edition', 3, 4, '12+', 'None', true, '2026-07-31 10:00:00-04', 120, '2026-07-31 12:00:00-04', 'GM Alice',
 'http://catan.com', 'alice@catan.com', true, 3, 3, 120, 'No', 10, 'ICC', 'Hall A', 'Table 3', '', 10, 2026, 'BGM', '2026-07-30 00:00:00-04'),

('RPG26ND200001', true, 'D&D Adventurers', 'Dungeons & Dragons Epic', 'Save the realm', 'Long D&D description',
 'RPG - Role Playing Game', 'D&D 5e', '5th Edition', 4, 6, 'Teen', 'Some', false, '2026-07-30 13:00:00-04', 240, '2026-07-30 17:00:00-04', 'Bob the DM',
 'http://dnd.com', 'bob@dnd.com', false, 1, 1, 240, 'No', 14, 'ICC', 'Room 101', 'Table A', '', 5, 2026, 'RPG', '2026-07-30 00:00:00-04'),

('RPG26ND200002', true, 'D&D Adventurers', 'Dungeons & Dragons Epic Intro', 'Intro to D&D', 'Long D&D description',
 'RPG - Role Playing Game', 'D&D 5e', '5th Edition', 4, 6, 'Teen', 'None', false, '2026-07-31 13:00:00-04', 240, '2026-07-31 17:00:00-04', 'Bob the DM',
 'http://dnd.com', 'bob@dnd.com', false, 1, 1, 240, 'No', 14, 'ICC', 'Room 101', 'Table B', '', 20, 2026, 'RPG', '2026-07-30 00:00:00-04')
ON CONFLICT (event_id) DO NOTHING;

-- 4. Starred Events
INSERT INTO public.starred_events (email, event_id, level, tier) VALUES
('leader@example.com', 'BGM26ND100001', 'group', 'must_have'),
('leader@example.com', 'BGM26ND100002', 'event', 'purchased'),
('member1@example.com', 'BGM26ND100001', 'group', 'very_interested'),
('member1@example.com', 'RPG26ND200001', 'event', 'must_have'),
('member2@example.com', 'BGM26ND100002', 'event', 'purchased'),
('solo@example.com', 'RPG26ND200002', 'event', 'somewhat_interested')
ON CONFLICT (event_id, email, level) DO NOTHING;
