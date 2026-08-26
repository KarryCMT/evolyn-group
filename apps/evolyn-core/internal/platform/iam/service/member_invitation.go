package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"evolyn/internal/contextx"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/repository"

	"gorm.io/gorm"
)

const maxMemberInvitationBatchSize = 200

type memberInvitationService struct {
	invitations repository.MemberInvitationRepository
	departments repository.DepartmentRepository
	accounts    repository.AccountRepository
	users       UserService
	audit       auditservice.Recorder
}

func NewMemberInvitationService(
	invitations repository.MemberInvitationRepository,
	departments repository.DepartmentRepository,
	accounts repository.AccountRepository,
	users UserService,
	audit auditservice.Recorder,
) MemberInvitationService {
	return &memberInvitationService{invitations: invitations, departments: departments, accounts: accounts, users: users, audit: audit}
}

func (s *memberInvitationService) Create(ctx context.Context, inviterMemberID uint, req MemberInvitationRequest) (*model.MemberInvitation, error) {
	return s.create(ctx, inviterMemberID, req, model.MemberInvitationSourceManual)
}

func (s *memberInvitationService) create(ctx context.Context, inviterMemberID uint, req MemberInvitationRequest, source string) (*model.MemberInvitation, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	if err := validateMemberInvitationRequest(req); err != nil {
		return nil, err
	}
	profile, err := s.resolveProfileDepartments(ctx, req)
	if err != nil {
		return nil, err
	}
	token, err := newInvitationToken()
	if err != nil {
		return nil, err
	}
	invitation := &model.MemberInvitation{
		InviterMemberID: inviterMemberID,
		Name:            strings.TrimSpace(req.Name),
		Identifier:      strings.TrimSpace(req.Identifier),
		Phone:           strings.TrimSpace(req.Phone),
		Email:           strings.TrimSpace(req.Email),
		Profile:         profile,
		InviteToken:     token,
		Source:          source,
		Status:          model.MemberInvitationPending,
	}
	invitation.TenantID = tenantID
	created, err := s.invitations.Create(ctx, invitation)
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{Module: "iam", Action: "create", ResourceType: "member_invitation", ResourceID: fmt.Sprint(created.ID), After: map[string]any{"name": created.Name, "source": created.Source}})
	}
	return created, nil
}

func (s *memberInvitationService) Import(ctx context.Context, inviterMemberID uint, content []byte) (*model.MemberInvitationBatchResult, error) {
	requests, parseErrors, err := parseMemberInvitationWorkbook(content)
	if err != nil {
		return nil, err
	}
	result := &model.MemberInvitationBatchResult{FailedRows: parseErrors}
	if len(requests) == 0 {
		return result, nil
	}
	accounts, err := s.accounts.List(ctx)
	if err != nil {
		return nil, err
	}
	knownPhones, knownEmails := make(map[string]struct{}), make(map[string]struct{})
	for _, account := range accounts {
		if account.Phone != "" {
			knownPhones[account.Phone] = struct{}{}
		}
		if account.Email != "" {
			knownEmails[strings.ToLower(account.Email)] = struct{}{}
		}
	}
	seenPhones, seenEmails := make(map[string]struct{}), make(map[string]struct{})
	for _, row := range requests {
		if err := validateMemberInvitationRequest(row.request); err != nil {
			result.FailedRows = append(result.FailedRows, fmt.Sprintf("第 %d 行：%v", row.row, err))
			continue
		}
		phone, email := strings.TrimSpace(row.request.Phone), strings.ToLower(strings.TrimSpace(row.request.Email))
		if (phone != "" && hasKey(knownPhones, phone)) || (email != "" && hasKey(knownEmails, email)) ||
			(phone != "" && hasKey(seenPhones, phone)) || (email != "" && hasKey(seenEmails, email)) {
			result.FailedRows = append(result.FailedRows, fmt.Sprintf("第 %d 行：手机号或邮箱已存在", row.row))
			continue
		}
		if phone != "" {
			seenPhones[phone] = struct{}{}
		}
		if email != "" {
			seenEmails[email] = struct{}{}
		}
		if _, err := s.create(ctx, inviterMemberID, row.request, model.MemberInvitationSourceBatch); err != nil {
			result.FailedRows = append(result.FailedRows, fmt.Sprintf("第 %d 行：%v", row.row, err))
			continue
		}
		result.SuccessCount++
	}
	return result, nil
}

func hasKey(values map[string]struct{}, key string) bool {
	_, ok := values[key]
	return ok
}

