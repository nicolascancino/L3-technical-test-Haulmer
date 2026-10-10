CREATE TABLE IF NOT EXISTS cards (
    id                uuid        PRIMARY KEY,
    card_holder_name  text        NOT NULL,
    brand             text        NOT NULL,
    pan_hash          text        NOT NULL UNIQUE,
    last4             char(4)     NOT NULL,
    exp_month         integer     NOT NULL,
    exp_year          integer     NOT NULL,
    created_at        timestamptz NOT NULL,
    updated_at        timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS cards_last4_created_at_id_idx ON cards (last4, created_at, id);
CREATE INDEX IF NOT EXISTS cards_created_at_id_idx ON cards (created_at, id);
CREATE INDEX IF NOT EXISTS cards_card_holder_name_lower_idx ON cards (lower(card_holder_name) text_pattern_ops);
