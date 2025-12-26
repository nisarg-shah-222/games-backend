-- Game requests table - stores requests to play games
CREATE TABLE IF NOT EXISTS game_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    requester_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    partner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, accepted, rejected, expired
    expires_at TIMESTAMP NOT NULL, -- Request valid for 24 hours
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_game_requests_game ON game_requests(game_id);
CREATE INDEX IF NOT EXISTS idx_game_requests_requester ON game_requests(requester_id);
CREATE INDEX IF NOT EXISTS idx_game_requests_partner ON game_requests(partner_id);
CREATE INDEX IF NOT EXISTS idx_game_requests_status ON game_requests(status);
CREATE INDEX IF NOT EXISTS idx_game_requests_expires ON game_requests(expires_at);

-- Ensure only one pending request per game-partner pair at a time
CREATE UNIQUE INDEX IF NOT EXISTS idx_game_requests_unique_pending 
    ON game_requests(game_id, requester_id, partner_id) 
    WHERE status = 'pending';

