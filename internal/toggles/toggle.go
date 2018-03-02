package toggles

// Features is an interface which allows you to enable specific behaviour for a given user.
// In particular, it controls whether Jenkins idling is enabled for a specific user.
type Features interface {
	// IsIdlerEnabled returns true if the Jenkins idler is enabled for the user with the specified uid, false otherwise.
	IsIdlerEnabled(uid string) (bool, error)
}
