-- Migration 12: IGA Campaign Items
-- Per-user review items within a campaign (who gets reviewed, what decision).

CREATE TABLE IF NOT EXISTS iga_campaign_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id  UUID NOT NULL REFERENCES iga_campaigns(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL,
    role_id      UUID NOT NULL,
    decision     TEXT,  -- approve/revoke/modify/pending
    decided_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_iga_items_campaign ON iga_campaign_items (campaign_id);
CREATE INDEX IF NOT EXISTS idx_iga_items_user ON iga_campaign_items (user_id, decision);
