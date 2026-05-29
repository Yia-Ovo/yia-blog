package handler

import (
	"blog-server/api/dto"
	"blog-server/app/service"
	"blog-server/common"
	"blog-server/common/gin/response"
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	adminAppSrv *service.AdminAppService
}

func NewAdminHandler(adminAppSrv *service.AdminAppService) *AdminHandler {
	return &AdminHandler{
		adminAppSrv: adminAppSrv,
	}
}

// Login 管理员登录 POST /api/blog/v1/admin/login
func (h *AdminHandler) Login(c *gin.Context) {
	var req dto.AdminLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	adminDTO, token, err := h.adminAppSrv.Login(ctx, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	respData := gin.H{
		"admin": adminDTO,
		"token": token,
	}
	response.WriteResp(c, respData, common.OK)
}

// GetInfo 获取管理员信息 GET /api/blog/v1/admin/getInfo
func (h *AdminHandler) GetInfo(c *gin.Context) {
	adminIdVal, exists := c.Get("admin_id")
	if !exists {
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}
	adminId, ok := adminIdVal.(uint64)
	if !ok {
		response.WriteResp(c, nil, common.ServerErr.WithMsg("管理员ID类型错误"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	adminInfo, err := h.adminAppSrv.GetAdminInfo(ctx, adminId)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}
	response.WriteResp(c, adminInfo, common.OK)
}

// UpdatePWD 管理员修改密码 POST /api/blog/v1/admin/updatePWD
func (h *AdminHandler) UpdatePWD(c *gin.Context) {
	var req dto.AdminUpdatePWDReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数校验失败："+err.Error()))
		return
	}

	adminIdVal, exists := c.Get("admin_id")
	if !exists {
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}
	adminId, ok := adminIdVal.(uint64)
	if !ok {
		response.WriteResp(c, nil, common.ServerErr.WithMsg("管理员ID类型错误"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.adminAppSrv.UpdatePWD(ctx, adminId, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}
	response.WriteResp(c, nil, common.OK)
}

// UpdateAdminInfo 更新管理员信息 POST /api/blog/v1/admin/updateAdminInfo
func (h *AdminHandler) UpdateAdminInfo(c *gin.Context) {
	var req dto.AdminUpdateInfoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteResp(c, nil, common.ParamErr.WithMsg("参数错误"))
		return
	}

	adminIdVal, exists := c.Get("admin_id")
	if !exists {
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}
	adminId, ok := adminIdVal.(uint64)
	if !ok {
		response.WriteResp(c, nil, common.ServerErr.WithMsg("管理员ID类型错误"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	res, err := h.adminAppSrv.UpdateAdminInfo(ctx, adminId, &req)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}

	response.WriteResp(c, res, common.OK)
}

// Logout 管理员登出 POST /api/blog/v1/admin/logout
func (h *AdminHandler) Logout(c *gin.Context) {
	token := c.GetHeader("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	if token == "" {
		response.WriteResp(c, nil, common.ParamErr)
		return
	}

	adminIdVal, exists := c.Get("admin_id")
	if !exists {
		response.WriteResp(c, nil, common.LoginNotLoginErr)
		return
	}
	adminId, ok := adminIdVal.(uint64)
	if !ok {
		response.WriteResp(c, nil, common.ServerErr.WithMsg("管理员ID类型错误"))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.adminAppSrv.Logout(ctx, adminId, token)
	if err != nil {
		response.WriteResp(c, nil, err.(common.Errno))
		return
	}
	response.WriteResp(c, nil, common.OK)
}
