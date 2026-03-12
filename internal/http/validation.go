package http

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var (
	urlSuffixRe = regexp.MustCompile(`^/[^ ]*$`) // starts with /, no spaces
)

// RegisterCustomValidators wires custom rules into Gin's validator.
func RegisterCustomValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("urlsuffix", func(fl validator.FieldLevel) bool {
			// allow empty (omitempty); only check non-empty
			s, _ := fl.Field().Interface().(string)
			if s == "" {
				return true
			}
			return urlSuffixRe.MatchString(s)
		})
		_ = v.RegisterValidation("authmethod", func(fl validator.FieldLevel) bool {
			s, _ := fl.Field().Interface().(string)
			switch s {
			case "none", "basic", "bearer":
				return true
			default:
				return false
			}
		})
		_ = v.RegisterValidation("httpmethod", func(fl validator.FieldLevel) bool {
			s, _ := fl.Field().Interface().(string)
			switch s {
			case "GET", "POST", "PUT", "PATCH", "DELETE":
				return true
			default:
				return false
			}
		})
	}
}
