package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/qzone-memory/internal/dto"
	"github.com/qzone-memory/internal/service"
	"github.com/qzone-memory/pkg/response"
)

func GetMemoryTimeline(c *gin.Context) {
	var req dto.QueryMemoryRequest
	if !bindQuery(c, &req) {
		return
	}
	data, err := service.GetMemoryTimeline(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}

func GetOnThisDay(c *gin.Context) {
	var req dto.QueryOnThisDayRequest
	if !bindQuery(c, &req) {
		return
	}
	data, err := service.GetOnThisDay(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}

func SearchMemory(c *gin.Context) {
	var req dto.QueryMemorySearchRequest
	if !bindQuery(c, &req) {
		return
	}
	data, err := service.SearchMemory(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}

func SearchFriendInteractions(c *gin.Context) {
	var req dto.QueryFriendInteractionsRequest
	if !bindQuery(c, &req) {
		return
	}
	data, err := service.SearchFriendInteractions(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}

func GetMemoryInteractions(c *gin.Context) {
	var req dto.QueryMemoryInteractionsRequest
	if !bindQuery(c, &req) {
		return
	}
	data, err := service.GetMemoryInteractions(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}

func GetMemoryReport(c *gin.Context) {
	var req dto.QueryByQQRequest
	if !bindQuery(c, &req) {
		return
	}
	data, err := service.GetMemoryReport(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.Success(c, data)
}
