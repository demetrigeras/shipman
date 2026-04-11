-- +goose Up
CREATE TABLE IF NOT EXISTS shipman.documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    charter_detail_id UUID REFERENCES shipman.charter_details(id) ON DELETE SET NULL,
    uploaded_by UUID NOT NULL REFERENCES shipman.users(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    storage_path TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'uploaded', -- uploaded | processing | processed | failed
    extracted_text TEXT,
    ai_analysis JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_documents_uploaded_by ON shipman.documents(uploaded_by);
CREATE INDEX idx_documents_charter_detail_id ON shipman.documents(charter_detail_id);
CREATE INDEX idx_documents_status ON shipman.documents(status);

CREATE TRIGGER trg_documents_updated_at
    BEFORE UPDATE ON shipman.documents
    FOR EACH ROW
    EXECUTE FUNCTION shipman.set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS trg_documents_updated_at ON shipman.documents;
DROP TABLE IF EXISTS shipman.documents;
