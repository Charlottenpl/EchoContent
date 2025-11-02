package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charlottepl/blog-system/internal/core/config"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"github.com/charlottepl/blog-system/internal/core/validator"
	"github.com/charlottepl/blog-system/internal/media/model"
	"github.com/charlottepl/blog-system/internal/media/repository"
)

// UploadService 文件上传服务
type UploadService struct {
	mediaRepo    repository.MediaRepository
	usageRepo    repository.MediaUsageRepository
	config       *config.Config
}

// NewUploadService 创建文件上传服务实例
func NewUploadService() *UploadService {
	return &UploadService{
		mediaRepo: repository.NewMediaRepository(),
		usageRepo: repository.NewMediaUsageRepository(),
		config:    config.GetConfig(),
	}
}

// UploadRequest 上传请求
type UploadRequest struct {
	File        interface{} `json:"file"`        // 文件对象（multipart.File等）
	Filename    string      `json:"filename"`    // 原始文件名
	Alt         string      `json:"alt"`         // 图片替代文本
	Caption     string      `json:"caption"`     // 图片说明
	IsPublic    bool        `json:"is_public"`   // 是否公开
	UsageType   string      `json:"usage_type"`  // 使用类型
	TargetID    *int        `json:"target_id"`   // 目标ID
	UploaderID  int         `json:"uploader_id"` // 上传者ID
}

// UploadResult 上传结果
type UploadResult struct {
	MediaFile *model.MediaFile `json:"media_file"`
	URL       string           `json:"url"`
	Message   string           `json:"message"`
}

// FileUploadRequest 文件上传请求结构
type FileUploadRequest struct {
	Alt       string `json:"alt" validate:"max=200"`
	Caption   string `json:"caption" validate:"max=500"`
	IsPublic  *bool  `json:"is_public"`
	UsageType string `json:"usage_type" validate:"required,oneof=post_avatar post_cover post_content moment_content user_avatar"`
	TargetID  *int   `json:"target_id"`
}

// MediaListRequest 媒体文件列表请求
type MediaListRequest struct {
	Page       int    `form:"page,default=1" validate:"min=1"`
	Size       int    `form:"size,default=20" validate:"min=1,max=100"`
	FileType   string `form:"file_type" validate:"omitempty,oneof=image video document other"`
	UploaderID *int   `form:"uploader_id"`
	Status     string `form:"status" validate:"omitempty,oneof=active deleted"`
	IsPublic   *bool  `form:"is_public"`
	Keyword    string `form:"keyword"`
}

// UploadFile 上传文件
func (s *UploadService) UploadFile(ctx context.Context, file io.Reader, filename string, uploaderID int, req *FileUploadRequest) (*UploadResult, error) {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return nil, fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 读取文件内容
	content, err := io.ReadAll(file)
	if err != nil {
		logger.Errorf("读取文件内容失败: %v", err)
		return nil, fmt.Errorf("读取文件内容失败: %w", err)
	}

	// 验证文件大小
	if len(content) > s.config.Upload.MaxFileSize {
		return nil, fmt.Errorf("文件大小超过限制 (%d bytes)", s.config.Upload.MaxFileSize)
	}

	// 检测文件类型
	mimeType := s.detectMimeType(filename, content)
	fileType := s.getFileType(mimeType)

	// 验证文件类型
	if !s.isAllowedFileType(mimeType) {
		return nil, fmt.Errorf("不支持的文件类型: %s", mimeType)
	}

	// 计算文件哈希
	hash := s.calculateHash(content)

	// 检查文件是否已存在
	existingMedia, err := s.mediaRepo.GetByHash(hash)
	if err != nil {
		logger.Errorf("检查文件是否存在失败: %v", err)
		return nil, fmt.Errorf("检查文件是否存在失败: %w", err)
	}

	if existingMedia != nil {
		// 文件已存在，创建使用记录并返回现有媒体文件
		if req.UsageType != "" {
			usage := model.NewMediaUsage(existingMedia.ID, req.UsageType, req.TargetID, &uploaderID)
			if err := s.usageRepo.Create(usage); err != nil {
				logger.Errorf("创建使用记录失败: %v", err)
				// 不返回错误，因为文件已经存在
			}
		}

		url := s.generateFileURL(existingMedia.StoragePath)
		return &UploadResult{
			MediaFile: existingMedia,
			URL:       url,
			Message:   "文件已存在，返回现有文件",
		}, nil
	}

	// 生成存储路径
	storagePath := s.generateStoragePath(filename, hash)

	// 保存文件到磁盘
	if err := s.saveFileToDisk(storagePath, content); err != nil {
		logger.Errorf("保存文件到磁盘失败: %v", err)
		return nil, fmt.Errorf("保存文件到磁盘失败: %w", err)
	}

	// 创建媒体文件记录
	mediaFile := model.NewMediaFile(
		filename,
		storagePath,
		mimeType,
		fileType,
		hash,
		int64(len(content)),
		&uploaderID,
	)

	// 设置元数据
	if req.Alt != "" {
		mediaFile.Alt = req.Alt
	}
	if req.Caption != "" {
		mediaFile.Caption = req.Caption
	}
	if req.IsPublic != nil {
		mediaFile.IsPublic = *req.IsPublic
	}

	// 如果是图片，尝试获取尺寸信息
	if fileType == "image" {
		if width, height, err := s.getImageDimensions(content); err == nil {
			mediaFile.SetDimensions(width, height)
		}
	}

	// 保存媒体文件记录
	if err := s.mediaRepo.Create(mediaFile); err != nil {
		// 删除已保存的文件
		os.Remove(storagePath)
		logger.Errorf("创建媒体文件记录失败: %v", err)
		return nil, fmt.Errorf("创建媒体文件记录失败: %w", err)
	}

	// 创建使用记录
	if req.UsageType != "" {
		usage := model.NewMediaUsage(mediaFile.ID, req.UsageType, req.TargetID, &uploaderID)
		if err := s.usageRepo.Create(usage); err != nil {
			logger.Errorf("创建使用记录失败: %v", err)
			// 不返回错误，因为文件已经上传成功
		}
	}

	// 生成访问URL
	url := s.generateFileURL(storagePath)

	logger.Infof("文件上传成功 (ID: %d, 文件名: %s, 类型: %s)",
		mediaFile.ID, mediaFile.Filename, mediaFile.FileType)

	return &UploadResult{
		MediaFile: mediaFile,
		URL:       url,
		Message:   "文件上传成功",
	}
}

