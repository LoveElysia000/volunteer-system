package service

import (
	"context"
	"errors"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
)

type VolunteerService struct {
	Service
}

func NewVolunteerService(ctx context.Context, c *app.RequestContext) *VolunteerService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &VolunteerService{
		Service{
			ctx:  ctx,
			c:    c,
			repo: repository.NewRepository(ctx, c),
		},
	}
}

// VolunteerList 以组织管理者身份查询志愿者列表，仅返回其所管理组织下的志愿者。
func (s *VolunteerService) VolunteerList(req *api.VolunteerListRequest) (*api.VolunteerListResponse, error) {
	// 参数校验
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// 获取当前组织管理者所属组织列表
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Error("查询志愿者列表失败: 获取当前用户ID失败: %v", err)
		return nil, err
	}

	organizations, err := s.repo.FindOrganizationByAccountID(s.repo.DB, userID)
	if err != nil {
		log.Error("查询志愿者列表失败: 查询组织异常: %v, user_id=%d", err, userID)
		return nil, err
	}
	if len(organizations) == 0 {
		log.Error("查询志愿者列表失败: 当前用户无组织信息, user_id=%d", userID)
		return nil, errors.New("当前用户无组织信息")
	}
	orgIDs := make([]int64, 0, len(organizations))
	for _, org := range organizations {
		orgIDs = append(orgIDs, org.ID)
	}

	// 构建查询参数map
	queryMap := make(map[string]any)

	// 如果有关键字，先通过模糊查询获取志愿者ID列表
	if req.Keyword != "" {
		ids, err := s.repo.FindVolunteerIDsByKeyword(s.repo.DB, req.Keyword)
		if err != nil {
			log.Error("关键字查询志愿者ID失败: %v", err)
			return nil, err
		}
		if len(ids) == 0 {
			// 没有匹配的志愿者
			return &api.VolunteerListResponse{
				Total: 0,
				List:  []*api.VolunteerListItem{},
			}, nil
		}
		queryMap["v.id IN ?"] = ids
	}

	// 根据查询参数查询志愿者列表
	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	volunteers, total, err := s.repo.GetVolunteerList(s.repo.DB, orgIDs, queryMap, pageSize, offset)
	if err != nil {
		log.Error("查询志愿者列表失败: %v, organization_ids=%v", err, orgIDs)
		return nil, err
	}

	// 组装返回数据
	resp := &api.VolunteerListResponse{
		Total: int32(total),
		List:  make([]*api.VolunteerListItem, 0, len(volunteers)),
	}
	if len(volunteers) == 0 {
		return resp, nil
	}
	for _, v := range volunteers {
		item := &api.VolunteerListItem{
			Id:           v.ID,
			AccountId:    v.AccountID,
			RealName:     v.RealName,
			Gender:       v.Gender,
			AvatarUrl:    v.AvatarURL,
			TotalHours:   v.TotalHours,
			ServiceCount: v.ServiceCount,
			CreditScore:  v.CreditScore,
			AuditStatus:  v.AuditStatus,
			Status:       v.Status,
			CreatedAt:    v.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    v.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		resp.List = append(resp.List, item)
	}

	return resp, nil
}

func (s *VolunteerService) VolunteerDetail(req *api.VolunteerDetailRequest) (*api.VolunteerDetailResponse, error) {
	// 查询志愿者信息
	volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, req.Id)
	if err != nil {
		log.Error("查询志愿者信息失败: %v, ID=%d", err, req.Id)
		return nil, err
	}

	if volunteer == nil {
		log.Error("查询志愿者信息失败: 志愿者不存在, id=%d", req.Id)
		return nil, errors.New("志愿者不存在")
	}

	// 格式化生日
	birthday := ""
	if volunteer.Birthday != nil {
		birthday = volunteer.Birthday.Format("2006-01-02")
	}

	// 组装返回数据
	resp := &api.VolunteerDetailResponse{
		Volunteer: &api.VolunteerInfo{
			Id:           volunteer.ID,
			AccountId:    volunteer.AccountID,
			RealName:     volunteer.RealName,
			Gender:       volunteer.Gender,
			Birthday:     birthday,
			IdCard:       volunteer.IDCard,
			AvatarUrl:    volunteer.AvatarURL,
			Introduction: volunteer.Introduction,
			TotalHours:   volunteer.TotalHours,
			ServiceCount: volunteer.ServiceCount,
			CreditScore:  volunteer.CreditScore,
			AuditStatus:  volunteer.AuditStatus,
			Status:       volunteer.Status,
			CreatedAt:    volunteer.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    volunteer.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	}

	return resp, nil
}

