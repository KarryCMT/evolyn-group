package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary 首页
// @Produce html
// @Tags 首页
// @Router /index [get]
func Index(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(
		`<html>
	<head>
		<title>Evolyn Server</title>
	</head>
	<body>
		<h1>Hello Evolyn</h1>
		<ul>
			<li><a href="/swagger/index.html">swagger</a></li>
			<li><a href="/metrics">metrics</a></li>
			<li><a href="/healthz">healthz</a></li>
			<li><a href="/">api list</a></li>
	  	</ul>
		<hr>
		<center>evolyn-core/1.0</center>
	</body>
<html>`))
}
