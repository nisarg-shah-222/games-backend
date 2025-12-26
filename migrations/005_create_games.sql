-- Games table - stores all available games
CREATE TABLE IF NOT EXISTS games (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    icon VARCHAR(10), -- emoji or icon identifier
    details JSONB NOT NULL DEFAULT '{}', -- JSON for all game-related details
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_games_name ON games(name);

-- Insert default games
INSERT INTO games (id, name, description, icon, details) VALUES
    ('550e8400-e29b-41d4-a716-446655440001', 'Bulls and Cows', 'Set a secret 4-digit number and guess your partner''s', '🎯', '{"type": "bulls_and_cows"}'::jsonb)
ON CONFLICT (id) DO NOTHING;

