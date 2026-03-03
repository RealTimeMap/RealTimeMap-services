package validator

import (
	"docgen/internal/models"
	"fmt"
	"strings"
)

var validHTTPMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
var validParamIn = []string{"path", "query", "header"}
var validDirections = []string{"server-to-client", "client-to-server"}

// ValidateAPI валидирует структуру api.yaml.
// Возвращает слайс ошибок — пустой, если всё корректно.
func ValidateAPI(api models.API) []error {
	var errs []error

	if api.HTTP == nil && api.SocketIO == nil && api.GRPC == nil {
		errs = append(errs, fmt.Errorf("api.yaml пустой: должен содержать хотя бы один раздел (http, socketio, grpc)"))
		return errs
	}

	if api.HTTP != nil {
		errs = append(errs, validateHTTP(api.HTTP)...)
	}

	if api.SocketIO != nil {
		errs = append(errs, validateSocketIO(api.SocketIO)...)
	}

	if api.GRPC != nil {
		errs = append(errs, validateGRPC(api.GRPC)...)
	}

	return errs
}

func validateHTTP(http *models.HTTPSpec) []error {
	var errs []error

	for gi, group := range http.Groups {
		groupPath := fmt.Sprintf("http.groups[%d]", gi)
		for ei, ep := range group.Endpoints {
			epPath := fmt.Sprintf("%s.endpoints[%d]", groupPath, ei)

			if ep.Path == "" {
				errs = append(errs, fmt.Errorf("%s: path обязателен", epPath))
			}

			if ep.Method == "" {
				errs = append(errs, fmt.Errorf("%s: method обязателен", epPath))
			} else if !isOneOf(strings.ToUpper(ep.Method), validHTTPMethods) {
				errs = append(errs, fmt.Errorf(
					"%s: недопустимый метод %q (допустимые: %s)",
					epPath, ep.Method, strings.Join(validHTTPMethods, ", "),
				))
			}

			for pi, param := range ep.Parameters {
				paramPath := fmt.Sprintf("%s.parameters[%d]", epPath, pi)

				if param.Name == "" {
					errs = append(errs, fmt.Errorf("%s: name обязателен", paramPath))
				}

				if param.In == "" {
					errs = append(errs, fmt.Errorf("%s: in обязателен", paramPath))
				} else if !isOneOf(param.In, validParamIn) {
					errs = append(errs, fmt.Errorf(
						"%s: недопустимое значение in=%q (допустимые: %s)",
						paramPath, param.In, strings.Join(validParamIn, ", "),
					))
				}
			}
		}
	}

	return errs
}

func validateSocketIO(sio *models.SocketIOSpec) []error {
	var errs []error

	for i, event := range sio.Events {
		eventPath := fmt.Sprintf("socketio.events[%d]", i)

		if event.Name == "" {
			errs = append(errs, fmt.Errorf("%s: name обязателен", eventPath))
		}

		if event.Direction == "" {
			errs = append(errs, fmt.Errorf("%s: direction обязателен", eventPath))
		} else if !isOneOf(event.Direction, validDirections) {
			errs = append(errs, fmt.Errorf(
				"%s: недопустимое direction=%q (допустимые: %s)",
				eventPath, event.Direction, strings.Join(validDirections, ", "),
			))
		}
	}

	return errs
}

func validateGRPC(grpc *models.GRPCSpec) []error {
	var errs []error

	for si, svc := range grpc.Services {
		svcPath := fmt.Sprintf("grpc.services[%d]", si)
		for mi, method := range svc.Methods {
			methodPath := fmt.Sprintf("%s.methods[%d]", svcPath, mi)
			if method.Name == "" {
				errs = append(errs, fmt.Errorf("%s: name обязателен", methodPath))
			}
		}
	}

	return errs
}

func isOneOf(val string, allowed []string) bool {
	for _, a := range allowed {
		if val == a {
			return true
		}
	}
	return false
}