// UploadMultipleFiles 批量上传文件
func (s *UploadService) UploadMultipleFiles(ctx context.Context, files []FileData, uploaderID int, req *FileUploadRequest) ([]*UploadResult, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("没有文件需要上传")
	}

	if len(files) > 10 { // 限制批量上传数量
		return nil, fmt.Errorf("批量上传文件数量不能超过10个")
	}

	results := make([]*UploadResult, 0, len(files))
	var errors []string

	for i, fileData := range files {
		result, err := s.UploadFile(ctx, fileData.Reader, fileData.Filename, uploaderID, req)
		if err != nil {
			logger.Errorf("批量上传文件失败 (索引: %d, 文件名: %s): %v", i, fileData.Filename, err)
			errors = append(errors, fmt.Sprintf("文件 %s 上传失败: %v", fileData.Filename, err))
			continue
		}
		results = append(results, result)
	}

	if len(errors) > 0 && len(results) == 0 {
		return nil, fmt.Errorf("所有文件上传失败: %s", strings.Join(errors, "; "))
	}

	if len(errors) > 0 {
		logger.Warnf("批量上传部分文件失败: %s", strings.Join(errors, "; "))
	}

	return results, nil
}

// GetMediaList 获取媒体文件列表
func (s *UploadService) GetMediaList(ctx context.Context, req *MediaListRequest, userID *int) ([]*model.MediaFile, int64, error) {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return nil, 0, fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 构建过滤器
	filters := &repository.MediaFilters{
		FileType: req.FileType,
		Status:   req.Status,
		IsPublic: req.IsPublic,
		Keyword:  req.Keyword,
	}

	if userID != nil {
		filters.UploaderID = userID
	} else if req.UploaderID != nil {
		filters.UploaderID = req.UploaderID
	}

	// 获取媒体文件列表
	medias, total, err := s.mediaRepo.List(req.Page, req.Size, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("获取媒体文件列表失败: %w", err)
	}

	return medias, total, nil
}

// GetMediaByID 根据ID获取媒体文件
func (s *UploadService) GetMediaByID(ctx context.Context, id int, userID *int) (*model.MediaFile, error) {
	media, err := s.mediaRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("获取媒体文件失败: %w", err)
	}

	// 权限检查：如果不是公开文件且不是上传者本人，则拒绝访问
	if !media.IsPublic && (userID == nil || media.UploaderID == nil || *userID != *media.UploaderID) {
		return nil, fmt.Errorf("无权限访问该媒体文件")
	}

	return media, nil
}

// UpdateMedia 更新媒体文件信息
func (s *UploadService) UpdateMedia(ctx context.Context, id int, req *FileUploadRequest, userID *int) (*model.MediaFile, error) {
	// 获取媒体文件
	media, err := s.mediaRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("媒体文件不存在: %w", err)
	}

	// 权限检查
	if !media.CanEdit(nil) { // 这里需要传入用户对象
		return nil, fmt.Errorf("无权限编辑该媒体文件")
	}

	// 更新字段
	if req.Alt != "" {
		media.Alt = req.Alt
	}
	if req.Caption != "" {
		media.Caption = req.Caption
	}
	if req.IsPublic != nil {
		media.SetPublic(*req.IsPublic)
	}

	// 保存更新
	if err := s.mediaRepo.Update(media); err != nil {
		return nil, fmt.Errorf("更新媒体文件失败: %w", err)
	}

	logger.Infof("媒体文件更新成功 (ID: %d)", id)
	return media, nil
}

