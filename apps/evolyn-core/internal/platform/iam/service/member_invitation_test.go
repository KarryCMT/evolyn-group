package service

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMemberInvitationWorkbook(t *testing.T) {
	var content bytes.Buffer
	writer := zip.NewWriter(&content)
	shared, err := writer.Create("xl/sharedStrings.xml")
	assert.NoError(t, err)
	_, err = shared.Write([]byte(`<?xml version="1.0"?><sst><si><t>姓名</t></si><si><t>编号</t></si><si><t>手机</t></si><si><t>邮箱</t></si><si><t>部门</t></si><si><t>李同学</t></si><si><t>LY-001</t></si><si><t>13800138000</t></si><si><t>li@example.com</t></si><si><t>研发部,产品部</t></si></sst>`))
	assert.NoError(t, err)
	sheet, err := writer.Create("xl/worksheets/sheet1.xml")
	assert.NoError(t, err)
	_, err = sheet.Write([]byte(`<?xml version="1.0"?><worksheet><sheetData><row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c><c r="C1" t="s"><v>2</v></c><c r="D1" t="s"><v>3</v></c><c r="E1" t="s"><v>4</v></c></row><row r="2"><c r="A2" t="s"><v>5</v></c><c r="B2" t="s"><v>6</v></c><c r="C2" t="s"><v>7</v></c><c r="D2" t="s"><v>8</v></c><c r="E2" t="s"><v>9</v></c></row></sheetData></worksheet>`))
	assert.NoError(t, err)
	assert.NoError(t, writer.Close())

	rows, failures, err := parseMemberInvitationWorkbook(content.Bytes())
	assert.NoError(t, err)
	assert.Empty(t, failures)
	if assert.Len(t, rows, 1) {
		assert.Equal(t, 2, rows[0].row)
		assert.Equal(t, "李同学", rows[0].request.Name)
		assert.Equal(t, "LY-001", rows[0].request.Identifier)
		assert.Equal(t, "13800138000", rows[0].request.Phone)
		assert.Equal(t, "li@example.com", rows[0].request.Email)
		assert.Equal(t, []string{"研发部", "产品部"}, rows[0].request.DepartmentNames)
	}
}

func TestValidateMemberInvitationRequest(t *testing.T) {
	assert.ErrorIs(t, validateMemberInvitationRequest(MemberInvitationRequest{Name: "成员"}), ErrMemberInvitationContactRequired)
	assert.ErrorIs(t, validateMemberInvitationRequest(MemberInvitationRequest{Name: "", Phone: "13800138000"}), ErrMemberInvitationInvalid)
	assert.NoError(t, validateMemberInvitationRequest(MemberInvitationRequest{Name: "成员", Email: "member@example.com"}))
}

func TestNormalizeExcelDate(t *testing.T) {
	assert.Equal(t, "2024-01-01", normalizeExcelDate("45292"))
	assert.Equal(t, "2024-01-01", normalizeExcelDate("2024-01-01"))
}
