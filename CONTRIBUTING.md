# 参与开发

先阅读 [README](README.md)、[本地联调](docs/local-integration.md)和[发布入门](docs/deployment-and-public-release-guide.md)。功能建议与普通缺陷通过 Issue 讨论；安全问题按 [SECURITY.md](SECURITY.md) 私下报告。

后端遵守 Handler → App → Domain → Repo 分层：Handler 只处理 HTTP；App 编排用例与缓存；Domain 保证权限、归属和状态规则；Repo 实现持久化。新入口也必须经过领域规则。数据库通过 docs/sql 手动迁移，不启用 AutoMigrate。

请保持改动范围清晰，描述问题、方案、测试结果和兼容性。后端变更在 blog-server 执行 go test ./...、go vet ./... 和 go build ./...；前端变更在 blog-web 执行 npm test 和 npm run build。需要真实数据库/Redis 的检查请按联调说明单独运行。

不要提交真实配置、密钥、数据库备份、上传目录、node_modules、dist 或个人资料。配置使用占位符模板。新增功能应补有意义的行为回归；说明未验证的环境或外部依赖。
