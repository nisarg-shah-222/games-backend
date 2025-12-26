-- Would You Rather game sessions table
CREATE TABLE IF NOT EXISTS wyr_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    partnership_id UUID NOT NULL REFERENCES partnerships(id) ON DELETE CASCADE,
    question TEXT NOT NULL,
    option_a TEXT NOT NULL,
    option_b TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, completed
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- Would You Rather game answers table
CREATE TABLE IF NOT EXISTS wyr_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES wyr_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    choice VARCHAR(1) NOT NULL, -- 'A' or 'B'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(session_id, user_id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_wyr_sessions_partnership ON wyr_sessions(partnership_id);
CREATE INDEX IF NOT EXISTS idx_wyr_sessions_status ON wyr_sessions(status);
CREATE INDEX IF NOT EXISTS idx_wyr_answers_session ON wyr_answers(session_id);
CREATE INDEX IF NOT EXISTS idx_wyr_answers_user ON wyr_answers(user_id);

