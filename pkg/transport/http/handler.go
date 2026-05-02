package http

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	nethttp "net/http"
)

const (
	healthy   = "healthy"
	unhealthy = "unhealthy"
)

type ServiceStatus struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func newServiceStatus(service, status string) ServiceStatus {
	return ServiceStatus{Status: status, Service: service}
}

func HealthHandler(serviceName string, db *gorm.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(nethttp.StatusServiceUnavailable, newServiceStatus(serviceName, unhealthy))
			return
		}
		c.JSON(nethttp.StatusOK, newServiceStatus(serviceName, healthy))
	}
}
