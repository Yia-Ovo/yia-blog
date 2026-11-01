-- 点赞关系持久化；先在目标数据库执行建表，再部署使用该表的代码。
-- comment 与 comment_like 都必须使用 InnoDB，关系与计数才能在事务中一起提交/回滚。
-- 本迁移保留 comment.likes，不重置、不凭空重建历史贡献者。
-- 切换前暂停点赞写入；仍存活的 Redis Bitmap 可导出为 (comment_id, user_id) 关系回填，
-- 回填仅 INSERT 关系，不再次增加 comment.likes。已过期 Bitmap 无法恢复，边界见收尾文档。
CREATE TABLE IF NOT EXISTS `comment_like` (
    `comment_id` BIGINT UNSIGNED NOT NULL COMMENT '评论ID',
    `user_id` BIGINT UNSIGNED NOT NULL COMMENT '点赞用户ID',
    `create_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '点赞时间',
    PRIMARY KEY (`comment_id`, `user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户评论点赞关系';
