-- Plays table - stores actual game plays
CREATE TABLE IF NOT EXISTS plays (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    partner1_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    partner2_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    play_data JSONB NOT NULL DEFAULT '{}', -- JSON for every play and play detail
    is_live BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_plays_game ON plays(game_id);
CREATE INDEX IF NOT EXISTS idx_plays_partner1 ON plays(partner1_id);
CREATE INDEX IF NOT EXISTS idx_plays_partner2 ON plays(partner2_id);
CREATE INDEX IF NOT EXISTS idx_plays_is_live ON plays(is_live);
CREATE INDEX IF NOT EXISTS idx_plays_partners ON plays(partner1_id, partner2_id);

-- Ensure only one live play per partner combination at a time
-- Using a partial unique index for live plays only
-- Note: This uses a functional index to normalize partner IDs
CREATE UNIQUE INDEX IF NOT EXISTS idx_plays_unique_live 
    ON plays(game_id, 
             CASE WHEN partner1_id < partner2_id THEN partner1_id ELSE partner2_id END,
             CASE WHEN partner1_id < partner2_id THEN partner2_id ELSE partner1_id END) 
    WHERE is_live = true;

