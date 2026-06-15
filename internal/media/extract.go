package media

import (
	"context"
	"encoding/json"

	"github.com/qzone-memory/config"
	"github.com/qzone-memory/database"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/model"
)

// jsonURLs 解析以 JSON 数组存储的 URL 列表（如 talk.Images / activity.Images）。
func jsonURLs(raw string) []string {
	if raw == "" {
		return nil
	}
	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return nil
	}
	return arr
}

// buildAssets 归一化 + 过滤 + 批内去重，构造待下载记录。
func buildAssets(userQQ, category string, urls []string, seen map[string]struct{}) []*model.MediaAsset {
	assets := make([]*model.MediaAsset, 0, len(urls))
	for _, raw := range urls {
		if !shouldDownload(raw) {
			continue
		}
		h := hashURL(raw)
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		assets = append(assets, &model.MediaAsset{
			UserQQ:    userQQ,
			URLHash:   h,
			SourceURL: normalizeURL(raw),
			Category:  category,
			Status:    "pending",
		})
	}
	return assets
}

// EnqueueAll 扫描某 QQ 现有的全部记录，登记所有值得本地化的媒体为待下载。
// 用于同步完成后的批量入队、以及老用户的存量回填。幂等——BatchUpsert 按
// (user_qq, url_hash) 去重，不覆盖已下载结果。返回本次登记的资源数。
func EnqueueAll(ctx context.Context, userQQ string) (int, error) {
	seen := make(map[string]struct{})
	var assets []*model.MediaAsset
	add := func(category string, urls []string) {
		assets = append(assets, buildAssets(userQQ, category, urls, seen)...)
	}

	db := database.DB.WithContext(ctx)

	var photos []model.Photo
	db.Where("user_qq = ?", userQQ).Find(&photos)
	for _, p := range photos {
		add("photo", []string{p.URL})
		add("thumb", []string{p.ThumbURL})
	}

	var albums []model.Album
	db.Where("user_qq = ?", userQQ).Find(&albums)
	for _, a := range albums {
		add("cover", []string{a.CoverURL})
	}

	var talks []model.Talk
	db.Where("user_qq = ?", userQQ).Find(&talks)
	for _, t := range talks {
		add("talk_image", jsonURLs(t.Images))
	}

	var activities []model.Activity
	db.Where("user_qq = ?", userQQ).Find(&activities)
	for _, a := range activities {
		add("activity_image", jsonURLs(a.Images))
	}

	var favorites []model.Favorite
	db.Where("user_qq = ?", userQQ).Find(&favorites)
	for _, f := range favorites {
		add("favorite_image", jsonURLs(f.Images))
	}

	var videos []model.Video
	db.Where("user_qq = ?", userQQ).Find(&videos)
	downloadVideos := config.GlobalConfig != nil && config.GlobalConfig.Media.DownloadVideos
	for _, v := range videos {
		add("video_cover", []string{v.PreviewURL})
		if downloadVideos {
			add("video", []string{v.URL})
		}
	}

	// 头像：访客 / 好友 / 评论 / 点赞 / 留言（头像也是档案的一部分，保留）
	var visitors []model.Visitor
	db.Where("user_qq = ?", userQQ).Find(&visitors)
	for _, x := range visitors {
		add("avatar", []string{x.Avatar})
	}
	var friends []model.Friend
	db.Where("user_qq = ?", userQQ).Find(&friends)
	for _, x := range friends {
		add("avatar", []string{x.Avatar})
	}
	var comments []model.Comment
	db.Where("user_qq = ?", userQQ).Find(&comments)
	for _, x := range comments {
		add("avatar", []string{x.AuthorAvatar})
	}
	var likes []model.Like
	db.Where("user_qq = ?", userQQ).Find(&likes)
	for _, x := range likes {
		add("avatar", []string{x.LikerAvatar})
	}
	var messages []model.Message
	db.Where("user_qq = ?", userQQ).Find(&messages)
	for _, x := range messages {
		add("avatar", []string{x.AuthorAvatar})
	}

	// 分批 upsert，避免单条 SQL 绑定变量过多。
	const chunk = 200
	for i := 0; i < len(assets); i += chunk {
		end := i + chunk
		if end > len(assets) {
			end = len(assets)
		}
		if err := dao.BatchUpsertMediaAssets(assets[i:end]); err != nil {
			return 0, err
		}
	}
	return len(assets), nil
}
