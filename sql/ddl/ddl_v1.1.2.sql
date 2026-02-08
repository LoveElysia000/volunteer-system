-- 1. 修改活动报名表状态备注
ALTER TABLE `activity_signups`
MODIFY COLUMN `status` TINYINT NOT NULL DEFAULT 1
COMMENT '状态: 1-待审核, 2-报名成功, 3-报名驳回, 4-已取消';

-- 2. 修改组织成员表状态备注
ALTER TABLE `org_members`
MODIFY COLUMN `status` TINYINT NOT NULL DEFAULT 1
COMMENT '成员状态: 1-待审核, 2-正式成员, 3-申请驳回, 4-已退出';

-- 3. 修改组织档案表审核状态备注
ALTER TABLE `organizations`
MODIFY COLUMN `audit_status` TINYINT NOT NULL DEFAULT 0
COMMENT '资质审核状态: 0-未提交, 1-审核中, 2-已通过, 3-已驳回';

-- 4. 修改志愿者档案表实名状态备注
ALTER TABLE `volunteers`
MODIFY COLUMN `audit_status` TINYINT NOT NULL DEFAULT 0
COMMENT '实名认证状态: 0-未认证, 1-审核中, 2-已通过, 3-已驳回';