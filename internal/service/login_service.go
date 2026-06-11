package service

import (
	"context"
	"encoding/base64"
	"sync"
	"time"

	"github.com/qzone-memory/internal/client/qzone"
	"github.com/qzone-memory/internal/dao"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/model"
)

type LoginSession struct {
	Client    *qzone.LoginClient
	CreatedAt time.Time
}

var (
	loginSession *LoginSession
	loginMu      sync.Mutex
)

func GenerateLoginQRCode() (map[string]string, error) {
	loginMu.Lock()
	defer loginMu.Unlock()

	client := qzone.NewLoginClient()
	png, err := client.GetQRCode()
	if err != nil {
		return nil, err
	}

	loginSession = &LoginSession{
		Client:    client,
		CreatedAt: time.Now(),
	}

	return map[string]string{
		"qr_image": "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
	}, nil
}

func PollLoginStatus(ctx context.Context) (*qzone.LoginStatus, error) {
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
		return nil, err
	}

	if status.Status == 3 || status.Status == 4 {
		loginSession = nil
	}

	if status.Status == 2 && status.LoginURL != "" {
		result, err := loginSession.Client.DoLogin(status.LoginURL)
		if err != nil {
			return nil, err
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
		if err := dao.UpsertUser(ctx, user); err != nil {
			return nil, err
		}

		status.QQ = result.QQ
		loginSession = nil
	}

	return status, nil
}

func GetCurrentUser(ctx context.Context, req dto.QueryByQQRequest) (*model.User, error) {
	if err := validateQQ(req.QQ); err != nil {
		return nil, err
	}
	return dao.GetUserByQQ(ctx, req.QQ)
}
