-- Migration: 004_seed_data.sql
-- Seed test data for development/testing
-- Note: tenant_id follows the pattern: tenant_<username> for users with public email domains
-- Change 'tenant_admin' to match your actual tenant_id if different

-- Check your tenant_id by logging in and looking at the JWT claims or API response

-- ============================================
-- CUSTOMERS
-- ============================================
INSERT INTO customers (tenant_id, customer_code, customer_type, name, trade_name, tax_id, email, phone, active, created_by)
VALUES ('tenant_admin', 'CUST001', 'COMPANY', 'Acme Corporation', 'Acme Corp', '12345678901234', 'contact@acme.com', '(11) 99999-0001', 1, 'seed_script');

INSERT INTO customers (tenant_id, customer_code, customer_type, name, trade_name, tax_id, email, phone, active, created_by)
VALUES ('tenant_admin', 'CUST002', 'COMPANY', 'TechStart Ltda', 'TechStart', '98765432109876', 'info@techstart.com.br', '(11) 98888-0002', 1, 'seed_script');

INSERT INTO customers (tenant_id, customer_code, customer_type, name, trade_name, tax_id, email, phone, active, created_by)
VALUES ('tenant_admin', 'CUST003', 'INDIVIDUAL', 'João Silva', NULL, '12345678901', 'joao.silva@email.com', '(21) 97777-0003', 1, 'seed_script');

INSERT INTO customers (tenant_id, customer_code, customer_type, name, trade_name, tax_id, email, phone, active, created_by)
VALUES ('tenant_admin', 'CUST004', 'COMPANY', 'Global Services SA', 'GlobalServ', '11122233344455', 'contato@globalserv.com.br', '(11) 96666-0004', 1, 'seed_script');

INSERT INTO customers (tenant_id, customer_code, customer_type, name, trade_name, tax_id, email, phone, active, created_by)
VALUES ('tenant_admin', 'CUST005', 'INDIVIDUAL', 'Maria Santos', NULL, '98765432100', 'maria.santos@gmail.com', '(31) 95555-0005', 1, 'seed_script');

COMMIT;

-- ============================================
-- SERVICES
-- ============================================
INSERT INTO services (tenant_id, service_code, name, description, category, unit_price, currency, price_unit, active, created_by)
VALUES ('tenant_admin', 'SVC001', 'Software Development', 'Custom software development services', 'Development', 250.00, 'BRL', 'HOUR', 1, 'seed_script');

INSERT INTO services (tenant_id, service_code, name, description, category, unit_price, currency, price_unit, active, created_by)
VALUES ('tenant_admin', 'SVC002', 'Technical Support', '24/7 technical support and maintenance', 'Support', 150.00, 'BRL', 'HOUR', 1, 'seed_script');

INSERT INTO services (tenant_id, service_code, name, description, category, unit_price, currency, price_unit, active, created_by)
VALUES ('tenant_admin', 'SVC003', 'Cloud Hosting', 'Managed cloud hosting services', 'Infrastructure', 500.00, 'BRL', 'MONTH', 1, 'seed_script');

INSERT INTO services (tenant_id, service_code, name, description, category, unit_price, currency, price_unit, active, created_by)
VALUES ('tenant_admin', 'SVC004', 'Consulting', 'IT consulting and advisory services', 'Consulting', 350.00, 'BRL', 'HOUR', 1, 'seed_script');

INSERT INTO services (tenant_id, service_code, name, description, category, unit_price, currency, price_unit, active, created_by)
VALUES ('tenant_admin', 'SVC005', 'Training', 'Technical training and workshops', 'Training', 2000.00, 'BRL', 'DAY', 1, 'seed_script');

INSERT INTO services (tenant_id, service_code, name, description, category, unit_price, currency, price_unit, active, created_by)
VALUES ('tenant_admin', 'SVC006', 'System Integration', 'API and system integration services', 'Development', 15000.00, 'BRL', 'PROJECT', 1, 'seed_script');

COMMIT;

