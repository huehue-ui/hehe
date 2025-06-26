package utils

// Error messages used across the application.
const (
	ErrMissingApp       = "missing app param"       // Error message for missing 'app' parameter.
	ErrMissingOS        = "missing os param"        // Error message for missing 'os' parameter.
	ErrMissingCountry   = "missing country param"   // Error message for missing 'country' parameter.
	ErrMethodNotAllowed = "method not allowed"    // Error message for using an unsupported HTTP method.
	InternalServerError = "internal server error" // Generic error message for server-side issues.
	DefaultApiPageLimit = 10                      // Default number of items per page for API responses.
)

// TargetingDimensions lists the dimensions available for campaign targeting.
// This is used in db.go, though the query logic there is currently hardcoded for these specific dimensions.
var TargetingDimensions = []string{"app_id", "country", "os"}
