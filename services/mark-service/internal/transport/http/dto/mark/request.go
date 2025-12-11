package mark

import (
	"mime/multipart"
	"time"
)

type RequestMark struct {
	MarkName       string                  `form:"mark_name" binding:"required"`
	AdditionalInfo *string                 `form:"additional_info" binding:"-"`
	CategoryId     int                     `form:"category_id" binding:"required"`
	StartAt        time.Time               `form:"start_at"`
	Duration       int                     `form:"duration" binding:"required"`
	Longitude      float64                 `form:"longitude" binding:"required,longitude"`
	Latitude       float64                 `form:"latitude" binding:"required,latitude"`
	Photos         []*multipart.FileHeader `form:"photos"`
}
