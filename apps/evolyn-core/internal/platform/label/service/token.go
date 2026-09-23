package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	enginelabel "evolyn/internal/engine/label"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/httpx"
	iammodel "evolyn/internal/platform/iam/model"
	labelapp "evolyn/internal/platform/label"
	"evolyn/internal/platform/label/model"

	"gorm.io/gorm"
)

func (s *templateService) attachQRURL(ctx context.Context, member *iammodel.User, appID, formID, templateID, recordID uint, system map[string]any) error {
	if s.tokens == nil || s.publicBaseURL == "" {
		return nil // 未启用扫码能力的部署保持已有 recordId 数据语义。
	}
	tokenValue, err := newQRToken()
	if err != nil {
		return err
	}
	tenantID, err := s.ensureMember(ctx, member)
	if err != nil {
		return err
	}
	token, err := s.tokens.Ensure(ctx, &model.QRToken{
		TenantID: tenantID, Token: tokenValue, AppID: appID, FormID: formID,
		RecordID: recordID, TemplateID: templateID, TargetType: model.QRTargetFormRecord,
		Enabled: true, CreatorMemberID: member.ID,
	})
	if err != nil {
		return err
	}
	if system == nil {
		return fmt.Errorf("label record system data is nil")
	}
	system["recordId"] = s.publicBaseURL + "/q/" + token.Token
	return nil
}

func (s *templateService) ResolveQRToken(ctx context.Context, member *iammodel.User, tokenValue string) (*model.QRTokenTarget, error) {
	if _, err := s.ensureMember(ctx, member); err != nil {
		return nil, labelapp.ErrQRTokenInvalid
	}
	tokenValue = strings.TrimSpace(tokenValue)
	if s.tokens == nil || s.apps == nil || !validQRToken(tokenValue) {
		return nil, labelapp.ErrQRTokenInvalid
	}
	token, err := s.tokens.GetByToken(ctx, tokenValue)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, labelapp.ErrQRTokenInvalid
	}
	if err != nil {
		return nil, err
	}
	if !token.Enabled || token.TargetType != model.QRTargetFormRecord || token.ExpireAt != nil && !time.Time(*token.ExpireAt).After(time.Now()) {
		return nil, labelapp.ErrQRTokenInvalid
	}
	// 表单域在这里重新执行数据范围和字段矩阵校验；任何不可见状态统一为
	// Token 无效，防止扫码者探测记录是否真实存在。
	if _, err := s.records.GetRecord(ctx, member, token.FormID, token.RecordID); err != nil {
		return nil, httpx.Wrap(labelapp.ErrQRTokenInvalid, err)
	}
	form, missing, err := s.forms.PublishedForm(ctx, token.FormID)
	if err != nil {
		return nil, err
	}
	if missing {
		return nil, labelapp.ErrQRTokenInvalid
	}
	appCode, err := s.apps.AppCodeByID(ctx, token.AppID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, labelapp.ErrQRTokenInvalid
		}
		return nil, err
	}
	if err := s.tokens.IncrementScan(ctx, token.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, labelapp.ErrQRTokenInvalid
		}
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{
			Module: "label", Action: "scan", ResourceType: "label-qr-token", ResourceID: strconv.FormatUint(uint64(token.ID), 10),
			AppID: token.AppID, After: map[string]any{"formId": token.FormID, "recordId": token.RecordID},
		})
	}
	return &model.QRTokenTarget{
		TargetType: token.TargetType, AppCode: appCode, FormCode: form.Code,
		RecordID: strconv.FormatUint(uint64(token.RecordID), 10),
	}, nil
}

func newQRToken() (string, error) {
	buffer := make([]byte, 24)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func validQRToken(value string) bool {
	if len(value) != 32 {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(decoded) == 24
}

func usesScanToken(schema *enginelabel.Schema) bool {
	if schema == nil {
		return false
	}
	for index := range schema.Elements {
		element := &schema.Elements[index]
		if element.Visible && element.Type == "qrcode" && element.Value != nil && element.Value.Type == "system" && element.Value.Key == "recordId" {
			return true
		}
	}
	return false
}
