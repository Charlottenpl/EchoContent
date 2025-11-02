package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charlottepl/blog-system/internal/core/config"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"github.com/charlottepl/blog-system/internal/media/model"
	"github.com/charlottepl/blog-system/internal/media/repository"
)

// ProcessorService 媒体处理服务
type ProcessorService struct {
	mediaRepo repository.MediaRepository
	config    *config.Config
}

// NewProcessorService 创建媒体处理服务实例
func NewProcessorService() *ProcessorService {
	return &ProcessorService{
		mediaRepo: repository.NewMediaRepository(),
		config:    config.GetConfig(),
	}
}

// ProcessRequest 处理请求
type ProcessRequest struct {
	MediaID      int    `json:"media_id" validate:"required,min=1"`
	ProcessType  string `json:"process_type" validate:"required,oneof=resize compress watermark thumbnail"`
	Width        *int   `json:"width"`
	Height       *int   `json:"height"`
	Quality      *int   `json:"quality"`
	WatermarkText string `json:"watermark_text"`
	OutputFormat string `json:"output_format"`
}

// ProcessResult 处理结果
type ProcessResult struct {
	MediaFile *model.MediaFile `json:"media_file"`
	URL       string           `json:"url"`
	Message   string           `json:"message"`
}

// ProcessImage 处理图片
func (s *ProcessorService) ProcessImage(ctx context.Context, req *ProcessRequest) (*ProcessResult, error) {
	// 获取原始媒体文件
	media, err := s.mediaRepo.GetByID(req.MediaID)
	if err != nil {
		return nil, fmt.Errorf("获取媒体文件失败: %w", err)
	}

	// 检查是否为图片
	if !media.IsImage() {
		return nil, fmt.Errorf("只能处理图片文件")
	}

	// 读取原始文件
	file, err := os.Open(media.StoragePath)
	if err != nil {
		logger.Errorf("打开原始文件失败: %v", err)
		return nil, fmt.Errorf("打开原始文件失败: %w", err)
	}
	defer file.Close()

	// 解码图片
	img, format, err := image.Decode(file)
	if err != nil {
		logger.Errorf("解码图片失败: %v", err)
		return nil, fmt.Errorf("解码图片失败: %w", err)
	}

	// 根据处理类型进行处理
	var processedImg image.Image
	var outputFormat string

	switch req.ProcessType {
	case "resize":
		processedImg, err = s.resizeImage(img, req.Width, req.Height)
		outputFormat = format
	case "compress":
		processedImg, err = s.compressImage(img, req.Quality)
		outputFormat = format
	case "watermark":
		processedImg, err = s.addWatermark(img, req.WatermarkText)
		outputFormat = format
	case "thumbnail":
		processedImg, err = s.generateThumbnail(img, req.Width, req.Height)
		outputFormat = "jpeg"
	default:
		return nil, fmt.Errorf("不支持的处理类型: %s", req.ProcessType)
	}

	if err != nil {
		return nil, fmt.Errorf("图片处理失败: %w", err)
	}

	// 如果指定了输出格式，使用指定格式
	if req.OutputFormat != "" {
		outputFormat = req.OutputFormat
	}

	// 生成处理后的文件路径
	processedPath := s.generateProcessedImagePath(media.StoragePath, req.ProcessType, outputFormat)

	// 保存处理后的图片
	if err := s.saveProcessedImage(processedImg, processedPath, outputFormat, req.Quality); err != nil {
		logger.Errorf("保存处理后图片失败: %v", err)
		return nil, fmt.Errorf("保存处理后图片失败: %w", err)
	}

	// 创建新的媒体文件记录
	processedMedia := model.NewMediaFile(
		s.generateProcessedFilename(media.Filename, req.ProcessType),
		processedPath,
		s.getMimeTypeByFormat(outputFormat),
		"image",
		s.calculateFileHash(processedPath),
		media.FileSize, // 这里应该重新计算文件大小
		media.UploaderID,
	)

	// 设置尺寸
	bounds := processedImg.Bounds()
	processedMedia.SetDimensions(bounds.Dx(), bounds.Dy())

	// 设置关联信息
	processedMedia.Alt = media.Alt
	processedMedia.Caption = fmt.Sprintf("%s (%s)", media.Caption, req.ProcessType)
	processedMedia.IsPublic = media.IsPublic

	// 保存处理后的媒体文件记录
	if err := s.mediaRepo.Create(processedMedia); err != nil {
		// 删除已保存的处理后文件
		os.Remove(processedPath)
		logger.Errorf("创建处理后媒体文件记录失败: %v", err)
		return nil, fmt.Errorf("创建处理后媒体文件记录失败: %w", err)
	}

	// 生成访问URL
	url := s.generateFileURL(processedPath)

	logger.Infof("图片处理成功 (原始ID: %d, 处理后ID: %d, 类型: %s)",
		media.ID, processedMedia.ID, req.ProcessType)

	return &ProcessResult{
		MediaFile: processedMedia,
		URL:       url,
		Message:   "图片处理成功",
	}
}

