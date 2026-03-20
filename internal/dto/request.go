package dto

// QueryByQQRequest 根据 QQ 查询请求
type QueryByQQRequest struct {
	QQ       string `form:"qq" binding:"required"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type QueryActivityRequest struct {
	QQ       string `form:"qq" binding:"required"`
	Type     string `form:"type"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type QueryFriendsRequest struct {
	QQ             string `form:"qq" binding:"required"`
	IncludeDeleted bool   `form:"include_deleted"`
	Page           int    `form:"page"`
	PageSize       int    `form:"page_size"`
}

type QueryMemoryRequest struct {
	QQ       string `form:"qq" binding:"required"`
	Type     string `form:"type"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// QueryByTargetRequest 按目标类型查询（评论/点赞/转发）
type QueryByTargetRequest struct {
	TargetType string `form:"target_type" binding:"required"`
	TargetID   string `form:"target_id" binding:"required"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

// QueryByAlbumRequest 按相册查询照片
type QueryByAlbumRequest struct {
	AlbumID  string `form:"album_id" binding:"required"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// QueryByTalkIDRequest 根据说说 ID 查询
type QueryByTalkIDRequest struct {
	TalkID string `form:"talk_id" binding:"required"`
}

// QueryByBlogIDRequest 根据日志 ID 查询
type QueryByBlogIDRequest struct {
	BlogID string `form:"blog_id" binding:"required"`
}

type QueryByFeedIDRequest struct {
	FeedID string `form:"feed_id" binding:"required"`
}

// QueryByAlbumIDRequest 根据相册 ID 查询
type QueryByAlbumIDRequest struct {
	AlbumID string `form:"album_id" binding:"required"`
}

// SyncRequest 触发同步请求
type SyncRequest struct {
	QQ string `json:"qq" binding:"required"`
}
