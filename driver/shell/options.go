package shell

// options holds optional configuration for shell drivers.
type options struct {
	scriptMode bool
}

// Option configures a shell driver.
type Option func(*options)

// WithScriptMode disables extra shell quoting of macro values.
// Use this when the command was written by the user as a literal bash expression
// (via run: "..." or run: | multiline), so the user's own quotes are respected.
func WithScriptMode() Option {
	return func(o *options) { o.scriptMode = true }
}