// MyProfile 我的个人信息（志愿者端）
func (s *VolunteerService) MyProfile(req *api.MyProfileRequest) (*api.MyProfileResponse, error) {
	// 参数校验
	if req == nil || req.Id <= 0 {
		return nil, errors.New("志愿者ID无效")
	}

	// 通过请求ID查询志愿者信息
	volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("志愿者不存在")
		}
		log.Error("查询我的个人信息失败: 查询志愿者信息异常: %v, volunteer_id=%d", err, req.Id)
		return nil, err
	}

	if volunteer == nil {
		log.Error("查询我的个人信息失败: 志愿者不存在, volunteer_id=%d", req.Id)
		return nil, errors.New("志愿者不存在")
	}

	// 权限校验：只能查询自己的档案
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Error("查询我的个人信息失败: 获取当前用户ID失败: %v", err)
		return nil, err
	}
	if volunteer.AccountID != userID {
		log.Error("查询我的个人信息失败: 无权查看他人信息, user_id=%d, volunteer_account_id=%d, volunteer_id=%d", userID, volunteer.AccountID, req.Id)
		return nil, errors.New("无权查看他人信息")
	}

	// 格式化生日
	birthday := ""
	if volunteer.Birthday != nil {
		birthday = volunteer.Birthday.Format("2006-01-02")
	}

	// 组装返回数据
	resp := &api.MyProfileResponse{
		Volunteer: &api.VolunteerInfo{
			Id:           volunteer.ID,
			AccountId:    volunteer.AccountID,
			RealName:     volunteer.RealName,
			Gender:       volunteer.Gender,
			Birthday:     birthday,
			IdCard:       volunteer.IDCard,
			AvatarUrl:    volunteer.AvatarURL,
			Introduction: volunteer.Introduction,
			TotalHours:   volunteer.TotalHours,
			ServiceCount: volunteer.ServiceCount,
			CreditScore:  volunteer.CreditScore,
			Status:       volunteer.Status,
			AuditStatus:  volunteer.AuditStatus,
			CreatedAt:    volunteer.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    volunteer.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	}

	return resp, nil
}

func (s *VolunteerService) VolunteerUpdate(req *api.VolunteerUpdateRequest) (*api.VolunteerUpdateResponse, error) {
	// 参数校验 + 构建更新查询
	if req.VolunteerId <= 0 {
		log.Error("更新志愿者信息失败: 志愿者ID无效, volunteer_id=%d", req.VolunteerId)
		return nil, errors.New("志愿者ID无效")
	}

	updateQuery := map[string]any{}

	// 校验真实姓名
	if req.RealName != "" {
		if len(req.RealName) > 50 {
			log.Error("更新志愿者信息失败: 真实姓名长度超限, volunteer_id=%d, length=%d", req.VolunteerId, len(req.RealName))
			return nil, errors.New("真实姓名长度不能超过50个字符")
		}
		updateQuery["real_name"] = req.RealName
	}

	// 校验性别
	if req.Gender >= 0 {
		if req.Gender > 2 {
			log.Error("更新志愿者信息失败: 性别值无效, volunteer_id=%d, gender=%d", req.VolunteerId, req.Gender)
			return nil, errors.New("性别值无效，0-未知, 1-男, 2-女")
		}
		updateQuery["gender"] = req.Gender
	}

	// 校验生日
	var birthday *time.Time
	if req.Birthday != "" {
		t, err := util.ParsePastDate(req.Birthday)
		if err != nil {
			log.Error("更新志愿者信息失败: 生日格式错误, volunteer_id=%d, birthday=%s, err=%v", req.VolunteerId, req.Birthday, err)
			return nil, errors.New("生日格式错误，请使用 YYYY-MM-DD 格式")
		}
		birthday = &t
		updateQuery["birthday"] = birthday
	}

	// 校验头像URL
	if req.AvatarUrl != "" {
		if len(req.AvatarUrl) > 255 {
			log.Error("更新志愿者信息失败: 头像URL长度超限, volunteer_id=%d, length=%d", req.VolunteerId, len(req.AvatarUrl))
			return nil, errors.New("头像URL长度不能超过255个字符")
		}
		updateQuery["avatar_url"] = req.AvatarUrl
	}

	// 校验个人简介
	if req.Introduction != "" {
		if len(req.Introduction) > 2000 {
			log.Error("更新志愿者信息失败: 个人简介长度超限, volunteer_id=%d, length=%d", req.VolunteerId, len(req.Introduction))
			return nil, errors.New("个人简介长度不能超过2000个字符")
		}
		updateQuery["introduction"] = req.Introduction
	}

	if len(updateQuery) == 0 {
		log.Error("更新志愿者信息失败: 没有需要更新的字段, volunteer_id=%d", req.VolunteerId)
		return nil, errors.New("没有需要更新的字段")
	}

	// 检查志愿者是否存在
	volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, req.VolunteerId)
	if err != nil {
		log.Error("查询志愿者信息失败: %v, ID=%d", err, req.VolunteerId)
		return nil, errors.New("查询志愿者信息失败")
	}

	if volunteer == nil {
		log.Error("更新志愿者信息失败: 志愿者不存在, volunteer_id=%d", req.VolunteerId)
		return nil, errors.New("志愿者不存在")
	}

	// 调用 repository 层更新
	err = s.repo.UpdateVolunteer(s.repo.DB, req.VolunteerId, updateQuery)
	if err != nil {
		log.Error("更新志愿者信息失败: %v, ID=%d", err, req.VolunteerId)
		return nil, errors.New("更新志愿者信息失败")
	}

	var resp api.VolunteerUpdateResponse
	return &resp, nil
}
