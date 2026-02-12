-- 1. 将组织成员关系索引升级为唯一索引，防止同一组织重复成员关系
ALTER TABLE `org_members`
DROP INDEX `idx_org_vol`,
ADD UNIQUE INDEX `uk_org_volunteer` (`org_id`, `volunteer_id`);
