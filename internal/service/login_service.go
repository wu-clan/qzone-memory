package service

import (
	"encoding/base64"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/common"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/model"
	"github.com/qzone-memory/pkg/response"
	"github.com/qzone-memory/qzone"
	"gorm.io/gorm"
)

type LoginSession struct {
	Client    *qzone.LoginClient
	CreatedAt time.Time
}

var (
	loginSession *LoginSession
	loginMu      sync.Mutex
)

func GenerateLoginQRCode() (map[string]string, *response.AppError) {
	loginMu.Lock()
	defer loginMu.Unlock()

	client := qzone.NewLoginClient()
	png, err := client.GetQRCode()
	if err != nil {
		return nil, &response.AppError{Code: http.StatusInternalServerError, Err: err}
	}

	loginSession = &LoginSession{
		Client:    client,
		CreatedAt: time.Now(),
	}

	return map[string]string{
		"qr_image": "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
	}, nil
}

func PollLoginStatus() (*qzone.LoginStatus, *response.AppError) {
	loginMu.Lock()
	defer loginMu.Unlock()

	if loginSession == nil {
		return &qzone.LoginStatus{
			Status:  3,
			Message: "二维码已过期",
		}, nil
	}
	if time.Since(loginSession.CreatedAt) > 5*time.Minute {
		loginSession = nil
		return &qzone.LoginStatus{
			Status:  3,
			Message: "二维码已过期",
		}, nil
	}

	status, err := loginSession.Client.PollStatus()
	if err != nil {
		return nil, &response.AppError{Code: http.StatusInternalServerError, Err: err}
	}

	if status.Status == 3 || status.Status == 4 {
		loginSession = nil
	}

	if status.Status == 2 && status.LoginURL != "" {
		result, err := loginSession.Client.DoLogin(status.LoginURL)
		if err != nil {
			return nil, &response.AppError{Code: http.StatusInternalServerError, Err: err}
		}

		user := &model.User{
			QQ:        result.QQ,
			Nickname:  status.Nickname,
			Cookie:    result.Cookie,
			GTK:       result.GTK,
			PSKey:     result.PSKey,
			LoginAt:   time.Now(),
			ExpiredAt: time.Now().Add(24 * time.Hour),
		}
		if err := dao.UpsertUser(user); err != nil {
			return nil, &response.AppError{Code: http.StatusInternalServerError, Err: err}
		}

		status.QQ = result.QQ
		loginSession = nil
	}

	return status, nil
}

func GetCurrentUser(c *gin.Context) (*model.User, *response.AppError) {
	var req dto.QueryByQQRequest
	if err := bindQuery(c, &req); err != nil {
		return nil, err
	}
	if err := validateQQ(req.QQ); err != nil {
		return nil, err
	}

	user, err := dao.GetUserByQQ(req.QQ)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &response.AppError{Code: http.StatusNotFound, Err: common.ErrNotFound}
		}
		return nil, &response.AppError{Code: http.StatusInternalServerError, Err: err}
	}

	return user, nil
}
