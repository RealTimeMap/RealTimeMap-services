package attraction

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
)

// Лимиты совпадают с size у колонок
const (
	maxSlugLen     = 255
	maxNameLen     = 255
	maxCityLen     = 128
	maxCategoryLen = 128
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func validateCreate(p CreateAttractionParams) error {
	var errs []apperror.DomainError

	errs = appendSlug(errs, p.Slug)
	errs = appendString(errs, "name", p.Name, maxNameLen)
	errs = appendString(errs, "city", p.City, maxCityLen)
	errs = appendString(errs, "category", p.Category, maxCategoryLen)
	errs = appendGeom(errs, p.Geom)
	errs = append(errs, validateSlides(p.Slides)...)

	if len(errs) > 0 {
		return apperror.NewMultipleErrors(errs...)
	}
	return nil
}

// validateUpdate проверяет только переданные поля
func validateUpdate(p UpdateAttractionParams) error {
	var errs []apperror.DomainError

	if p.Slug != nil {
		errs = appendSlug(errs, *p.Slug)
	}
	if p.Name != nil {
		errs = appendString(errs, "name", *p.Name, maxNameLen)
	}
	if p.City != nil {
		errs = appendString(errs, "city", *p.City, maxCityLen)
	}
	if p.Category != nil {
		errs = appendString(errs, "category", *p.Category, maxCategoryLen)
	}
	if p.Status != nil && !Status(*p.Status).Valid() {
		errs = append(errs, ErrStatusInvalid(*p.Status))
	}
	if p.Geom != nil {
		errs = appendGeom(errs, *p.Geom)
	}
	if p.Slides != nil {
		errs = append(errs, validateSlides(*p.Slides)...)
	}

	if len(errs) > 0 {
		return apperror.NewMultipleErrors(errs...)
	}
	return nil
}

func appendSlug(errs []apperror.DomainError, slug string) []apperror.DomainError {
	errs = appendString(errs, "slug", slug, maxSlugLen)
	if slug != "" && !slugRegex.MatchString(slug) {
		errs = append(errs, ErrSlugInvalid(slug))
	}
	return errs
}

func appendGeom(errs []apperror.DomainError, geom types.Point) []apperror.DomainError {
	if lon, lat := geom.Lon(), geom.Lat(); lon < -180 || lon > 180 || lat < -90 || lat > 90 {
		return append(errs, ErrGeomInvalid(lon, lat))
	}
	return errs
}

func validateSlides(slides []CreateSlideParams) []apperror.DomainError {
	var errs []apperror.DomainError

	positions := make(map[uint]struct{}, len(slides))
	for i, sp := range slides {
		field := fmt.Sprintf("slides[%d]", i)

		if _, dup := positions[sp.Position]; dup {
			errs = append(errs, ErrPositionDuplicate(field+".position", sp.Position))
		}
		positions[sp.Position] = struct{}{}

		if sp.Duration == 0 {
			errs = append(errs, ErrDurationInvalid(field+".duration"))
		}

		errs = append(errs, validateBlocks(field, sp.Blocks)...)
	}
	return errs
}

func validateBlocks(slideField string, blocks []CreateSlideBlockParams) []apperror.DomainError {
	var errs []apperror.DomainError

	positions := make(map[uint]struct{}, len(blocks))
	for i, bp := range blocks {
		field := fmt.Sprintf("%s.blocks[%d]", slideField, i)

		if _, dup := positions[bp.Position]; dup {
			errs = append(errs, ErrPositionDuplicate(field+".position", bp.Position))
		}
		positions[bp.Position] = struct{}{}

		if !BlockType(bp.BlockType).Valid() {
			errs = append(errs, ErrBlockTypeInvalid(field+".blockType", bp.BlockType))
		}
		if len(bp.Content) == 0 {
			errs = append(errs, ErrRequired(field+".content"))
		}
	}
	return errs
}

func appendString(errs []apperror.DomainError, field, value string, max int) []apperror.DomainError {
	switch {
	case strings.TrimSpace(value) == "":
		return append(errs, ErrRequired(field))
	case len([]rune(value)) > max:
		return append(errs, ErrTooLong(field, max, value))
	}
	return errs
}
