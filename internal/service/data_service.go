package service

import (
	"context"

	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/media"
	"github.com/qzone-memory/internal/model"
	"github.com/qzone-memory/pkg/logger"
	"go.uber.org/zap"
)

// userScopedModels 所有按 user_qq 归属的表，用于「彻底删除我的数据」。
func userScopedModels() []interface{} {
	return []interface{}{
		&model.Activity{}, &model.FriendGroup{}, &model.Friend{}, &model.Visitor{},
		&model.Video{}, &model.Favorite{}, &model.Diary{}, &model.Talk{}, &model.Blog{},
		&model.Album{}, &model.Photo{}, &model.Message{}, &model.Comment{},
		&model.Like{}, &model.Share{}, &model.Mention{},
	}
}

// DeleteAllData 彻底删除某 QQ 的全部数据：库中各表 + 同步状态 + 媒体登记与文件 + 登录态。
func DeleteAllData(ctx context.Context, qq string, confirm bool) error {
	if err := validateQQ(qq); err != nil {
		return err
	}
	if !confirm {
		return common.ErrInvalidParam
	}

	for _, m := range userScopedModels() {
		if err := database.DB.WithContext(ctx).Where("user_qq = ?", qq).Delete(m).Error; err != nil {
			return err
		}
	}
	if err := dao.DeleteMediaAssetsByQQ(ctx, qq); err != nil {
		return err
	}
	if err := dao.DeleteSyncStatesByQQ(ctx, qq); err != nil {
		return err
	}
	if err := dao.DeleteSyncLogsByQQ(ctx, qq); err != nil {
		return err
	}
	// 登录态：用户表用 qq 列
	if err := database.DB.WithContext(ctx).Where("qq = ?", qq).Delete(&model.User{}).Error; err != nil {
		return err
	}
	// 本地媒体文件（失败不致命，记录即可）
	if err := media.RemoveUserMedia(qq); err != nil {
		logger.Warn("删除本地媒体目录失败", zap.String("qq", qq), zap.Error(err))
	}
	return nil
}