func (s *memberInvitationService) GetPublicLink(ctx context.Context) (*model.TenantPublicInvitationLink, error) {
	link, err := s.invitations.GetPublicLink(ctx)
	if err == nil || !isNotFound(err) {
		return link, err
	}
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	token, err := newInvitationToken()
	if err != nil {
		return nil, err
	}
	link = &model.TenantPublicInvitationLink{Token: token, Enabled: false}
	link.TenantID = tenantID
	return s.invitations.CreatePublicLink(ctx, link)
}

func (s *memberInvitationService) UpdatePublicLink(ctx context.Context, inviterMemberID uint, enabled bool) (*model.TenantPublicInvitationLink, error) {
	link, err := s.GetPublicLink(ctx)
	if err != nil {
		return nil, err
	}
	link.Enabled = enabled
	link.CreatorMemberID = inviterMemberID
	updated, err := s.invitations.UpdatePublicLink(ctx, link)
	if err == nil && s.audit != nil {
		s.audit.Record(ctx, auditservice.Entry{Module: "iam", Action: "update", ResourceType: "tenant_public_invitation_link", ResourceID: fmt.Sprint(updated.ID), After: map[string]any{"enabled": enabled}})
	}
	return updated, err
}

// AcceptPublicLink 将已完成注册的账号加入公开链接所对应的租户。链接查找时
// 显式剥离调用链租户，随后重新注入目标租户再复用成员创建的配额与同租户校验。
func (s *memberInvitationService) AcceptPublicLink(ctx context.Context, accountID uint, nickname, token string) (*model.User, error) {
	if s.users == nil || accountID == 0 || strings.TrimSpace(token) == "" {
		return nil, ErrMemberInvitationInvalid
	}
	link, err := s.invitations.GetPublicLinkByToken(ctx, strings.TrimSpace(token))
	if err != nil {
		return nil, err
	}
	if !link.Enabled {
		return nil, ErrMemberInvitationInvalid
	}
	return s.users.AddMember(contextx.NewTenantContext(ctx, link.TenantID), &AddMemberRequest{AccountID: accountID, Nickname: strings.TrimSpace(nickname)})
}

func (s *memberInvitationService) resolveProfileDepartments(ctx context.Context, req MemberInvitationRequest) (model.MemberInviteProfile, error) {
	profile := model.MemberInviteProfile{DepartmentIDs: req.DepartmentIDs, DepartmentNames: req.DepartmentNames, Alias: strings.TrimSpace(req.Alias), EmployeeNo: strings.TrimSpace(req.EmployeeNo), Gender: strings.TrimSpace(req.Gender), Title: strings.TrimSpace(req.Title), EmploymentType: strings.TrimSpace(req.EmploymentType), HiredAt: strings.TrimSpace(req.HiredAt), WorkLocation: strings.TrimSpace(req.WorkLocation), Birthday: strings.TrimSpace(req.Birthday), Education: strings.TrimSpace(req.Education)}
	for _, departmentID := range profile.DepartmentIDs {
		if _, err := s.departments.GetByID(ctx, departmentID); err != nil {
			return profile, fmt.Errorf("部门 %d 不存在或无权限", departmentID)
		}
	}
	return profile, nil
}

func validateMemberInvitationRequest(req MemberInvitationRequest) error {
	if name := strings.TrimSpace(req.Name); name == "" || utf8.RuneCountInString(name) > 80 {
		return ErrMemberInvitationInvalid
	}
	if strings.TrimSpace(req.Phone) == "" && strings.TrimSpace(req.Email) == "" {
		return ErrMemberInvitationContactRequired
	}
	if email := strings.TrimSpace(req.Email); email != "" && (!strings.Contains(email, "@") || strings.HasPrefix(email, "@") || strings.HasSuffix(email, "@")) {
		return ErrMemberInvitationInvalid
	}
	for _, value := range []string{req.Identifier, req.Alias, req.EmployeeNo, req.Gender, req.Title, req.EmploymentType, req.WorkLocation, req.Education} {
		if utf8.RuneCountInString(strings.TrimSpace(value)) > 50 {
			return ErrMemberInvitationInvalid
		}
	}
	return nil
}

func newInvitationToken() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func isNotFound(err error) bool { return err == gorm.ErrRecordNotFound }

type batchInvitationRow struct {
	row     int
	request MemberInvitationRequest
}

type xlsxCell struct {
	Ref       string `xml:"r,attr"`
	Type      string `xml:"t,attr"`
	Value     string `xml:"v"`
	InlineStr string `xml:"is>t"`
}
type xlsxRow struct {
	Number int        `xml:"r,attr"`
	Cells  []xlsxCell `xml:"c"`
}
type xlsxSheet struct {
	Rows []xlsxRow `xml:"sheetData>row"`
}
type xlsxSharedStrings struct {
	Items []struct {
		Text string `xml:"t"`
		Runs []struct {
			Text string `xml:"t"`
		} `xml:"r"`
	} `xml:"si"`
}

