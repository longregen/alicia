ALTER TABLE user_preferences
    ADD COLUMN IF NOT EXISTS max_tool_iterations INTEGER NOT NULL DEFAULT 10;
