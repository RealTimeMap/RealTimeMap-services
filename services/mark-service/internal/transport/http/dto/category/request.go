package category

import "mime/multipart"

type RequestCategory struct {
	CategoryName string                `form:"category_name"`
	Color        string                `form:"color"`
	Icon         *multipart.FileHeader `form:"icon"`
}
