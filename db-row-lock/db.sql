CREATE TABLE events (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    date TIMESTAMP NOT NULL,
    venue TEXT NOT NULL,
    total_seats INTEGER NOT NULL,
    available_seats INTEGER NOT NULL
);

CREATE TABLE tickets (
    id UUID PRIMARY KEY,
    event_id UUID REFERENCES events(id),
    seat_number TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('AVAILABLE', 'RESERVED', 'BOOKED'))
);

CREATE TABLE reservations (
    id UUID PRIMARY KEY,
    ticket_id UUID REFERENCES tickets(id),
    user_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'CONFIRMED', 'CANCELLED'))
);

-- Seed 1 event and 10 tickets
INSERT INTO events (id, name, date, venue, total_seats, available_seats)
VALUES ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Concert', '2021-12-31 20:00:00', 'Venue', 10, 10);

INSERT INTO tickets (id, event_id, seat_number, status)
VALUES ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'A1', 'AVAILABLE'),
         ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'A2', 'AVAILABLE'),
         ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'A3', 'AVAILABLE'),
         ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'A4', 'AVAILABLE'),
         ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a16', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'A5', 'AVAILABLE'),
         ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a17', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'A6', 'AVAILABLE'),
         ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a18', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'A7', 'AVAILABLE'),
         ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a19', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'A8', 'AVAILABLE'),
         ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a20', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'A9', 'AVAILABLE'),
         ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a21', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'A10', 'AVAILABLE');