-- ============================================
-- CONTRACTS
-- ============================================
-- Contract 1: Active service contract with Acme (100*250 + 200*150 = 55000)
INSERT INTO contracts (tenant_id, contract_number, contract_type, customer_id, start_date, end_date, total_value, billing_cycle, status, created_by)
SELECT 'tenant_admin', 'CTR-2024-001', 'SERVICE', c.id, DATE '2024-01-15', DATE '2025-01-14', 55000.00, 'MONTHLY', 'ACTIVE', 'seed_script'
FROM customers c WHERE c.tenant_id = 'tenant_admin' AND c.customer_code = 'CUST001';

-- Contract 2: Recurring contract with TechStart
INSERT INTO contracts (tenant_id, contract_number, contract_type, customer_id, start_date, end_date, total_value, billing_cycle, status, created_by)
SELECT 'tenant_admin', 'CTR-2024-002', 'RECURRING', c.id, DATE '2024-03-01', DATE '2025-02-28', 36000.00, 'MONTHLY', 'ACTIVE', 'seed_script'
FROM customers c WHERE c.tenant_id = 'tenant_admin' AND c.customer_code = 'CUST002';

-- Contract 3: Project contract with João Silva
INSERT INTO contracts (tenant_id, contract_number, contract_type, customer_id, start_date, end_date, total_value, billing_cycle, status, created_by)
SELECT 'tenant_admin', 'CTR-2024-003', 'PROJECT', c.id, DATE '2024-06-01', DATE '2024-12-31', 25000.00, 'PROJECT', 'ACTIVE', 'seed_script'
FROM customers c WHERE c.tenant_id = 'tenant_admin' AND c.customer_code = 'CUST003';

-- Contract 4: Draft contract with GlobalServ
INSERT INTO contracts (tenant_id, contract_number, contract_type, customer_id, start_date, total_value, billing_cycle, status, created_by)
SELECT 'tenant_admin', 'CTR-2024-004', 'SERVICE', c.id, DATE '2024-09-01', 75000.00, 'QUARTERLY', 'DRAFT', 'seed_script'
FROM customers c WHERE c.tenant_id = 'tenant_admin' AND c.customer_code = 'CUST004';

-- Contract 5: Pending contract with Maria Santos
INSERT INTO contracts (tenant_id, contract_number, contract_type, customer_id, start_date, end_date, total_value, billing_cycle, status, created_by)
SELECT 'tenant_admin', 'CTR-2024-005', 'SERVICE', c.id, DATE '2024-08-01', DATE '2025-07-31', 12000.00, 'MONTHLY', 'PENDING', 'seed_script'
FROM customers c WHERE c.tenant_id = 'tenant_admin' AND c.customer_code = 'CUST005';

COMMIT;

-- ============================================
-- CONTRACT ITEMS
-- ============================================
-- Items for Contract 1 (Acme - Development + Support)
INSERT INTO contract_items (tenant_id, contract_id, service_id, quantity, unit_price, description, status)
SELECT 'tenant_admin', c.id, s.id, 100, 250.00, 'Custom ERP development', 'IN_PROGRESS'
FROM contracts c, services s 
WHERE c.tenant_id = 'tenant_admin' AND c.contract_number = 'CTR-2024-001'
  AND s.tenant_id = 'tenant_admin' AND s.service_code = 'SVC001';

INSERT INTO contract_items (tenant_id, contract_id, service_id, quantity, unit_price, description, status)
SELECT 'tenant_admin', c.id, s.id, 200, 150.00, 'Technical support package', 'IN_PROGRESS'
FROM contracts c, services s 
WHERE c.tenant_id = 'tenant_admin' AND c.contract_number = 'CTR-2024-001'
  AND s.tenant_id = 'tenant_admin' AND s.service_code = 'SVC002';

-- Items for Contract 2 (TechStart - Cloud + Support)
INSERT INTO contract_items (tenant_id, contract_id, service_id, quantity, unit_price, description, status)
SELECT 'tenant_admin', c.id, s.id, 12, 2000.00, 'Premium cloud hosting - 12 months', 'IN_PROGRESS'
FROM contracts c, services s 
WHERE c.tenant_id = 'tenant_admin' AND c.contract_number = 'CTR-2024-002'
  AND s.tenant_id = 'tenant_admin' AND s.service_code = 'SVC003';

