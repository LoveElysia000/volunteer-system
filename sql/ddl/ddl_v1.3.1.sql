-- ============================================
-- DDL Version: v1.3.1
-- Description: add RBAC role/permission/account-role tables
-- Created: 2026-03-08
-- ============================================

-- 1) RBAC 角色表
CREATE TABLE IF NOT EXISTS `rbac_roles` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `role_code` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '角色编码（唯一）',
    `role_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '角色名称',
    `description` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '角色描述',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1-启用，0-禁用',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_rbac_roles_role_code` (`role_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='RBAC 角色定义表';

-- 2) RBAC 权限点表
CREATE TABLE IF NOT EXISTS `rbac_permissions` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `resource` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '资源标识',
    `action` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '动作标识',
    `description` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '权限描述',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1-启用，0-禁用',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_rbac_permissions_resource_action` (`resource`, `action`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='RBAC 权限点定义表';

-- 3) RBAC 角色-权限关联表
CREATE TABLE IF NOT EXISTS `rbac_role_permissions` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `role_id` BIGINT NOT NULL DEFAULT 0 COMMENT '角色ID（关联 rbac_roles.id）',
    `permission_id` BIGINT NOT NULL DEFAULT 0 COMMENT '权限ID（关联 rbac_permissions.id）',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_rbac_role_permissions_role_permission` (`role_id`, `permission_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='RBAC 角色权限映射表';

-- 4) RBAC 账号-角色-作用域关联表
CREATE TABLE IF NOT EXISTS `rbac_account_roles` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `account_id` BIGINT NOT NULL DEFAULT 0 COMMENT '账号ID（关联 sys_accounts.id）',
    `role_id` BIGINT NOT NULL DEFAULT 0 COMMENT '角色ID（关联 rbac_roles.id）',
    `scope_type` VARCHAR(16) NOT NULL DEFAULT 'org' COMMENT '作用域类型：global/org',
    `scope_id` BIGINT NOT NULL DEFAULT 0 COMMENT '作用域ID（global 固定为0，org 为 organizations.id）',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1-启用，0-禁用',
    `granted_by` BIGINT NOT NULL DEFAULT 0 COMMENT '授权人账号ID',
    `expires_at` DATETIME NULL COMMENT '过期时间（为空表示不过期）',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_rbac_account_roles_binding` (`account_id`, `role_id`, `scope_type`, `scope_id`),
    KEY `idx_rbac_account_roles_scope` (`scope_type`, `scope_id`),
    KEY `idx_rbac_account_roles_account_status` (`account_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='RBAC 账号角色授权表';
