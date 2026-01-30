package request

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// GetQueryInt returns an integer query parameter or the default value
func GetQueryInt(r *http.Request, key string, defaultVal int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}

	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}

	return intVal
}

// GetQueryIntWithRange returns an integer query parameter clamped to a range
func GetQueryIntWithRange(r *http.Request, key string, defaultVal, minVal, maxVal int) int {
	val := GetQueryInt(r, key, defaultVal)

	if val < minVal {
		return minVal
	}
	if val > maxVal {
		return maxVal
	}

	return val
}

// GetQueryString returns a string query parameter or the default value
func GetQueryString(r *http.Request, key string, defaultVal string) string {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// GetQueryTime parses a time query parameter (RFC3339 format)
func GetQueryTime(r *http.Request, key string) *time.Time {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil
	}

	t, err := time.Parse(time.RFC3339, val)
	if err != nil {
		// Try parsing date only format
		t, err = time.Parse("2006-01-02", val)
		if err != nil {
			return nil
		}
	}

	return &t
}

// GetURLParam returns a URL parameter from chi router
func GetURLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

// GetURLParamInt returns a URL parameter as an integer
func GetURLParamInt(r *http.Request, key string) (int64, error) {
	val := chi.URLParam(r, key)
	return strconv.ParseInt(val, 10, 64)
}

// GetQueryBool returns a boolean query parameter or the default value
func GetQueryBool(r *http.Request, key string, defaultVal bool) bool {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}

	boolVal, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}

	return boolVal
}
