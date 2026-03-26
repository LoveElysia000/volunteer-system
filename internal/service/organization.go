package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
)

type OrganizationService struct {
	Service
}

// maxBatchOrganizationStatusIDs limits each batch request to avoid oversized IN SQL and lock pressure.
const maxBatchOrganizationStatusIDs = 500

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

// ----- 组织端私有能力：鉴权与基础工具 -----

func (s *OrganizationService) requireOrganizationManagePermission(orgID int64) (int64, error) {
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return 0, err
	}
	if err := s.requireOrgPermission(
		userID,
		orgID,
		model.PermissionResourceOrganization,
		model.PermissionActionManage,
	); err != nil {
		return 0, err
	}
	return userID, nil
}

func (s *OrganizationService) resolveManageableOrganizationScope() (bool, []int64, error) {
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil || userID <= 0 {
		return false, nil, errUnauthorized("未登录或认证无效")
	}

	hasGlobalManage, err := s.repo.HasPermissionByScope(
		s.repo.DB,
		userID,
		model.RBACScopeGlobal,
		0,
		model.PermissionResourceOrganization,
		model.PermissionActionManage,
	)
	if err != nil {
		return false, nil, err
	}
	if hasGlobalManage {
		return true, nil, nil
	}

	scopedOrgIDs, err := s.repo.ListOrgScopeIDsByPermission(
		s.repo.DB,
		userID,
		model.PermissionResourceOrganization,
		model.PermissionActionManage,
		0,
	)
	if err != nil {
		return false, nil, err
	}
	return false, scopedOrgIDs, nil
}

func buildManageableOrgFilter(hasGlobalManage bool, scopedOrgIDs []int64) (map[string]any, error) {
	filter := make(map[string]any)
	if hasGlobalManage {
		return filter, nil
	}

	ids := util.UniquePositiveInt64(scopedOrgIDs)
	if len(ids) == 0 {
		return nil, errForbidden("无权访问组织")
	}
	filter["org.id IN ?"] = ids
	return filter, nil
}

func buildPublicOrganizationQueryMap(req *api.OrganizationListRequest) map[string]any {
	queryMap := map[string]any{
		"org.status IN ?": []int32{model.OrganizationNormal},
	}
	if req == nil {
		return queryMap
	}
	return queryMap
}

func parseOrganizationSearchDate(dateStr, fieldName string, endOfDay bool) (*time.Time, error) {
	trimmed := strings.TrimSpace(dateStr)
	if trimmed == "" {
		return nil, nil
	}

	if t, err := util.ParseDateTime(trimmed); err == nil {
		return &t, nil
	}

	d, err := util.ParseDate(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s 时间格式错误", fieldName)
	}
	if endOfDay {
		d = d.Add(24*time.Hour - time.Second)
	}
	return &d, nil
}

func (s *OrganizationService) getOrganizationByID(orgID int64) (*model.Organization, error) {
	organization, err := s.repo.GetOrganizationByID(s.repo.DB, orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("组织不存在")
		}
		log.Error("查询组织信息失败: %v, ID=%d", err, orgID)
		return nil, errors.New("查询组织信息失败")
	}
	return organization, nil
}

func (s *OrganizationService) getManageableOrganization(orgID int64) (*model.Organization, int64, error) {
	if orgID <= 0 {
		return nil, 0, errors.New("组织ID无效")
	}
	organization, err := s.getOrganizationByID(orgID)
	if err != nil {
		return nil, 0, err
	}
	operatorID, err := s.requireOrganizationManagePermission(organization.ID)
	if err != nil {
		return nil, 0, err
	}
	return organization, operatorID, nil
}

// ===== 组织端：查询能力 =====

func (s *OrganizationService) OrganizationList(req *api.OrganizationListRequest) (*api.OrganizationListResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	hasGlobalManage, scopedOrgIDs, err := s.resolveManageableOrganizationScope()
	if err != nil {
		return nil, err
	}
	queryMap, err := buildManageableOrgFilter(hasGlobalManage, scopedOrgIDs)
	if err != nil {
		return nil, err
	}

	trimmedKeyword := strings.TrimSpace(req.Keyword)
	if trimmedKeyword != "" {
		ids, err := s.findManageableOrganizationIDsByKeyword(queryMap, trimmedKeyword)
		if err != nil {
			log.Error("构建组织查询条件失败: %v", err)
			return nil, err
		}
		if len(ids) == 0 {
			return &api.OrganizationListResponse{
				Total: 0,
				List:  []*api.OrganizationListItem{},
			}, nil
		}
		queryMap["org.id IN ?"] = ids
	}
	if len(req.Status) > 0 {
		queryMap["org.status IN ?"] = req.Status
	}

	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	organizations, total, err := s.repo.GetOrganizationList(s.repo.DB, queryMap, pageSize, offset)
	if err != nil {
		log.Error("查询组织列表失败: %v", err)
		return nil, err
	}

	list := make([]*api.OrganizationListItem, 0, len(organizations))
	for _, org := range organizations {
		if org == nil {
			continue
		}
		item, buildErr := buildOrganizationListItem(org)
		if buildErr != nil {
			log.Error("组装组织列表项失败: %v, org_id=%d", buildErr, org.ID)
			return nil, buildErr
		}
		list = append(list, item)
	}

	return &api.OrganizationListResponse{
		Total: int32(total),
		List:  list,
	}, nil
}

