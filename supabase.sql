create table users(
    id uuid primary key default gen_random_uuid(),
    firebase_uid text unique,
    username text unique,
    display_name text,
    email text unique,
    avatar_url text,
    total_leagues_won integer default 0,
    created_at timestamptz default now(),
    updated_at timestamptz default now());
create table teams(
    id uuid primary key default gen_random_uuid(),
    name text not null,
    short_name text,
    logo_url text,
    country text,
    created_at timestamptz default now());
create table players(
    id uuid primary key default gen_random_uuid(),
    team_id uuid references teams(id),
    name text not null,
    role text,
    batting_style text,
    bowling_style text,
    image_url text,
    base_price bigint,
    status text,
    created_at timestamptz default now());
create table modes(
    id uuid primary key default gen_random_uuid(),
    name text not null,
    description text,
    lineup_size integer,
    reset_frequency text,
    created_at timestamptz default now()
);
create table leagues(
    id uuid primary key default gen_random_uuid(),
    mode_id uuid not null references modes(id),
    created_by uuid not null references users(id),
    name text not null,
    auction_budget bigint not null,
    max_users integer not null,
    start_date timestamptz,
    end_date timestamptz,
    status text not null,
    created_at timestamptz default now()
);
create table league_users(
    league_id uuid not null references leagues(id) on delete cascade,
    user_id uuid not null references users(id) on delete cascade,
    remaining_budget bigint not null,
    franchise_value bigint not null,
    rank integer,
    version integer default 1,
    joined_at timestamptz default now(),
    primary key (league_id, user_id)
);
create table matches(
    id uuid primary key default gen_random_uuid(),
    team1_id uuid not null references teams(id),
    team2_id uuid not null references teams(id),
    match_date timestamptz not null,
    venue text,
    status text not null,
    created_at timestamptz default now()
);
create table player_performances(
    id uuid primary key default gen_random_uuid(),
    player_id uuid not null references players(id),
    match_id uuid not null references matches(id) on delete cascade,
    runs integer default 0,
    wickets integer default 0,
    catches integer default 0,
    strike_rate numeric(6,2),
    economy numeric(6,2),
    valuation_change bigint default 0,
    created_at timestamptz default now()
);
create table auctions(
    id uuid primary key default gen_random_uuid(),
    league_id uuid not null references leagues(id) on delete cascade,
    started_at timestamptz,
    ended_at timestamptz,
    status text not null,
    created_at timestamptz default now()
);
create table auction_queue(
    id uuid primary key default gen_random_uuid(),
    auction_id uuid not null references auctions(id) on delete cascade,
    player_id uuid not null references players(id),
    base_price bigint not null,
    winning_bid bigint,
    winning_user_id uuid references users(id),
    status text not null,
    started_at timestamptz,
    ended_at timestamptz
);
create table bids(
    id uuid primary key default gen_random_uuid(),
    auction_queue_id uuid not null references auction_queue(id) on delete cascade,
    user_id uuid not null references users(id),
    bid_amount bigint not null,
    created_at timestamptz default now()
);
create table auction_events(
    id uuid primary key default gen_random_uuid(),
    auction_id uuid not null references auctions(id) on delete cascade,
    event_type text not null,
    payload_json jsonb,
    created_at timestamptz default now()
);
create table player_ownerships(
    id uuid primary key default gen_random_uuid(),
    league_id uuid not null references leagues(id) on delete cascade,
    user_id uuid not null references users(id),
    player_id uuid not null references players(id),
    purchase_price bigint not null,
    acquired_at timestamptz default now(),
    unique (league_id, player_id)
);
create table lineups(
    id uuid primary key default gen_random_uuid(),
    league_id uuid not null references leagues(id) on delete cascade,
    user_id uuid not null references users(id),
    match_id uuid not null references matches(id) on delete cascade,
    player_id uuid not null references players(id),
    is_captain boolean default false,
    is_vice_captain boolean default false,
    created_at timestamptz default now(),
    unique (league_id, user_id, match_id, player_id)
);
create table trades(
    id uuid primary key default gen_random_uuid(),
    league_id uuid not null references leagues(id) on delete cascade,
    initiated_by uuid not null references users(id),
    offered_to uuid not null references users(id),
    offered_player_id uuid not null references players(id),
    requested_player_id uuid not null references players(id),
    cash_adjustment bigint default 0,
    status text not null,
    created_at timestamptz default now(),
    responded_at timestamptz
);
create table transactions(
    id uuid primary key default gen_random_uuid(),
    league_id uuid not null references leagues(id) on delete cascade,
    user_id uuid not null references users(id),
    amount bigint not null,
    type text not null,
    reference_id text,
    description text,
    created_at timestamptz default now()
);
create table chat_messages(
    id uuid primary key default gen_random_uuid(),
    league_id uuid not null references leagues(id) on delete cascade,
    user_id uuid not null references users(id),
    message text not null,
    created_at timestamptz default now()
);
create table achievements(
    id uuid primary key default gen_random_uuid(),
    name text not null,
    description text,
    badge_icon text,
    created_at timestamptz default now()
);
create table user_achievements(
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    achievement_id uuid not null references achievements(id) on delete cascade,
    earned_at timestamptz default now()
);