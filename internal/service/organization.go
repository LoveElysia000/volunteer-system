package service

import (
	"context"
	"errors"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
)

type OrganizationService struct {
	Service
}

func NewOrganizationService(ctx context.Context, c *app.RequestContext) *OrganizationService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &OrganizationService{
		Service{
			ctx:  ctx,
			c:    c,
			repo: repository.NewRepository(ctx, c),
		},
	}
}

func (s *OrganizationService) OrganizationList(req *api.OrganizationListRequest) (*api.OrganizationListResponse, error) {
	// 参数校验
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// 构建查询参数map
	queryMap := make(map[string]any)

	// 如果有关键字，先通过模糊查询获取组织ID列表
	if req.Keyword != "" {
		ids, err := s.repo.FindOrganizationIDsByKeyword(s.repo.DB, req.Keyword)
		if err != nil {
			log.Error("关键字查询组织ID失败: %v", err)
			return nil, err
		}
		if len(ids) == 0 {
			// 没有匹配的组织
			return &api.OrganizationListResponse{
				Total: 0,
				List:  []*api.OrganizationListItem{},
			}, nil
		}
		queryMap["org.id IN (?)"] = ids
	}

	// 状态支持多选筛选（0-停用，1-正常）
	if len(req.Status) > 0 {
		queryMap["org.status IN (?)"] = req.Status
	}

	// 根据查询参数查询组织列表
	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	organizations, total, err := s.repo.GetOrganizationList(s.repo.DB, queryMap, pageSize, offset)
	if err != nil {
		log.Error("查询组织列表失败: %v", err)
		return nil, err
	}

	// 组装返回数据
	resp := &api.OrganizationListResponse{
		Total: int32(total),
		List:  make([]*api.OrganizationListItem, 0, len(organizations)),
	}

	for _, org := range organizations {
		item := &api.OrganizationListItem{
			Id:               org.ID,
			Name:             org.OrgName,
			OrganizationCode: org.LicenseCode,
			ContactPerson:    org.ContactPerson,
			ContactPhone:     org.ContactPhone,
			Address:          org.Address,
			Status:           org.Status,
			OrganizationType: "",
			Region:           "",
			CreatedAt:        org.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		resp.List = append(resp.List, item)
	}

	return resp, nil
}

func (s *OrganizationService) OrganizationDetail(req *api.OrganizationDetailRequest) (*api.OrganizationDetailResponse, error) {
	// 查询组织信息
	organization, err := s.repo.GetOrganizationByID(s.repo.DB, req.Id)
	if err != nil {
		log.Error("查询组织信息失败: %v, ID=%d", err, req.Id)
		return nil, err
	}

	if organization == nil {
		log.Warn("组织不存在: ID=%d", req.Id)
		return nil, errors.New("组织不存在")
	}

	// 组装返回数据
	resp := &api.OrganizationDetailResponse{
		Organization: &api.OrganizationInfo{
			Id:               organization.ID,
			AccountId:        organization.AccountID,
			Name:             organization.OrgName,
			OrganizationCode: organization.LicenseCode,
			ContactPerson:    organization.ContactPerson,
			ContactPhone:     organization.ContactPhone,
			Email:            "",
			Address:          organization.Address,
			Status:           organization.Status,
			OrganizationType: "",
			Region:           "",
			Description:      organization.Introduction,
			WebsiteUrl:       "",
			LogoUrl:          organization.LogoURL,
			CreatedAt:        organization.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:        organization.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	}

	return resp, nil
}

func (s *OrganizationService) CreateOrganization(req *api.OrganizationCreateRequest) (*api.OrganizationCreateResponse, error) {
	// 获取当前登录用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Warn("获取当前用户ID失败: %v", err)
		return nil, err
	}

	// 参数校验
	if req.Name == "" {
		return nil, errors.New("组织名称不能为空")
	}
	if req.ContactPerson == "" {
		return nil, errors.New("联系人不能为空")
	}
	if req.ContactPhone == "" {
		return nil, errors.New("联系电话不能为空")
	}

	// 创建组织
	org := &model.Organization{
		AccountID:     userID,
		OrgName:       req.Name,
		LicenseCode:   req.OrganizationCode,
		ContactPerson: req.ContactPerson,
		ContactPhone:  req.ContactPhone,
		Address:       req.Address,
		LogoURL:       req.LogoUrl,
		Introduction:  req.Description,
		Status:        model.OrganizationNormal, // 默认启用
	}

	err = s.repo.CreateOrganization(s.repo.DB, org)
	if err != nil {
		log.Error("创建组织失败: %v", err)
		return nil, errors.New("创建组织失败")
	}

	log.Info("组织创建成功: ID=%d, 名称=%s", org.ID, req.Name)

	return &api.OrganizationCreateResponse{
		Id:      org.ID,
		Message: "创建成功",
	}, nil
}

func (s *OrganizationService) UpdateOrganization(req *api.OrganizationUpdateRequest) (*api.OrganizationUpdateResponse, error) {
	// 参数校验
	if req.Id <= 0 {
		return nil, errors.New("组织ID无效")
	}

	// 检查组织是否存在
	organization, err := s.repo.GetOrganizationByID(s.repo.DB, req.Id)
	if err != nil {
		log.Error("查询组织信息失败: %v, ID=%d", err, req.Id)
		return nil, errors.New("查询组织信息失败")
	}

	if organization == nil {
		return nil, errors.New("组织不存在")
	}

	// 构建更新查询
	updateQuery := make(map[string]any)

	if req.Name != "" {
		updateQuery["org_name"] = req.Name
	}
	if req.OrganizationCode != "" {
		updateQuery["license_code"] = req.OrganizationCode
	}
	if req.ContactPerson != "" {
		updateQuery["contact_person"] = req.ContactPerson
	}
	if req.ContactPhone != "" {
		updateQuery["contact_phone"] = req.ContactPhone
	}
	if req.Address != "" {
		updateQuery["address"] = req.Address
	}
	if req.Description != "" {
		updateQuery["introduction"] = req.Description
	}
	if req.LogoUrl != "" {
		updateQuery["logo_url"] = req.LogoUrl
	}

	if len(updateQuery) == 0 {
		return nil, errors.New("没有需要更新的字段")
	}

	err = s.repo.UpdateOrganization(s.repo.DB, req.Id, updateQuery)
	if err != nil {
		log.Error("更新组织信息失败: %v, ID=%d", err, req.Id)
		return nil, errors.New("更新组织信息失败")
	}

	log.Info("组织信息更新成功: ID=%d", req.Id)

	return &api.OrganizationUpdateResponse{
		Message: "更新成功",
	}, nil
}

func (s *OrganizationService) DeleteOrganization(req *api.DeleteOrganizationRequest) (*api.DeleteOrganizationResponse, error) {
	// 参数校验
	if req.Id <= 0 {
		return nil, errors.New("组织ID无效")
	}

	// 检查组织是否存在
	organization, err := s.repo.GetOrganizationByID(s.repo.DB, req.Id)
	if err != nil {
		log.Error("查询组织信息失败: %v, ID=%d", err, req.Id)
		return nil, errors.New("查询组织信息失败")
	}

	if organization == nil {
		return nil, errors.New("组织不存在")
	}

	// 删除组织
	err = s.repo.DeleteOrganization(s.repo.DB, req.Id)
	if err != nil {
		log.Error("删除组织失败: %v, ID=%d", err, req.Id)
		return nil, errors.New("删除组织失败")
	}

	log.Info("组织删除成功: ID=%d", req.Id)

	return &api.DeleteOrganizationResponse{
		Message: "删除成功",
	}, nil
}

func (s *OrganizationService) DisableOrganization(req *api.DisableOrganizationRequest) (*api.DisableOrganizationResponse, error) {
	// 参数校验
	if req.Id <= 0 {
		return nil, errors.New("组织ID无效")
	}

	// 检查组织是否存在
	organization, err := s.repo.GetOrganizationByID(s.repo.DB, req.Id)
	if err != nil {
		log.Error("查询组织信息失败: %v, ID=%d", err, req.Id)
		return nil, errors.New("查询组织信息失败")
	}

	if organization == nil {
		return nil, errors.New("组织不存在")
	}

	err = s.repo.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.UpdateOrganization(tx, req.Id, map[string]any{"status": model.OrganizationDisabled}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Error("停用组织失败: %v, ID=%d", err, req.Id)
		return nil, errors.New("停用组织失败")
	}

	log.Info("组织停用成功: ID=%d, 原因=%s", req.Id, req.Reason)

	return &api.DisableOrganizationResponse{
		Message: "停用成功",
	}, nil
}

func (s *OrganizationService) EnableOrganization(req *api.EnableOrganizationRequest) (*api.EnableOrganizationResponse, error) {
	// 参数校验
	if req.Id <= 0 {
		return nil, errors.New("组织ID无效")
	}

	// 检查组织是否存在
	organization, err := s.repo.GetOrganizationByID(s.repo.DB, req.Id)
	if err != nil {
		log.Error("查询组织信息失败: %v, ID=%d", err, req.Id)
		return nil, errors.New("查询组织信息失败")
	}

	if organization == nil {
		return nil, errors.New("组织不存在")
	}

	err = s.repo.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.UpdateOrganization(tx, req.Id, map[string]any{"status": model.OrganizationNormal}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Error("启用组织失败: %v, ID=%d", err, req.Id)
		return nil, errors.New("启用组织失败")
	}

	log.Info("组织启用成功: ID=%d, 原因=%s", req.Id, req.Reason)

	return &api.EnableOrganizationResponse{
		Message: "启用成功",
	}, nil
}

func (s *OrganizationService) SearchOrganizations(req *api.OrganizationSearchRequest) (*api.OrganizationSearchResponse, error) {
	// 参数校验
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// 构建查询参数map
	queryMap := make(map[string]any)

	// 如果有关键字，先通过模糊查询获取组织ID列表
	if req.Keyword != "" {
		ids, err := s.repo.FindOrganizationIDsByKeyword(s.repo.DB, req.Keyword)
		if err != nil {
			log.Error("关键字查询组织ID失败: %v", err)
			return nil, err
		}
		if len(ids) == 0 {
			// 没有匹配的组织
			return &api.OrganizationSearchResponse{
				Total: 0,
				List:  []*api.OrganizationListItem{},
			}, nil
		}
		queryMap["org.id IN ?"] = ids
	}

	// 状态支持多选筛选（0-停用，1-正常）
	if len(req.Status) > 0 {
		queryMap["org.status in (?)"] = req.Status
	}

	if req.StartDate != "" {
		queryMap["org.created_at >= ?"] = req.StartDate
	}
	if req.EndDate != "" {
		queryMap["org.created_at <= ?"] = req.EndDate
	}

	// 根据查询参数查询组织列表
	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	organizations, total, err := s.repo.GetOrganizationList(s.repo.DB, queryMap, pageSize, offset)
	if err != nil {
		log.Error("搜索组织失败: %v", err)
		return nil, err
	}

	// 组装返回数据
	resp := &api.OrganizationSearchResponse{
		Total: int32(total),
		List:  make([]*api.OrganizationListItem, 0, len(organizations)),
	}

	for _, org := range organizations {
		item := &api.OrganizationListItem{
			Id:               org.ID,
			Name:             org.OrgName,
			OrganizationCode: org.LicenseCode,
			ContactPerson:    org.ContactPerson,
			ContactPhone:     org.ContactPhone,
			Address:          org.Address,
			Status:           org.Status,
			OrganizationType: "",
			Region:           "",
			CreatedAt:        org.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		resp.List = append(resp.List, item)
	}

	return resp, nil
}

func (s *OrganizationService) BulkDeleteOrganizations(req *api.BulkDeleteOrganizationRequest) (*api.BulkDeleteOrganizationResponse, error) {
	// 参数校验
	if len(req.Ids) == 0 {
		return nil, errors.New("组织ID列表不能为空")
	}

	// 转换ID列表
	orgIDs := make([]int64, 0, len(req.Ids))
	for _, id := range req.Ids {
		orgIDs = append(orgIDs, id)
	}

	// 批量删除组织
	successCount, failedCount, err := s.repo.BulkDeleteOrganizations(s.repo.DB, orgIDs)
	if err != nil {
		log.Error("批量删除组织失败: %v", err)
		return nil, errors.New("批量删除组织失败")
	}

	log.Info("批量删除组织成功: 成功=%d, 失败=%d", successCount, failedCount)

	return &api.BulkDeleteOrganizationResponse{
		SuccessCount: int32(successCount),
		FailedCount:  int32(failedCount),
		Message:      "批量删除完成",
	}, nil
}