func (s *OrganizationService) PublicOrganizationList(req *api.OrganizationListRequest) (*api.OrganizationListResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	queryMap := buildPublicOrganizationQueryMap(req)

	trimmedKeyword := strings.TrimSpace(req.Keyword)
	if trimmedKeyword != "" {
		ids, err := s.repo.FindOrganizationIDsByKeyword(s.repo.DB, trimmedKeyword)
		if err != nil {
			log.Error("构建公开组织查询条件失败: %v", err)
			return nil, err
		}
		if len(ids) == 0 {
			return &api.OrganizationListResponse{
				Total: 0,
				List:  []*api.OrganizationListItem{},
			}, nil
		}
		queryMap["org.id IN ?"] = ids
	}

	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	organizations, total, err := s.repo.GetOrganizationList(s.repo.DB, queryMap, pageSize, offset)
	if err != nil {
		log.Error("查询公开组织列表失败: %v", err)
		return nil, err
	}

	list := make([]*api.OrganizationListItem, 0, len(organizations))
	for _, org := range organizations {
		if org == nil {
			continue
		}
		item, buildErr := buildPublicOrganizationListItem(org)
		if buildErr != nil {
			log.Error("组装公开组织列表项失败: %v, org_id=%d", buildErr, org.ID)
			return nil, buildErr
		}
		list = append(list, item)
	}

	return &api.OrganizationListResponse{
		Total: int32(total),
		List:  list,
	}, nil
}

func (s *OrganizationService) OrganizationDetail(req *api.OrganizationDetailRequest) (*api.OrganizationDetailResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if req.Id <= 0 {
		return nil, errors.New("组织ID无效")
	}
	organization, _, err := s.getManageableOrganization(req.Id)
	if err != nil {
		return nil, err
	}
	account, err := s.repo.FindByID(s.repo.DB, organization.AccountID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error("查询组织详情失败: 查询账户异常: %v, org_id=%d, account_id=%d", err, organization.ID, organization.AccountID)
			return nil, err
		}
		account = nil
	}
	resp, err := buildOrganizationDetailResponse(organization, account)
	if err != nil {
		log.Error("查询组织详情失败: 组装响应异常: %v, org_id=%d", err, organization.ID)
		return nil, err
	}
	return resp, nil
}

// ===== 组织端：单组织变更能力 =====

func (s *OrganizationService) CreateOrganization(req *api.OrganizationCreateRequest) (*api.OrganizationCreateResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
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
	if userID <= 0 {
		return nil, errors.New("用户ID无效")
	}

	var createdOrgID int64
	err = s.withTransaction(func(tx *gorm.DB) error {
		contactPhone := req.ContactPhone
		if strings.TrimSpace(req.ContactPhone) != "" {
			pair, phoneErr := util.ProcessSensitiveField(strings.TrimSpace(req.ContactPhone))
			if phoneErr != nil {
				return errors.New("联系电话处理失败")
			}
			contactPhone = pair.Encrypted
		}
		org := &model.Organization{
			AccountID:     userID,
			OrgName:       req.Name,
			LicenseCode:   req.OrganizationCode,
			ContactPerson: req.ContactPerson,
			ContactPhone:  contactPhone,
			Address:       req.Address,
			LogoURL:       req.LogoUrl,
			Introduction:  req.Description,
			Status:        model.OrganizationNormal, // 默认启用
		}

		if err := s.repo.CreateOrganization(tx, org); err != nil {
			return err
		}
		if err := s.bindOrganizationOwnerRole(tx, userID, org.ID); err != nil {
			return err
		}
		createdOrgID = org.ID
		return nil
	})
	if err != nil {
		log.Error("创建组织失败: %v, user_id=%d", err, userID)
		return nil, errors.New("创建组织失败")
	}

	log.Info("组织创建成功: ID=%d, 名称=%s", createdOrgID, req.Name)

	return &api.OrganizationCreateResponse{
		Id:      createdOrgID,
		Message: "创建成功",
	}, nil
}

