-- Add token version columns for JWT invalidation.
-- Run once before deploying code that reads/writes token_version.

ALTER TABLE `user`
  ADD COLUMN `token_version` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '令牌版本号' AFTER `status`;

ALTER TABLE `admin`
  ADD COLUMN `token_version` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '令牌版本号' AFTER `status`;