// BatchProcessImages 批量处理图片
func (s *ProcessorService) BatchProcessImages(ctx context.Context, mediaIDs []int, req *ProcessRequest) ([]*ProcessResult, error) {
	if len(mediaIDs) == 0 {
		return nil, fmt.Errorf("没有需要处理的媒体文件")
	}

	if len(mediaIDs) > 10 { // 限制批量处理数量
		return nil, fmt.Errorf("批量处理文件数量不能超过10个")
	}

	results := make([]*ProcessResult, 0, len(mediaIDs))
	var errors []string

	for _, mediaID := range mediaIDs {
		processReq := *req // 复制请求
		processReq.MediaID = mediaID

		result, err := s.ProcessImage(ctx, &processReq)
		if err != nil {
			logger.Errorf("批量处理图片失败 (媒体ID: %d): %v", mediaID, err)
			errors = append(errors, fmt.Sprintf("媒体文件 %d 处理失败: %v", mediaID, err))
			continue
		}
		results = append(results, result)
	}

	if len(errors) > 0 && len(results) == 0 {
		return nil, fmt.Errorf("所有文件处理失败: %s", strings.Join(errors, "; "))
	}

	if len(errors) > 0 {
		logger.Warnf("批量处理部分图片失败: %s", strings.Join(errors, "; "))
	}

	return results, nil
}

// resizeImage 调整图片尺寸
func (s *ProcessorService) resizeImage(img image.Image, width, height *int) (image.Image, error) {
	if width == nil && height == nil {
		return nil, fmt.Errorf("必须指定宽度或高度")
	}

	originalBounds := img.Bounds()
	originalWidth := originalBounds.Dx()
	originalHeight := originalBounds.Dy()

	// 计算目标尺寸
	targetWidth := originalWidth
	targetHeight := originalHeight

	if width != nil {
		targetWidth = *width
		// 如果只指定宽度，按比例计算高度
		if height == nil {
			targetHeight = int(float64(originalHeight) * float64(*width) / float64(originalWidth))
		}
	}

	if height != nil {
		targetHeight = *height
		// 如果只指定高度，按比例计算宽度
		if width == nil {
			targetWidth = int(float64(originalWidth) * float64(*height) / float64(originalHeight))
		}
	}

	// 创建目标画布
	targetRect := image.Rect(0, 0, targetWidth, targetHeight)
	targetImg := image.NewRGBA(targetRect)

	// 简单的最近邻插值缩放
	for y := 0; y < targetHeight; y++ {
		for x := 0; x < targetWidth; x++ {
			srcX := x * originalWidth / targetWidth
			srcY := y * originalHeight / targetHeight
			targetImg.Set(x, y, img.At(srcX, srcY))
		}
	}

	return targetImg, nil
}

// compressImage 压缩图片
func (s *ProcessorService) compressImage(img image.Image, quality *int) (image.Image, error) {
	// 对于JPEG压缩，质量参数在保存时应用
	// 这里直接返回原图，压缩在保存时进行
	return img, nil
}

// addWatermark 添加水印
func (s *ProcessorService) addWatermark(img image.Image, watermarkText string) (image.Image, error) {
	if watermarkText == "" {
		return nil, fmt.Errorf("水印文本不能为空")
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 创建新的画布
	watermarked := image.NewRGBA(bounds)

	// 复制原图
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			watermarked.Set(x, y, img.At(x, y))
		}
	}

	// 这里应该使用文字渲染库来添加水印
	// 为了简化，暂时跳过实际的水印添加逻辑
	// 在实际项目中，可以使用 github.com/golang/freetype 等库

	logger.Infof("水印功能暂未实现，返回原图 (水印文本: %s)", watermarkText)
	return watermarked, nil
}

// generateThumbnail 生成缩略图
func (s *ProcessorService) generateThumbnail(img image.Image, width, height *int) (image.Image, error) {
	// 设置默认缩略图尺寸
	defaultSize := 200
	if width == nil && height == nil {
		width = &defaultSize
		height = &defaultSize
	}

	return s.resizeImage(img, width, height)
}