func (s *OrganizationService) bindOrganizationOwnerRole(tx *gorm.DB, accountID, orgID int64) error {
	role, err := s.repo.GetRBACRoleByCode(tx, model.RBACRoleOrgOwner)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("组织管理员角色未初始化")
		}
		return err
	}
	if role == nil || role.Status != 1 {
		return errors.New("组织管理员角色不可用")
	}
	return s.repo.UpsertRBACAccountRoleBinding(
		tx,
		accountID,
		role.ID,
		model.RBACScopeOrg,
		orgID,
		1,
		accountID,
		nil,
	)
}

func (s *OrganizationService) UpdateOrganization(req *api.OrganizationUpdateRequest) (*api.OrganizationUpdateResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	_, _, err := s.getManageableOrganization(req.Id)
	if err != nil {
		return nil, err
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
		pair, phoneErr := util.ProcessSensitiveField(strings.TrimSpace(req.ContactPhone))
		if phoneErr != nil {
			return nil, errors.New("联系电话处理失败")
		}
		updateQuery["contact_phone"] = pair.Encrypted
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

func (s *OrganizationService) OrganizationAccountUpdate(req *api.OrganizationAccountUpdateRequest) (*api.OrganizationAccountUpdateResponse, error) {
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	account, err := s.repo.FindByID(s.repo.DB, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("账户不存在")
		}
		log.Error("更新组织账户信息失败: 查询账户异常: %v, user_id=%d", err, userID)
		return nil, err
	}

	updates, err := buildOrganizationAccountUpdateMutations(req)
	if err != nil {
		return nil, err
	}

	if email, ok := updates["email"].(string); ok && email != "" && email != account.Email {
		exists, checkErr := s.repo.CheckEmailExists(s.repo.DB, email)
		if checkErr != nil {
			log.Error("更新组织账户信息失败: 检查邮箱异常: %v, user_id=%d", checkErr, userID)
			return nil, errors.New("检查邮箱失败")
		}
		if exists {
			return nil, errors.New("邮箱已存在")
		}
	}

	if mobileHash, ok := updates["mobile_hash"].(string); ok && mobileHash != "" && mobileHash != account.MobileHash {
		exists, checkErr := s.repo.CheckMobileExists(s.repo.DB, mobileHash)
		if checkErr != nil {
			log.Error("更新组织账户信息失败: 检查手机号异常: %v, user_id=%d", checkErr, userID)
			return nil, errors.New("检查手机号失败")
		}
		if exists {
			return nil, errors.New("手机号已存在")
		}
	}

	if err := s.repo.UpdateAccountByID(s.repo.DB, userID, updates); err != nil {
		log.Error("更新组织账户信息失败: %v, user_id=%d", err, userID)
		return nil, errors.New("更新组织账户信息失败")
	}

	return &api.OrganizationAccountUpdateResponse{}, nil
}

func (s *OrganizationService) findManageableOrganizationIDsByKeyword(baseQueryMap map[string]any, keyword string) ([]int64, error) {
	ids, err := s.repo.FindOrganizationIDsByKeyword(s.repo.DB, keyword)
	if err != nil {
		return nil, err
	}

	queryMap := make(map[string]any, len(baseQueryMap))
	for key, value := range baseQueryMap {
		queryMap[key] = value
	}
	organizations, err := s.repo.ListOrganizations(s.repo.DB, queryMap)
	if err != nil {
		return nil, err
	}

	idSet := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	for _, org := range organizations {
		if org == nil {
			continue
		}
		matched, matchErr := organizationPhoneContainsKeyword(org.ContactPhone, keyword)
		if matchErr != nil {
			log.Warn("组织联系电话匹配关键字失败: org_id=%d err=%v", org.ID, matchErr)
			continue
		}
		if matched {
			idSet[org.ID] = struct{}{}
		}
	}

	result := make([]int64, 0, len(idSet))
	for id := range idSet {
		result = append(result, id)
	}
	return result, nil
}

func (s *OrganizationService) DeleteOrganization(req *api.DeleteOrganizationRequest) (*api.DeleteOrganizationResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	_, _, err := s.getManageableOrganization(req.Id)
	if err != nil {
		return nil, err
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
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	organization, _, err := s.getManageableOrganization(req.Id)
	if err != nil {
		return nil, err
	}
	if organization.AccountID <= 0 {
		return nil, errors.New("组织账号信息异常")
	}

	err = s.repo.UpdateOrganization(s.repo.DB, req.Id, map[string]any{"status": model.OrganizationDisabled})
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
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	organization, _, err := s.getManageableOrganization(req.Id)
	if err != nil {
		return nil, err
	}
	if organization.AccountID <= 0 {
		return nil, errors.New("组织账号信息异常")
	}

	err = s.repo.UpdateOrganization(s.repo.DB, req.Id, map[string]any{"status": model.OrganizationNormal})
	if err != nil {
		log.Error("启用组织失败: %v, ID=%d", err, req.Id)
		return nil, errors.New("启用组织失败")
	}

	log.Info("组织启用成功: ID=%d, 原因=%s", req.Id, req.Reason)

	return &api.EnableOrganizationResponse{
		Message: "启用成功",
	}, nil
}

// ===== 组织端：搜索与批量操作 =====

func (s *OrganizationService) SearchOrganizations(req *api.OrganizationSearchRequest) (*api.OrganizationSearchResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	hasGlobalManage, scopedOrgIDs, err := s.resolveManageableOrganizationScope()
	if err != nil {
		return nil, err
	}
	queryMap, err := buildManageableOrgFilter(hasGlobalManage, scopedOrgIDs)
	if err != nil {
		return nil, err
	}

	trimmedKeyword := strings.TrimSpace(req.Keyword)
	if trimmedKeyword != "" {
		ids, err := s.findManageableOrganizationIDsByKeyword(queryMap, trimmedKeyword)
		if err != nil {
			log.Error("构建组织搜索条件失败: %v", err)
			return nil, err
		}
		if len(ids) == 0 {
			return &api.OrganizationSearchResponse{
				Total: 0,
				List:  []*api.OrganizationListItem{},
			}, nil
		}
		queryMap["org.id IN ?"] = ids
	}
	if len(req.Status) > 0 {
		queryMap["org.status IN ?"] = req.Status
	}

	startAt, err := parseOrganizationSearchDate(req.StartDate, "startDate", false)
	if err != nil {
		return nil, err
	}
	endAt, err := parseOrganizationSearchDate(req.EndDate, "endDate", true)
	if err != nil {
		return nil, err
	}
	if startAt != nil && endAt != nil && startAt.After(*endAt) {
		return nil, errors.New("startDate 不能晚于 endDate")
	}
	if startAt != nil {
		queryMap["org.created_at >= ?"] = *startAt
	}
	if endAt != nil {
		queryMap["org.created_at <= ?"] = *endAt
	}

	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	organizations, total, err := s.repo.GetOrganizationList(s.repo.DB, queryMap, pageSize, offset)
	if err != nil {
		log.Error("搜索组织失败: %v", err)
		return nil, err
	}

	list := make([]*api.OrganizationListItem, 0, len(organizations))
	for _, org := range organizations {
		if org == nil {
			continue
		}
		item, buildErr := buildOrganizationListItem(org)
		if buildErr != nil {
			log.Error("组装组织搜索结果失败: %v, org_id=%d", buildErr, org.ID)
			return nil, buildErr
		}
		list = append(list, item)
	}

	return &api.OrganizationSearchResponse{
		Total: int32(total),
		List:  list,
	}, nil
}

func (s *OrganizationService) BulkDeleteOrganizations(req *api.BulkDeleteOrganizationRequest) (*api.BulkDeleteOrganizationResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	// 参数校验
	if len(req.Ids) == 0 {
		return nil, errors.New("组织ID列表不能为空")
	}

	// 转换ID列表
	orgIDs := make([]int64, 0, len(req.Ids))
	for _, id := range req.Ids {
		orgIDs = append(orgIDs, id)
	}
	// 批量删除前按组织维度逐条鉴权，支持 super_admin 全局权限。
	for _, orgID := range orgIDs {
		if _, err := s.requireOrganizationManagePermission(orgID); err != nil {
			return nil, err
		}
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

func (s *OrganizationService) BatchDisableOrganizations(req *api.BatchDisableOrganizationRequest) (*api.BatchDisableOrganizationResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	successCount, failedIDs, err := s.batchChangeOrganizationStatus(req.Ids, req.Reason, model.OrganizationDisabled, "停用")
	if err != nil {
		return nil, err
	}

	return &api.BatchDisableOrganizationResponse{
		SuccessCount: int32(successCount),
		FailedIds:    failedIDs,
		Message:      "批量停用完成",
	}, nil
}

func (s *OrganizationService) BatchEnableOrganizations(req *api.BatchEnableOrganizationRequest) (*api.BatchEnableOrganizationResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	successCount, failedIDs, err := s.batchChangeOrganizationStatus(req.Ids, req.Reason, model.OrganizationNormal, "启用")
	if err != nil {
		return nil, err
	}

	return &api.BatchEnableOrganizationResponse{
		SuccessCount: int32(successCount),
		FailedIds:    failedIDs,
		Message:      "批量启用完成",
	}, nil
}

// ----- 组织端私有能力：批量流程 -----

func (s *OrganizationService) resolveOrgBatchOperatorID() (int64, error) {
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Warn("组织批量停启失败: 获取当前用户失败: %v", err)
		return 0, err
	}
	if userID <= 0 {
		return 0, errors.New("用户ID无效")
	}
	return userID, nil
}

func (s *OrganizationService) batchChangeOrganizationStatus(
	ids []int64,
	reason string,
	orgStatus int32,
	action string,
) (int, []int64, error) {
	// 步骤1：基础参数与操作者权限校验。
	if len(ids) == 0 {
		return 0, nil, errors.New("组织ID列表不能为空")
	}
	operatorID, err := s.resolveOrgBatchOperatorID()
	if err != nil {
		return 0, nil, err
	}

	// 步骤2：归一化ID输入（记录无效ID失败、有效ID去重、校验批量上限）。
	failed := make([]int64, 0)
	for _, id := range ids {
		if id <= 0 {
			failed = append(failed, id)
		}
	}
	validOrgIDs := util.UniquePositiveInt64(ids)
	if len(validOrgIDs) == 0 {
		log.Warn("组织批量%s失败: 组织ID均无效, reason=%q", action, reason)
		return 0, failed, nil
	}
	// 基于去重后的有效ID做上限校验，防止批量请求过大导致SQL过长和锁压力升高。
	if len(validOrgIDs) > maxBatchOrganizationStatusIDs {
		return 0, nil, fmt.Errorf("组织ID数量超过上限(%d)", maxBatchOrganizationStatusIDs)
	}

	// 步骤3：批量加载组织快照，构建查询索引。
	orgs, err := s.repo.GetOrganizationsByIDs(s.repo.DB, validOrgIDs)
	if err != nil {
		log.Error("组织批量%s失败: 查询组织失败: %v", action, err)
		return 0, nil, errors.New("查询组织信息失败")
	}
	orgMap := make(map[int64]*model.Organization, len(orgs))
	for _, org := range orgs {
		if org == nil {
			continue
		}
		orgMap[org.ID] = org
	}

	// 步骤4：逐条校验可操作性并构建执行计划。
	plan := struct {
		idempotentOK int
		updOrgIDs    []int64
	}{
		updOrgIDs: make([]int64, 0, len(validOrgIDs)),
	}
	for _, orgID := range validOrgIDs {
		org := orgMap[orgID]
		if org == nil || org.AccountID <= 0 {
			failed = append(failed, orgID)
			continue
		}
		if err := s.requireOrgPermission(
			operatorID,
			org.ID,
			model.PermissionResourceOrganization,
			model.PermissionActionManage,
		); err != nil {
			failed = append(failed, orgID)
			continue
		}

		if org.Status == orgStatus {
			plan.idempotentOK++
			continue
		}
		plan.updOrgIDs = append(plan.updOrgIDs, org.ID)
	}

	// 步骤5：批量落库并汇总结果。
	if len(plan.updOrgIDs) == 0 {
		successCount := plan.idempotentOK
		log.Info(
			"组织批量%s完成: total=%d success=%d failed=%d reason=%q",
			action,
			len(validOrgIDs),
			successCount,
			len(failed),
			reason,
		)
		return successCount, failed, nil
	}

	err = s.repo.BatchUpdateOrganizationStatusByIDs(s.repo.DB, plan.updOrgIDs, orgStatus)
	if err != nil {
		log.Error(
			"组织批量%s失败: 批量更新状态异常: %v, org_count=%d",
			action,
			err,
			len(plan.updOrgIDs),
		)
		failed = append(failed, plan.updOrgIDs...)
		plan.updOrgIDs = plan.updOrgIDs[:0]
	}

	successCount := plan.idempotentOK + len(plan.updOrgIDs)
	log.Info(
		"组织批量%s完成: total=%d success=%d failed=%d reason=%q",
		action,
		len(validOrgIDs),
		successCount,
		len(failed),
		reason,
	)
	return successCount, failed, nil
}
