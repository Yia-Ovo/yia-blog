package convert

import (
	"blog-server/api/dto"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/infra/model"
)

func AIStateToModel(s *do.AIState) model.AIStateModel {
	m := model.AIStateModel{Version: 3, AdminID: s.AdminID, PrivateKnowledge: s.PrivateKnowledge, Memory: s.Memory, GeneralMemory: s.GeneralMemory, Day: s.Day, Requests: s.Requests, RecentRequests: s.RecentRequests, Conversations: []model.AIConversationModel{}}
	for _, c := range s.Conversations {
		cm := model.AIConversationModel{ID: c.ID, ContextMode: c.ContextMode, OriginImportID: c.OriginImportID, Title: c.Title, UpdateAt: c.UpdateAt, Messages: []model.AIMessageModel{}}
		for _, v := range c.Messages {
			mm := model.AIMessageModel{ID: v.ID, RequestID: v.RequestID, Role: v.Role, Content: v.Content, Status: v.Status, CreateAt: v.CreateAt, Sources: []model.AISourceModel{}}
			for _, src := range v.Sources {
				mm.Sources = append(mm.Sources, model.AISourceModel{Path: src.Path, Line: src.Line, UpdatedAt: src.UpdatedAt, Excerpt: src.Excerpt})
			}
			cm.Messages = append(cm.Messages, mm)
		}
		m.Conversations = append(m.Conversations, cm)
	}
	return m
}

func AIModelToState(m model.AIStateModel) do.AIState {
	// Version 1 的所有对话都来自原私人助手，缺失标记时按私有处理。
	s := do.AIState{AdminID: m.AdminID, PrivateKnowledge: m.Version == 1 || m.PrivateKnowledge, Memory: m.Memory, GeneralMemory: m.GeneralMemory, Day: m.Day, Requests: m.Requests, RecentRequests: m.RecentRequests, Conversations: []do.AIConversation{}}
	// 旧版普通管理员记忆归入通用区；站主旧记忆保留在项目区，避免闲聊继承私人背景。
	if m.Version < 3 && !s.PrivateKnowledge {
		s.GeneralMemory = s.Memory
		s.Memory = ""
	}
	for _, c := range m.Conversations {
		mode := c.ContextMode
		if m.Version < 3 || mode == "" {
			mode = consts.AIContextGeneral
			if s.PrivateKnowledge {
				mode = consts.AIContextProject
			}
		}
		cd := do.AIConversation{ID: c.ID, ContextMode: mode, OriginImportID: c.OriginImportID, Title: c.Title, UpdateAt: c.UpdateAt, Messages: []do.AIMessage{}}
		for _, v := range c.Messages {
			md := do.AIMessage{ID: v.ID, RequestID: v.RequestID, Role: v.Role, Content: v.Content, Status: v.Status, CreateAt: v.CreateAt, Sources: []do.AISource{}}
			for _, src := range v.Sources {
				md.Sources = append(md.Sources, do.AISource{Path: src.Path, Line: src.Line, UpdatedAt: src.UpdatedAt, Excerpt: src.Excerpt})
			}
			cd.Messages = append(cd.Messages, md)
		}
		s.Conversations = append(s.Conversations, cd)
	}
	return s
}

func AIConversationToDTO(c do.AIConversation, includeMessages bool) dto.AIConversation {
	v := dto.AIConversation{ID: c.ID, ContextMode: c.ContextMode, OriginImportID: c.OriginImportID, Title: c.Title, UpdateAt: c.UpdateAt}
	if includeMessages {
		v.Messages = []dto.AIMessage{}
		for _, m := range c.Messages {
			x := dto.AIMessage{ID: m.ID, Role: m.Role, Content: m.Content, Status: m.Status, CreateAt: m.CreateAt, Sources: []dto.AISource{}}
			for _, src := range m.Sources {
				x.Sources = append(x.Sources, dto.AISource{Path: src.Path, Line: src.Line, UpdatedAt: src.UpdatedAt, Excerpt: src.Excerpt})
			}
			v.Messages = append(v.Messages, x)
		}
	}
	return v
}
