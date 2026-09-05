-- 1. ตารางเก็บบัญชี (Accounts)
-- balance เก็บเป็น BIGINT (หน่วยสตางค์ หรือ Cents) เพื่อป้องกัน Floating Point Error
CREATE TABLE IF NOT EXISTS accounts (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    account_type VARCHAR(50) NOT NULL, -- เช่น 'SAVINGS', 'INVESTMENT', 'SYSTEM_RESERVE'
    currency VARCHAR(10) NOT NULL DEFAULT 'THB',
    balance BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. ตารางหัวธุรกรรม (Transactions) + Idempotency
CREATE TABLE IF NOT EXISTS transactions (
    id VARCHAR(64) PRIMARY KEY, -- UUID
    idempotency_key VARCHAR(128) UNIQUE NOT NULL, -- กันกดยิงซ้ำ
    transaction_type VARCHAR(50) NOT NULL, -- 'TRANSFER', 'DCA_BUY', 'DEPOSIT'
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 3. ตารางรายการบัญชีคู่ (Entries) -> ห้าม UPDATE / DELETE (Append-Only)
CREATE TABLE IF NOT EXISTS entries (
    id SERIAL PRIMARY KEY,
    transaction_id VARCHAR(64) NOT NULL REFERENCES transactions(id),
    account_id INT NOT NULL REFERENCES accounts(id),
    amount BIGINT NOT NULL, -- ค่าบวก (+) = เงินเข้า (Debit), ค่าลบ (-) = เงินออก (Credit)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index สำหรับค้นหาประวัติ Ledger ได้ไว
CREATE INDEX IF NOT EXISTS idx_entries_account_id ON entries(account_id);
CREATE INDEX IF NOT EXISTS idx_entries_transaction_id ON entries(transaction_id);

-- 4. ตารางแผนการลงทุนอัตโนมัติ (DCA Plans)
CREATE TABLE IF NOT EXISTS dca_plans (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    from_account_id INT NOT NULL REFERENCES accounts(id),
    asset_symbol VARCHAR(20) NOT NULL, -- เช่น 'VOO', 'BTC', 'SET50'
    amount BIGINT NOT NULL CHECK (amount > 0),
    interval_seconds INT NOT NULL DEFAULT 60, -- รอบเวลาตัดเงิน (วินาที)
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE', -- 'ACTIVE', 'PAUSED'
    last_run_at TIMESTAMP WITH TIME ZONE,
    next_run_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Seed บัญชีจำลองสำหรับทดสอบ
INSERT INTO accounts (id, user_id, account_type, currency, balance) VALUES
(1, 101, 'SAVINGS', 'THB', 10000000),     -- User 101 มีเงิน 100,000.00 บาท (10,000,000 สตางค์)
(2, 102, 'SAVINGS', 'THB', 5000000),      -- User 102 มีเงิน 50,000.00 บาท
(3, 101, 'INVESTMENT', 'THB', 0)          -- บัญชีพอร์ตลงทุนของ User 101
ON CONFLICT (id) DO NOTHING;