package controller

import (
	"net/http"

	platformcontroller "evolyn/internal/platform/controller"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/iam/model"
	"evolyn/internal/platform/iam/service"

	"github.com/gin-gonic/gin"
)

// DepartmentController 部门管理（租户内组织架构，P3-2）
type DepartmentController struct {
	departmentService service.DepartmentService
}

func NewDepartmentController(departmentService service.DepartmentService) platformcontroller.Controller {
	return &DepartmentController{
		departmentService: departmentService,
	}
}

// @Summary 部门列表
// @Produce json
// @Tags 部门
// @Security JWT
// @Success 200 {object} httpx.Response{data=[]model.Department}
// @Router /api/v1/departments [get]
func (d *DepartmentController) List(c *gin.Context) {
	depts, err := d.departmentService.List(c.Request.Context())
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}
	httpx.ResponseSuccess(c, depts)
}

// @Summary 部门树
// @Produce json
// @Tags 部门
// @Security JWT
// @Success 200 {object} httpx.Response{data=[]service.DepartmentNode}
// @Router /api/v1/departments/tree [get]
func (d *DepartmentController) Tree(c *gin.Context) {
	tree, err := d.departmentService.Tree(c.Request.Context())
	if err != nil {
		httpx.ResponseFailed(c, http.StatusInternalServerError, err)
		return
	}
	httpx.ResponseSuccess(c, tree)
}

// @Summary 创建部门
// @Accept json
// @Produce json
// @Tags 部门
// @Security JWT
// @Param dept body model.Department true "department info"
// @Success 200 {object} httpx.Response{data=model.Department}
// @Router /api/v1/departments [post]
func (d *DepartmentController) Create(c *gin.Context) {
	dept := new(model.Department)
	if err := c.BindJSON(dept); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	dept, err := d.departmentService.Create(c.Request.Context(), dept)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, dept)
}

// @Summary 更新部门
// @Accept json
// @Produce json
// @Tags 部门
// @Security JWT
// @Param id path int true "department id"
// @Param dept body model.Department true "department info"
// @Success 200 {object} httpx.Response{data=model.Department}
// @Router /api/v1/departments/{id} [put]
func (d *DepartmentController) Update(c *gin.Context) {
	dept := new(model.Department)
	if err := c.BindJSON(dept); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}

	dept, err := d.departmentService.Update(c.Request.Context(), c.Param("id"), dept)
	if err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, dept)
}

// @Summary 删除部门
// @Produce json
// @Tags 部门
// @Security JWT
// @Param id path int true "department id"
// @Success 200 {object} httpx.Response
// @Router /api/v1/departments/{id} [delete]
func (d *DepartmentController) Delete(c *gin.Context) {
	if err := d.departmentService.Delete(c.Request.Context(), c.Param("id")); err != nil {
		httpx.ResponseFailed(c, http.StatusBadRequest, err)
		return
	}
	httpx.ResponseSuccess(c, nil)
}

func (d *DepartmentController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/departments", d.List)
	api.GET("/departments/tree", d.Tree)
	api.POST("/departments", d.Create)
	api.PUT("/departments/:id", d.Update)
	api.DELETE("/departments/:id", d.Delete)
}

func (d *DepartmentController) Name() string {
	return "Department"
}
