-- 文章「用户作者模型」：新增作者用户ID 列。
-- 普通用户创建文章时记录其为作者；与 admin_id 二选一，非零者为作者。
-- 若 article 表已存在 user_id 列（模型已声明），请跳过本脚本。
-- Run once before deploying code that reads/writes article.user_id.

ALTER TABLE `article`
  ADD COLUMN `user_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '作者用户ID(与admin_id二选一，非零者为作者)' AFTER `id`,
  ADD INDEX `idx_user_id` (`user_id`);
