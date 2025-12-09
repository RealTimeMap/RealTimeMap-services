package pagination

type Params struct {
	Page     int `json:"page" query:"page"`
	PageSize int `json:"page_size" query:"page_size"`
}

func (p *Params) ApplyDefaults(cfg Config) {
	if p.Page <= 0 {
		p.Page = 1
	}

	if p.PageSize <= 0 {
		p.PageSize = cfg.DefaultPageSize
	}
	if p.PageSize < cfg.MinPageSize {
		p.PageSize = cfg.MinPageSize
	}
	if p.PageSize > cfg.MaxPageSize {
		p.PageSize = cfg.MaxPageSize
	}
}

func (p *Params) Defaults() {
	p.ApplyDefaults(DefaultConfig)
}

// Validate проверяет корректность параметров с учетом конфигурации
func (p *Params) Validate(cfg Config) error {
	if p.Page <= 0 {
		return ErrInvalidPage
	}
	if p.PageSize <= 0 {
		return ErrInvalidPageSize
	}
	if p.PageSize < cfg.MinPageSize {
		return ErrPageSizeTooSmall
	}
	if p.PageSize > cfg.MaxPageSize {
		return ErrPageSizeTooLarge
	}
	return nil
}

func (p *Params) Offset() int {
	return p.PageSize * (p.Page - 1)
}

func (p *Params) Limit() int {
	return p.PageSize
}

// ForSql метод возвращает высчитанный limit и offset для формирования запросов
func (p *Params) ForSql() (offset, limit int) {
	return p.Offset(), p.Limit()
}
