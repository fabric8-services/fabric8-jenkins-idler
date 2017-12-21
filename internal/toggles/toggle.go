package toggles

type Features interface {
	// IsIdlerEnabled returns true if the Jenkins idler is enabled for the user with the specified uid, false otherwise.
	IsIdlerEnabled(uid string) (bool, error)
}