INSERT INTO contract_items (tenant_id, contract_id, service_id, quantity, unit_price, description, status)
SELECT 'tenant_admin', c.id, s.id, 80, 150.00, 'Monthly support hours', 'IN_PROGRESS'
FROM contracts c, services s 
WHERE c.tenant_id = 'tenant_admin' AND c.contract_number = 'CTR-2024-002'
  AND s.tenant_id = 'tenant_admin' AND s.service_code = 'SVC002';

-- Items for Contract 3 (João - Integration project)
INSERT INTO contract_items (tenant_id, contract_id, service_id, quantity, unit_price, description, status)
SELECT 'tenant_admin', c.id, s.id, 1, 15000.00, 'ERP to CRM integration', 'PENDING'
FROM contracts c, services s 
WHERE c.tenant_id = 'tenant_admin' AND c.contract_number = 'CTR-2024-003'
  AND s.tenant_id = 'tenant_admin' AND s.service_code = 'SVC006';

INSERT INTO contract_items (tenant_id, contract_id, service_id, quantity, unit_price, description, status)
SELECT 'tenant_admin', c.id, s.id, 5, 2000.00, 'Integration training days', 'PENDING'
FROM contracts c, services s 
WHERE c.tenant_id = 'tenant_admin' AND c.contract_number = 'CTR-2024-003'
  AND s.tenant_id = 'tenant_admin' AND s.service_code = 'SVC005';

COMMIT;

-- ============================================
-- PRINT JOBS
-- ============================================
-- Completed print job
INSERT INTO contract_print_jobs (tenant_id, contract_id, status, format, output_path, file_size, page_count, queued_at, started_at, completed_at, requested_by)
SELECT 'tenant_admin', c.id, 'COMPLETED', 'PDF', '/output/CTR-2024-001.pdf', 245678, 12, 
       CURRENT_TIMESTAMP - INTERVAL '2' DAY, 
       CURRENT_TIMESTAMP - INTERVAL '2' DAY + INTERVAL '5' SECOND,
       CURRENT_TIMESTAMP - INTERVAL '2' DAY + INTERVAL '15' SECOND, 
       'admin'
FROM contracts c WHERE c.tenant_id = 'tenant_admin' AND c.contract_number = 'CTR-2024-001';

-- Processing print job
INSERT INTO contract_print_jobs (tenant_id, contract_id, status, format, queued_at, started_at, requested_by)
SELECT 'tenant_admin', c.id, 'PROCESSING', 'PDF', 
       CURRENT_TIMESTAMP - INTERVAL '5' MINUTE,
       CURRENT_TIMESTAMP - INTERVAL '4' MINUTE,
       'admin'
FROM contracts c WHERE c.tenant_id = 'tenant_admin' AND c.contract_number = 'CTR-2024-002';

-- Queued print job
INSERT INTO contract_print_jobs (tenant_id, contract_id, status, format, queued_at, requested_by)
SELECT 'tenant_admin', c.id, 'QUEUED', 'DOCX', CURRENT_TIMESTAMP, 'admin'
FROM contracts c WHERE c.tenant_id = 'tenant_admin' AND c.contract_number = 'CTR-2024-003';

-- Failed print job with error
INSERT INTO contract_print_jobs (tenant_id, contract_id, status, format, queued_at, started_at, retry_count, error_message, requested_by)
SELECT 'tenant_admin', c.id, 'FAILED', 'PDF', 
       CURRENT_TIMESTAMP - INTERVAL '1' DAY,
       CURRENT_TIMESTAMP - INTERVAL '1' DAY + INTERVAL '2' SECOND,
       3, 'Template rendering failed: missing customer address', 'admin'
FROM contracts c WHERE c.tenant_id = 'tenant_admin' AND c.contract_number = 'CTR-2024-004';

COMMIT;

-- ============================================
-- VERIFICATION QUERIES
-- ============================================
-- Run these to verify the data was inserted correctly:
-- SELECT COUNT(*) as customer_count FROM customers WHERE tenant_id = 'tenant_admin';
-- SELECT COUNT(*) as service_count FROM services WHERE tenant_id = 'tenant_admin';
-- SELECT COUNT(*) as contract_count FROM contracts WHERE tenant_id = 'tenant_admin';
-- SELECT COUNT(*) as print_job_count FROM contract_print_jobs WHERE tenant_id = 'tenant_admin';
