-- 评论增加用户作者ID，用于记录评论归属并校验用户只能修改/删除自己的评论。
-- 当前用户评论写入 user_id，管理员评论保留 admin_id；二者按作者类型使用。
-- 若 comment 表已存在 user_id 列，请跳过本脚本。

ALTER TABLE `comment`
  ADD COLUMN `user_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户作者ID(与admin_id二选一)' AFTER `id`,
  ADD INDEX `idx_user_id` (`user_id`);