// parseMemberInvitationWorkbook 只读取模板所需的首个工作表，不执行 Excel 公式，
// 从而让服务端避免依赖桌面端 Office 或额外二进制组件。
func parseMemberInvitationWorkbook(content []byte) ([]batchInvitationRow, []string, error) {
	if len(content) == 0 || len(content) > 5*1024*1024 {
		return nil, nil, ErrMemberInvitationImportFile
	}
	archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, nil, ErrMemberInvitationImportFile
	}
	files := make(map[string]*zip.File, len(archive.File))
	for _, file := range archive.File {
		files[file.Name] = file
	}
	shared := []string{}
	if file := files["xl/sharedStrings.xml"]; file != nil {
		var raw xlsxSharedStrings
		if err := decodeXLSXFile(file, &raw); err != nil {
			return nil, nil, ErrMemberInvitationImportFile
		}
		for _, item := range raw.Items {
			text := item.Text
			for _, run := range item.Runs {
				text += run.Text
			}
			shared = append(shared, text)
		}
	}
	var sheetFile *zip.File
	for name, file := range files {
		if strings.HasPrefix(name, "xl/worksheets/") && strings.HasSuffix(name, ".xml") {
			sheetFile = file
			break
		}
	}
	if sheetFile == nil {
		return nil, nil, ErrMemberInvitationImportFile
	}
	var sheet xlsxSheet
	if err := decodeXLSXFile(sheetFile, &sheet); err != nil {
		return nil, nil, ErrMemberInvitationImportFile
	}
	headerRow := -1
	headers := map[string]string{}
	for _, row := range sheet.Rows {
		current := readXLSXRow(row, shared)
		if current["A"] == "姓名" {
			headerRow = row.Number
			for column, value := range current {
				headers[value] = column
			}
			break
		}
	}
	if headerRow < 0 || headers["姓名"] == "" || (headers["手机"] == "" && headers["邮箱"] == "") {
		return nil, nil, ErrMemberInvitationImportFile
	}
	result := make([]batchInvitationRow, 0)
	failed := make([]string, 0)
	for _, row := range sheet.Rows {
		if row.Number <= headerRow {
			continue
		}
		values := readXLSXRow(row, shared)
		request := MemberInvitationRequest{Name: values[headers["姓名"]], Identifier: values[headers["编号"]], Phone: values[headers["手机"]], Email: values[headers["邮箱"]], Alias: values[headers["别名"]], EmployeeNo: values[headers["工号"]], Gender: values[headers["性别"]], Title: values[headers["职务"]], EmploymentType: values[headers["聘用形式"]], HiredAt: normalizeExcelDate(values[headers["入职日期"]]), WorkLocation: values[headers["工作地点"]], Birthday: normalizeExcelDate(values[headers["出生日期"]]), Education: values[headers["学历"]]}
		if departments := strings.TrimSpace(values[headers["部门"]]); departments != "" {
			request.DepartmentNames = strings.Split(departments, ",")
		}
		if isEmptyInvitationRequest(request) {
			continue
		}
		if len(result) >= maxMemberInvitationBatchSize {
			failed = append(failed, fmt.Sprintf("第 %d 行：一次最多导入 %d 名成员", row.Number, maxMemberInvitationBatchSize))
			continue
		}
		result = append(result, batchInvitationRow{row: row.Number, request: request})
	}
	return result, failed, nil
}

func decodeXLSXFile(file *zip.File, target interface{}) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	return xml.NewDecoder(reader).Decode(target)
}

func readXLSXRow(row xlsxRow, shared []string) map[string]string {
	values := make(map[string]string, len(row.Cells))
	for _, cell := range row.Cells {
		column := strings.TrimRightFunc(cell.Ref, func(r rune) bool { return r >= '0' && r <= '9' })
		value := cell.Value
		if cell.Type == "s" {
			var index int
			if _, err := fmt.Sscan(value, &index); err == nil && index >= 0 && index < len(shared) {
				value = shared[index]
			}
		}
		if cell.Type == "inlineStr" {
			value = cell.InlineStr
		}
		values[column] = strings.TrimSpace(value)
	}
	return values
}

func isEmptyInvitationRequest(req MemberInvitationRequest) bool {
	return req.Name == "" && req.Phone == "" && req.Email == ""
}

// normalizeExcelDate 兼容模板以 Excel 日期序列号保存日期的情况；文本日期保持
// 原值，便于导入端保留用户已经填写的 YYYY-MM-DD 格式。
func normalizeExcelDate(value string) string {
	value = strings.TrimSpace(value)
	serial, err := strconv.ParseFloat(value, 64)
	if err != nil || serial < 1 || serial > 100000 {
		return value
	}
	return time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, int(serial)).Format("2006-01-02")
}
