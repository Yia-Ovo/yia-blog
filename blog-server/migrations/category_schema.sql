-- 分类表结构归档（非迁移脚本）。
-- 用途：docs/sql 下原本只有 add_* 增量脚本，缺少 category 建表定义，导致 review 时无法在仓库内核对表结构。
-- 本文件记录当前线上表定义，与 infra/model/category_model.go 的 gorm tag 逐列对齐；改表时请同步更新两侧。
--
-- 已知限制：idx_name 是 name 单列唯一索引，与 gorm.DeletedAt 软删除叠加后，
-- 已软删除的旧行仍占用该名称，无法再创建同名分类。
-- 如需支持“删除后重建同名”，须改为 (name, deleted_at) 联合唯一索引，属独立表结构变更。

CREATE TABLE `category` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `name` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '分类名称',
  `sort` int NOT NULL DEFAULT '0' COMMENT '排序值，越小越靠前',
  `status` tinyint(1) NOT NULL DEFAULT '1' COMMENT '状态：1-正常，0-禁用',
  `create_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='文章分类表';
