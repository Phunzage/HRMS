package service

import (
	"errors"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// 版本登记业务错误
var (
	ErrUpdatePlatformInvalid = errors.New("平台参数不合法（仅支持 win/android）")
	ErrUpdateVersionRequired = errors.New("版本号不能为空")
	ErrUpdateBuildInvalid    = errors.New("build_number 必须大于 0")
	ErrUpdateURLRequired     = errors.New("下载地址不能为空")
	ErrUpdateBuildNotNewer   = errors.New("build_number 必须高于该平台当前最新版本")
	ErrUpdateVersionExists   = errors.New("该平台已存在相同版本号")
)

// CreateUpdateVersionRequest 登记新版本请求
type CreateUpdateVersionRequest struct {
	Platform     string `json:"platform"`
	Version      string `json:"version"`
	BuildNumber  int    `json:"build_number"`
	URL          string `json:"url"`
	ReleaseNotes string `json:"release_notes"`
}

// UpdateService 客户端版本业务服务
type UpdateService struct {
	repo *repository.UpdateRepo
}

// NewUpdateService 创建版本服务
func NewUpdateService(repo *repository.UpdateRepo) *UpdateService {
	return &UpdateService{repo: repo}
}

// Check 检查某平台是否有新版本：返回服务器最新版本记录与是否有更新
// latest 为空表示该平台尚无任何版本记录，此时 hasUpdate 恒为 false
func (s *UpdateService) Check(platform string, clientBuild int) (latest *models.AppVersion, hasUpdate bool, err error) {
	latest, err = s.repo.Latest(platform)
	if err != nil || latest == nil {
		return latest, false, err
	}
	return latest, latest.BuildNumber > clientBuild, nil
}

// Latest 查询某平台最新版本记录（无记录返回 nil）
func (s *UpdateService) Latest(platform string) (*models.AppVersion, error) {
	return s.repo.Latest(platform)
}

// List 版本分页列表（platform 为空查全部）
func (s *UpdateService) List(platform string, page, pageSize int) ([]models.AppVersion, int64, error) {
	return s.repo.List(platform, page, pageSize)
}

// Create 登记新版本（校验必填字段 + 平台合法 + build 单调递增 + 平台内版本号唯一）
func (s *UpdateService) Create(req CreateUpdateVersionRequest) (*models.AppVersion, error) {
	if req.Platform != "win" && req.Platform != "android" {
		return nil, ErrUpdatePlatformInvalid
	}
	if req.Version == "" {
		return nil, ErrUpdateVersionRequired
	}
	if req.BuildNumber <= 0 {
		return nil, ErrUpdateBuildInvalid
	}
	if req.URL == "" {
		return nil, ErrUpdateURLRequired
	}
	// build_number 须高于该平台当前最新版本，保证客户端能稳定感知升级
	latest, err := s.repo.Latest(req.Platform)
	if err != nil {
		return nil, err
	}
	if latest != nil && req.BuildNumber <= latest.BuildNumber {
		return nil, ErrUpdateBuildNotNewer
	}
	v := &models.AppVersion{
		Platform:     req.Platform,
		Version:      req.Version,
		BuildNumber:  req.BuildNumber,
		URL:          req.URL,
		ReleaseNotes: req.ReleaseNotes,
	}
	if err := s.repo.Create(v); err != nil {
		// 平台内版本号唯一（idx_plat_ver）冲突
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrUpdateVersionExists
		}
		return nil, err
	}
	return v, nil
}
