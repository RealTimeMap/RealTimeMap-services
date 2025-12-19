package mark

import (
	"mime/multipart"
	"time"
)

type RequestMark struct {
	MarkName       string                  `form:"markName" binding:"required"`
	AdditionalInfo *string                 `form:"additionalInfo" binding:"-"`
	CategoryId     int                     `form:"categoryId" binding:"required"`
	StartAt        time.Time               `form:"startAt"`
	Duration       int                     `form:"duration" binding:"required"`
	Longitude      float64                 `form:"longitude" binding:"required,longitude"`
	Latitude       float64                 `form:"latitude" binding:"required,latitude"`
	Photos         []*multipart.FileHeader `form:"photos"`
}

type RequestUpdateMark struct {
	MarkName       *string `form:"markName,omitempty" binding:"-"`
	AdditionalInfo *string `form:"additionalInfo,omitempty" binding:"-"`
	CategoryId     *int    `form:"categoryId,omitempty" binding:"-"`
	Duration       *int    `form:"duration,omitempty" binding:"-"`
	// Управление фотками
	PhotosToDelete []string                `form:"photosToDelete" binding:"-"`
	Photos         []*multipart.FileHeader `form:"photos" binding:"-"`
}
