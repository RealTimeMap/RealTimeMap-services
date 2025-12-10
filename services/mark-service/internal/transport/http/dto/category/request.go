package category

import "mime/multipart"

type RequestCategory struct {
	CategoryName string                `form:"category_name" binding:"required,max=64"`
	Color        string                `form:"color" binding:"required,max=7"`
	Icon         *multipart.FileHeader `form:"icon" binding:"required"`
}
