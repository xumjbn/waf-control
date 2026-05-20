-- waf-control · policies.category 防护模块字符串
-- 与 deploy/modsec/rules.d/<category>/ 的目录名一一对应
-- (sqli / xss / rce / lfi-rfi / bot / rate-limit / ip-reputation / virtual-patches)
-- 用户自建规则可填 'custom'，前端按此分类做 chip 过滤

BEGIN;

ALTER TABLE policies ADD COLUMN IF NOT EXISTS category VARCHAR(32) NOT NULL DEFAULT 'custom';
CREATE INDEX IF NOT EXISTS idx_policies_category ON policies(category);

COMMIT;
