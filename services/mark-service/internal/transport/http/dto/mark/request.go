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

type Coords struct {
	Longitude float64 `json:"lon" binding:"required,longitude"`
	Latitude  float64 `json:"lat" binding:"required,latitude"`
}

type Screen struct {
	LeftTop     Coords `json:"leftTop" binding:"required"`
	Center      Coords `json:"center" binding:"required"`
	RightBottom Coords `json:"rightBottom" binding:"required"`
}

type FilterParams struct {
	Screen  Screen    `json:"screen" binding:"required"`
	StartAt time.Time `json:"startAt" binding:"required"`
	EndAt   time.Time `json:"endAt" binding:"-"`
}
