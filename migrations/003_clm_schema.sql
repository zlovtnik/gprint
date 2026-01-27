-- CLM (Contract Lifecycle Management) Schema
-- Migration: 003_clm_schema.sql
-- 
-- This migration creates the comprehensive contract lifecycle management tables
-- supporting: parties, contracts, obligations, workflows, documents, and audit trail.

-- ==============================================================================
-- PARTIES (Organizations/Individuals)
-- ==============================================================================
CREATE TABLE clm_parties (
    party_id            RAW(16) DEFAULT SYS_GUID() PRIMARY KEY,
    tenant_id           VARCHAR2(100) NOT NULL,
    party_type          VARCHAR2(20) NOT NULL, -- ORGANIZATION, INDIVIDUAL
    name                VARCHAR2(500) NOT NULL,
    legal_name          VARCHAR2(500),
    tax_id              VARCHAR2(50),
    registration_number VARCHAR2(100),
    
    -- Contact
    email               VARCHAR2(255),
    phone               VARCHAR2(50),
    website             VARCHAR2(255),
    
    -- Address
    address_line1       VARCHAR2(255),
    address_line2       VARCHAR2(255),
    city                VARCHAR2(100),
    state_province      VARCHAR2(100),
    postal_code         VARCHAR2(20),
    country_code        VARCHAR2(2),
    
    -- Risk Assessment
    risk_score          NUMBER(3), -- 0-100 (enforced by CHK_CLM_PARTY_RISK_SCORE)
    risk_level          VARCHAR2(20), -- LOW, MEDIUM, HIGH, CRITICAL
    
    -- Metadata
    tags                VARCHAR2(1000), -- JSON array
    notes               CLOB,
    
    -- Audit
    created_by          RAW(16) NOT NULL,
    created_at          TIMESTAMP DEFAULT SYSTIMESTAMP NOT NULL,
    updated_at          TIMESTAMP,
    is_active           NUMBER(1) DEFAULT 1 NOT NULL,
    
    CONSTRAINT chk_clm_party_type CHECK (party_type IN ('ORGANIZATION', 'INDIVIDUAL')),
    CONSTRAINT chk_clm_risk_level CHECK (risk_level IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    CONSTRAINT chk_clm_party_risk_score CHECK (risk_score BETWEEN 0 AND 100)
);

CREATE INDEX idx_clm_party_tenant ON clm_parties(tenant_id);
CREATE INDEX idx_clm_party_name ON clm_parties(tenant_id, UPPER(name));
CREATE INDEX idx_clm_party_tax_id ON clm_parties(tenant_id, tax_id);
CREATE INDEX idx_clm_party_active ON clm_parties(tenant_id, is_active);

-- ==============================================================================
-- CONTRACT TYPES
-- ==============================================================================
CREATE TABLE clm_contract_types (
    contract_type_id    RAW(16) DEFAULT SYS_GUID() PRIMARY KEY,
    tenant_id           VARCHAR2(100) NOT NULL,
    name                VARCHAR2(100) NOT NULL,
    description         VARCHAR2(500),
    category            VARCHAR2(50), -- SALES, PROCUREMENT, HR, PARTNERSHIP, etc.
    default_template_id RAW(16),
    workflow_config     CLOB, -- JSON workflow definition
    retention_years     NUMBER(3) DEFAULT 7,
    is_active           NUMBER(1) DEFAULT 1 NOT NULL,
    created_at          TIMESTAMP DEFAULT SYSTIMESTAMP NOT NULL,
    
    CONSTRAINT uk_clm_contract_type UNIQUE (tenant_id, name)
);

CREATE INDEX idx_clm_ctype_tenant ON clm_contract_types(tenant_id);

-- ==============================================================================
-- CONTRACTS (Master Table)
-- ==============================================================================
CREATE TABLE clm_contracts (
    contract_id         RAW(16) DEFAULT SYS_GUID() PRIMARY KEY,
    tenant_id           VARCHAR2(100) NOT NULL,
    contract_number     VARCHAR2(50) NOT NULL,
    title               VARCHAR2(500) NOT NULL,
    description         CLOB,
    contract_type_id    RAW(16) NOT NULL,
    status              VARCHAR2(50) NOT NULL, -- DRAFT, IN_REVIEW, APPROVED, EXECUTED, ACTIVE, EXPIRED, TERMINATED, CANCELLED
    version             NUMBER(10) DEFAULT 1 NOT NULL,
    parent_contract_id  RAW(16), -- For amendments/renewals
    
    -- Parties
    primary_party_id    RAW(16) NOT NULL,
    counterparty_id     RAW(16) NOT NULL,
    
    -- Dates
    start_date          DATE NOT NULL,
    end_date            DATE NOT NULL,
    execution_date      DATE,
    effective_date      DATE,
    notice_period_days  NUMBER(5),
    
    -- Financial
    total_value         NUMBER(20,2),
    currency_code       VARCHAR2(3),
    payment_terms       VARCHAR2(100),
    
    -- Metadata
    tags                VARCHAR2(1000), -- JSON array
    custom_fields       CLOB, -- JSON object
    
    -- Audit
    created_by          RAW(16) NOT NULL,
    created_at          TIMESTAMP DEFAULT SYSTIMESTAMP NOT NULL,
    updated_by          RAW(16),
    updated_at          TIMESTAMP,
    is_deleted          NUMBER(1) DEFAULT 0 NOT NULL,
    
    CONSTRAINT uk_clm_contract_number UNIQUE (tenant_id, contract_number),
    CONSTRAINT fk_clm_contract_type FOREIGN KEY (contract_type_id) 
        REFERENCES clm_contract_types(contract_type_id),
    CONSTRAINT fk_clm_primary_party FOREIGN KEY (primary_party_id) 
        REFERENCES clm_parties(party_id),
    CONSTRAINT fk_clm_counterparty FOREIGN KEY (counterparty_id) 
        REFERENCES clm_parties(party_id),
    CONSTRAINT fk_clm_parent_contract FOREIGN KEY (parent_contract_id) 
        REFERENCES clm_contracts(contract_id),
    CONSTRAINT chk_clm_status CHECK (status IN ('DRAFT', 'IN_REVIEW', 'APPROVED', 
        'EXECUTED', 'ACTIVE', 'EXPIRED', 'TERMINATED', 'CANCELLED')),
    CONSTRAINT chk_clm_contract_end_after_start CHECK (end_date >= start_date)
);

CREATE INDEX idx_clm_contract_tenant ON clm_contracts(tenant_id);
CREATE INDEX idx_clm_contract_status ON clm_contracts(tenant_id, status, end_date);
CREATE INDEX idx_clm_contract_parties ON clm_contracts(tenant_id, primary_party_id, counterparty_id);
CREATE INDEX idx_clm_contract_dates ON clm_contracts(tenant_id, start_date, end_date);
CREATE INDEX idx_clm_contract_type ON clm_contracts(tenant_id, contract_type_id);

-- ==============================================================================
-- CONTRACT VERSIONS (Full History)
-- ==============================================================================
CREATE TABLE clm_contract_versions (
    version_id          RAW(16) DEFAULT SYS_GUID() PRIMARY KEY,
    tenant_id           VARCHAR2(100) NOT NULL,
    contract_id         RAW(16) NOT NULL,
    version_number      NUMBER(10) NOT NULL,
    version_data        CLOB NOT NULL, -- Complete contract snapshot as JSON
    changes_summary     CLOB,
    created_by          RAW(16) NOT NULL,
    created_at          TIMESTAMP DEFAULT SYSTIMESTAMP NOT NULL,
    
    CONSTRAINT fk_clm_cv_contract FOREIGN KEY (contract_id) 
        REFERENCES clm_contracts(contract_id),
    CONSTRAINT uk_clm_contract_version UNIQUE (contract_id, version_number)
);

CREATE INDEX idx_clm_cv_contract ON clm_contract_versions(contract_id);

-- ==============================================================================
-- CONTRACT DOCUMENTS
-- ==============================================================================
CREATE TABLE clm_documents (
    document_id         RAW(16) DEFAULT SYS_GUID() PRIMARY KEY,
    tenant_id           VARCHAR2(100) NOT NULL,
    contract_id         RAW(16) NOT NULL,
    document_type       VARCHAR2(50) NOT NULL, -- MAIN_CONTRACT, AMENDMENT, EXHIBIT, SUPPORTING
    filename            VARCHAR2(255) NOT NULL,
    file_size           NUMBER(15),
    mime_type           VARCHAR2(100),
    storage_path        VARCHAR2(1000) NOT NULL, -- S3/Object Storage path
    checksum            VARCHAR2(64), -- SHA-256 hash
    version             NUMBER(5) DEFAULT 1 NOT NULL,
    
    -- Document processing
    text_extracted      NUMBER(1) DEFAULT 0,
    text_content        CLOB, -- For full-text search
    
    uploaded_by         RAW(16) NOT NULL,
    uploaded_at         TIMESTAMP DEFAULT SYSTIMESTAMP NOT NULL,
    
    CONSTRAINT fk_clm_doc_contract FOREIGN KEY (contract_id) 
        REFERENCES clm_contracts(contract_id),
    CONSTRAINT chk_clm_doc_type CHECK (document_type IN ('MAIN_CONTRACT', 'AMENDMENT', 'EXHIBIT', 'SUPPORTING'))
);

CREATE INDEX idx_clm_doc_contract ON clm_documents(contract_id);
CREATE INDEX idx_clm_doc_tenant ON clm_documents(tenant_id);

-- ==============================================================================
-- DOCUMENT TEMPLATES
-- ==============================================================================
CREATE TABLE clm_templates (
    template_id         RAW(16) DEFAULT SYS_GUID() PRIMARY KEY,
    tenant_id           VARCHAR2(100) NOT NULL,
    name                VARCHAR2(255) NOT NULL,
    description         VARCHAR2(1000),
    contract_type_id    RAW(16),
    template_content    CLOB NOT NULL, -- HTML with merge fields
    merge_fields        CLOB, -- JSON array of field definitions
    version             NUMBER(5) DEFAULT 1 NOT NULL,
    is_active           NUMBER(1) DEFAULT 1 NOT NULL,
    created_at          TIMESTAMP DEFAULT SYSTIMESTAMP NOT NULL,
    
    CONSTRAINT fk_clm_template_type FOREIGN KEY (contract_type_id) 
        REFERENCES clm_contract_types(contract_type_id)
);

CREATE INDEX idx_clm_template_tenant ON clm_templates(tenant_id);

-- Add FK constraint for default_template_id (after clm_templates is created)
ALTER TABLE clm_contract_types ADD CONSTRAINT fk_clm_ctype_template 
    FOREIGN KEY (default_template_id) REFERENCES clm_templates(template_id);

-- ==============================================================================
-- WORKFLOW INSTANCES
-- ==============================================================================
CREATE TABLE clm_workflow_instances (
    workflow_id         RAW(16) DEFAULT SYS_GUID() PRIMARY KEY,
    tenant_id           VARCHAR2(100) NOT NULL,
    contract_id         RAW(16) NOT NULL,
    workflow_type       VARCHAR2(50) NOT NULL, -- APPROVAL, REVIEW, SIGNATURE, RENEWAL
    status              VARCHAR2(50) NOT NULL, -- PENDING, IN_PROGRESS, COMPLETED, CANCELLED
    current_step        NUMBER(3) DEFAULT 1,
    total_steps         NUMBER(3),
    
    started_by          RAW(16) NOT NULL,
    started_at          TIMESTAMP DEFAULT SYSTIMESTAMP NOT NULL,
    completed_at        TIMESTAMP,
    
    CONSTRAINT fk_clm_wf_contract FOREIGN KEY (contract_id) 
        REFERENCES clm_contracts(contract_id),
    CONSTRAINT chk_clm_wf_type CHECK (workflow_type IN ('APPROVAL', 'REVIEW', 'SIGNATURE', 'RENEWAL')),
    CONSTRAINT chk_clm_wf_status CHECK (status IN ('PENDING', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED'))
);

CREATE INDEX idx_clm_wf_contract ON clm_workflow_instances(contract_id);
CREATE INDEX idx_clm_wf_tenant ON clm_workflow_instances(tenant_id);
CREATE INDEX idx_clm_wf_status ON clm_workflow_instances(tenant_id, status);

-- ==============================================================================
-- WORKFLOW STEPS
-- ==============================================================================
CREATE TABLE clm_workflow_steps (
    step_id             RAW(16) DEFAULT SYS_GUID() PRIMARY KEY,
    tenant_id           VARCHAR2(100) NOT NULL,
    workflow_id         RAW(16) NOT NULL,
    step_number         NUMBER(3) NOT NULL,
    step_type           VARCHAR2(50) NOT NULL, -- APPROVAL, REVIEW, SIGNATURE, NOTIFICATION
    step_name           VARCHAR2(255) NOT NULL,
    
    assigned_to         RAW(16), -- User ID
    assigned_to_role    VARCHAR2(100), -- Or role name
    
    status              VARCHAR2(50) NOT NULL, -- PENDING, IN_PROGRESS, APPROVED, REJECTED, COMPLETED, SKIPPED
    due_date            TIMESTAMP,
    
    action_taken        VARCHAR2(50),
    comments            CLOB,
    action_by           RAW(16),
    action_at           TIMESTAMP,
    
    parallel_group      NUMBER(3), -- For parallel approvals
    
    CONSTRAINT fk_clm_ws_workflow FOREIGN KEY (workflow_id) 
        REFERENCES clm_workflow_instances(workflow_id),
    CONSTRAINT uk_clm_workflow_step UNIQUE (workflow_id, step_number),
    CONSTRAINT chk_clm_step_type CHECK (step_type IN ('APPROVAL', 'REVIEW', 'SIGNATURE', 'NOTIFICATION')),
    CONSTRAINT chk_clm_step_status CHECK (status IN ('PENDING', 'IN_PROGRESS', 'APPROVED', 'REJECTED', 'COMPLETED', 'SKIPPED'))
);

CREATE INDEX idx_clm_ws_workflow ON clm_workflow_steps(workflow_id);
CREATE INDEX idx_clm_ws_assigned ON clm_workflow_steps(assigned_to, status);

-- ==============================================================================
-- CONTRACT OBLIGATIONS
-- ==============================================================================
CREATE TABLE clm_obligations (
    obligation_id       RAW(16) DEFAULT SYS_GUID() PRIMARY KEY,
    tenant_id           VARCHAR2(100) NOT NULL,
    contract_id         RAW(16) NOT NULL,
    obligation_type     VARCHAR2(50) NOT NULL, -- DELIVERABLE, PAYMENT, MILESTONE, COMPLIANCE, SLA
    title               VARCHAR2(500) NOT NULL,
    description         CLOB,
    
    responsible_party_id RAW(16) NOT NULL, -- Who must fulfill
    
    due_date            DATE NOT NULL,
    completion_date     DATE,
    status              VARCHAR2(50) NOT NULL, -- PENDING, IN_PROGRESS, COMPLETED, OVERDUE, WAIVED
    
    -- For financial obligations
    amount              NUMBER(20,2),
    currency_code       VARCHAR2(3),
    
    -- Recurrence
    is_recurring        NUMBER(1) DEFAULT 0,
    recurrence_pattern  VARCHAR2(50), -- DAILY, WEEKLY, MONTHLY, QUARTERLY, YEARLY
    recurrence_end_date DATE,
    
    priority            VARCHAR2(20) DEFAULT 'MEDIUM',
    
    created_by          RAW(16) NOT NULL,
    created_at          TIMESTAMP DEFAULT SYSTIMESTAMP NOT NULL,
    updated_at          TIMESTAMP,
    
    CONSTRAINT fk_clm_obl_contract FOREIGN KEY (contract_id) 
        REFERENCES clm_contracts(contract_id),
    CONSTRAINT fk_clm_obl_party FOREIGN KEY (responsible_party_id) 
        REFERENCES clm_parties(party_id),
    CONSTRAINT chk_clm_obl_type CHECK (obligation_type IN ('DELIVERABLE', 'PAYMENT', 'MILESTONE', 'COMPLIANCE', 'SLA')),
    CONSTRAINT chk_clm_obl_status CHECK (status IN ('PENDING', 'IN_PROGRESS', 'COMPLETED', 'OVERDUE', 'WAIVED')),
    CONSTRAINT chk_clm_obl_priority CHECK (priority IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL'))
);

CREATE INDEX idx_clm_obl_contract ON clm_obligations(contract_id);
CREATE INDEX idx_clm_obl_tenant ON clm_obligations(tenant_id);
CREATE INDEX idx_clm_obl_due_date ON clm_obligations(tenant_id, due_date, status);
CREATE INDEX idx_clm_obl_party ON clm_obligations(responsible_party_id);

-- ==============================================================================
-- OBLIGATION UPDATES/TRACKING
-- ==============================================================================
CREATE TABLE clm_obligation_updates (
    update_id           RAW(16) DEFAULT SYS_GUID() PRIMARY KEY,
    tenant_id           VARCHAR2(100) NOT NULL,
    obligation_id       RAW(16) NOT NULL,
    update_type         VARCHAR2(50) NOT NULL, -- STATUS_CHANGE, COMMENT, ATTACHMENT, REMINDER
    old_status          VARCHAR2(50),
    new_status          VARCHAR2(50),
    notes               CLOB,
    updated_by          RAW(16) NOT NULL,
    updated_at          TIMESTAMP DEFAULT SYSTIMESTAMP NOT NULL,
    
    CONSTRAINT fk_clm_ou_obligation FOREIGN KEY (obligation_id) 
        REFERENCES clm_obligations(obligation_id)
);

CREATE INDEX idx_clm_ou_obligation ON clm_obligation_updates(obligation_id);

-- ==============================================================================
-- AUDIT TRAIL (Immutable)
-- ==============================================================================
CREATE TABLE clm_audit_trail (
    audit_id            RAW(16) DEFAULT SYS_GUID() PRIMARY KEY,
    tenant_id           VARCHAR2(100) NOT NULL,
    
    entity_type         VARCHAR2(50) NOT NULL, -- CONTRACT, DOCUMENT, WORKFLOW, PARTY, OBLIGATION
    entity_id           RAW(16) NOT NULL,
    
    action              VARCHAR2(100) NOT NULL, -- CREATED, UPDATED, DELETED, APPROVED, etc.
    action_category     VARCHAR2(50), -- DATA_CHANGE, STATUS_CHANGE, ACCESS, SECURITY
    
    user_id             RAW(16) NOT NULL,
    user_name           VARCHAR2(255),
    user_role           VARCHAR2(100),
    
    ip_address          VARCHAR2(45),
    user_agent          VARCHAR2(500),
    
    old_values          CLOB, -- JSON of changed fields
    new_values          CLOB, -- JSON of changed fields
    
    audit_timestamp     TIMESTAMP DEFAULT SYSTIMESTAMP NOT NULL,
    
    session_id          VARCHAR2(100)
);

CREATE INDEX idx_clm_audit_entity ON clm_audit_trail(entity_type, entity_id);
CREATE INDEX idx_clm_audit_tenant ON clm_audit_trail(tenant_id);
CREATE INDEX idx_clm_audit_user ON clm_audit_trail(user_id, audit_timestamp);
CREATE INDEX idx_clm_audit_timestamp ON clm_audit_trail(audit_timestamp);

-- ==============================================================================
-- VIEWS
-- ==============================================================================

-- Active Contracts View
CREATE OR REPLACE VIEW v_clm_active_contracts AS
SELECT 
    c.*,
    ct.name as contract_type_name,
    ct.category as contract_type_category,
    pp.name as primary_party_name,
    cp.name as counterparty_name,
    CASE 
        WHEN c.end_date <= SYSDATE + 30 THEN 'EXPIRING_SOON'
        WHEN c.end_date <= SYSDATE + 90 THEN 'EXPIRING'
        ELSE 'ACTIVE'
    END as expiry_status,
    TRUNC(c.end_date - SYSDATE) as days_to_expiry
FROM clm_contracts c
JOIN clm_contract_types ct ON c.contract_type_id = ct.contract_type_id
JOIN clm_parties pp ON c.primary_party_id = pp.party_id
JOIN clm_parties cp ON c.counterparty_id = cp.party_id
WHERE c.status IN ('ACTIVE', 'EXECUTED')
AND c.is_deleted = 0;

-- Pending Approvals View
CREATE OR REPLACE VIEW v_clm_pending_approvals AS
SELECT 
    ws.*,
    wi.contract_id,
    c.contract_number,
    c.title as contract_title
FROM clm_workflow_steps ws
JOIN clm_workflow_instances wi ON ws.workflow_id = wi.workflow_id
JOIN clm_contracts c ON wi.contract_id = c.contract_id
WHERE ws.status IN ('PENDING', 'IN_PROGRESS')
AND wi.status != 'CANCELLED';

-- Overdue Obligations View
CREATE OR REPLACE VIEW v_clm_overdue_obligations AS
SELECT 
    o.*,
    c.contract_number,
    c.title as contract_title,
    p.name as responsible_party_name,
    TRUNC(SYSDATE - o.due_date) as days_overdue
FROM clm_obligations o
JOIN clm_contracts c ON o.contract_id = c.contract_id
JOIN clm_parties p ON o.responsible_party_id = p.party_id
WHERE o.status IN ('PENDING', 'IN_PROGRESS')
AND o.due_date < SYSDATE;

-- ==============================================================================
-- SEQUENCES (for generating contract numbers if needed)
-- ==============================================================================
CREATE SEQUENCE seq_clm_contract_number START WITH 1000 INCREMENT BY 1;
