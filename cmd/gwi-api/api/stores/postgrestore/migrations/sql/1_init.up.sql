CREATE TABLE volunteers (
    email    TEXT PRIMARY KEY,
    password TEXT NOT NULL
);

CREATE TABLE teams (
    id   TEXT PRIMARY KEY,
    name TEXT
);

CREATE TABLE team_members (
    volunteer_email TEXT REFERENCES volunteers(email) ON DELETE CASCADE,
    team_id         TEXT REFERENCES teams(id) ON DELETE CASCADE,

    PRIMARY KEY (team_id, volunteer_email)
);