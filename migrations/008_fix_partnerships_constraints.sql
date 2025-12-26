-- Fix partnerships table constraints
-- Drop migration-created unique constraints (now managed by GORM)
-- This migration ensures GORM can manage the constraints going forward
DO $$
BEGIN
    -- Drop migration-created unique constraints if they exist
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'partnerships_user1_id_key' 
        AND conrelid = 'partnerships'::regclass
    ) THEN
        ALTER TABLE partnerships DROP CONSTRAINT partnerships_user1_id_key;
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'partnerships_user2_id_key' 
        AND conrelid = 'partnerships'::regclass
    ) THEN
        ALTER TABLE partnerships DROP CONSTRAINT partnerships_user2_id_key;
    END IF;
END $$;

-- GORM will create the unique constraints via AutoMigrate with its own naming

