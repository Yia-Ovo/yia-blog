package service

import (
	"blog-server/api/dto"
	"blog-server/common"
	"blog-server/common/utils/convert"
	"blog-server/domain/do"
	"context"
)

func (s *AIAppService) ImportedThreads(ctx context.Context, adminID uint64) ([]dto.AIImportedThread, error) {
	access, err := s.access(ctx, adminID)
	if err != nil {
		return nil, err
	}
	if err := s.domain.PrivateWorkspace(access); err != nil {
		return nil, err
	}
	result := []dto.AIImportedThread{}
	if s.imports == nil {
		return result, nil
	}
	values, err := s.imports.List(ctx, adminID)
	if err != nil {
		return nil, aiStorageError(err)
	}
	for _, value := range values {
		if err := s.domain.ValidateImportedThread(&value, access, false); err != nil {
			return nil, err
		}
		result = append(result, dto.AIImportedThread{ID: value.ID, Title: value.Title, UpdatedAt: value.UpdatedAt})
	}
	return result, nil
}

func (s *AIAppService) ImportedThread(ctx context.Context, adminID uint64, id string, before int) (*dto.AIImportedThread, error) {
	access, err := s.access(ctx, adminID)
	if err != nil {
		return nil, err
	}
	if err := s.domain.PrivateWorkspace(access); err != nil {
		return nil, err
	}
	if s.imports == nil {
		return nil, common.ParamErr.WithMsg("本地历史尚未同步。")
	}
	value, err := s.imports.Get(ctx, adminID, id)
	if err != nil {
		return nil, aiStorageError(err)
	}
	if err := s.domain.ValidateImportedThread(value, access, true); err != nil {
		return nil, err
	}
	messages, next, err := s.domain.ImportedPage(value, access, before)
	if err != nil {
		return nil, err
	}
	conversation := convert.AIConversationToDTO(do.AIConversation{Messages: messages}, true)
	return &dto.AIImportedThread{ID: value.ID, Title: value.Title, UpdatedAt: value.UpdatedAt, TotalMessages: len(value.Messages), Messages: conversation.Messages, NextBefore: next}, nil
}

// SyncImportedThread 每次写入都重新取现行身份；客户端无法指定另一个管理员目录。
func (s *AIAppService) SyncImportedThread(ctx context.Context, value do.AIImportedThread) error {
	access, err := s.access(ctx, value.AdminID)
	if err != nil {
		return err
	}
	if err := s.domain.ValidateImportedThread(&value, access, true); err != nil {
		return err
	}
	if s.imports == nil {
		return common.ServerErr.WithMsg("本地历史同步尚未配置。")
	}
	return aiStorageError(s.imports.Put(ctx, value))
}
