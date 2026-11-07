package consts

import "time"

const (
	AIRequestTimeout    = 3 * time.Second
	AIGenerationTimeout = 100 * time.Second
	AILeaseDuration     = 2 * time.Minute
	AIStateMaxBytes     = 16 << 20
	AIMessageMaxRunes   = 4000
	AIAnswerMaxRunes    = 12000
	AIMemoryMaxRunes    = 4000
	AIMaxConversations  = 30
	AIMaxMessages       = 200
	AIStatusPending     = "pending"
	AIStatusCompleted   = "completed"
	AIStatusFailed      = "failed"
	AIRoleUser          = "user"
	AIRoleAssistant     = "assistant"
	AIKeyEnv            = "YIA_AI_API_KEY"
	AIContextGeneral    = "general"
	AIContextProject    = "project"
	AIWorkerKeyEnv      = "YIA_AI_WORKER_KEY"
	AIImportMaxBytes    = 8 << 20
	AIImportMaxMessages = 20000
	AIImportMaxThreads  = 200
	AIJobQueued         = "queued"
	AIJobRunning        = "running"
	AIJobReview         = "review"
	AIJobPublishQueued  = "publish_queued"
	AIJobPublishing     = "publishing"
	AIJobCompleted      = "completed"
	AIJobFailed         = "failed"
	AIJobDailyLimit     = 2
	AIJobMaxRecords     = 30
	AIJobLeaseDuration  = 20 * time.Minute
)