// DeleteMedia 删除媒体文件
func (s *UploadService) DeleteMedia(ctx context.Context, id int, userID *int) error {
	// 获取媒体文件
	media, err := s.mediaRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("媒体文件不存在: %w", err)
	}

	// 权限检查
	if !media.CanDelete(nil) { // 这里需要传入用户对象
		return fmt.Errorf("无权限删除该媒体文件")
	}

	// 检查是否有使用记录
	usages, err := s.usageRepo.GetByMediaID(id)
	if err != nil {
		logger.Errorf("获取媒体使用记录失败: %v", err)
	} else if len(usages) > 0 {
		// 如果有使用记录，执行软删除
		if err := s.mediaRepo.SoftDelete(id); err != nil {
			return fmt.Errorf("软删除媒体文件失败: %w", err)
		}
		logger.Infof("媒体文件软删除成功 (ID: %d)，因为存在使用记录", id)
		return nil
	}

	// 删除物理文件
	if err := os.Remove(media.StoragePath); err != nil {
		logger.Errorf("删除物理文件失败: %v", err)
		// 继续删除数据库记录
	}

	// 删除数据库记录
	if err := s.mediaRepo.Delete(id); err != nil {
		return fmt.Errorf("删除媒体文件记录失败: %w", err)
	}

	logger.Infof("媒体文件删除成功 (ID: %d)", id)
	return nil
}

// SearchMedia 搜索媒体文件
func (s *UploadService) SearchMedia(ctx context.Context, keyword string, req *MediaListRequest) ([]*model.MediaFile, int64, error) {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return nil, 0, fmt.Errorf("请求参数验证失败: %w", err)
	}

	medias, total, err := s.mediaRepo.Search(keyword, req.Page, req.Size)
	if err != nil {
		return nil, 0, fmt.Errorf("搜索媒体文件失败: %w", err)
	}

	return medias, total, nil
}

// GetMediaStats 获取媒体文件统计
func (s *UploadService) GetMediaStats(ctx context.Context, uploaderID *int) (*repository.MediaStats, error) {
	var stats *repository.MediaStats
	var err error

	if uploaderID != nil {
		stats, err = s.mediaRepo.GetUploaderStats(*uploaderID)
	} else {
		stats, err = s.mediaRepo.GetStats()
	}

	if err != nil {
		return nil, fmt.Errorf("获取媒体文件统计失败: %w", err)
	}

	return stats, nil
}

// detectMimeType 检测MIME类型
func (s *UploadService) detectMimeType(filename string, content []byte) string {
	// 首先通过文件扩展名检测
	mimeType := mime.TypeByExtension(filepath.Ext(filename))

	// 如果通过扩展名无法检测，尝试通过内容检测
	if mimeType == "" {
		mimeType = http.DetectContentType(content)
	}

	return mimeType
}

// getFileType 根据MIME类型获取文件类型
func (s *UploadService) getFileType(mimeType string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	}
	if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}
	if strings.HasPrefix(mimeType, "audio/") {
		return "video" // 音频归类为视频
	}
	if strings.Contains(mimeType, "document") || strings.Contains(mimeType, "pdf") ||
		strings.Contains(mimeType, "text") || strings.Contains(mimeType, "spreadsheet") ||
		strings.Contains(mimeType, "presentation") {
		return "document"
	}

	return "other"
}

// isAllowedFileType 检查是否为允许的文件类型
func (s *UploadService) isAllowedFileType(mimeType string) bool {
	allowedTypes := s.config.Upload.AllowedTypes
	for _, allowedType := range allowedTypes {
		if strings.HasPrefix(mimeType, allowedType) {
			return true
		}
	}
	return false
}

// calculateHash 计算文件哈希
func (s *UploadService) calculateHash(content []byte) string {
	hash := md5.Sum(content)
	return fmt.Sprintf("%x", hash)
}

// generateStoragePath 生成存储路径
func (s *UploadService) generateStoragePath(filename, hash string) string {
	// 按日期组织目录结构
	date := time.Now().Format("2006/01/02")

	// 使用哈希值的前两位作为子目录
	prefix := hash[:2]

	// 获取文件扩展名
	ext := filepath.Ext(filename)
	if ext == "" {
		// 如果没有扩展名，尝试从MIME类型推断
		// 这里简化处理
		ext = ".bin"
	}

	// 生成唯一文件名
	uniqueFilename := hash[2:] + ext

	return filepath.Join(s.config.Upload.StoragePath, date, prefix, uniqueFilename)
}

// saveFileToDisk 保存文件到磁盘
func (s *UploadService) saveFileToDisk(path string, content []byte) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// generateFileURL 生成文件访问URL
func (s *UploadService) generateFileURL(storagePath string) string {
	// 移除存储路径前缀，生成相对路径
	relativePath := strings.TrimPrefix(storagePath, s.config.Upload.StoragePath)
	relativePath = strings.TrimPrefix(relativePath, "/")

	return s.config.Upload.BaseURL + "/" + relativePath
}

// getImageDimensions 获取图片尺寸（简化实现）
func (s *UploadService) getImageDimensions(content []byte) (int, int, error) {
	// 这里应该使用图片处理库来获取尺寸
	// 为了简化，暂时返回默认值
	return 0, 0, fmt.Errorf("图片尺寸检测功能未实现")
}

// FileData 文件数据结构
type FileData struct {
	Reader   io.Reader
	Filename string
}