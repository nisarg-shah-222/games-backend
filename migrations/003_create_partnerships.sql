-- Add display_name column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS display_name VARCHAR(100);

-- Partner requests table
CREATE TABLE IF NOT EXISTS partner_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_email VARCHAR(255) NOT NULL,
    recipient_id UUID REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, accepted, rejected, cancelled
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(sender_id, recipient_email)
);

-- Partnerships table (active pairings)
-- Note: Unique constraints on user1_id and user2_id are managed by GORM via uniqueIndex tags
CREATE TABLE IF NOT EXISTS partnerships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user1_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user2_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_partner_requests_sender ON partner_requests(sender_id);
CREATE INDEX IF NOT EXISTS idx_partner_requests_recipient_email ON partner_requests(recipient_email);
CREATE INDEX IF NOT EXISTS idx_partner_requests_recipient_id ON partner_requests(recipient_id);
CREATE INDEX IF NOT EXISTS idx_partner_requests_status ON partner_requests(status);
CREATE INDEX IF NOT EXISTS idx_partnerships_user1 ON partnerships(user1_id);
CREATE INDEX IF NOT EXISTS idx_partnerships_user2 ON partnerships(user2_id);