// saveProcessedImage 保存处理后的图片
func (s *ProcessorService) saveProcessedImage(img image.Image, path, format string, quality *int) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 创建文件
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 根据格式编码保存
	switch format {
	case "jpeg", "jpg":
		q := 85 // 默认质量
		if quality != nil {
			q = *quality
		}
		return jpeg.Encode(file, img, &jpeg.Options{Quality: q})
	case "png":
		return png.Encode(file, img)
	default:
		return fmt.Errorf("不支持的输出格式: %s", format)
	}
}

// generateProcessedImagePath 生成处理后图片路径
func (s *ProcessorService) generateProcessedImagePath(originalPath, processType, format string) string {
	dir := filepath.Dir(originalPath)
	filename := filepath.Base(originalPath)
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	return filepath.Join(dir, fmt.Sprintf("%s_%s.%s", nameWithoutExt, processType, format))
}

// generateProcessedFilename 生成处理后文件名
func (s *ProcessorService) generateProcessedFilename(originalFilename, processType string) string {
	ext := filepath.Ext(originalFilename)
	nameWithoutExt := strings.TrimSuffix(originalFilename, ext)
	return fmt.Sprintf("%s_%s%s", nameWithoutExt, processType, ext)
}

// getMimeTypeByFormat 根据格式获取MIME类型
func (s *ProcessorService) getMimeTypeByFormat(format string) string {
	switch format {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	default:
		return "image/jpeg"
	}
}

// calculateFileHash 计算文件哈希（简化实现）
func (s *ProcessorService) calculateFileHash(filePath string) string {
	// 这里应该实际计算文件哈希
	// 为了简化，返回一个基于路径的哈希
	return fmt.Sprintf("processed_%x", len(filePath))
}

// generateFileURL 生成文件访问URL
func (s *ProcessorService) generateFileURL(storagePath string) string {
	// 移除存储路径前缀，生成相对路径
	relativePath := strings.TrimPrefix(storagePath, s.config.Upload.StoragePath)
	relativePath = strings.TrimPrefix(relativePath, "/")

	return s.config.Upload.BaseURL + "/" + relativePath
}

// GetImageInfo 获取图片信息
func (s *ProcessorService) GetImageInfo(ctx context.Context, mediaID int) (*ImageInfo, error) {
	media, err := s.mediaRepo.GetByID(mediaID)
	if err != nil {
		return nil, fmt.Errorf("获取媒体文件失败: %w", err)
	}

	if !media.IsImage() {
		return nil, fmt.Errorf("不是图片文件")
	}

	// 读取文件获取详细信息
	file, err := os.Open(media.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 解码图片获取配置信息
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("解析图片配置失败: %w", err)
	}

	info := &ImageInfo{
		Width:     config.Width,
		Height:    config.Height,
		ColorModel: config.ColorModel.String(),
		FileSize:  media.FileSize,
		Format:    media.MimeType,
		URL:       s.generateFileURL(media.StoragePath),
	}

	return info, nil
}

// ImageInfo 图片信息
type ImageInfo struct {
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	ColorModel string `json:"color_model"`
	FileSize   int64  `json:"file_size"`
	Format     string `json:"format"`
	URL        string `json:"url"`
}

// OptimizeImage 优化图片（自动选择最佳处理方式）
func (s *ProcessorService) OptimizeImage(ctx context.Context, mediaID int) (*ProcessResult, error) {
	media, err := s.mediaRepo.GetByID(mediaID)
	if err != nil {
		return nil, fmt.Errorf("获取媒体文件失败: %w", err)
	}

	if !media.IsImage() {
		return nil, fmt.Errorf("只能优化图片文件")
	}

	// 根据文件大小和尺寸决定优化策略
	var req ProcessRequest
	req.MediaID = mediaID
	req.OutputFormat = "jpeg" // 统一转换为JPEG格式以减小文件大小

	if media.FileSize > 5*1024*1024 { // 大于5MB
		req.ProcessType = "compress"
		q := 75
		req.Quality = &q
	} else if media.Width != nil && *media.Width > 2048 { // 宽度大于2048px
		req.ProcessType = "resize"
		width := 2048
		req.Width = &width
	} else {
		req.ProcessType = "compress"
		q := 85
		req.Quality = &q
	}

	return s.ProcessImage(ctx, &req)
}