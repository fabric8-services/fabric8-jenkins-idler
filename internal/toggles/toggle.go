package toggles

// Features : A feature toggle is a technique that attempts to provide an alternative to maintaining
// multiple source-code branches (known as feature branches), such that a feature can be tested even
// before it is completed and ready for release. Feature toggle is used to hide, enable or disable the
// feature during run time. For example, during the development process, a developer can enable the
// feature for testing and disable it for other users
// Continuous release and continuous deployment provide developers with rapid feedback about their coding.
// This requires the integration of their code changes as early as possible.
// Feature branches introduce a bypass to this process.
type Features interface {
	// IsIdlerEnabled returns true if the Jenkins idler is enabled for the user with the specified uid, false otherwise.
	IsIdlerEnabled(uid string) (bool, error)
}
