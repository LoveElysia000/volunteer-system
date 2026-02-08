-- 1. 先将 NULL 值更新为默认值（避免修改 NOT NULL 时报错）
UPDATE audit_records SET old_status = 0 WHERE old_status IS NULL;
UPDATE audit_records SET old_content = '' WHERE old_content IS NULL;
UPDATE audit_records SET new_content = '' WHERE new_content IS NULL;
UPDATE audit_records SET reject_reason = '' WHERE reject_reason IS NULL;

-- 2. 修改字段为 NOT NULL
ALTER TABLE audit_records 
    MODIFY COLUMN old_status INT NOT NULL COMMENT '变更前状态';

ALTER TABLE audit_records 
    MODIFY COLUMN old_content TEXT NOT NULL COMMENT '变更前数据快照(JSON形式)';

ALTER TABLE audit_records 
    MODIFY COLUMN new_content TEXT NOT NULL COMMENT '变更后数据快照(JSON形式)';

ALTER TABLE audit_records 
    MODIFY COLUMN reject_reason VARCHAR(500) NOT NULL COMMENT '驳回原因/备注';

-- 新增操作类型字段
ALTER TABLE audit_records ADD COLUMN operation_type TINYINT NOT NULL DEFAULT 0 COMMENT '操作类型: 1-新增, 2-更新, 3-删除';
