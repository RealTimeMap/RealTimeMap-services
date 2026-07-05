package category

type RequestCategory struct {
	CategoryName string `json:"categoryName" binding:"required,max=64"`
	Color        string `json:"color" binding:"required,max=7"`
	Icon         string `json:"icon" binding:"required,max=128"`
}
